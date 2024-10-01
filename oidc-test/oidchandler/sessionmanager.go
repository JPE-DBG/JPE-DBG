package oidchandler

import (
	"encoding/gob"
	"github.com/alexedwards/scs/v2"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

func initSessionManager(maxAge int) (*scs.SessionManager, error) {

	store := *scs.New()
	store.Lifetime = time.Duration(maxAge) * time.Second
	store.Cookie.Name = sessionCookie
	store.Cookie.HttpOnly = true
	store.Cookie.Secure = false
	store.Cookie.SameSite = http.SameSiteNoneMode

	// Register the types for session store
	gob.Register(&oauth2.Token{})
	gob.Register(&time.Time{})

	return &store, nil
}
