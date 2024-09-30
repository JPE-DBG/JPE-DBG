package main

import (
	"net/http"
)

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, err := authHandler.Session(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Delete the session by setting MaxAge to -1
	session.Options.MaxAge = -1
	if err = session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   stateCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// logout from the identity provider
	http.Redirect(w, r, logOutUrl, http.StatusFound)
}
