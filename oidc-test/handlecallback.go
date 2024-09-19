package main

import (
	"context"
	"golang.org/x/oauth2"
	"net/http"
)

func handleCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	tokenExchangeCtx := context.WithValue(baseContext, oauth2.HTTPClient, customClient)
	token, err := oauth2Config.Exchange(tokenExchangeCtx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := getSession(r)
	if err != nil {
		http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	groups, err := getGroups(token)
	if err != nil {
		http.Error(w, "Failed to get groups: "+err.Error(), http.StatusInternalServerError)
		return
	}

	values := map[string]any{
		sessionGroupKey: groups,
		sessionTokenKey: *token,
	}
	err = saveSession(w, r, session, values)
	if err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/menu", http.StatusSeeOther)
}
