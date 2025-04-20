import { ROWS, COLS, gameState, moveRange, zoom, offsetX, offsetY, selectedTile } from './state.js';

let fps = 0;
let lastFrameTime = performance.now();
let frameCount = 0;
let droppedFrames = 0;
let lastFpsReport = performance.now();

export function drawGrid(ctx, canvas) {
    // --- FPS & dropped frame logic ---
    const now = performance.now();
    frameCount++;
    if (now - lastFrameTime > 25) droppedFrames++;
    lastFrameTime = now;
    if (now - lastFpsReport > 1000) {
        fps = frameCount;
        frameCount = 0;
        lastFpsReport = now;
    }
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    if (!gameState) return;

    // --- Optimization: Precompute lookup maps --- 
    const unitMap = new Map(gameState.units.map(u => [`${u.col}_${u.row}`, u]));
    const buildingMap = new Map(gameState.buildings.map(b => [`${b.col}_${b.row}`, b]));
    // -------------------------------------------

    // Rough culling based on indices (keep this as a first pass)
    const minCol = Math.floor((-offsetX - margin - hexSize) / (hexSize * 1.5));
    const maxCol = Math.ceil((canvas.width - offsetX - margin + hexSize) / (hexSize * 1.5));
    const minRow = Math.floor((-offsetY - margin - hexHeight) / hexHeight);
    const maxRow = Math.ceil((canvas.height - offsetY - margin + hexHeight) / hexHeight);

    // Draw Tiles
    for (let col = Math.max(0, minCol); col < Math.min(COLS, maxCol); col++) {
        for (let row = Math.max(0, minRow); row < Math.min(ROWS, maxRow); row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;

            // --- Optimization: More precise viewport culling (commented out for now, can add if needed) ---
            // if (x + hexSize < 0 || x - hexSize > canvas.width || y + hexHeight < 0 || y - hexHeight > canvas.height) {
            //     continue; // Skip drawing if hex is completely off-screen
            // }
            // -----------------------------------------------------------------------------------------

            let tileType = gameState.tiles[col]?.[row]?.type; // Add safe navigation
            if (!tileType) continue; // Skip if tile data is missing

            let color = '#222';
            if (tileType === 'land') color = '#81c784';
            else if (tileType === 'water') color = '#1976d2';
            else if (tileType === 'void') color = '#111';
            drawHex(x, y, hexSize, color, ctx);
        }
    }

    // Draw Move Range (after tiles, before units/buildings)
    ctx.globalAlpha = 0.5; // Make move range semi-transparent
    for (let i = 0; i < moveRange.length; i++) {
        const {col, row} = moveRange[i];
        // Apply similar culling as tiles
        if (col < Math.max(0, minCol) || col >= Math.min(COLS, maxCol) || row < Math.max(0, minRow) || row >= Math.min(ROWS, maxRow)) continue;

        let x = hexSize * 1.5 * col + offsetX + margin;
        let y = hexHeight * row + offsetY + margin;
        if (col % 2 !== 0) y += hexHeight / 2;

        // Optional: Add precise culling here too if needed
        // if (x + hexSize < 0 || x - hexSize > canvas.width || y + hexHeight < 0 || y - hexHeight > canvas.height) continue;

        drawHexOutline(x, y, hexSize, '#ffffff', 3, ctx);
    }
    ctx.globalAlpha = 1.0; // Reset alpha

    // Draw Selection Outline (if a tile is selected)
    if (selectedTile) {
        const {col, row} = selectedTile;
        if (col >= Math.max(0, minCol) && col < Math.min(COLS, maxCol) && row >= Math.max(0, minRow) && row < Math.min(ROWS, maxRow)) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            // Optional: Add precise culling here too if needed
            drawHexOutline(x, y, hexSize, '#ffff00', 4, ctx); // Yellow outline for selection
        }
    }

    // Draw Units and Buildings (using the maps)
    for (let col = Math.max(0, minCol); col < Math.min(COLS, maxCol); col++) {
        for (let row = Math.max(0, minRow); row < Math.min(ROWS, maxRow); row++) {
            let tileType = gameState.tiles[col]?.[row]?.type;
            if (tileType !== 'land') continue; // Only draw units/buildings on land

            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;

            // Optional: Add precise culling here too if needed
            // if (x + hexSize < 0 || x - hexSize > canvas.width || y + hexHeight < 0 || y - hexHeight > canvas.height) continue;

            // --- Optimization: Use maps for lookup ---
            const unit = unitMap.get(`${col}_${row}`);
            const building = buildingMap.get(`${col}_${row}`);
            // ----------------------------------------

            if (unit) {
                drawUnit(x, y, hexSize, unit.moved, unit.owner === gameState.currentPlayer, ctx);
            } else if (building) {
                drawBuilding(x, y, hexSize, ctx);
            }
        }
    }

    renderFpsCounter(ctx);
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