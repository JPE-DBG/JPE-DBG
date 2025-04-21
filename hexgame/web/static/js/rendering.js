import { ROWS, COLS, gameState, moveRange, zoom, offsetX, offsetY } from './state.js';
import * as perfMeasurement from './perfMeasurement.js';

export function drawGrid(ctx, canvas) {
    // Start performance monitoring
    perfMeasurement.monitorFrameStart();
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    
    if (!gameState) return;
    
    const minX = Math.floor(-offsetX / hexSize / 1.5) - 2;
    const maxX = Math.ceil((canvas.width - offsetX) / hexSize / 1.5) + 2;
    const minY = Math.floor(-offsetY / hexHeight) - 2;
    const maxY = Math.ceil((canvas.height - offsetY) / hexHeight) + 2;
    
    // Draw map tiles
    for (let col = 0; col < COLS; col++) {
        if (col < minX || col > maxX) continue;
        for (let row = 0; row < ROWS; row++) {
            if (row < minY || row > maxY) continue;
            
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Skip if offscreen
            if (y + hexSize < 0 || y - hexSize > canvas.height) continue;
            
            let tileType = gameState.tiles[col][row].type;
            let color = '#222';
            if (tileType === 'land') color = '#81c784';
            else if (tileType === 'water') color = '#1976d2';
            else if (tileType === 'void') color = '#111';
            
            drawHex(x, y, hexSize, color, ctx);
            
            // Track tile render for performance measurement
            perfMeasurement.trackTileRender();
        }
    }
    
    // Draw move range
    for (let i = 0; i < moveRange.length; i++) {
        const {col, row} = moveRange[i];
        if (col < minX || col > maxX || row < minY || row > maxY) continue;
        let x = hexSize * 1.5 * col + offsetX + margin;
        let y = hexHeight * row + offsetY + margin;
        if (col % 2 !== 0) y += hexHeight / 2;
        drawHexOutline(x, y, hexSize, '#ffffff', 3, ctx);
    }
    
    // Draw units and buildings
    for (let col = 0; col < COLS; col++) {
        if (col < minX || col > maxX) continue;
        for (let row = 0; row < ROWS; row++) {
            if (row < minY || row > maxY) continue;
            
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Skip if offscreen
            if (y + hexSize < 0 || y - hexSize > canvas.height) continue;
            
            let tileType = gameState.tiles[col][row].type;
            if (tileType === 'land') {
                const unit = gameState.units.find(u => u.col === col && u.row === row);
                const building = gameState.buildings.find(b => b.col === col && b.row === row);
                if (unit) {
                    drawUnit(x, y, hexSize, unit.moved, unit.owner === gameState.currentPlayer, ctx);
                } else if (building) {
                    drawBuilding(x, y, hexSize, ctx);
                }
            }
        }
    }
    
    // End performance monitoring
    perfMeasurement.monitorFrameEnd();
    
    // Render performance overlay if needed
    perfMeasurement.renderPerfOverlay(ctx);
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