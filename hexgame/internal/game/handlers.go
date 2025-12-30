package game

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
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
			// Check cost: 20 wood
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Wood >= 20 {
					gameState.Players[i].Wood -= 20
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "ship", Tier: 1, Health: 10, Attack: 3, Defense: 2})
					break
				}
			}
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
			// Check cost: 10 gold
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= 10 {
					gameState.Players[i].Gold -= 10
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "troop", Tier: 1, Health: 5, Attack: 2, Defense: 1})
					break
				}
			}
		}
	} else if req.Type == "place_city" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			// Check cost: 50 gold
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= 50 {
					gameState.Players[i].Gold -= 50
					gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "city"})
					break
				}
			}
		}
	} else if req.Type == "place_port" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			// Check cost: 30 gold, 10 wood
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= 30 && p.Wood >= 10 {
					gameState.Players[i].Gold -= 30
					gameState.Players[i].Wood -= 10
					gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "port"})
					break
				}
			}
		}
	} else if req.Type == "place_fort" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			// Check cost: 40 gold, 5 iron
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= 40 && p.Iron >= 5 {
					gameState.Players[i].Gold -= 40
					gameState.Players[i].Iron -= 5
					gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "fort"})
					break
				}
			}
		}
	} else if req.Type == "attack" {
		// Attack unit or building at target
		var attacker *Unit
		attackerIndex := -1
		for i, u := range gameState.Units {
			if u.Col == req.FromCol && u.Row == req.FromRow && u.Owner == gameState.CurrentPlayer {
				attacker = &gameState.Units[i]
				attackerIndex = i
				break
			}
		}
		if attacker != nil {
			// Check if attacking unit
			for i, u := range gameState.Units {
				if u.Col == req.ToCol && u.Row == req.ToRow && u.Owner != gameState.CurrentPlayer {
					// Combat: calculate damage
					damageToDefender := attacker.Attack - u.Defense
					if damageToDefender < 1 {
						damageToDefender = 1
					}
					damageToAttacker := u.Attack - attacker.Defense
					if damageToAttacker < 1 {
						damageToAttacker = 1
					}
					gameState.Units[i].Health -= damageToDefender
					attacker.Health -= damageToAttacker
					if gameState.Units[i].Health <= 0 {
						gameState.Units = append(gameState.Units[:i], gameState.Units[i+1:]...)
					}
					if attacker.Health <= 0 {
						gameState.Units = append(gameState.Units[:attackerIndex], gameState.Units[attackerIndex+1:]...)
					}
					break
				}
			}
			// Check if attacking building
			for i, b := range gameState.Buildings {
				if b.Col == req.ToCol && b.Row == req.ToRow && b.Owner != gameState.CurrentPlayer {
					// Damage building: reduce level
					damage := attacker.Attack
					gameState.Buildings[i].Level -= damage
					if gameState.Buildings[i].Level <= 0 {
						gameState.Buildings = append(gameState.Buildings[:i], gameState.Buildings[i+1:]...)
					}
					break
				}
			}
		}
	} else if req.Type == "place_advanced_ship" {
		// Check research >= 50, cost 40 wood, 10 iron
		hasPort := false
		for _, b := range gameState.Buildings {
			if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "port" && b.Owner == gameState.CurrentPlayer {
				hasPort = true
				break
			}
		}
		if hasPort && (gameState.Tiles[req.ToCol][req.ToRow].Type == "land" || gameState.Tiles[req.ToCol][req.ToRow].Type == "water") && !unitAt(req.ToCol, req.ToRow) {
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Research >= 50 && p.Wood >= 40 && p.Iron >= 10 {
					gameState.Players[i].Wood -= 40
					gameState.Players[i].Iron -= 10
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "ship", Tier: 2, Health: 15, Attack: 5, Defense: 3})
					break
				}
			}
		}
	} else if req.Type == "place_advanced_troop" {
		hasCity := false
		for _, b := range gameState.Buildings {
			if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "city" && b.Owner == gameState.CurrentPlayer {
				hasCity = true
				break
			}
		}
		if hasCity && gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) {
			// Check research >= 50, cost 20 gold, 5 iron
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Research >= 50 && p.Gold >= 20 && p.Iron >= 5 {
					gameState.Players[i].Gold -= 20
					gameState.Players[i].Iron -= 5
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "troop", Tier: 2, Health: 8, Attack: 4, Defense: 2})
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
	// Produce resources
	for i, p := range gameState.Players {
		if p.ID == gameState.CurrentPlayer {
			for _, b := range gameState.Buildings {
				if b.Owner == p.ID {
					if b.Type == "city" {
						gameState.Players[i].Gold += 10
					} else if b.Type == "port" {
						gameState.Players[i].Wood += 5
					} else if b.Type == "fort" {
						gameState.Players[i].Iron += 3
					}
				}
			}
			// Produce research
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer {
					gameState.Players[i].Research += 5 // 5 research per turn
					break
				}
			}
			break
		}
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
