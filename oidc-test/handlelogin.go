package main

import (
	"net/http"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, authHandler.PrepareSession(w), http.StatusFound)
}
