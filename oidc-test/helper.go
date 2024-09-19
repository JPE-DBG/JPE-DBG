package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is present in the environment, the value (which may be empty) is returned.
// Otherwise, the defaultValue is returned. The function supports int and string types.
// For any other type, the function will log an info message and return the defaultValue.
func getEnv[T any](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	var result T
	switch any(result).(type) {
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Failed to convert %s to int: %v", key, err)
		}
		return any(v).(T)
	case string:
		return any(value).(T)
	default:
		log.Printf("Unsupported type for environment variable %s, using default", key)
		return defaultValue
	}
	return result
}

// generateState generates a random state string.
func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // ignore error as there will only be an issue if the source cannot be read, but we are using a byte slice, so no issue.
	return base64.URLEncoding.EncodeToString(b)
}

// generateRandomSecret generates a random secret key for session store.
func generateRandomSecret() string {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		log.Fatalf("Failed to generate random secret: %v", err)
	}
	return base64.StdEncoding.EncodeToString(secret)
}

func getGroups(token *oauth2.Token) ([]string, error) {
	userInfo, err := getUserInfo(token)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	groupsList := make([]string, 0)
	err = userInfo.Claims(&m)
	if err != nil {
		return groupsList, err
	}
	groups, ok := m["groups"]
	if !ok {
		return nil, errors.New("groups claim not found")
	}
	for _, v := range groups.([]any) {
		groupsList = append(groupsList, v.(string))
	}
	return groupsList, nil
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

// getSession retrieves the session from the request using the session ID cookie.
func getSession(r *http.Request) (*sessions.Session, error) {
	sessionID, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, errors.New("session ID cookie not found")
	}

	session, err := sessionStore.Get(r, sessionID.Value)
	if err != nil {
		return nil, errors.New("Failed to get session: " + err.Error())
	}
	if session.IsNew {
		session.Values[sessionTSKey] = time.Now()
	}
	return session, nil
}

// saveSession saves the session with the provided values.
func saveSession(w http.ResponseWriter, r *http.Request, session *sessions.Session, values map[string]any) error {
	for k, v := range values {
		session.Values[k] = v
	}
	return session.Save(r, w)
}

// getToken returns the token from the session.
func getToken(session *sessions.Session) *oauth2.Token {
	token, ok := session.Values[sessionTokenKey].(*oauth2.Token)
	if !ok {
		return nil
	}
	return token
}

// func needUserInfoUpdate checks if the user info needs to be updated.
// This is done by checking if the session "ts" key is older than the specified duration.
func needUserInfoUpdate(session *sessions.Session, checkInterval int) bool {
	ts, ok := session.Values[sessionTSKey].(*time.Time)
	if !ok {
		return true
	}
	duration := time.Duration(checkInterval) * time.Second
	return time.Since(*ts) > duration
}

// getUserInfo retrieves the user info from the provider.
func getUserInfo(token *oauth2.Token) (*oidc.UserInfo, error) {
	// get a context with the TLS enabled client
	ctx := context.WithValue(baseContext, oauth2.HTTPClient, customClient)
	// create an auto-refreshing client using the TLS enabled base client
	client := oauth2Config.Client(ctx, token)
	// create a context with the client
	reqCtx := context.WithValue(ctx, oauth2.HTTPClient, client)
	// get the user info from the provider
	return oidcProvider.UserInfo(reqCtx, oauth2.StaticTokenSource(token))
}

// updateUserInfo updates the user info in the session.
// User info is fetched from id provider and stored in the session.
func updateUserInfo(w http.ResponseWriter, r *http.Request, session *sessions.Session, token *oauth2.Token) (*sessions.Session, error) {
	groups, err := getGroups(token)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		sessionGroupKey: groups,
		sessionTSKey:    time.Now(),
	}
	err = saveSession(w, r, session, values)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func formatUserInfo(i *oidc.UserInfo) string {
	var m map[string]any
	err := i.Claims(&m)
	if err != nil {
		return err.Error()
	}
	var sb strings.Builder

	// Find the maximum key length
	maxKeyLen := 0
	for k := range m {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	// Format the output with aligned colons
	for k, v := range m {
		sb.WriteString(k)
		sb.WriteString(strings.Repeat(" ", maxKeyLen-len(k)))
		sb.WriteString(": ")
		if str, ok := v.(string); ok {
			sb.WriteString(str)
		} else {
			sb.WriteString(fmt.Sprintf("%v", v))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// initTLSTransport initializes a custom HTTP transport with TLS configuration.
// It loads the CA certificate from the specified path and sets up the TLS configuration
// to use the CA certificate pool. The function returns the configured HTTP transport.
func initTLSTransport(err error) *http.Transport {
	// load CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            caCertPool,
	}
	customTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return customTransport
}

// cleanupStaleSessions removes expired session files from the session store.
func cleanupStaleSessions(ctx context.Context, storePath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupSessions(storePath)
		case <-ctx.Done():
			return
		}
	}
}

func cleanupSessions(storePath string) {
	err := filepath.WalkDir(storePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			removeStaleSession(path, d)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error cleaning up stale sessions: %v", err)
	}
}

func removeStaleSession(path string, d fs.DirEntry) {
	info, err := d.Info()
	if err != nil {
		log.Printf("Error getting file info for %s: %v", path, err)
		return
	}
	if time.Since(info.ModTime()) > time.Duration(sessionStore.Options.MaxAge)*time.Second {
		if err := os.Remove(path); err != nil {
			log.Printf("Error removing file %s: %v", path, err)
		}
	}
}
