package game

var (
	MapCols = 500
	MapRows = 500
)

// SetInitialMapSize sets the initial map dimensions
func SetInitialMapSize(cols, rows int) {
	MapCols = cols
	MapRows = rows
}

type Tile struct {
	Type string `json:"type"`
}

type Unit struct {
	Col   int  `json:"col"`
	Row   int  `json:"row"`
	Moved bool `json:"moved"`
	Owner int  `json:"owner"`
}

type Building struct {
	Col   int `json:"col"`
	Row   int `json:"row"`
	Owner int `json:"owner"`
	Level int `json:"level"`
}

type GameState struct {
	Cols          int        `json:"cols"`
	Rows          int        `json:"rows"`
	Tiles         [][]Tile   `json:"tiles"`
	Units         []Unit     `json:"units"`
	Buildings     []Building `json:"buildings"`
	Turn          int        `json:"turn"`
	CurrentPlayer int        `json:"currentPlayer"`
}

var gameState *GameState

func newGameState(cols, rows int) *GameState {
	tiles := generateMapV3(cols, rows)
	return &GameState{
		Cols:          cols,
		Rows:          rows,
		Tiles:         tiles,
		Units:         []Unit{},
		Buildings:     []Building{},
		Turn:          1,
		CurrentPlayer: 1,
	}
}

func unitAt(col, row int) bool {
	for _, u := range gameState.Units {
		if u.Col == col && u.Row == row {
			return true
		}
	}
	return false
}

func buildingAt(col, row int) bool {
	for _, b := range gameState.Buildings {
		if b.Col == col && b.Row == row {
			return true
		}
	}
	return false
}

// Returns a list of [col, row] pairs for valid move range
func getMoveRange(col, row, rng int) [][2]int {
	if gameState == nil {
		return nil
	}
	visited := make([][]bool, gameState.Cols)
	for i := range visited {
		visited[i] = make([]bool, gameState.Rows)
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
		for _, n := range hexNeighbors(c, r, gameState.Cols, gameState.Rows) {
			nc, nr := n[0], n[1]
			if !visited[nc][nr] && gameState.Tiles[nc][nr].Type == "land" && !unitAt(nc, nr) && !buildingAt(nc, nr) {
				visited[nc][nr] = true
				queue = append(queue, [3]int{nc, nr, dist + 1})
			}
		}
	}
	return result
}
