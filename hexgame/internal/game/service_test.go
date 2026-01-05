package game

import (
	"testing"
)

func TestPlaceStrategies(t *testing.T) {
	tests := []struct {
		name        string
		strategy    string
		req         MoveRequest
		setup       func(*GameImpl)
		expectError bool
	}{
		{
			name:     "place troop on land",
			strategy: "place_troop",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Buildings = append(g.state.Buildings, Building{Col: 0, Row: 0, Type: "city", Owner: 0})
				g.state.Players[0].Gold = 10
				g.state.Players[0].Wood = 5
			},
			expectError: false,
		},
		{
			name:     "place troop insufficient resources",
			strategy: "place_troop",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Players[0].Gold = 0
			},
			expectError: true,
		},
		{
			name:     "place advanced troop with city",
			strategy: "place_advanced_troop",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Buildings = append(g.state.Buildings, Building{Col: 0, Row: 0, Type: "city", Owner: 0})
				g.state.Players[0].Gold = 20
				g.state.Players[0].Wood = 0
				g.state.Players[0].Iron = 5
				g.state.Players[0].Research = 50
			},
			expectError: false,
		},
		{
			name:     "place advanced troop without city",
			strategy: "place_advanced_troop",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Players[0].Gold = 50
			},
			expectError: true,
		},
		{
			name:     "place ship on water",
			strategy: "place_ship",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "water"
				g.state.Players[0].Gold = 20
				g.state.Players[0].Wood = 20
			},
			expectError: true,
		},
		{
			name:     "place ship at port",
			strategy: "place_ship",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Buildings = append(g.state.Buildings, Building{Col: 0, Row: 0, Type: "port", Owner: 0})
				g.state.Players[0].Gold = 10
				g.state.Players[0].Wood = 20
			},
			expectError: false,
		},
		{
			name:     "place advanced ship on water",
			strategy: "place_advanced_ship",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "water"
				g.state.Players[0].Gold = 50
				g.state.Players[0].Wood = 40
				g.state.Players[0].Iron = 10
				g.state.Players[0].Research = 50
			},
			expectError: true,
		},
		{
			name:     "place advanced ship at port",
			strategy: "place_advanced_ship",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Buildings = append(g.state.Buildings, Building{Col: 0, Row: 0, Type: "port", Owner: 0})
				g.state.Players[0].Gold = 20
				g.state.Players[0].Wood = 40
				g.state.Players[0].Iron = 10
				g.state.Players[0].Research = 50
			},
			expectError: false,
		},
		{
			name:     "place fort on land",
			strategy: "place_fort",
			req: MoveRequest{
				ToCol: 0,
				ToRow: 0,
			},
			setup: func(g *GameImpl) {
				g.state.Tiles[0][0].Type = "land"
				g.state.Players[0].Gold = 40
				g.state.Players[0].Wood = 5
				g.state.Players[0].Iron = 5
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGame()
			g.state.CurrentPlayer = 0
			g.state.Players = append(g.state.Players, Player{ID: 0, Gold: 0, Wood: 0, Iron: 0, Research: 0})
			tt.setup(g)
			strategy := g.placeStrategies[tt.strategy]
			if strategy == nil {
				t.Fatalf("strategy %s not found", tt.strategy)
			}
			err := strategy.Execute(g, tt.req)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
