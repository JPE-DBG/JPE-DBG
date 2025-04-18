package main

import (
	"log"
	"net/http"
)

const MapCols = 100
const MapRows = 100

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/map", mapHandler)
	http.HandleFunc("/api/game", gameHandler)
	http.HandleFunc("/api/move", moveHandler)
	http.HandleFunc("/api/move-range", moveRangeHandler)
	http.HandleFunc("/api/endturn", endTurnHandler)

	log.Println("Serving on http://localhost:8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
