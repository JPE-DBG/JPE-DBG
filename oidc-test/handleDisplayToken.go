package main

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"net/http"
)

func handleDisplayToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	session, err := getSession(r)
	if err != nil {
		http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the token from the session
	token, ok := session.Values["token"].(*oauth2.Token)
	if !ok || token == nil {
		http.Error(w, "No token found in session", http.StatusNotFound)
		return
	}

	// Convert token to JSON
	tokenJSON, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Write token info to response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
  <!DOCTYPE html>
  <html>
  <head>
   <title>Token Info</title>
   <style>
    body {
     display: flex;
     flex-direction: column;
     justify-content: space-between;
     align-items: center;
     height: 100vh;
     margin: 0;
     font-family: Arial, sans-serif;
     overflow: hidden;
    }
    .header, .footer {
     text-align: center;
    }
    .content {
     flex-grow: 1;
     display: flex;
     justify-content: center;
     align-items: center;
     overflow: auto;
     width: 100%;
     padding: 10px;
     border-left: 2px solid #000;
     border-right: 2px solid #000;
    }
    .content pre {
     text-align: left;
     white-space: pre-wrap;
     word-wrap: break-word;
     max-width: 100%;
    }
   </style>
  </head>
  <body>
   <div class="header">
    <h1>Token Info</h1>
   </div>
   <div class="content">
    <pre>` + string(tokenJSON) + `</pre>
   </div>
   <div class="footer">
    <a href="/menu">Menu</a>
   </div>
  </body>
  </html>
 `))
}
