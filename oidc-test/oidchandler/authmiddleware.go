package oidchandler

import "net/http"

func (h *OidcHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groups, ok := h.sessionManager.Get(r.Context(), sessionGroupKey).([]string)
		if !ok || len(groups) == 0 {
			http.Error(w, "no group info", http.StatusUnauthorized)
			return
		}

		if !isGroupMember(groups, readerGroup) {
			http.Error(w, "not a seti user", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
