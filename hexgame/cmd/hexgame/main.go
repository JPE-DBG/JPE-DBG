package main

import (
	"flag"
	"hexgame/internal/game"
	"log"
	"net/http"
)

func main() {
	// Add command line flags
	cols := flag.Int("cols", game.MapCols, "Initial map columns (30-50000)")
	rows := flag.Int("rows", game.MapRows, "Initial map rows (30-50000)")
	flag.Parse()

	// Validate and set map size
	if *cols < 30 || *cols > 50000 {
		log.Printf("Invalid columns value %d, using default %d", *cols, game.MapCols)
		*cols = game.MapCols
	}
	if *rows < 30 || *rows > 50000 {
		log.Printf("Invalid rows value %d, using default %d", *rows, game.MapRows)
		*rows = game.MapRows
	}

	// Set initial map size
	game.SetInitialMapSize(*cols, *rows)

	fs := http.FileServer(http.Dir("../../web/static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/map", game.MapHandler)
	http.HandleFunc("/api/game", game.GameHandler)
	http.HandleFunc("/api/move", game.MoveHandler)
	http.HandleFunc("/api/move-range", game.MoveRangeHandler)
	http.HandleFunc("/api/endturn", game.EndTurnHandler)

	log.Printf("Starting with map size %dx%d", *cols, *rows)
	log.Println("Serving on http://localhost:8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
