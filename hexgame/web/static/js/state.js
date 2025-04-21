// --- State management for game, map, and UI ---
export let ROWS = 0, COLS = 0;
export let mapState = [];
export let selectedBarType = null;
export let selectedTile = null;
export let moveRange = [];
export let zoom = 1, offsetX = 0, offsetY = 0;
export let isPanning = false;
export let panStart = {x: 0, y: 0, ox: 0, oy: 0};
export let mapData = null, gameState = null, mapCenteredOnce = false;

export function setSelectedBarType(type) { selectedBarType = type; }
export function setSelectedTile(tile) { selectedTile = tile; }
export function setMoveRange(range) { moveRange = range; }
export function setZoom(z) { zoom = z; }
export function setOffset(x, y) { offsetX = x; offsetY = y; }
export function setPanning(pan) { isPanning = pan; }
export function setPanStart(obj) { panStart = obj; }
export function setMapData(data) { mapData = data; }
export function setGameState(state) { gameState = state; }
export function setMapCenteredOnce(val) { mapCenteredOnce = val; }

export async function fetchGame(draw = true, scheduleDrawGrid) {
    const res = await fetch('/api/game');
    gameState = await res.json();
    COLS = gameState.cols;
    ROWS = gameState.rows;
    mapData = { tiles: gameState.tiles };

    // Update UI input fields with current map size
    const colsInput = document.getElementById('mapCols');
    const rowsInput = document.getElementById('mapRows');
    if (colsInput) colsInput.value = COLS;
    if (rowsInput) rowsInput.value = ROWS;

    if (draw && typeof scheduleDrawGrid === 'function') scheduleDrawGrid();
}

export async function fetchMap(scheduleDrawGrid) {
    const res = await fetch('/api/map');
    mapData = await res.json();
    COLS = mapData.cols;
    ROWS = mapData.rows;
    for (let col = 0; col < COLS; col++) {
        mapState[col] = [];
        for (let row = 0; row < ROWS; row++) {
            mapState[col][row] = null;
        }
    }
    if (typeof scheduleDrawGrid === 'function') scheduleDrawGrid();
}

export function getHexAt(mx, my) {
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    if (!mapData) return null;
    for (let col = 0; col < COLS; col++) {
        for (let row = 0; row < ROWS; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            if (pointInHex(mx, my, x, y, hexSize)) {
                if (mapData.tiles[col][row].type === 'land') return {col, row};
                return null;
            }
        }
    }
    return null;
}

export function pointInHex(mx, my, cx, cy, size) {
    return Math.hypot(mx - cx, my - cy) < size;
}