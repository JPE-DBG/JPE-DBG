package oidchandler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"time"
)

func initSessionStore(path string, maxSize, maxAge int) (*sessions.FilesystemStore, error) {
	storeKey, err := generateRandomSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random secret: %w", err)
	}
	store := *sessions.NewFilesystemStore(path, []byte(storeKey))
	store.MaxLength(maxSize)
	store.Options.MaxAge = maxAge
	store.Options.HttpOnly = true
	store.Options.Secure = false

	// Register the types for session store
	gob.Register(&oauth2.Token{})
	gob.Register(&time.Time{})

	return &store, nil
}

// generateRandomSecret generates a random secret key for session store.
func generateRandomSecret() (string, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(secret), nil
}
