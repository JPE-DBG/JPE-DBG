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
			config := UnitConfigs["ship"][1]
			// Check cost
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "ship", Tier: 1, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
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
			config := UnitConfigs["troop"][1]
			// Check cost
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "troop", Tier: 1, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
					break
				}
			}
		}
	} else if req.Type == "place_city" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			config := BuildingConfigs["city"]
			// Check cost
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "city"})
					break
				}
			}
		}
	} else if req.Type == "place_port" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			config := BuildingConfigs["port"]
			// Check cost
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Buildings = append(gameState.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: gameState.CurrentPlayer, Level: 1, Type: "port"})
					break
				}
			}
		}
	} else if req.Type == "place_fort" {
		if gameState.Tiles[req.ToCol][req.ToRow].Type == "land" && !unitAt(req.ToCol, req.ToRow) && !buildingAt(req.ToCol, req.ToRow) {
			config := BuildingConfigs["fort"]
			// Check cost
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
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
			config := UnitConfigs["ship"][2]
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Research >= config.CostResearch && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "ship", Tier: 2, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
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
			config := UnitConfigs["troop"][2]
			// Check research >= 50, cost 20 gold, 5 iron
			for i, p := range gameState.Players {
				if p.ID == gameState.CurrentPlayer && p.Research >= config.CostResearch && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
					gameState.Players[i].Gold -= config.CostGold
					gameState.Players[i].Wood -= config.CostWood
					gameState.Players[i].Iron -= config.CostIron
					gameState.Units = append(gameState.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: gameState.CurrentPlayer, Type: "troop", Tier: 2, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
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
					config := BuildingConfigs[b.Type]
					gameState.Players[i].Gold += config.ProductionGold
					gameState.Players[i].Wood += config.ProductionWood
					gameState.Players[i].Iron += config.ProductionIron
				}
			}
			// Produce research
			gameState.Players[i].Research += 5 // 5 research per turn
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

func JoinHandler(w http.ResponseWriter, r *http.Request) {
	if gameState == nil {
		// Initialize gameState if it doesn't exist yet
		gameState = newGameState(MapCols, MapRows)
	}

	type JoinRequest struct {
		Name string `json:"name"`
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Find the next available player ID
	nextPlayerID := 1
	usedIDs := make(map[int]bool)
	for _, p := range gameState.Players {
		usedIDs[p.ID] = true
	}
	for usedIDs[nextPlayerID] {
		nextPlayerID++
	}

	// Create new player
	newPlayer := Player{
		ID:       nextPlayerID,
		Name:     req.Name,
		Color:    getPlayerColor(nextPlayerID),
		Capital:  getPlayerCapital(nextPlayerID, gameState.Cols, gameState.Rows, gameState),
		Gold:     100,
		Wood:     50,
		Iron:     20,
		Research: 0,
	}

	gameState.Players = append(gameState.Players, newPlayer)

	// If this is the first player, make them the current player
	if len(gameState.Players) == 1 {
		gameState.CurrentPlayer = nextPlayerID
	}

	// Create initial city and unit for the new player at their capital
	c := newPlayer.Capital
	log.Printf("Creating capital for player %d at coordinates [%d,%d]", nextPlayerID, c[0], c[1])

	// The capital position is already guaranteed to be valid land
	gameState.Buildings = append(gameState.Buildings, Building{
		Col: c[0], Row: c[1], Owner: nextPlayerID, Level: 1, Type: "city",
	})
	gameState.Units = append(gameState.Units, Unit{
		Col: c[0], Row: c[1], Moved: false, Owner: nextPlayerID,
		Type: "troop", Tier: 1, Health: 5, Attack: 2, Defense: 1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"playerId":  nextPlayerID,
		"color":     newPlayer.Color,
		"capital":   newPlayer.Capital,
		"gameState": gameState,
	})
	select {
	case broadcast <- gameState:
	default:
	}
}

func getPlayerColor(playerID int) string {
	colors := []string{"#ff0000", "#0000ff", "#00ff00", "#ffff00", "#ff00ff", "#00ffff", "#ffa500", "#800080"}
	return colors[(playerID-1)%len(colors)]
}

func getPlayerCapital(playerID int, cols, rows int, gameState *GameState) [2]int {
	// Scan the map for available land positions and choose one based on player ID
	var landPositions [][2]int

	// Scan the entire map for land tiles that don't have buildings or units
	for col := 0; col < cols; col++ {
		for row := 0; row < rows; row++ {
			if gameState.Tiles[col][row].Type == "land" && !unitAt(col, row) && !buildingAt(col, row) {
				landPositions = append(landPositions, [2]int{col, row})
			}
		}
	}

	if len(landPositions) == 0 {
		log.Printf("No land positions available for player %d capital!", playerID)
		return [2]int{0, 0} // fallback
	}

	// Choose positions within preferred quadrants but avoid edges/corners
	var preferredPositions [][2]int
	margin := cols / 8 // Avoid positions too close to edges

	switch playerID % 4 {
	case 1: // Top-left quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > margin && pos[0] < cols/2-margin && pos[1] > margin && pos[1] < rows/2-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 2: // Top-right quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > cols/2+margin && pos[0] < cols-margin && pos[1] > margin && pos[1] < rows/2-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 3: // Bottom-left quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > margin && pos[0] < cols/2-margin && pos[1] > rows/2+margin && pos[1] < rows-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 0: // Bottom-right quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > cols/2+margin && pos[0] < cols-margin && pos[1] > rows/2+margin && pos[1] < rows-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	}

	// If we found positions in the preferred quadrant, choose one
	if len(preferredPositions) > 0 {
		// Use player ID to select different positions within the quadrant
		// This distributes players more naturally
		index := (playerID - 1) * 7 % len(preferredPositions) // Multiply by prime for better distribution
		return preferredPositions[index]
	}

	// Fallback: use any available land position, avoiding edges
	var safePositions [][2]int
	for _, pos := range landPositions {
		if pos[0] > margin && pos[0] < cols-margin && pos[1] > margin && pos[1] < rows-margin {
			safePositions = append(safePositions, pos)
		}
	}

	if len(safePositions) > 0 {
		index := (playerID - 1) % len(safePositions)
		return safePositions[index]
	}

	// Ultimate fallback: use any available land position
	index := (playerID - 1) % len(landPositions)
	return landPositions[index]
}
