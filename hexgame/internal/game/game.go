package game

var (
	MapCols = 50
	MapRows = 50
)

type Tile struct {
	Type string `json:"type"`
}

type Unit struct {
	Col     int    `json:"col"`
	Row     int    `json:"row"`
	Moved   bool   `json:"moved"`
	Owner   int    `json:"owner"`
	Type    string `json:"type"`   // "ship" or "troop"
	Tier    int    `json:"tier"`   // 1: basic, 2: advanced, 3: elite
	Health  int    `json:"health"` // max 10 for ships, 5 for troops
	Attack  int    `json:"attack"`
	Defense int    `json:"defense"`
}

type Building struct {
	Col   int    `json:"col"`
	Row   int    `json:"row"`
	Owner int    `json:"owner"`
	Level int    `json:"level"`
	Type  string `json:"type"` // "city", "port", "fort"
}

type Player struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Capital  [2]int `json:"capital"`
	Gold     int    `json:"gold"`
	Wood     int    `json:"wood"`
	Iron     int    `json:"iron"`
	Research int    `json:"research"`
}

type GameState struct {
	Cols          int        `json:"cols"`
	Rows          int        `json:"rows"`
	Tiles         [][]Tile   `json:"tiles"`
	Units         []Unit     `json:"units"`
	Buildings     []Building `json:"buildings"`
	Players       []Player   `json:"players"`
	Turn          int        `json:"turn"`
	CurrentPlayer int        `json:"currentPlayer"`
}

func newGameState(cols, rows int) *GameState {
	tiles := generateMapV3(cols, rows)
	players := []Player{} // Start with no default players
	units := []Unit{}
	buildings := []Building{}
	return &GameState{
		Cols:          cols,
		Rows:          rows,
		Tiles:         tiles,
		Units:         units,
		Buildings:     buildings,
		Players:       players,
		Turn:          1,
		CurrentPlayer: 1, // Will be updated when first player joins
	}
}

type UnitStats struct {
	Health  int
	Attack  int
	Defense int
}

type UnitConfig struct {
	CostGold     int
	CostWood     int
	CostIron     int
	CostResearch int
	Stats        UnitStats
}

type BuildingConfig struct {
	CostGold       int
	CostWood       int
	CostIron       int
	CostResearch   int
	ProductionGold int
	ProductionWood int
	ProductionIron int
}

var UnitConfigs = map[string]map[int]UnitConfig{
	"troop": {
		1: {CostGold: 10, CostWood: 0, CostIron: 0, CostResearch: 0, Stats: UnitStats{Health: 5, Attack: 2, Defense: 1}},
		2: {CostGold: 20, CostWood: 0, CostIron: 5, CostResearch: 50, Stats: UnitStats{Health: 8, Attack: 4, Defense: 2}},
	},
	"ship": {
		1: {CostGold: 0, CostWood: 20, CostIron: 0, CostResearch: 0, Stats: UnitStats{Health: 10, Attack: 3, Defense: 2}},
		2: {CostGold: 0, CostWood: 40, CostIron: 10, CostResearch: 50, Stats: UnitStats{Health: 15, Attack: 5, Defense: 3}},
	},
}

var BuildingConfigs = map[string]BuildingConfig{
	"city": {CostGold: 50, CostWood: 0, CostIron: 0, CostResearch: 0, ProductionGold: 10, ProductionWood: 0, ProductionIron: 0},
	"port": {CostGold: 30, CostWood: 10, CostIron: 0, CostResearch: 0, ProductionGold: 0, ProductionWood: 5, ProductionIron: 0},
	"fort": {CostGold: 40, CostWood: 0, CostIron: 5, CostResearch: 0, ProductionGold: 0, ProductionWood: 0, ProductionIron: 3},
}
