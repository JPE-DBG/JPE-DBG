package main

import "github.com/gorilla/mux"

func initRoutes() *mux.Router {
	r := mux.NewRouter()

	// non-protected routes
	r.HandleFunc("/", handleMain)
	r.HandleFunc("/login", handleLogin)
	r.HandleFunc("/callback", authHandler.HandleCallback)
	r.Use(authHandler.SessionManagerMiddleware())

	// protected routes
	protected := r.NewRoute().Subrouter()
	protected.Use(authHandler.AuthMiddleware)
	protected.HandleFunc("/info", handleInfo)
	protected.HandleFunc("/logout", handleLogout)
	protected.HandleFunc("/display-token", handleDisplayToken)
	protected.HandleFunc("/menu", handleMenu)
	return r
}
