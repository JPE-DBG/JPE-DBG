package handler

import "net/http"

func registerRoutes(h handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.homepageHandler)
	return mux
}
