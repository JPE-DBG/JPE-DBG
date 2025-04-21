import { ROWS, COLS, gameState, moveRange, zoom, offsetX, offsetY } from './state.js';
import * as perfMeasurement from './perfMeasurement.js';

// Pre-computed hex vertices for better performance
let hexPoints = [];
for (let i = 0; i < 6; i++) {
    const angle = Math.PI / 3 * i;
    hexPoints.push({
        x: Math.cos(angle),
        y: Math.sin(angle)
    });
}

export function drawGrid(ctx, canvas) {
    // Start performance monitoring
    perfMeasurement.monitorFrameStart();
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    
    if (!gameState) return;
    
    // More precise culling calculations
    const minX = Math.max(0, Math.floor((-offsetX - margin) / (hexSize * 1.5)) - 1);
    const maxX = Math.min(COLS - 1, Math.ceil((canvas.width - offsetX + margin) / (hexSize * 1.5)) + 1);
    const minY = Math.max(0, Math.floor((-offsetY - margin - hexHeight/2) / hexHeight) - 1);
    const maxY = Math.min(ROWS - 1, Math.ceil((canvas.height - offsetY + margin) / hexHeight) + 1);
    
    // Group tiles by type for batch rendering
    const landTiles = [];
    const waterTiles = [];
    const voidTiles = [];
    
    // First pass: collect tiles by type
    for (let col = minX; col <= maxX; col++) {
        for (let row = minY; row <= maxY; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Fast bounds check
            if (x + hexSize < 0 || x - hexSize > canvas.width || 
                y + hexSize < 0 || y - hexSize > canvas.height) continue;
            
            // Track tile for rendering
            const tileType = gameState.tiles[col][row].type;
            const pos = { x, y, col, row };
            
            if (tileType === 'land') landTiles.push(pos);
            else if (tileType === 'water') waterTiles.push(pos);
            else if (tileType === 'void') voidTiles.push(pos);
            
            // Track tile render for performance measurement
            perfMeasurement.trackTileRender();
        }
    }
    
    // Second pass: batch render tiles by type
    // Render void tiles
    if (voidTiles.length > 0) {
        ctx.fillStyle = '#111';
        ctx.strokeStyle = '#222';
        batchDrawHexes(ctx, voidTiles, hexSize);
    }
    
    // Render water tiles
    if (waterTiles.length > 0) {
        ctx.fillStyle = '#1976d2';
        ctx.strokeStyle = '#222';
        batchDrawHexes(ctx, waterTiles, hexSize);
    }
    
    // Render land tiles
    if (landTiles.length > 0) {
        ctx.fillStyle = '#81c784';
        ctx.strokeStyle = '#222';
        batchDrawHexes(ctx, landTiles, hexSize);
    }
    
    // Draw move range with batch approach
    if (moveRange.length > 0) {
        ctx.save();
        ctx.lineWidth = 3;
        ctx.strokeStyle = '#ffffff';
        
        const moveRangeTiles = [];
        for (let i = 0; i < moveRange.length; i++) {
            const {col, row} = moveRange[i];
            if (col < minX || col > maxX || row < minY || row > maxY) continue;
            
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Skip if offscreen
            if (x + hexSize < 0 || x - hexSize > canvas.width || 
                y + hexSize < 0 || y - hexSize > canvas.height) continue;
            
            moveRangeTiles.push({ x, y });
        }
        
        batchDrawHexOutlines(ctx, moveRangeTiles, hexSize);
        ctx.restore();
    }
    
    // Draw units and buildings - We keep this approach as these are fewer and varied
    for (let col = minX; col <= maxX; col++) {
        for (let row = minY; row <= maxY; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Skip if offscreen
            if (x + hexSize < 0 || x - hexSize > canvas.width || 
                y + hexSize < 0 || y - hexSize > canvas.height) continue;
            
            let tileType = gameState.tiles[col][row].type;
            if (tileType === 'land') {
                // Use indexOf instead of find for better performance
                const unit = findUnitAt(col, row);
                const building = findBuildingAt(col, row);
                
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

// Utility functions for faster entity lookup
function findUnitAt(col, row) {
    for (let i = 0; i < gameState.units.length; i++) {
        const unit = gameState.units[i];
        if (unit.col === col && unit.row === row) return unit;
    }
    return null;
}

function findBuildingAt(col, row) {
    for (let i = 0; i < gameState.buildings.length; i++) {
        const building = gameState.buildings[i];
        if (building.col === col && building.row === row) return building;
    }
    return null;
}

// Draw multiple hexes with the same style in a single batch
function batchDrawHexes(ctx, positions, size) {
    ctx.beginPath();
    
    for (let i = 0; i < positions.length; i++) {
        const { x, y } = positions[i];
        
        // Draw first point
        let px = x + size * hexPoints[0].x;
        let py = y + size * hexPoints[0].y;
        ctx.moveTo(px, py);
        
        // Draw remaining points
        for (let j = 1; j < 6; j++) {
            px = x + size * hexPoints[j].x;
            py = y + size * hexPoints[j].y;
            ctx.lineTo(px, py);
        }
        
        // Close this hex
        ctx.closePath();
    }
    
    // Fill and stroke all hexes at once
    ctx.fill();
    ctx.stroke();
}

// Draw multiple hex outlines with the same style in a single batch
function batchDrawHexOutlines(ctx, positions, size) {
    ctx.beginPath();
    
    for (let i = 0; i < positions.length; i++) {
        const { x, y } = positions[i];
        
        // Draw first point
        let px = x + size * hexPoints[0].x;
        let py = y + size * hexPoints[0].y;
        ctx.moveTo(px, py);
        
        // Draw remaining points
        for (let j = 1; j < 6; j++) {
            px = x + size * hexPoints[j].x;
            py = y + size * hexPoints[j].y;
            ctx.lineTo(px, py);
        }
        
        // Close this hex
        ctx.closePath();
    }
    
    // Stroke all hex outlines at once
    ctx.stroke();
}

// Legacy method kept for compatibility but optimized
export function drawHex(x, y, size, color, ctx) {
    ctx.beginPath();
    
    for (let i = 0; i < 6; i++) {
        const px = x + size * hexPoints[i].x;
        const py = y + size * hexPoints[i].y;
        if (i === 0) ctx.moveTo(px, py);
        else ctx.lineTo(px, py);
    }
    
    ctx.closePath();
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = '#222';
    ctx.stroke();
}

// Legacy method kept for compatibility but optimized
export function drawHexOutline(x, y, size, outlineColor, lineWidth, ctx) {
    ctx.save();
    ctx.beginPath();
    
    for (let i = 0; i < 6; i++) {
        const px = x + size * hexPoints[i].x;
        const py = y + size * hexPoints[i].y;
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