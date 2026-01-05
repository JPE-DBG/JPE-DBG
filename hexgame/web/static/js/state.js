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

// Viewport state for culling optimization
let viewportState = {
    minVisibleCol: 0,
    maxVisibleCol: 0,
    minVisibleRow: 0,
    maxVisibleRow: 0,
    lastZoom: 1,
    lastWidth: 0,
    lastHeight: 0
};

// Cache for hex positions
let hexPositionCache = new Map();

// Persistence with localStorage
export function saveGameState() {
    try {
        const stateToSave = {
            ROWS, COLS, mapState, selectedBarType, selectedTile, moveRange, zoom, offsetX, offsetY, isPanning, panStart, mapData, gameState, mapCenteredOnce
        };
        localStorage.setItem('hexgame_state', JSON.stringify(stateToSave));
    } catch (e) {
        console.error('Failed to save game state:', e);
    }
}

export function loadGameState() {
    try {
        const saved = localStorage.getItem('hexgame_state');
        if (saved) {
            const parsed = JSON.parse(saved);
            // Update individual variables
            ROWS = parsed.ROWS || 0;
            COLS = parsed.COLS || 0;
            mapState = parsed.mapState || [];
            selectedBarType = parsed.selectedBarType || null;
            selectedTile = parsed.selectedTile || null;
            moveRange = parsed.moveRange || [];
            zoom = parsed.zoom || 1;
            offsetX = parsed.offsetX || 0;
            offsetY = parsed.offsetY || 0;
            isPanning = parsed.isPanning || false;
            panStart = parsed.panStart || {x: 0, y: 0, ox: 0, oy: 0};
            mapData = parsed.mapData || null;
            gameState = parsed.gameState || null;
            mapCenteredOnce = parsed.mapCenteredOnce || false;
        }
    } catch (e) {
        console.error('Failed to load game state:', e);
    }
}

export async function fetchGame(regen = false, callback) {
    try {
        const url = regen ? '/api/game?regen=1' : '/api/game';
        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to fetch game');
        const newGameState = await response.json();
        setGameState(newGameState);
        if (callback) callback();
    } catch (error) {
        console.error('Error fetching game:', error);
    }
}

export function setSelectedBarType(type) { selectedBarType = type; }
export function setSelectedTile(tile) { selectedTile = tile; }
export function setMoveRange(range) { moveRange = range; }

// Optimized zoom handling with position preservation
export function setZoom(newZoom, centerX, centerY) {
    const oldZoom = zoom;
    zoom = newZoom;
    
    // If center point provided, maintain that point's position after zoom
    if (centerX !== undefined && centerY !== undefined) {
        offsetX = centerX - (centerX - offsetX) * (newZoom / oldZoom);
        offsetY = centerY - (centerY - offsetY) * (newZoom / oldZoom);
    }
    
    // Clear position cache when zoom changes
    hexPositionCache.clear();
}

export function setOffset(x, y) {
    offsetX = x;
    offsetY = y;
    // Update visible bounds when offset changes
    updateVisibleBounds();
}

export function setPanning(pan) { isPanning = pan; }
export function setPanStart(obj) { panStart = obj; }
export function setMapData(data) { mapData = data; }
export function setGameState(state) { 
    gameState = state; 
    COLS = gameState.cols;
    ROWS = gameState.rows;
    mapData = { tiles: gameState.tiles };
    
    // Clear caches when map changes
    hexPositionCache.clear();
    updateVisibleBounds();
}
export function clearHexPositionCache() {
    hexPositionCache.clear();
}
export function setMapSize(cols, rows) { COLS = cols; ROWS = rows; }
export function setMapCenteredOnce(val) { mapCenteredOnce = val; }

// Update UI elements based on current game state
function updateGameUI() {
    if (!gameState) return;
    
    // Update UI input fields with current map size
    const colsInput = document.getElementById('mapCols');
    const rowsInput = document.getElementById('mapRows');
    if (colsInput) colsInput.value = COLS;
    if (rowsInput) rowsInput.value = ROWS;

    // Update turn and player info
    const turnInfo = document.getElementById('turnInfo');
    const playerInfo = document.getElementById('playerInfo');
    const resourcesInfo = document.getElementById('resourcesInfo');
    if (turnInfo) turnInfo.textContent = `Turn: ${gameState.turn}`;
    if (playerInfo && gameState.players) {
        const currentPlayer = gameState.players.find(p => p.id === gameState.currentPlayer);
        if (currentPlayer) {
            const isCurrentUser = currentPlayer.id === window.currentPlayerId;
            playerInfo.textContent = `${isCurrentUser ? 'Your' : currentPlayer.name + "'s"} turn (Player ${currentPlayer.id})`;
        }
    }
    if (resourcesInfo && gameState.players && window.currentPlayerId) {
        const myPlayer = gameState.players.find(p => p.id === window.currentPlayerId);
        if (myPlayer) {
            resourcesInfo.textContent = `Gold: ${myPlayer.gold} Wood: ${myPlayer.wood} Iron: ${myPlayer.iron} Research: ${myPlayer.research}`;
        }
    }
}

// Get cached or calculate hex position
export function getHexPosition(col, row) {
    const key = `${col},${row},${zoom}`;
    if (hexPositionCache.has(key)) {
        return hexPositionCache.get(key);
    }
    
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight; // Match the margin used in rendering.js
    let x = hexSize * 1.5 * col + offsetX + margin;
    let y = hexHeight * row + offsetY + margin;
    if (col % 2 !== 0) y += hexHeight / 2;
    
    const pos = { x, y };
    hexPositionCache.set(key, pos);
    return pos;
}

// Update visible bounds for culling optimization
function updateVisibleBounds() {
    if (!gameState) return;
    
    const canvas = document.getElementById('hexCanvas');
    if (!canvas) return;
    
    // Only recalculate if view parameters changed significantly
    if (zoom === viewportState.lastZoom && 
        canvas.width === viewportState.lastWidth && 
        canvas.height === viewportState.lastHeight) {
        return;
    }
    
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight;
    
    // Calculate visible columns and rows with margin
    viewportState.minVisibleCol = Math.max(0, Math.floor((-offsetX - margin) / (hexSize * 1.5)));
    viewportState.maxVisibleCol = Math.min(COLS - 1, Math.ceil((canvas.width - offsetX + margin) / (hexSize * 1.5)));
    viewportState.minVisibleRow = Math.max(0, Math.floor((-offsetY - margin) / hexHeight));
    viewportState.maxVisibleRow = Math.min(ROWS - 1, Math.ceil((canvas.height - offsetY + margin) / hexHeight));
    
    viewportState.lastZoom = zoom;
    viewportState.lastWidth = canvas.width;
    viewportState.lastHeight = canvas.height;
}

export function getVisibleBounds() {
    return viewportState;
}

// Optimized hex hit detection
export function getHexAt(mx, my) {
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight; // Match the margin used in rendering.js
    
    if (!mapData || !mapData.tiles || !gameState || !gameState.tiles) return null;
    
    // Quick bounds check using visible area
    const bounds = getVisibleBounds();
    for (let col = bounds.minVisibleCol; col <= bounds.maxVisibleCol; col++) {
        for (let row = bounds.minVisibleRow; row <= bounds.maxVisibleRow; row++) {
            // Check array bounds to avoid errors
            if (col < 0 || col >= COLS || row < 0 || row >= ROWS) continue;
            if (!gameState.tiles[col] || !gameState.tiles[col][row]) continue;
            
            const pos = getHexPosition(col, row);
            if (pointInHex(mx, my, pos.x, pos.y, hexSize)) {
                return {col, row};
            }
        }
    }
    return null;
}

// Fetch game state without updating the input fields
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

    // Clear caches when map changes
    hexPositionCache.clear();
    updateVisibleBounds();

    if (typeof scheduleDrawGrid === 'function') scheduleDrawGrid();
}

// More precise hex hit detection
export function pointInHex(mx, my, cx, cy, size) {
    // Use squared distance for faster calculation (avoiding Math.sqrt)
    const dx = mx - cx;
    const dy = my - cy;
    const distSquared = dx * dx + dy * dy;
    return distSquared <= size * size;
}

// Calculate hex distance between two tiles
export function hexDistance(col1, row1, col2, row2) {
    const dx = col2 - col1;
    const dy = row2 - row1;
    // For hex grid with even rows offset, distance is max(|dx|, |dy|, |dx + dy|)
    return Math.max(Math.abs(dx), Math.abs(dy), Math.abs(dx + dy));
}