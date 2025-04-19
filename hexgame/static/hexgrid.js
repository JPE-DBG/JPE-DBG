const canvas = document.getElementById('hexCanvas');
const ctx = canvas.getContext('2d');

const HEX_SIZE = 30; // radius
const HEX_HEIGHT = Math.sqrt(3) * HEX_SIZE;
const HEX_WIDTH = 2 * HEX_SIZE;
let ROWS = 0;
let COLS = 0;

// --- Map State ---
// Now each unit is an object: {type: 'unit', moved: false} or {type: 'building'}
const mapState = [];
for (let col = 0; col < COLS; col++) {
    mapState[col] = [];
    for (let row = 0; row < ROWS; row++) {
        mapState[col][row] = null;
    }
}

let selectedBarType = null; // 'unit' or 'building'
let selectedTile = null; // {col, row} or null
let moveRange = [];

// --- Map View State ---
let zoom = 1;
let offsetX = 0;
let offsetY = 0;
let isPanning = false;
let panStart = {x: 0, y: 0, ox: 0, oy: 0};

const FIXED_HEX_SIZE = 30;
const FIXED_HEX_HEIGHT = Math.sqrt(3) * FIXED_HEX_SIZE;
const FIXED_HEX_WIDTH = 2 * FIXED_HEX_SIZE;

let mapData = null;
let gameState = null;
let mapCenteredOnce = false;

async function initGame() {
    await fetchGame();
    centerMapView();
    mapCenteredOnce = true;
    drawGrid();
}

// Fetch map from Go backend
async function fetchMap() {
    const res = await fetch('/api/map');
    mapData = await res.json();
    COLS = mapData.cols;
    ROWS = mapData.rows;

    // Initialize mapState for units/buildings
    for (let col = 0; col < COLS; col++) {
        mapState[col] = [];
        for (let row = 0; row < ROWS; row++) {
            mapState[col][row] = null;
        }
    }
    drawGrid();
}

// Fetch game state from Go backend
async function fetchGame() {
    const res = await fetch('/api/game');
    gameState = await res.json();
    COLS = gameState.cols;
    ROWS = gameState.rows;
    mapData = { tiles: gameState.tiles };
    drawGrid();
}

// --- UI: Bottom Bar Selection ---
document.querySelectorAll('.icon-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');
        selectedBarType = btn.dataset.type;
        console.log('Selected bar type:', selectedBarType); // DEBUG
        selectedTile = null;
        moveRange = [];
        drawGrid();
    });
});

document.getElementById('nextTurnBtn').addEventListener('click', async () => {
    await fetch('/api/endturn', { method: 'POST' });
    await fetchGame();
    selectedTile = null;
    moveRange = [];
    drawGrid();
});

document.getElementById('regenMapBtn').addEventListener('click', async () => {
    const cols = parseInt(document.getElementById('mapCols').value, 10) || 100;
    const rows = parseInt(document.getElementById('mapRows').value, 10) || 100;
    await fetch(`/api/game?regen=1&cols=${cols}&rows=${rows}`); // pass new size to backend
    await fetchGame();
    selectedTile = null;
    moveRange = [];
    drawGrid();
});

// --- Canvas Click: Place, Select, or Move ---
canvas.addEventListener('click', async (e) => {
    if (!gameState) return;
    const rect = canvas.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    const tile = getHexAt(mx, my);
    console.log('Clicked tile:', tile); // DEBUG
    if (!tile) return;
    const {col, row} = tile;
    const unit = gameState.units.find(u => u.col === col && u.row === row);
    const building = gameState.buildings.find(b => b.col === col && b.row === row);
    if (selectedBarType) {
        let type = selectedBarType === 'unit' ? 'place_unit' : 'place_building';
        console.log('Placing:', type, 'at', col, row); // DEBUG
        const resp = await fetch('/api/move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type, toCol: col, toRow: row })
        });
        console.log('API response status:', resp.status); // DEBUG
        selectedBarType = null;
        document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
        await fetchGame();
        selectedTile = {col, row};
        moveRange = [];
        drawGrid();
    } else if (unit) {
        selectedTile = {col, row};
        // Only show move range if unit has not moved and is owned by current player
        if (!unit.moved && unit.owner === gameState.currentPlayer) {
            // Fetch move range from backend
            const res = await fetch('/api/move-range', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ col, row, range: 5 })
            });
            const data = await res.json();
            moveRange = data.tiles.map(([c, r]) => ({col: c, row: r}));
        } else {
            moveRange = [];
        }
        drawGrid();
    } else if (selectedTile && moveRange.length > 0 && !unit && !building) {
        const inRange = moveRange.some(t => t.col === col && t.row === row);
        if (inRange) {
            await fetch('/api/move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type: 'move', fromCol: selectedTile.col, fromRow: selectedTile.row, toCol: col, toRow: row })
            });
            await fetchGame();
            selectedTile = {col, row};
            moveRange = [];
            drawGrid();
        }
    } else {
        selectedTile = null;
        moveRange = [];
        drawGrid();
    }
});

// --- Mouse Wheel Zoom ---
canvas.addEventListener('wheel', (e) => {
    e.preventDefault();
    const prevZoom = zoom;
    if (e.deltaY < 0) zoom *= 1.1;
    else zoom /= 1.1;
    zoom = Math.max(0.1, Math.min(2.5, zoom));
    // Zoom to mouse position
    const rect = canvas.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    offsetX = (offsetX - mx) * (zoom / prevZoom) + mx;
    offsetY = (offsetY - my) * (zoom / prevZoom) + my;
    drawGrid();
}, { passive: false });

// --- Right Click and Drag Pan ---
canvas.addEventListener('mousedown', (e) => {
    if (e.button === 2) {
        isPanning = true;
        panStart = {x: e.clientX, y: e.clientY, ox: offsetX, oy: offsetY};
        e.preventDefault();
    }
});
window.addEventListener('mousemove', (e) => {
    if (isPanning) {
        offsetX = panStart.ox + (e.clientX - panStart.x);
        offsetY = panStart.oy + (e.clientY - panStart.y);
        drawGrid();
    }
});
window.addEventListener('mouseup', (e) => {
    if (e.button === 2) {
        isPanning = false;
    }
});
canvas.addEventListener('contextmenu', e => e.preventDefault());

// --- Hex Picking ---
function getHexAt(mx, my) {
    const hexSize = FIXED_HEX_SIZE * zoom;
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
function pointInHex(mx, my, cx, cy, size) {
    return Math.hypot(mx - cx, my - cy) < size;
}

// --- Draw only visible tiles ---
function drawHexOutline(x, y, size, outlineColor, lineWidth = 4) {
    ctx.save();
    ctx.beginPath();
    for (let i = 0; i < 6; i++) {
        const angle = Math.PI / 3 * i;
        const px = x + size * Math.cos(angle);
        const py = y + size * Math.sin(angle);
        if (i === 0) ctx.moveTo(px, py);
        else ctx.lineTo(px, py);
    }
    ctx.closePath();
    ctx.lineWidth = lineWidth;
    ctx.strokeStyle = outlineColor;
    ctx.stroke();
    ctx.restore();
}

function drawGrid() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const hexSize = FIXED_HEX_SIZE * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    if (!gameState) return;
    // Calculate visible bounds
    const minX = -offsetX / hexSize / 1.5 - 2;
    const maxX = (canvas.width - offsetX) / hexSize / 1.5 + 2;
    const minY = -offsetY / hexHeight - 2;
    const maxY = (canvas.height - offsetY) / hexHeight + 2;
    // Pass 1: Draw all tile bases
    for (let col = 0; col < COLS; col++) {
        if (col < minX || col > maxX) continue;
        for (let row = 0; row < ROWS; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            if (y + hexSize < 0 || y - hexSize > canvas.height) continue;
            let tileType = gameState.tiles[col][row].type;
            //let color = '#222';
            if (tileType === 'land') color = '#81c784';
            else if (tileType === 'water') color = '#1976d2';
            else if (tileType === 'void') color = '#111';

            // Only the selected tile gets the yellow fill
            // if (selectedTile && selectedTile.col === col && selectedTile.row === row) {
            //     color = '#ffd54f';
            // }
            drawHex(x, y, hexSize, color);
        }
    }
    // Pass 2: Draw orange outline for move range (no fill, no fill color set)
    for (let i = 0; i < moveRange.length; i++) {
        const {col, row} = moveRange[i];
        if (col < minX || col > maxX || row < minY || row > maxY) continue;
        let x = hexSize * 1.5 * col + offsetX + margin;
        let y = hexHeight * row + offsetY + margin;
        if (col % 2 !== 0) y += hexHeight / 2;
        //drawHexOutline(x, y, hexSize, '#ff9800', 4);
        drawHexOutline(x, y, hexSize, '#ffffff', 3);
    }
    // Pass 3: Draw units/buildings only on land
    for (let col = 0; col < COLS; col++) {
        if (col < minX || col > maxX) continue;
        for (let row = 0; row < ROWS; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            if (y + hexSize < 0 || y - hexSize > canvas.height) continue;
            let tileType = gameState.tiles[col][row].type;
            if (tileType === 'land') {
                const unit = gameState.units.find(u => u.col === col && u.row === row);
                const building = gameState.buildings.find(b => b.col === col && b.row === row);
                if (unit) {
                    drawUnit(x, y, hexSize, unit.moved, unit.owner === gameState.currentPlayer);
                } else if (building) {
                    drawBuilding(x, y, hexSize);
                }
            }
        }
    }
}

function drawHex(x, y, size, color) {
    ctx.beginPath();
    for (let i = 0; i < 6; i++) {
        const angle = Math.PI / 3 * i;
        const px = x + size * Math.cos(angle);
        const py = y + size * Math.sin(angle);
        if (i === 0) ctx.moveTo(px, py);
        else ctx.lineTo(px, py);
    }
    ctx.closePath();
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = '#222';
    ctx.stroke();
}

function drawUnit(x, y, size, moved, isCurrentPlayer) {
    ctx.beginPath();
    ctx.arc(x, y, size/3, 0, 2*Math.PI);
    ctx.fillStyle = moved ? '#bdbdbd' : (isCurrentPlayer ? '#e53935' : '#888');
    ctx.fill();
    ctx.strokeStyle = '#fff';
    ctx.stroke();
}
function drawBuilding(x, y, size) {
    ctx.beginPath();
    ctx.rect(x-size/4, y-size/4, size/2, size/2);
    ctx.fillStyle = '#3949ab';
    ctx.fill();
    ctx.strokeStyle = '#fff';
    ctx.stroke();
}

// Remove grid scaling and resize logic, keep canvas size fixed to window
function resizeCanvas() {
    const bar = document.getElementById('bottom-bar');
    const barRect = bar ? bar.getBoundingClientRect() : { height: 35, top: 0 };
    // Set canvas height so it ends exactly at the top of the bottom bar
    canvas.width = window.innerWidth;
    canvas.height = bar ? barRect.top : (window.innerHeight - barRect.height);
    if (!mapCenteredOnce) {
        centerMapView();
        mapCenteredOnce = true;
    }
    drawGrid();
}

function centerMapView() {
    const hexSize = FIXED_HEX_SIZE * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    // Center of map in pixels
    const mapCenterX = (COLS * 1.5 * hexSize) / 2 + margin;
    const mapCenterY = (ROWS * hexHeight) / 2 + margin;
    // Center of canvas in pixels
    const canvasCenterX = canvas.width / 2;
    const canvasCenterY = canvas.height / 2;
    offsetX = canvasCenterX - mapCenterX;
    offsetY = canvasCenterY - mapCenterY;
}

window.addEventListener('resize', resizeCanvas);

// On load, call initGame instead of fetchGame
initGame();
resizeCanvas();
