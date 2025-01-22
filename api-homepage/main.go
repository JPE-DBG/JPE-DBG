package main

import (
	"fmt"
	"net/http"
	"os"

	"api-homepage/handler"
)

func main() {
	fmt.Println("Starting server on :10000")
	if err := http.ListenAndServe(":10000", handler.NewHandler()); err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}
}
