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

// Add offscreen canvas for double buffering
let offscreenCanvas = document.createElement('canvas');
let offscreenCtx = offscreenCanvas.getContext('2d');
let lastDrawnState = {
    offsetX: 0,
    offsetY: 0,
    zoom: 1,
    visibleTiles: new Set(), // Track visible tiles for optimization
    lastDrawnTiles: new Map() // Cache of last drawn positions
};

// Resize offscreen canvas
export function resizeOffscreenCanvas(width, height) {
    offscreenCanvas.width = width;
    offscreenCanvas.height = height;
    // Clear cache when resizing
    lastDrawnState.visibleTiles.clear();
    lastDrawnState.lastDrawnTiles.clear();
}

export function drawGrid(ctx, canvas) {
    // Start performance monitoring
    perfMeasurement.monitorFrameStart();
    
    // Ensure offscreen canvas matches main canvas size
    if (offscreenCanvas.width !== canvas.width || offscreenCanvas.height !== canvas.height) {
        resizeOffscreenCanvas(canvas.width, canvas.height);
    }
    
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight; // Increased margin for smoother scrolling
    
    if (!gameState) return;

    // Calculate scroll delta
    const deltaX = offsetX - lastDrawnState.offsetX;
    const deltaY = offsetY - lastDrawnState.offsetY;
    const zoomChanged = zoom !== lastDrawnState.zoom;
    
    if (!zoomChanged && Math.abs(deltaX) < canvas.width && Math.abs(deltaY) < canvas.height) {
        // Shift existing content
        offscreenCtx.save();
        offscreenCtx.globalCompositeOperation = 'copy';
        offscreenCtx.drawImage(offscreenCanvas, deltaX, deltaY);
        offscreenCtx.restore();
        
        // Calculate regions that need redrawing
        const regions = [];
        if (deltaX !== 0) {
            if (deltaX > 0) {
                regions.push({x: 0, y: 0, width: deltaX, height: canvas.height}); // Left strip
            } else {
                regions.push({x: canvas.width + deltaX, y: 0, width: -deltaX, height: canvas.height}); // Right strip
            }
        }
        if (deltaY !== 0) {
            if (deltaY > 0) {
                regions.push({x: 0, y: 0, width: canvas.width, height: deltaY}); // Top strip
            } else {
                regions.push({x: 0, y: canvas.height + deltaY, width: canvas.width, height: -deltaY}); // Bottom strip
            }
        }
        
        // Clear and redraw only the regions that need updating
        regions.forEach(region => {
            offscreenCtx.clearRect(region.x, region.y, region.width, region.height);
            drawVisibleHexesInRegion(region.x, region.y, region.width, region.height);
        });
    } else {
        // Complete redraw needed
        offscreenCtx.clearRect(0, 0, canvas.width, canvas.height);
        drawVisibleHexesInRegion(0, 0, canvas.width, canvas.height);
    }
    
    // Update last drawn state
    lastDrawnState.offsetX = offsetX;
    lastDrawnState.offsetY = offsetY;
    lastDrawnState.zoom = zoom;
    
    // Copy to main canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(offscreenCanvas, 0, 0);
    
    // Draw dynamic elements (units, buildings, move range) directly on main canvas
    drawDynamicElements(ctx, canvas);
    
    // End performance monitoring
    perfMeasurement.monitorFrameEnd();
    
    // Render performance overlay if needed
    perfMeasurement.renderPerfOverlay(ctx);
}

// Draw visible hexes in a specific region
function drawVisibleHexesInRegion(startX, startY, width, height) {
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight;
    
    // More precise culling calculations for the region
    const minX = Math.max(0, Math.floor((startX - offsetX - margin) / (hexSize * 1.5)));
    const maxX = Math.min(COLS - 1, Math.ceil((startX + width - offsetX + margin) / (hexSize * 1.5)));
    const minY = Math.max(0, Math.floor((startY - offsetY - margin) / hexHeight));
    const maxY = Math.min(ROWS - 1, Math.ceil((startY + height - offsetY + margin) / hexHeight));
    
    // Group tiles by type for batch rendering
    const tilesByType = {
        land: [],
        water: [],
        void: []
    };
    
    // Collect visible tiles
    const newVisibleTiles = new Set();
    
    for (let col = minX; col <= maxX; col++) {
        for (let row = minY; row <= maxY; row++) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Fast bounds check for the region
            if (x + hexSize < startX || x - hexSize > startX + width || 
                y + hexSize < startY || y - hexSize > startY + height) continue;
            
            // Track tile and prepare for rendering
            const tileId = `${col},${row}`;
            newVisibleTiles.add(tileId);
            
            const tileType = gameState.tiles[col][row].type;
            tilesByType[tileType].push({ x, y });
            
            // Track tile render for performance measurement
            perfMeasurement.trackTileRender();
        }
    }
    
    // Batch render tiles by type
    const colors = {
        void: ['#111', '#222'],
        water: ['#1976d2', '#222'],
        land: ['#81c784', '#222']
    };
    
    Object.entries(tilesByType).forEach(([type, tiles]) => {
        if (tiles.length > 0) {
            const [fillColor, strokeColor] = colors[type];
            offscreenCtx.fillStyle = fillColor;
            offscreenCtx.strokeStyle = strokeColor;
            batchDrawHexes(offscreenCtx, tiles, hexSize);
        }
    });
    
    // Update visible tiles tracking
    lastDrawnState.visibleTiles = newVisibleTiles;
}

// Draw units, buildings and move range
function drawDynamicElements(ctx, canvas) {
    const hexSize = 30 * zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    
    // Draw move range
    if (moveRange.length > 0) {
        ctx.save();
        ctx.lineWidth = 3;
        ctx.strokeStyle = '#ffffff';
        
        const moveRangeTiles = [];
        for (const {col, row} of moveRange) {
            let x = hexSize * 1.5 * col + offsetX + margin;
            let y = hexHeight * row + offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            if (isHexVisible(x, y, hexSize, canvas)) {
                moveRangeTiles.push({ x, y });
            }
        }
        
        batchDrawHexOutlines(ctx, moveRangeTiles, hexSize);
        ctx.restore();
    }
    
    // Draw units and buildings
    if (gameState.units.length > 0 || gameState.buildings.length > 0) {
        // Only process tiles that are potentially visible
        const visibleCols = Math.ceil(canvas.width / (hexSize * 1.5)) + 2;
        const visibleRows = Math.ceil(canvas.height / hexHeight) + 2;
        const centerCol = Math.floor(-offsetX / (hexSize * 1.5));
        const centerRow = Math.floor(-offsetY / hexHeight);
        
        const minCol = Math.max(0, centerCol - Math.floor(visibleCols / 2));
        const maxCol = Math.min(COLS - 1, centerCol + Math.ceil(visibleCols / 2));
        const minRow = Math.max(0, centerRow - Math.floor(visibleRows / 2));
        const maxRow = Math.min(ROWS - 1, centerRow + Math.ceil(visibleRows / 2));
        
        // Draw units
        for (const unit of gameState.units) {
            if (unit.col >= minCol && unit.col <= maxCol && unit.row >= minRow && unit.row <= maxRow) {
                let x = hexSize * 1.5 * unit.col + offsetX + margin;
                let y = hexHeight * unit.row + offsetY + margin;
                if (unit.col % 2 !== 0) y += hexHeight / 2;
                
                if (isHexVisible(x, y, hexSize, canvas)) {
                    drawUnit(x, y, hexSize, unit.moved, unit.owner === gameState.currentPlayer, ctx);
                }
            }
        }
        
        // Draw buildings
        for (const building of gameState.buildings) {
            if (building.col >= minCol && building.col <= maxCol && building.row >= minRow && building.row <= maxRow) {
                let x = hexSize * 1.5 * building.col + offsetX + margin;
                let y = hexHeight * building.row + offsetY + margin;
                if (building.col % 2 !== 0) y += hexHeight / 2;
                
                if (isHexVisible(x, y, hexSize, canvas)) {
                    drawBuilding(x, y, hexSize, ctx);
                }
            }
        }
    }
}

// Helper to check if a hex is visible on screen
function isHexVisible(x, y, size, canvas) {
    return x + size >= 0 && x - size <= canvas.width && 
           y + size >= 0 && y - size <= canvas.height;
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