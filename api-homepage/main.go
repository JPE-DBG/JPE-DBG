package main

import (
	"fmt"
	"net/http"
	"os"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	http.HandleFunc("/", mainHandler)
	fmt.Println("Starting server on :10000")
	if err := http.ListenAndServe(":10000", nil); err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}
}
