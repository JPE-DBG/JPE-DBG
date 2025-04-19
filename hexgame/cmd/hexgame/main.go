package main

import (
	"hexgame/internal/game"
	"log"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("../../web/static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/map", game.MapHandler)
	http.HandleFunc("/api/game", game.GameHandler)
	http.HandleFunc("/api/move", game.MoveHandler)
	http.HandleFunc("/api/move-range", game.MoveRangeHandler)
	http.HandleFunc("/api/endturn", game.EndTurnHandler)

	log.Println("Serving on http://localhost:8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
