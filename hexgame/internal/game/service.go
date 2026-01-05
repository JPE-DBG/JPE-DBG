package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
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

type Game interface {
	GetState(ctx context.Context) (*GameState, error)
	Move(ctx context.Context, req MoveRequest) error
	// Add other methods as needed
}

type GameImpl struct {
	state           *GameState
	logger          *slog.Logger
	moveStrategies  map[string]Strategy
	placeStrategies map[string]Strategy
	clients         map[*websocket.Conn]bool
	clientsMutex    sync.Mutex
	upgrader        websocket.Upgrader
}

type Option func(*GameImpl)

func WithLogger(l *slog.Logger) Option {
	return func(g *GameImpl) { g.logger = l }
}

func WithInitialSize(cols, rows int) Option {
	return func(g *GameImpl) {
		g.state = newGameState(cols, rows)
	}
}

type Strategy interface {
	Execute(g *GameImpl, req MoveRequest) error
}

type MoveUnitStrategy struct{}

func (s *MoveUnitStrategy) Execute(g *GameImpl, req MoveRequest) error {
	for i, u := range g.state.Units {
		if u.Col == req.FromCol && u.Row == req.FromRow && !u.Moved && u.Owner == g.state.CurrentPlayer {
			valid := false
			switch u.Type {
			case "ship":
				valid = g.state.Tiles[req.ToCol][req.ToRow].Type == "water" && !g.unitAt(req.ToCol, req.ToRow) && !g.buildingAt(req.ToCol, req.ToRow)
			case "troop":
				valid = g.state.Tiles[req.ToCol][req.ToRow].Type == "land" && !g.unitAt(req.ToCol, req.ToRow) && !g.buildingAt(req.ToCol, req.ToRow)
			}
			if valid {
				g.state.Units[i].Col = req.ToCol
				g.state.Units[i].Row = req.ToRow
				g.state.Units[i].Moved = true
				return nil
			}
			return errors.New("invalid move")
		}
	}
	return errors.New("unit not found or already moved")
}

type PlaceShipStrategy struct{}

func (s *PlaceShipStrategy) Execute(g *GameImpl, req MoveRequest) error {
	canPlaceShip := false
	for _, b := range g.state.Buildings {
		if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "port" && b.Owner == g.state.CurrentPlayer {
			canPlaceShip = true
			break
		}
	}
	if !canPlaceShip || g.unitAt(req.ToCol, req.ToRow) {
		return errors.New("ships must be placed at a port")
	}
	config := UnitConfigs["ship"][1]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Units = append(g.state.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: g.state.CurrentPlayer, Type: "ship", Tier: 1, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlaceTroopStrategy struct{}

func (s *PlaceTroopStrategy) Execute(g *GameImpl, req MoveRequest) error {
	hasCity := false
	for _, b := range g.state.Buildings {
		if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "city" && b.Owner == g.state.CurrentPlayer {
			hasCity = true
			break
		}
	}
	if !hasCity || g.unitAt(req.ToCol, req.ToRow) {
		return errors.New("troops must be placed on a city")
	}
	config := UnitConfigs["troop"][1]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Units = append(g.state.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: g.state.CurrentPlayer, Type: "troop", Tier: 1, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlaceCityStrategy struct{}

func (s *PlaceCityStrategy) Execute(g *GameImpl, req MoveRequest) error {
	if g.state.Tiles[req.ToCol][req.ToRow].Type != "land" || g.unitAt(req.ToCol, req.ToRow) || g.buildingAt(req.ToCol, req.ToRow) {
		return errors.New("cities must be placed on land")
	}
	config := BuildingConfigs["city"]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron && p.Research >= config.CostResearch {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Players[i].Research -= config.CostResearch
			g.state.Buildings = append(g.state.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: g.state.CurrentPlayer, Level: 1, Type: "city"})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlacePortStrategy struct{}

func (s *PlacePortStrategy) Execute(g *GameImpl, req MoveRequest) error {
	if g.state.Tiles[req.ToCol][req.ToRow].Type != "land" || g.unitAt(req.ToCol, req.ToRow) || g.buildingAt(req.ToCol, req.ToRow) {
		return errors.New("ports must be placed on land adjacent to water")
	}
	// Check if adjacent to water
	adjacentToWater := false
	for _, n := range hexNeighbors(req.ToCol, req.ToRow, g.state.Cols, g.state.Rows) {
		nc, nr := n[0], n[1]
		if g.state.Tiles[nc][nr].Type == "water" {
			adjacentToWater = true
			break
		}
	}
	if !adjacentToWater {
		return errors.New("ports must be placed on land adjacent to water")
	}
	config := BuildingConfigs["port"]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron && p.Research >= config.CostResearch {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Players[i].Research -= config.CostResearch
			g.state.Buildings = append(g.state.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: g.state.CurrentPlayer, Level: 1, Type: "port"})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlaceFortStrategy struct{}

func (s *PlaceFortStrategy) Execute(g *GameImpl, req MoveRequest) error {
	if g.state.Tiles[req.ToCol][req.ToRow].Type != "land" || g.unitAt(req.ToCol, req.ToRow) || g.buildingAt(req.ToCol, req.ToRow) {
		return errors.New("forts must be placed on land")
	}
	config := BuildingConfigs["fort"]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron && p.Research >= config.CostResearch {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Players[i].Research -= config.CostResearch
			g.state.Buildings = append(g.state.Buildings, Building{Col: req.ToCol, Row: req.ToRow, Owner: g.state.CurrentPlayer, Level: 1, Type: "fort"})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlaceAdvancedTroopStrategy struct{}

func (s *PlaceAdvancedTroopStrategy) Execute(g *GameImpl, req MoveRequest) error {
	hasCity := false
	for _, b := range g.state.Buildings {
		if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "city" && b.Owner == g.state.CurrentPlayer {
			hasCity = true
			break
		}
	}
	if !hasCity || g.unitAt(req.ToCol, req.ToRow) {
		return errors.New("advanced troops must be placed on a city")
	}
	config := UnitConfigs["troop"][2]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron && p.Research >= config.CostResearch {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Players[i].Research -= config.CostResearch
			g.state.Units = append(g.state.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: g.state.CurrentPlayer, Type: "troop", Tier: 2, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

type PlaceAdvancedShipStrategy struct{}

func (s *PlaceAdvancedShipStrategy) Execute(g *GameImpl, req MoveRequest) error {
	canPlaceShip := false
	for _, b := range g.state.Buildings {
		if b.Col == req.ToCol && b.Row == req.ToRow && b.Type == "port" && b.Owner == g.state.CurrentPlayer {
			canPlaceShip = true
			break
		}
	}
	if !canPlaceShip || g.unitAt(req.ToCol, req.ToRow) {
		return errors.New("advanced ships must be placed at a port")
	}
	config := UnitConfigs["ship"][2]
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer && p.Gold >= config.CostGold && p.Wood >= config.CostWood && p.Iron >= config.CostIron && p.Research >= config.CostResearch {
			g.state.Players[i].Gold -= config.CostGold
			g.state.Players[i].Wood -= config.CostWood
			g.state.Players[i].Iron -= config.CostIron
			g.state.Players[i].Research -= config.CostResearch
			g.state.Units = append(g.state.Units, Unit{Col: req.ToCol, Row: req.ToRow, Moved: false, Owner: g.state.CurrentPlayer, Type: "ship", Tier: 2, Health: config.Stats.Health, Attack: config.Stats.Attack, Defense: config.Stats.Defense})
			return nil
		}
	}
	return errors.New("insufficient resources")
}

func NewGame(opts ...Option) *GameImpl {
	g := &GameImpl{
		logger:  slog.Default(),
		state:   nil,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow connections from any origin for development
			},
		},
		moveStrategies: map[string]Strategy{
			"move": &MoveUnitStrategy{},
		},
		placeStrategies: map[string]Strategy{
			"place_ship":           &PlaceShipStrategy{},
			"place_troop":          &PlaceTroopStrategy{},
			"place_city":           &PlaceCityStrategy{},
			"place_port":           &PlacePortStrategy{},
			"place_fort":           &PlaceFortStrategy{},
			"place_advanced_ship":  &PlaceAdvancedShipStrategy{},
			"place_advanced_troop": &PlaceAdvancedTroopStrategy{},
		},
	}
	for _, opt := range opts {
		opt(g)
	}
	if g.state == nil {
		g.state = newGameState(MapCols, MapRows)
	}
	return g
}

func (g *GameImpl) GetState(ctx context.Context) (*GameState, error) {
	return g.state, nil
}

func (g *GameImpl) Move(ctx context.Context, req MoveRequest) error {
	g.logger.Info("Processing move", "type", req.Type, "fromCol", req.FromCol, "fromRow", req.FromRow, "toCol", req.ToCol, "toRow", req.ToRow)
	if req.Type == "move" {
		if strategy, ok := g.moveStrategies[req.Type]; ok {
			return strategy.Execute(g, req)
		}
	}
	return errors.New("invalid move type")
}

func (g *GameImpl) Place(ctx context.Context, req MoveRequest) error {
	g.logger.Info("Processing place", "type", req.Type, "toCol", req.ToCol, "toRow", req.ToRow)
	if strategy, ok := g.placeStrategies[req.Type]; ok {
		return strategy.Execute(g, req)
	}
	return errors.New("invalid place type")
}

func (g *GameImpl) unitAt(col, row int) bool {
	for _, u := range g.state.Units {
		if u.Col == col && u.Row == row {
			return true
		}
	}
	return false
}

func (g *GameImpl) buildingAt(col, row int) bool {
	for _, b := range g.state.Buildings {
		if b.Col == col && b.Row == row {
			return true
		}
	}
	return false
}

func (g *GameImpl) getMoveRange(col, row, rng int, unitType string) [][2]int {
	if g.state == nil {
		return nil
	}
	visited := make([][]bool, g.state.Cols)
	for i := range visited {
		visited[i] = make([]bool, g.state.Rows)
	}
	result := [][2]int{}
	queue := [][3]int{{col, row, 0}}
	visited[col][row] = true
	for len(queue) > 0 {
		c, r, dist := queue[0][0], queue[0][1], queue[0][2]
		queue = queue[1:]
		if dist > 0 {
			result = append(result, [2]int{c, r})
		}
		if dist == rng {
			continue
		}
		for _, n := range hexNeighbors(c, r, g.state.Cols, g.state.Rows) {
			nc, nr := n[0], n[1]
			if !visited[nc][nr] {
				valid := false
				switch unitType {
				case "ship":
					valid = (g.state.Tiles[nc][nr].Type == "land" || g.state.Tiles[nc][nr].Type == "water") && !g.unitAt(nc, nr) && !g.buildingAt(nc, nr)
				case "troop":
					valid = g.state.Tiles[nc][nr].Type == "land" && !g.unitAt(nc, nr) && !g.buildingAt(nc, nr)
				}
				if valid {
					visited[nc][nr] = true
					queue = append(queue, [3]int{nc, nr, dist + 1})
				}
			}
		}
	}
	return result
}

// Handlers

func (g *GameImpl) MapHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		g.logger.Error("Failed to encode map response", "error", err)
		http.Error(w, fmt.Errorf("failed to encode map response: %w", err).Error(), http.StatusInternalServerError)
		return
	}
	g.logger.Info("Generated map", "cols", cols, "rows", rows, "duration", time.Since(start))
}

func (g *GameImpl) GameHandler(w http.ResponseWriter, r *http.Request) {
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
		g.state = newGameState(cols, rows)
	} else if g.state == nil {
		// Initialize gameState on first load with default size
		g.state = newGameState(cols, rows)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(g.state); err != nil {
		g.logger.Error("Failed to encode game state", "error", err)
		http.Error(w, fmt.Errorf("failed to encode game state: %w", err).Error(), http.StatusInternalServerError)
		return
	}
}

func (g *GameImpl) MoveHandler(w http.ResponseWriter, r *http.Request) {
	if g.state == nil {
		g.logger.Warn("Game state is nil")
		http.Error(w, "Game not initialized", http.StatusBadRequest)
		return
	}
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.logger.Error("Bad request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	g.logger.Info("Move request", "type", req.Type, "from", fmt.Sprintf("(%d,%d)", req.FromCol, req.FromRow), "to", fmt.Sprintf("(%d,%d)", req.ToCol, req.ToRow))
	if err := g.Move(r.Context(), req); err != nil {
		g.logger.Error("Move failed", "error", err)
		http.Error(w, fmt.Errorf("move failed: %w", err).Error(), http.StatusBadRequest)
		return
	}
	g.broadcastGameState()
	w.WriteHeader(http.StatusOK)
}

func (g *GameImpl) PlaceHandler(w http.ResponseWriter, r *http.Request) {
	if g.state == nil {
		g.logger.Warn("Game state is nil")
		http.Error(w, "Game not initialized", http.StatusBadRequest)
		return
	}
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.logger.Error("Bad request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	g.logger.Info("Place request", "type", req.Type, "to", fmt.Sprintf("(%d,%d)", req.ToCol, req.ToRow))
	if err := g.Place(r.Context(), req); err != nil {
		g.logger.Error("Place failed", "error", err)
		http.Error(w, fmt.Errorf("place failed: %w", err).Error(), http.StatusBadRequest)
		return
	}
	g.broadcastGameState()
	w.WriteHeader(http.StatusOK)
}

// Placeholder for other handlers
func (g *GameImpl) MoveRangeHandler(w http.ResponseWriter, r *http.Request) {
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
		g.logger.Error("Bad request for move range", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	tiles := g.getMoveRange(req.Col, req.Row, req.Range, req.UnitType)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Resp{Tiles: tiles}); err != nil {
		g.logger.Error("Failed to encode move range", "error", err)
		http.Error(w, fmt.Errorf("failed to encode move range: %w", err).Error(), http.StatusInternalServerError)
		return
	}
}

func (g *GameImpl) EndTurnHandler(w http.ResponseWriter, r *http.Request) {
	if g.state == nil {
		http.Error(w, "Game not initialized", http.StatusBadRequest)
		return
	}
	// Reset all units' Moved flag
	for i := range g.state.Units {
		g.state.Units[i].Moved = false
	}
	// Produce resources
	for i, p := range g.state.Players {
		if p.ID == g.state.CurrentPlayer {
			for _, b := range g.state.Buildings {
				if b.Owner == p.ID {
					config := BuildingConfigs[b.Type]
					g.state.Players[i].Gold += config.ProductionGold
					g.state.Players[i].Wood += config.ProductionWood
					g.state.Players[i].Iron += config.ProductionIron
				}
			}
			// Produce research
			g.state.Players[i].Research += 5 // 5 research per turn
			break
		}
	}
	g.state.Turn++
	// Cycle to next player
	g.state.CurrentPlayer++
	if g.state.CurrentPlayer > len(g.state.Players) {
		g.state.CurrentPlayer = 1
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(g.state); err != nil {
		g.logger.Error("Failed to encode game state after end turn", "error", err)
		http.Error(w, fmt.Errorf("failed to encode game state after end turn: %w", err).Error(), http.StatusInternalServerError)
		return
	}
	g.broadcastGameState()
}

func (g *GameImpl) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	// Add client to the clients map
	g.clientsMutex.Lock()
	g.clients[conn] = true
	g.clientsMutex.Unlock()

	g.logger.Info("WebSocket client connected", "remote_addr", r.RemoteAddr)

	// Send initial game state
	if err := conn.WriteJSON(g.state); err != nil {
		g.logger.Error("Failed to send initial game state", "error", err)
		conn.Close()
		return
	}

	// Handle client disconnection
	defer func() {
		g.clientsMutex.Lock()
		delete(g.clients, conn)
		g.clientsMutex.Unlock()
		conn.Close()
		g.logger.Info("WebSocket client disconnected", "remote_addr", r.RemoteAddr)
	}()

	// Keep connection alive and handle any incoming messages (though we don't expect any from clients)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// broadcastGameState sends the current game state to all connected WebSocket clients
func (g *GameImpl) broadcastGameState() {
	g.clientsMutex.Lock()
	defer g.clientsMutex.Unlock()

	for conn := range g.clients {
		if err := conn.WriteJSON(g.state); err != nil {
			g.logger.Error("Failed to broadcast game state", "error", err)
			conn.Close()
			delete(g.clients, conn)
		}
	}
}

func (g *GameImpl) JoinHandler(w http.ResponseWriter, r *http.Request) {
	if g.state == nil {
		g.state = newGameState(MapCols, MapRows)
	}

	type JoinRequest struct {
		Name string `json:"name"`
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.logger.Error("Bad join request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Find the next available player ID
	nextPlayerID := 1
	usedIDs := make(map[int]bool)
	for _, p := range g.state.Players {
		usedIDs[p.ID] = true
	}
	for usedIDs[nextPlayerID] {
		nextPlayerID++
	}

	// Create new player
	newPlayer := Player{
		ID:       nextPlayerID,
		Name:     req.Name,
		Color:    g.getPlayerColor(nextPlayerID),
		Capital:  g.getPlayerCapital(nextPlayerID),
		Gold:     100,
		Wood:     50,
		Iron:     20,
		Research: 0,
	}

	g.state.Players = append(g.state.Players, newPlayer)

	// If this is the first player, make them the current player
	if len(g.state.Players) == 1 {
		g.state.CurrentPlayer = nextPlayerID
	}

	// Create initial city and unit for the new player at their capital
	c := newPlayer.Capital
	g.logger.Info("Creating capital", "player", nextPlayerID, "pos", c)

	g.state.Buildings = append(g.state.Buildings, Building{
		Col: c[0], Row: c[1], Owner: nextPlayerID, Level: 1, Type: "city",
	})
	g.state.Units = append(g.state.Units, Unit{
		Col: c[0], Row: c[1], Moved: false, Owner: nextPlayerID,
		Type: "troop", Tier: 1, Health: 5, Attack: 2, Defense: 1,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"playerId":  nextPlayerID,
		"color":     newPlayer.Color,
		"capital":   newPlayer.Capital,
		"gameState": g.state,
	}); err != nil {
		g.logger.Error("Failed to encode join response", "error", err)
		http.Error(w, fmt.Errorf("failed to encode join response: %w", err).Error(), http.StatusInternalServerError)
		return
	}
}

func (g *GameImpl) getPlayerColor(playerID int) string {
	colors := []string{"#ff0000", "#0000ff", "#00ff00", "#ffff00", "#ff00ff", "#00ffff", "#ffa500", "#800080"}
	return colors[(playerID-1)%len(colors)]
}

func (g *GameImpl) getPlayerCapital(playerID int) [2]int {
	// Scan the map for available land positions and choose one based on player ID
	var landPositions [][2]int

	// Scan the entire map for land tiles that don't have buildings or units
	for col := 0; col < g.state.Cols; col++ {
		for row := 0; row < g.state.Rows; row++ {
			if g.state.Tiles[col][row].Type == "land" && !g.unitAt(col, row) && !g.buildingAt(col, row) {
				landPositions = append(landPositions, [2]int{col, row})
			}
		}
	}

	if len(landPositions) == 0 {
		g.logger.Warn("No land positions available for capital", "player", playerID)
		return [2]int{0, 0} // fallback
	}

	// Choose positions within preferred quadrants but avoid edges/corners
	var preferredPositions [][2]int
	margin := g.state.Cols / 8 // Avoid positions too close to edges

	switch playerID % 4 {
	case 1: // Top-left quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > margin && pos[0] < g.state.Cols/2-margin && pos[1] > margin && pos[1] < g.state.Rows/2-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 2: // Top-right quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > g.state.Cols/2+margin && pos[0] < g.state.Cols-margin && pos[1] > margin && pos[1] < g.state.Rows/2-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 3: // Bottom-left quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > margin && pos[0] < g.state.Cols/2-margin && pos[1] > g.state.Rows/2+margin && pos[1] < g.state.Rows-margin {
				preferredPositions = append(preferredPositions, pos)
			}
		}
	case 0: // Bottom-right quadrant, but not too close to edges
		for _, pos := range landPositions {
			if pos[0] > g.state.Cols/2+margin && pos[0] < g.state.Cols-margin && pos[1] > g.state.Rows/2+margin && pos[1] < g.state.Rows-margin {
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
		if pos[0] > margin && pos[0] < g.state.Cols-margin && pos[1] > margin && pos[1] < g.state.Rows-margin {
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
