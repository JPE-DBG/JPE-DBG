package main

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	serverHost        = getEnv("HOST", "localhost")
	serverPort        = getEnv("PORT", "8080")
	clientID          = getEnv("CLIENT_ID", "")
	clientSecret      = getEnv("CLIENT_SECRET", "")
	redirectURL       = getEnv("REDIRECT_URL", fmt.Sprintf("http://%s:%s/callback", serverHost, serverPort))
	providerURL       = getEnv("PROVIDER_URL", "")
	sessionStorePath  = getEnv("SESSION_STORE_PATH", "")
	sessionStoreSize  = getEnv("SESSION_STORE_SIZE", defaultSessionStoreSize)
	caCertPath        = getEnv("CA_CERT_PATH", "")
	logOutUrl         = getEnv("LOGOUT_URL", "")
	sessionMaxAge     = getEnv("SESSION_MAX_AGE_SEC", defaultSessionMaxAgeSeconds)
	cleanupInterval   = getEnv("CLEANUP_INTERVAL_SEC", defaultCleanupSeconds)
	userCheckInterval = getEnv("USER_CHECK_INTERVAL_SEC", defaultUserCheckIntervalSeconds)
)
var (
	oauth2Config *oauth2.Config
	oidcProvider *oidc.Provider
	sessionStore *sessions.FilesystemStore
	baseContext  context.Context
	customClient *http.Client
)

func main() {
	var (
		err  error
		stop context.CancelFunc
	)

	baseContext, stop = signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	customTransport := initTLSTransport(err) //will be needed later in handleCallback, so don't remove this line!
	customClient = &http.Client{
		Transport: customTransport,
	}

	oidcProvider, err = oidc.NewProvider(oidc.ClientContext(baseContext, customClient), providerURL)
	if err != nil {
		log.Fatalf("Failed to get provider: %v", err)
	}

	oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"}, // Switch to oidc.ScopeSeti for IAM testing!
	}

	sessionStore = sessions.NewFilesystemStore(sessionStorePath, []byte(generateRandomSecret()))
	sessionStore.MaxLength(sessionStoreSize)
	sessionStore.Options.MaxAge = sessionMaxAge
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = true
	gob.Register(&oauth2.Token{})
	gob.Register(&time.Time{})

	go cleanupStaleSessions(baseContext, sessionStorePath, time.Duration(cleanupInterval)*time.Second)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", serverHost, serverPort),
		Handler: initRoutes(),
	}

	go func() {
		log.Printf("Server started at http://%s:%s", serverHost, serverPort)
		if errServe := server.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", errServe)
		}
	}()

	<-baseContext.Done()

	log.Println("Shutting down server...")
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = server.Shutdown(shutdownContext); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}
	log.Println("Server stopped")
}
