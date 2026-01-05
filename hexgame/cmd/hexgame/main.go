package main

import (
	"flag"
	"hexgame/internal/game"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Add command line flags
	cols := flag.Int("cols", game.MapCols, "Initial map columns (30-50000)")
	rows := flag.Int("rows", game.MapRows, "Initial map rows (30-50000)")
	flag.Parse()

	// Validate and set map size
	if *cols < 30 || *cols > 50000 {
		slog.Warn("Invalid columns value, using default", "cols", *cols, "default", game.MapCols)
		*cols = game.MapCols
	}
	if *rows < 30 || *rows > 50000 {
		slog.Warn("Invalid rows value, using default", "rows", *rows, "default", game.MapRows)
		*rows = game.MapRows
	}

	// Create game service
	gameSvc := game.NewGame(game.WithInitialSize(*cols, *rows))

	staticDir := filepath.Join(".", "web", "static")
	fs := http.FileServer(http.Dir(staticDir))

	http.Handle("/", fs)
	http.HandleFunc("/api/map", gameSvc.MapHandler)
	http.HandleFunc("/api/game", gameSvc.GameHandler)
	http.HandleFunc("/api/move", gameSvc.MoveHandler)
	http.HandleFunc("/api/place", gameSvc.PlaceHandler)
	http.HandleFunc("/api/move-range", gameSvc.MoveRangeHandler)
	http.HandleFunc("/api/endturn", gameSvc.EndTurnHandler)
	http.HandleFunc("/ws", gameSvc.WebSocketHandler)
	http.HandleFunc("/api/join", gameSvc.JoinHandler)

	slog.Info("Starting server", "cols", *cols, "rows", *rows)
	slog.Info("Serving on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
