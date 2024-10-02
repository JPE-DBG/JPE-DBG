package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"log"
	"net/http"
	"oidc1/oidchandler"
	"os"
	"os/signal"
)

var (
	serverHost    = getEnv("HOST", "localhost")
	serverPort    = getEnv("PORT", "8080")
	clientID      = getEnv("CLIENT_ID", "")
	clientSecret  = getEnv("CLIENT_SECRET", "")
	redirectURL   = getEnv("REDIRECT_URL", fmt.Sprintf("http://%s:%s/callback", serverHost, serverPort))
	providerURL   = getEnv("PROVIDER_URL", "")
	caCertPath    = getEnv("CA_CERT_PATH", "")
	loginURL      = getEnv("LOGIN_URL", defaultLoginURL)
	logOutUrl     = getEnv("LOGOUT_URL", "")
	sessionMaxAge = getEnv("SESSION_MAX_AGE_SEC", defaultSessionMaxAgeSeconds)
	authHandler   *oidchandler.OidcHandler
)
var (
	baseContext context.Context
)

func main() {
	var (
		err  error
		stop context.CancelFunc
	)

	baseContext, stop = signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	authHandler, err = oidchandler.NewOidcHandler(baseContext, oidchandler.Config{
		ProviderURL:  providerURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CaCertPath:   caCertPath,
		RedirectURL:  redirectURL,
		//Scopes:            []string{oidc.ScopeOpenID, "profile", "email", oidcScopeSeti},
		Scopes:        []string{oidc.ScopeOpenID, oidcScopeSeti},
		SessionMaxAge: sessionMaxAge,
		LoginURL:      loginURL,
	})

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
