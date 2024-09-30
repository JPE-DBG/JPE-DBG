package main

import (
	"log"
	"net/http"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := authHandler.Session(r)
		if err != nil || session.IsNew {
			http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if authHandler.NeedUserInfoUpdate(session, userCheckInterval) {
			log.Println("Updating user info")
			session, err = authHandler.UpdateUserInfo(baseContext, w, r)
			if err != nil {
				http.Error(w, "Failed to update user info: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			log.Println("User info is up to date")
		}

		groups, ok := session.Values[sessionGroupKey].([]string)
		if !ok || len(groups) == 0 {
			http.Error(w, "no group info", http.StatusUnauthorized)
			return
		}

		if !isGroupMember(groups, readerGroup) {
			http.Error(w, "not an admin", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
