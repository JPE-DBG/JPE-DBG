package oidchandler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type OidcHandler struct {
	oidcProvider      *oidc.Provider
	oauth2Config      *oauth2.Config
	sessionStore      *sessions.FilesystemStore
	sessionStorePath  string
	tlsClient         *http.Client
	userCheckInterval int
}

type Config struct {
	ProviderURL       string
	ClientID          string
	ClientSecret      string
	SessionStorePath  string
	CaCertPath        string
	RedirectURL       string
	Scopes            []string
	SessionStoreSize  int
	SessionMaxAge     int
	CleanupInterval   int
	UserCheckInterval int
}

// NewOidcHandler initializes a new OidcHandler with the provided context and configuration.
// It sets up the OIDC provider, OAuth2 configuration, session store, and TLS-enabled client.
// Returns the initialized OidcHandler or an error if any initialization step fails.
func NewOidcHandler(ctx context.Context, config Config) (*OidcHandler, error) {
	var (
		handler OidcHandler
		err     error
	)
	handler.userCheckInterval = config.UserCheckInterval
	handler.sessionStorePath = config.SessionStorePath

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

	// initialize the session store
	handler.sessionStore, err = initSessionStore(config.SessionStorePath, config.SessionStoreSize, config.SessionMaxAge)
	if err != nil {
		return nil, err
	}

	return &handler, nil
}

// PrepareSession initializes a new session and sets cookies for state and session ID.
// It returns the OAuth2 authorization URL with the state parameter.
func (h *OidcHandler) PrepareSession(w http.ResponseWriter) string {
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes) // ignore error as there will only be an issue if the source cannot be read, but we are using a byte slice, so no issue.
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// TODO: move to session and save
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   sessionStateMaxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	})

	sessionID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	})

	return h.oauth2Config.AuthCodeURL(state)
}

// Session retrieves the session associated with the request.
// It looks for the session ID in the request cookies and fetches the session from the session store.
// If the session is new, it initializes the session timestamp.
// Returns the session or an error if the session cannot be retrieved.
func (h *OidcHandler) Session(r *http.Request) (*sessions.Session, error) {
	sessionID, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, errors.New("session ID cookie not found")
	}
	session, err := h.sessionStore.Get(r, sessionID.Value)
	if err != nil {
		return nil, err
	}
	if session.IsNew {
		session.Values[sessionTSKey] = time.Now()
	}
	return session, nil
}

// SaveSession saves the provided session with the given values.
// It iterates over the values map and sets each key-value pair in the session.
// Finally, it saves the session using the session store.
// Returns an error if the session cannot be saved.
func (h *OidcHandler) SaveSession(w http.ResponseWriter, r *http.Request, session *sessions.Session, values map[string]interface{}) error {
	for k, v := range values {
		session.Values[k] = v
	}
	return h.sessionStore.Save(r, w, session)
}

// Groups retrieves the groups claim from the user information.
// It takes a context and an OAuth2 token as parameters.
// Returns a slice of group names or an error if the groups claim is not found or cannot be retrieved.
func (h *OidcHandler) Groups(ctx context.Context, token *oauth2.Token) ([]string, error) {
	userInfo, err := h.UserInfo(ctx, token)
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
	for _, v := range groups.([]interface{}) {
		groupsList = append(groupsList, v.(string))
	}
	return groupsList, nil
}

// UserInfo retrieves the user information from the OIDC provider using the provided OAuth2 token.
// It uses the TLS-enabled client for secure communication.
// Returns the user information or an error if the retrieval fails.
func (h *OidcHandler) UserInfo(ctx context.Context, token *oauth2.Token) (*oidc.UserInfo, error) {
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, h.tlsClient)
	client := h.oauth2Config.Client(clientCtx, token)
	reqCtx := context.WithValue(clientCtx, oauth2.HTTPClient, client)
	return h.oidcProvider.UserInfo(reqCtx, oauth2.StaticTokenSource(token))
}

// UpdateUserInfo updates the user information in the session.
// It retrieves the session, checks for the token, fetches the user's groups,
// and saves the updated session information.
// Returns the updated session or an error if the update fails.
func (h *OidcHandler) UpdateUserInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) (*sessions.Session, error) {
	session, err := h.Session(r)
	if err != nil {
		return nil, err
	}

	token := h.Token(session)
	if token == nil {
		return nil, errors.New("no token info")
	}
	groups, err := h.Groups(ctx, token)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		sessionGroupKey: groups,
		sessionTSKey:    time.Now(),
	}
	err = h.SaveSession(w, r, session, values)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Token retrieves the OAuth2 token from the session.
// It returns the token if found, otherwise returns nil.
func (h *OidcHandler) Token(s *sessions.Session) *oauth2.Token {
	token, ok := s.Values[sessionTokenKey].(*oauth2.Token)
	if !ok {
		return nil
	}
	return token
}

// NeedUserInfoUpdate checks if the user information needs to be updated.
// It compares the current time with the session timestamp and the provided check interval.
// Returns true if the user information needs to be updated, otherwise false.
func (h *OidcHandler) NeedUserInfoUpdate(session *sessions.Session, checkInterval int) bool {
	ts, ok := session.Values[sessionTSKey].(time.Time)
	if !ok {
		return true
	}
	duration := time.Duration(checkInterval) * time.Second
	return time.Since(ts) > duration
}

// ExchangeToken exchanges the authorization code for an OAuth2 token.
// It validates the state parameter to prevent CSRF attacks and uses the TLS-enabled client for the token exchange.
// Returns the OAuth2 token or an error if the exchange fails.
func (h *OidcHandler) ExchangeToken(ctx context.Context, r *http.Request) (*oauth2.Token, error) {
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

// CleanupStaleSessions periodically cleans up stale sessions from the session store.
// It uses a ticker to trigger the cleanup process at the specified interval.
// The cleanup process stops when the context is done.
func (h *OidcHandler) CleanupStaleSessions(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupSessionStore()
		case <-ctx.Done():
			return
		}
	}
}

// cleanupSessionStore walks through the session store directory and removes stale sessions.
// It logs any errors encountered during the cleanup process.
func (h *OidcHandler) cleanupSessionStore() {
	err := filepath.WalkDir(h.sessionStorePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			h.removeStaleSession(path, d)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error cleaning up stale sessions: %v", err)
	}
}

// removeStaleSession removes a stale session file if it is older than the session's MaxAge.
// It logs any errors encountered during the removal process.
func (h *OidcHandler) removeStaleSession(path string, d fs.DirEntry) {
	info, err := d.Info()
	if err != nil {
		log.Printf("Error getting file info for %s: %v", path, err)
		return
	}
	if time.Since(info.ModTime()) > time.Duration(h.sessionStore.Options.MaxAge)*time.Second {
		if err := os.Remove(path); err != nil {
			log.Printf("Error removing file %s: %v", path, err)
		}
	}
}
