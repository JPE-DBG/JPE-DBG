package main

import "net/http"

func handleMenu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Menu</title>
			<style>
				body {
					display: flex;
					justify-content: center;
					align-items: center;
					height: 100vh;
					margin: 0;
					font-family: Arial, sans-serif;
				}
				.menu {
					text-align: center;
				}
				.menu a {
					display: block;
					margin: 10px 0;
					text-decoration: none;
					color: #007BFF;
				}
			</style>
		</head>
		<body>
			<div class="menu">
				<h1>Menu</h1>
				<a href="/info">Info</a>
				<a href="/display-token">Display Token</a>
				<a href="/logout">Logout</a>
			</div>
		</body>
		</html>
	`))
}
