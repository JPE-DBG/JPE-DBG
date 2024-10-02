package oidchandler

import "net/http"

func (h *OidcHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.LoginDone(r.Context()) {
			http.Redirect(w, r, h.loginURL, http.StatusFound)
			return
		}
		if !h.HasReadAccess(r.Context()) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
