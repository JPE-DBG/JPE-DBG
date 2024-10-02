package oidchandler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"net/http"
)

type OidcHandler struct {
	oidcProvider      *oidc.Provider
	oauth2Config      *oauth2.Config
	sessionManager    *scs.SessionManager
	tlsClient         *http.Client
	userCheckInterval int
	loginURL          string
}

type Config struct {
	ProviderURL   string
	ClientID      string
	ClientSecret  string
	CaCertPath    string
	RedirectURL   string
	Scopes        []string
	SessionMaxAge int
	LoginURL      string
}

// NewOidcHandler initializes a new OidcHandler with the given configuration.
// It sets up the TLS client, OIDC provider, OAuth2 configuration, and session manager.
// Returns a pointer to the OidcHandler or an error if any initialization step fails.
func NewOidcHandler(ctx context.Context, config Config) (*OidcHandler, error) {
	var (
		handler OidcHandler
		err     error
	)

	handler.loginURL = config.LoginURL

	// initialize the tls enabled client
	handler.tlsClient, err = initTLSTransport(config.CaCertPath)
	if err != nil {
		return nil, err
	}

	// initialize the oidc provider
	handler.oidcProvider, err = oidc.NewProvider(oidc.ClientContext(ctx, handler.tlsClient), config.ProviderURL)
	if err != nil {
		return nil, err
	}

	// initialize the oauth2 config
	handler.oauth2Config = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     handler.oidcProvider.Endpoint(),
		Scopes:       config.Scopes,
	}

	// initialize the session manager
	handler.sessionManager, err = initSessionManager(config.SessionMaxAge)
	if err != nil {
		return nil, err
	}

	return &handler, nil
}

// SessionManagerMiddleware returns a middleware that loads and saves session data.
func (h *OidcHandler) SessionManagerMiddleware() func(next http.Handler) http.Handler {
	return h.sessionManager.LoadAndSave
}

// LoginDone checks if the login indicator exists in the session.
// Returns true if the login indicator is found, otherwise false.
func (h *OidcHandler) LoginDone(ctx context.Context) bool {
	return h.sessionManager.Exists(ctx, loginIndicator)
}

// saveToSession saves the provided key-value pairs to the session.
// It iterates over the map and stores each key-value pair in the session.
func (h *OidcHandler) saveToSession(ctx context.Context, values map[string]any) {
	for k, v := range values {
		h.sessionManager.Put(ctx, k, v)
	}
}

// DestroySession destroys the current session.
// It removes all session data associated with the context.
// Returns an error if the session destruction fails.
func (h *OidcHandler) DestroySession(ctx context.Context) error {
	return h.sessionManager.Destroy(ctx)
}

// Token retrieves the OAuth2 token from the session.
// It returns the token if found, otherwise returns nil.
func (h *OidcHandler) Token(ctx context.Context) *oauth2.Token {
	token, ok := h.sessionManager.Get(ctx, sessionTokenKey).(*oauth2.Token)
	if !ok {
		return nil
	}
	return token
}

// PrepareAuthCodeUrl generates a URL for the OAuth2 authorization code flow.
// It creates a random state, encodes it in base64, and sets it as a cookie.
// Returns the authorization code URL.
func (h *OidcHandler) PrepareAuthCodeUrl(w http.ResponseWriter) string {
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes) // ignore error as there will only be an issue if the source cannot be read, but we are using a byte slice, so no issue.
	state := base64.URLEncoding.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   sessionStateMaxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	})

	return h.oauth2Config.AuthCodeURL(state)
}

// clearStateCookie clears the state cookie by setting its value to an empty string and its MaxAge to -1.
func (h *OidcHandler) clearStateCookie(w http.ResponseWriter) {
	// clear the state cookie, it's not needed anymore
	http.SetCookie(w, &http.Cookie{
		Name:   stateCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// isGroupMember checks if the user is a member of the group.
func isGroupMember(groups []string, group string) bool {
	for _, g := range groups {
		if g == group {
			return true
		}
	}
	return false
}

// exchangeToken exchanges the authorization code for an OAuth2 token.
// It validates the state parameter to prevent CSRF attacks and uses the TLS-enabled client for secure communication.
// Returns the OAuth2 token or an error if the exchange fails.
func (h *OidcHandler) exchangeToken(ctx context.Context, r *http.Request) (*oauth2.Token, error) {
	state, err := r.Cookie(stateCookie)
	if err != nil {
		return nil, errors.New("state cookie not found")
	}

	if r.URL.Query().Get("state") != state.Value {
		return nil, errors.New("state is not valid")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, errors.New("code not found")
	}

	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, h.tlsClient)
	token, err := h.oauth2Config.Exchange(exchangeCtx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	return token, nil
}

// groups retrieves the user's groups from the OIDC provider using the provided OAuth2 token.
// It calls the UserInfo method to get the user information and extracts the groups from the claims.
// Returns a slice of group names or an error if the retrieval or extraction fails.
func (h *OidcHandler) groups(ctx context.Context, token *oauth2.Token) ([]string, error) {
	userInfo, err := h.userInfo(ctx, token)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	groupsList := make([]string, 0)
	err = userInfo.Claims(&m)
	if err != nil {
		return groupsList, err
	}
	groups, ok := m[groupsKey]
	if !ok {
		return nil, errors.New("groups claim not found")
	}
	groupSlice, ok := groups.([]interface{})
	if !ok {
		return nil, errors.New("groups claim is not a slice")
	}
	for _, v := range groupSlice {
		groupsList = append(groupsList, v.(string))
	}
	return groupsList, nil
}

// userInfo retrieves the user information from the OIDC provider using the provided OAuth2 token.
// It uses the TLS-enabled client for secure communication.
// Returns the user information or an error if the retrieval fails.
func (h *OidcHandler) userInfo(ctx context.Context, token *oauth2.Token) (*oidc.UserInfo, error) {
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, h.tlsClient)
	client := h.oauth2Config.Client(clientCtx, token)
	reqCtx := context.WithValue(clientCtx, oauth2.HTTPClient, client)
	return h.oidcProvider.UserInfo(reqCtx, oauth2.StaticTokenSource(token))
}

// UserInfoFromSession retrieves the user information from the session.
// It gets the OAuth2 token from the session and uses it to fetch the user info from the OIDC provider.
// Returns the user information or an error if the token is not found or the retrieval fails.
func (h *OidcHandler) UserInfoFromSession(ctx context.Context) (*oidc.UserInfo, error) {
	token := h.Token(ctx)
	if token == nil {
		return nil, errors.New("no token info")
	}
	return h.userInfo(ctx, token)
}

func (h *OidcHandler) HasReadAccess(ctx context.Context) bool {
	return h.sessionManager.GetBool(ctx, readAccess)
}
