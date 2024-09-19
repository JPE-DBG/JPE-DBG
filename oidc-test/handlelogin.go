package main

import (
	"github.com/google/uuid"
	"net/http"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	sessionID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   sessionStateMaxAge,
		HttpOnly: true,
		Secure:   true,
	})
	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
}
