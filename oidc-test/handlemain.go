package main

import (
	"net/http"
)

func handleMain(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Main Page</title>
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
			</style>
		</head>
		<body>
			<div class="header">
				<h1>Main Page</h1>
			</div>
			<div class="content">
				<a href="/login">Login with OIDC</a>
			</div>
			<div class="footer">
				<a href="/menu">Menu</a>
			</div>
		</body>
		</html>
	`))
}
