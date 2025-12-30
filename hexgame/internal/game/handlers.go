package game

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"time"
)

type MapResponse struct {
	Cols  int      `json:"cols"`
	Rows  int      `json:"rows"`
	Tiles [][]Tile `json:"tiles"`
}

type MoveRequest struct {
	Type    string `json:"type"` // "move", "place_ship", "place_troop", "place_city", "place_port", "place_fort", "attack"
	FromCol int    `json:"fromCol,omitempty"`
	FromRow int    `json:"fromRow,omitempty"`
	ToCol   int    `json:"toCol"`
	ToRow   int    `json:"toRow"`
}

func MapHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	cols := MapCols
	rows := MapRows
	if c := r.URL.Query().Get("cols"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v >= 30 && v <= 50000 {
			cols = v
		}
	}
	if rws := r.URL.Query().Get("rows"); rws != "" {
		if v, err := strconv.Atoi(rws); err == nil && v >= 30 && v <= 50000 {
			rows = v
		}
	}
	tiles := generateMapV3(cols, rows)
	resp := MapResponse{
		Cols:  cols,
		Rows:  rows,
		Tiles: tiles,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	log.Printf("mapHandler: generated %dx%d map in %v", cols, rows, time.Since(start))
}

func GameHandler(w http.ResponseWriter, r *http.Request) {
	regen := r.URL.Query().Get("regen")
	cols := MapCols
	rows := MapRows
	if regen == "1" {
		// Parse cols/rows from query params if present
		if c := r.URL.Query().Get("cols"); c != "" {
			if v, err := strconv.Atoi(c); err == nil && v >= 30 && v <= 50000 {
				cols = v
			}
		}
		if rws := r.URL.Query().Get("rows"); rws != "" {
			if v, err := strconv.Atoi(rws); err == nil && v >= 30 && v <= 50000 {
				rows = v
			}
		}
		gameState = newGameState(cols, rows)
	} else if gameState == nil {
		// Initialize gameState on first load with default size
		gameState = newGameState(cols, rows)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
}

func MoveHandler(w http.ResponseWriter, r *http.Request) {
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
				valid := false
				if u.Type == "ship" {
					valid = (gameState.Tiles[req.ToCol][req.ToRow].Type == "land" || gameState.Tiles[req.ToCol][req.ToRow].Type == "water") && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow)
				} else if u.Type == "troop" {
					valid = gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow)
				}
				if valid {
					gameState.Units[i].Col = req.ToCol
					gameState.Units[i].Row = req.ToRow
					gameState.Units[i].Moved = true
				}
				break
			}
		}
	} else if req.Type == "place_ship" {
		// Check if there's a port at the position
		hasPort := false
		for _, b := range gameState.Buildings {
			if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "port" && b.Owner == gameState.CurrentPlayer {
				hasPort = true
				break
			}
		}
		if hasPort && (gameState.Tiles[req.ToCol][req.ToRow].Type == "land" || gameState.Tiles[req.ToCol][req.ToRow].Type == "water") && !unitAt(req.ToCol, req.ToRow) {
			gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "ship", Health: 10})
		}
	} else if req.Type == "place_troop" {
		// Check if there's a city at the position
		hasCity := false
		for _, b := range gameState.Buildings {
			if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "city" && b.Owner == gameState.CurrentPlayer {
				hasCity = true
				break
			}
		}
		if hasCity && gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) {
			gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "troop", Health: 5})
		}
	} else if req.Type == "place_city" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "city"})
		}
	} else if req.Type == "place_port" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "port"})
		}
	} else if req.Type == "place_fort" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "fort"})
		}
	} else if req.Type == "attack" {
		// Attack unit or building at target
		attacker := -1
		for i, u := range gameState.Units {
			if u.Col == req.FromCol && u.Row == req.FromRow && u.Owner == gameState.CurrentPlayer {
				attacker = i
				break
			}
		}
		if attacker >= 0 {
			// Check if attacking unit
			for i, u := range gameState.Units {
				if u.Col == req.ToCol && u.Row == req.ToRow && u.Owner != gameState.CurrentPlayer {
					// Simple combat: attacker health -= 1, defender health -= 1
					gameState.Units[attacker].Health--
					gameState.Units[i].Health--
					if gameState.Units[attacker].Health <= 0 {
						gameState.Units = append(gameState.Units[:attacker], gameState.Units[attacker+1:]...)
					}
					if gameState.Units[i].Health <= 0 {
						gameState.Units = append(gameState.Units[:i], gameState.Units[i+1:]...)
					}
					break
				}
			}
			// Check if attacking building
			for i, b := range gameState.Buildings {
				if b.Col == req.ToCol && b.Row == req.ToRow && b.Owner != gameState.CurrentPlayer {
					// Damage building: reduce level
					gameState.Buildings[i].Level--
					if gameState.Buildings[i].Level <= 0 {
						gameState.Buildings = append(gameState.Buildings[:i], gameState.Buildings[i+1:]...)
					}
					break
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
	select {
	case broadcast <- gameState:
	default:
	}
}

func EndTurnHandler(w http.ResponseWriter, r *http.Request) {
	if gameState == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Reset all units' Moved flag
	for i := range gameState.Units {
		gameState.Units[i].Moved = false
	}
	gameState.Turn++
	// Cycle to next player
	gameState.CurrentPlayer++
	if gameState.CurrentPlayer > len(gameState.Players) {
		gameState.CurrentPlayer = 1
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gameState)
	select {
	case broadcast <- gameState:
	default:
	}
}

func MoveRangeHandler(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Col      int    `json:"col"`
		Row      int    `json:"row"`
		Range    int    `json:"range"`
		UnitType string `json:"unitType"`
	}
	type Resp struct {
		Tiles [][2]int `json:"tiles"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tiles := getMoveRange(req.Col, req.Row, req.Range, req.UnitType)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Resp{Tiles: tiles})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan *GameState)

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			delete(clients, conn)
			break
		}
	}
}

func init() {
	go func() {
		for {
			gameState := <-broadcast
			for client := range clients {
				if err := client.WriteJSON(gameState); err != nil {
					client.Close()
					delete(clients, client)
				}
			}
		}
	}()
}
