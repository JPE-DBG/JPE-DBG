package main

import (
	"net/http"
)

func handleCallback(w http.ResponseWriter, r *http.Request) {
	token, err := authHandler.ExchangeToken(baseContext, r)
	if err != nil {
		http.Error(w, "failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	groups, err := authHandler.Groups(baseContext, token)
	if err != nil {
		http.Error(w, "failed to get groups: "+err.Error(), http.StatusInternalServerError)
		return
	}

	values := map[string]any{
		sessionGroupKey: groups,
		sessionTokenKey: *token,
	}

	session, err := authHandler.Session(r)
	err = authHandler.SaveSession(w, r, session, values)
	if err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/menu", http.StatusSeeOther)
}
