package main

import (
	"net/http"
)

func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	session, err := getSession(r)
	if err != nil {
		http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token := getToken(session)
	if token == nil {
		http.Error(w, "no token info", http.StatusUnauthorized)
		return
	}

	// get user info from provider
	userInfo, err := getUserInfo(token)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	formattedUserInfo := formatUserInfo(userInfo)

	// write user info to response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>User Info</title>
			<style>
				body {
					display: flex;
					flex-direction: column;
					justify-content: space-between;
					align-items: center;
					height: 100vh;
					margin: 0;
					font-family: Arial, sans-serif;
				}
				.header, .footer {
					text-align: center;
				}
				.content {
					flex-grow: 1;
					display: flex;
					justify-content: center;
					align-items: center;
				}
				.content pre {
					text-align: left;
				}
			</style>
		</head>
		<body>
			<div class="header">
				<h1>User Info</h1>
			</div>
			<div class="content">
				<pre>` + formattedUserInfo + `</pre>
			</div>
			<div class="footer">
				<a href="/menu">Menu</a>
			</div>
		</body>
		</html>
	`))
}
