import * as state from './state.js'; // Import the whole state module

let fps = 0;
let lastFrameTime = performance.now();
let frameCount = 0;
let droppedFrames = 0;
let lastFpsReport = performance.now();

// --- Removed renderOffscreenMap function ---
// -----------------------------------------

export function drawGrid(ctx, canvas) {
    const now = performance.now();
    // --- FPS & dropped frame logic ---
    frameCount++;
    if (now - lastFrameTime > 25) droppedFrames++;
    lastFrameTime = now;
    if (now - lastFpsReport > 1000) {
        fps = frameCount;
        frameCount = 0;
        lastFpsReport = now;
    }

    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const hexSize = 30 * state.zoom; // Current zoom for visible canvas
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    if (!state.gameState) return;

    // --- Removed Offscreen Canvas Rendering Logic ---
    // ---------------------------------------------

    // --- Restore Original Tile Drawing Loop with Culling ---
    const unitMap = new Map(state.gameState.units.map(u => [`${u.col}_${u.row}`, u]));
    const buildingMap = new Map(state.gameState.buildings.map(b => [`${b.col}_${b.row}`, b]));

    const minCol = Math.floor((-state.offsetX - margin - hexSize) / (hexSize * 1.5));
    const maxCol = Math.ceil((canvas.width - state.offsetX - margin + hexSize) / (hexSize * 1.5));
    const minRow = Math.floor((-state.offsetY - margin - hexHeight) / hexHeight);
    const maxRow = Math.ceil((canvas.height - state.offsetY - margin + hexHeight) / hexHeight);

    // Draw Tiles
    for (let col = Math.max(0, minCol); col < Math.min(state.COLS, maxCol); col++) {
        for (let row = Math.max(0, minRow); row < Math.min(state.ROWS, maxRow); row++) {
            let x = hexSize * 1.5 * col + state.offsetX + margin;
            let y = hexHeight * row + state.offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;

            let tileType = state.gameState.tiles[col]?.[row]?.type;
            if (!tileType) continue;

            let color = '#222';
            if (tileType === 'land') color = '#81c784';
            else if (tileType === 'water') color = '#1976d2';
            else if (tileType === 'void') color = '#111';
            drawHex(x, y, hexSize, color, ctx);
        }
    }
    // ------------------------------------------------------

    // --- Draw Dynamic Elements (Move Range, Selection, Units, Buildings) ---
    // These still need to be drawn every frame on the main canvas
    // Use the *current* zoom and offset for positioning these elements.

    // Precompute maps for dynamic elements (still useful)
    ctx.globalAlpha = 0.5;
    for (let i = 0; i < state.moveRange.length; i++) {
        const {col, row} = state.moveRange[i];
        if (col < Math.max(0, minCol) || col >= Math.min(state.COLS, maxCol) || row < Math.max(0, minRow) || row >= Math.min(state.ROWS, maxRow)) continue;
        let x = hexSize * 1.5 * col + state.offsetX + margin;
        let y = hexHeight * row + state.offsetY + margin;
        if (col % 2 !== 0) y += hexHeight / 2;
        drawHexOutline(x, y, hexSize, '#ffffff', 3, ctx);
    }
    ctx.globalAlpha = 1.0;

    // Draw Selection Outline
    if (state.selectedTile) {
        const {col, row} = state.selectedTile;
        if (col >= Math.max(0, minCol) && col < Math.min(state.COLS, maxCol) && row >= Math.max(0, minRow) && row < Math.min(state.ROWS, maxRow)) {
            let x = hexSize * 1.5 * col + state.offsetX + margin;
            let y = hexHeight * row + state.offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            drawHexOutline(x, y, hexSize, '#ffff00', 4, ctx);
        }
    }

    // Draw Units and Buildings
    for (let col = Math.max(0, minCol); col < Math.min(state.COLS, maxCol); col++) {
        for (let row = Math.max(0, minRow); row < Math.min(state.ROWS, maxRow); row++) {
            let tileType = state.gameState.tiles[col]?.[row]?.type;
            if (tileType !== 'land') continue;

            let x = hexSize * 1.5 * col + state.offsetX + margin;
            let y = hexHeight * row + state.offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;

            const unit = unitMap.get(`${col}_${row}`);
            const building = buildingMap.get(`${col}_${row}`);

            if (unit) {
                drawUnit(x, y, hexSize, unit.moved, unit.owner === state.gameState.currentPlayer, ctx);
            } else if (building) {
                drawBuilding(x, y, hexSize, ctx);
            }
        }
    }
    // ---------------------------------------------------------------------

    renderFpsCounter(ctx); // Draw FPS counter last
}

export function drawHex(x, y, size, color, ctx) {
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

export function drawHexOutline(x, y, size, outlineColor, lineWidth, ctx) {
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

export function drawUnit(x, y, size, moved, isCurrentPlayer, ctx) {
    ctx.beginPath();
    ctx.arc(x, y, size/3, 0, 2*Math.PI);
    
    // logic for determining fillStyle
    let unitColor;
    if (moved) {
        unitColor = '#bdbdbd'; // Grey if moved
    } else if (isCurrentPlayer) {
        unitColor = '#e53935'; // Red if current player's unit and hasn't moved
    } else {
        unitColor = '#888';    // Darker grey for other players' units that haven't moved
    }
    ctx.fillStyle = unitColor;
    
    ctx.fill();
    ctx.strokeStyle = '#fff';
    ctx.stroke();
}

export function drawBuilding(x, y, size, ctx) {
    ctx.beginPath();
    ctx.rect(x-size/4, y-size/4, size/2, size/2);
    ctx.fillStyle = '#3949ab';
    ctx.fill();
    ctx.strokeStyle = '#fff';
    ctx.stroke();
}

export function renderFpsCounter(ctx) {
    ctx.save();
    ctx.font = '16px monospace';
    ctx.fillStyle = 'rgba(0,0,0,0.7)';
    ctx.fillRect(10, 10, 120, 40);
    ctx.fillStyle = '#fff';
    ctx.fillText(`FPS: ${fps}`, 20, 30);
    ctx.fillText(`Dropped: ${droppedFrames}`, 20, 50);
    ctx.restore();
}