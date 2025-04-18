package main

import (
	"math"
	"math/rand"
	"time"
)

func generateMap(cols, rows int) [][]Tile {
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	centerX := float64(cols) / 2
	centerY := float64(rows) / 2
	maxDist := math.Min(centerX, centerY) * 0.95
	noiseScale := 0.13

	tiles := make([][]Tile, cols)
	for x := 0; x < cols; x++ {
		tiles[x] = make([]Tile, rows)
		for y := 0; y < rows; y++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > maxDist {
				tiles[x][y] = Tile{Type: "void"}
				continue
			}
			val := 0.5 + 0.5*math.Sin(float64(x)*noiseScale+float64(y)*noiseScale+float64(seed%1000))
			val += rng.Float64()*0.2 - 0.1
			if val > 0.55 {
				tiles[x][y] = Tile{Type: "land"}
			} else if val > 0.45 {
				tiles[x][y] = Tile{Type: "water"}
			} else {
				tiles[x][y] = Tile{Type: "void"}
			}
		}
	}
	ensureLandConnected(tiles, cols, rows)
	return tiles
}

// generateMapV2 creates a single large, connected continent with natural edges and a minimum land ratio.
func generateMapV2(cols, rows int) [][]Tile {
	const minLandRatio = 0.20 // at least 20% of map should be land
	maxAttempts := 15
	for attempt := 0; attempt < maxAttempts; attempt++ {
		tiles := make([][]Tile, cols)
		for x := 0; x < cols; x++ {
			tiles[x] = make([]Tile, rows)
			for y := 0; y < rows; y++ {
				tiles[x][y] = Tile{Type: "water"}
			}
		}
		// Step 1: Seed a central landmass using random walk
		centerX, centerY := cols/2, rows/2
		landCount := 0
		maxLand := int(float64(cols*rows) * 0.6) // up to 60% land
		x, y := centerX, centerY
		tiles[x][y].Type = "land"
		landCount++
		for i := 0; i < maxLand; i++ {
			dir := rand.Intn(6)
			// Hex directions (flat-topped)
			dx, dy := 0, 0
			switch dir {
			case 0:
				dx = 1
			case 1:
				dx = -1
			case 2:
				dy = 1
			case 3:
				dy = -1
			case 4:
				dx, dy = 1, (x%2)*2-1 // up-right or down-right
			case 5:
				dx, dy = -1, (x%2)*2-1 // up-left or down-left
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				x, y = nx, ny
				if tiles[x][y].Type != "land" {
					tiles[x][y].Type = "land"
					landCount++
				}
			}
		}
		// Step 2: Add noise to edges for natural shape
		for x := 0; x < cols; x++ {
			for y := 0; y < rows; y++ {
				if tiles[x][y].Type == "land" {
					for _, n := range hexNeighbors(x, y, cols, rows) {
						nx, ny := n[0], n[1]
						if tiles[nx][ny].Type == "water" && rand.Float64() < 0.25 {
							tiles[nx][ny].Type = "land"
							landCount++
						}
					}
				}
			}
		}
		// Step 3: Flood fill from center to ensure connectivity
		visited := make([][]bool, cols)
		for i := range visited {
			visited[i] = make([]bool, rows)
		}
		queue := [][2]int{{centerX, centerY}}
		visited[centerX][centerY] = true
		connectedLand := 1
		for len(queue) > 0 {
			cx, cy := queue[0][0], queue[0][1]
			queue = queue[1:]
			for _, n := range hexNeighbors(cx, cy, cols, rows) {
				nx, ny := n[0], n[1]
				if !visited[nx][ny] && tiles[nx][ny].Type == "land" {
					visited[nx][ny] = true
					queue = append(queue, [2]int{nx, ny})
					connectedLand++
				}
			}
		}
		// Convert unconnected land to water
		for x := 0; x < cols; x++ {
			for y := 0; y < rows; y++ {
				if tiles[x][y].Type == "land" && !visited[x][y] {
					tiles[x][y].Type = "water"
					landCount--
				}
			}
		}
		// Step 4: Set void outside a circular map
		centerXF, centerYF := float64(centerX), float64(centerY)
		maxDist := 0.95 * (float64(cols) + float64(rows)) / 4
		for x := 0; x < cols; x++ {
			for y := 0; y < rows; y++ {
				dx, dy := float64(x)-centerXF, float64(y)-centerYF
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > maxDist {
					tiles[x][y].Type = "void"
				}
			}
		}
		// Step 5: Check land ratio
		if float64(landCount)/float64(cols*rows) >= minLandRatio {
			return tiles
		}
	}
	// fallback: return all water
	fallback := make([][]Tile, cols)
	for x := 0; x < cols; x++ {
		fallback[x] = make([]Tile, rows)
		for y := 0; y < rows; y++ {
			fallback[x][y] = Tile{Type: "water"}
		}
	}
	return fallback
}

// generateMapV3 creates a connected continent, no void mask, and no random void holes.
func generateMapV3(cols, rows int) [][]Tile {
	const minLandRatio = 0.20
	maxAttempts := 15
	for attempt := 0; attempt < maxAttempts; attempt++ {
		tiles := make([][]Tile, cols)
		for x := 0; x < cols; x++ {
			tiles[x] = make([]Tile, rows)
			for y := 0; y < rows; y++ {
				tiles[x][y] = Tile{Type: "water"}
			}
		}
		// Step 1: Seed a central landmass using random walk
		centerX, centerY := cols/2, rows/2
		landCount := 0
		maxLand := int(float64(cols*rows) * 0.6)
		x, y := centerX, centerY
		tiles[x][y].Type = "land"
		landCount++
		for i := 0; i < maxLand; i++ {
			dir := rand.Intn(6)
			dx, dy := 0, 0
			switch dir {
			case 0:
				dx = 1
			case 1:
				dx = -1
			case 2:
				dy = 1
			case 3:
				dy = -1
			case 4:
				dx, dy = 1, (x%2)*2-1
			case 5:
				dx, dy = -1, (x%2)*2-1
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				x, y = nx, ny
				if tiles[x][y].Type != "land" {
					tiles[x][y].Type = "land"
					landCount++
				}
			}
		}
		// Step 2: Add noise to edges for natural shape
		for x := 0; x < cols; x++ {
			for y := 0; y < rows; y++ {
				if tiles[x][y].Type == "land" {
					for _, n := range hexNeighbors(x, y, cols, rows) {
						nx, ny := n[0], n[1]
						if tiles[nx][ny].Type == "water" && rand.Float64() < 0.25 {
							tiles[nx][ny].Type = "land"
							landCount++
						}
					}
				}
			}
		}
		// Step 3: Flood fill from center to ensure connectivity
		visited := make([][]bool, cols)
		for i := range visited {
			visited[i] = make([]bool, rows)
		}
		queue := [][2]int{{centerX, centerY}}
		visited[centerX][centerY] = true
		connectedLand := 1
		for len(queue) > 0 {
			cx, cy := queue[0][0], queue[0][1]
			queue = queue[1:]
			for _, n := range hexNeighbors(cx, cy, cols, rows) {
				nx, ny := n[0], n[1]
				if !visited[nx][ny] && tiles[nx][ny].Type == "land" {
					visited[nx][ny] = true
					queue = append(queue, [2]int{nx, ny})
					connectedLand++
				}
			}
		}
		// Convert unconnected land to water
		for x := 0; x < cols; x++ {
			for y := 0; y < rows; y++ {
				if tiles[x][y].Type == "land" && !visited[x][y] {
					tiles[x][y].Type = "water"
					landCount--
				}
			}
		}
		// Step 4: Check land ratio
		if float64(landCount)/float64(cols*rows) >= minLandRatio {
			return tiles
		}
	}
	// fallback: return all water
	fallback := make([][]Tile, cols)
	for x := 0; x < cols; x++ {
		fallback[x] = make([]Tile, rows)
		for y := 0; y < rows; y++ {
			fallback[x][y] = Tile{Type: "water"}
		}
	}
	return fallback
}

func ensureLandConnected(tiles [][]Tile, cols, rows int) {
	visited := make([][]bool, cols)
	for i := range visited {
		visited[i] = make([]bool, rows)
	}
	var sx, sy int
	found := false
	for x := 0; x < cols && !found; x++ {
		for y := 0; y < rows && !found; y++ {
			if tiles[x][y].Type == "land" {
				sx, sy = x, y
				found = true
			}
		}
	}
	if !found {
		return
	}
	queue := [][2]int{{sx, sy}}
	visited[sx][sy] = true
	for len(queue) > 0 {
		cx, cy := queue[0][0], queue[0][1]
		queue = queue[1:]
		for _, n := range hexNeighbors(cx, cy, cols, rows) {
			nx, ny := n[0], n[1]
			if !visited[nx][ny] && tiles[nx][ny].Type == "land" {
				visited[nx][ny] = true
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			if tiles[x][y].Type == "land" && !visited[x][y] {
				tiles[x][y].Type = "water"
			}
		}
	}
}

func hexNeighbors(x, y, cols, rows int) [][2]int {
	even := x%2 == 0
	neighbors := [][2]int{
		{x + 1, y}, {x - 1, y}, {x, y + 1}, {x, y - 1},
	}
	if even {
		neighbors = append(neighbors, [2]int{x + 1, y - 1}, [2]int{x - 1, y - 1})
	} else {
		neighbors = append(neighbors, [2]int{x + 1, y + 1}, [2]int{x - 1, y + 1})
	}
	var result [][2]int
	for _, n := range neighbors {
		nx, ny := n[0], n[1]
		if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
			result = append(result, [2]int{nx, ny})
		}
	}
	return result
}
