package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type MapResponse struct {
	Cols  int      `json:"cols"`
	Rows  int      `json:"rows"`
	Tiles [][]Tile `json:"tiles"`
}

type MoveRequest struct {
	Type    string `json:"type"` // "move", "place_unit", "place_building"
	FromCol int    `json:"fromCol,omitempty"`
	FromRow int    `json:"fromRow,omitempty"`
	ToCol   int    `json:"toCol"`
	ToRow   int    `json:"toRow"`
}

func mapHandler(w http.ResponseWriter, r *http.Request) {
	tiles := generateMapV3(MapCols, MapRows)
	resp := MapResponse{
		Cols:  MapCols,
		Rows:  MapRows,
		Tiles: tiles,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	regen := r.URL.Query().Get("regen")
	if regen == "1" || gameState == nil {
		gameState = newGameState(MapCols, MapRows)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
}

func moveHandler(w http.ResponseWriter, r *http.Request) {
	if gameState == nil {
		log.Println("moveHandler: gameState is nil")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("moveHandler: bad request:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("moveHandler: received %+v\n", req)
	if req.Type == "move" {
		log.Printf("Trying to move unit from (%d,%d) to (%d,%d)", req.FromCol, req.FromRow, req.ToCol, req.ToRow)
	}
	if req.Type == "place_unit" || req.Type == "place_building" {
		log.Printf("Checking placement at %d,%d: tile=%s unit=%v building=%v",
			req.ToCol, req.ToRow,
			gameState.Tiles[req.ToCol][req.ToRow].Type,
			unitAt(req.ToCol, req.ToRow),
			buildingAt(req.ToCol, req.ToRow),
		)
	}
	if req.Type == "move" {
		for i, u := range gameState.Units {
			if u.Col == req.FromCol && u.Row == req.FromRow && !u.Moved && u.Owner == gameState.CurrentPlayer {
				if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
					gameState.Units[i].Col = req.ToCol
					gameState.Units[i].Row = req.ToRow
					gameState.Units[i].Moved = true
				}
				break
			}
		}
	} else if req.Type == "place_unit" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer})
		}
	} else if req.Type == "place_building" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
}

func endTurnHandler(w http.ResponseWriter, r *http.Request) {
	if gameState == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// For one player, always reset all units' Moved flag
	for i := range gameState.Units {
		gameState.Units[i].Moved = false
	}
	gameState.Turn++
	// Keep CurrentPlayer always 1
	gameState.CurrentPlayer = 1
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
}

func moveRangeHandler(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Col   int `json:"col"`
		Row   int `json:"row"`
		Range int `json:"range"`
	}
	type Resp struct {
		Tiles [][2]int `json:"tiles"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tiles := getMoveRange(req.Col, req.Row, req.Range)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Resp{Tiles: tiles})
}
