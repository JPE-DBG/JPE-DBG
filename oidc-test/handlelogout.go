package main

import (
	"log"
	"net/http"
)

func handleLogout(w http.ResponseWriter, r *http.Request) {
	err := authHandler.DestroySession(r.Context())
	if err != nil {
		log.Printf("Failed to destroy session: %v", err)
	}

	// TODO: Implement the logout URL for the identity provider
	// logout from the identity provider
	//http.Redirect(w, r, logOutUrl, http.StatusFound)
}
