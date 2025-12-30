// Import state as a module instead of importing specific variables
import * as state from './state.js';
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

// Make the offscreen canvas available globally for debugging/clearing
window.offscreenCanvas = document.createElement('canvas');
let offscreenCtx = window.offscreenCanvas.getContext('2d');
let lastDrawnState = {
    offsetX: 0,
    offsetY: 0,
    zoom: 1,
    visibleTiles: new Set(), // Track visible tiles for optimization
    lastDrawnTiles: new Map() // Cache of last drawn positions
};

// Resize offscreen canvas
export function resizeOffscreenCanvas(width, height) {
    window.offscreenCanvas.width = width;
    window.offscreenCanvas.height = height;
    // Clear cache when resizing
    lastDrawnState.visibleTiles.clear();
    lastDrawnState.lastDrawnTiles.clear();
}

export function drawGrid(ctx, canvas) {
    // Start performance monitoring
    perfMeasurement.monitorFrameStart();
    
    // Ensure offscreen canvas matches main canvas size
    if (window.offscreenCanvas.width !== canvas.width || window.offscreenCanvas.height !== canvas.height) {
        resizeOffscreenCanvas(canvas.width, canvas.height);
    }
    
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight; // Increased margin for smoother scrolling
    
    if (!state.gameState) return;

    // Calculate scroll delta
    const deltaX = state.offsetX - lastDrawnState.offsetX;
    const deltaY = state.offsetY - lastDrawnState.offsetY;
    const zoomChanged = state.zoom !== lastDrawnState.zoom;
    
    if (!zoomChanged && Math.abs(deltaX) < canvas.width && Math.abs(deltaY) < canvas.height) {
        // Shift existing content
        offscreenCtx.save();
        offscreenCtx.globalCompositeOperation = 'copy';
        offscreenCtx.drawImage(window.offscreenCanvas, deltaX, deltaY);
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
    lastDrawnState.offsetX = state.offsetX;
    lastDrawnState.offsetY = state.offsetY;
    lastDrawnState.zoom = state.zoom;
    
    // Copy to main canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(window.offscreenCanvas, 0, 0);
    
    // Draw dynamic elements (units, buildings, move range) directly on main canvas
    drawDynamicElements(ctx, canvas);
    
    // Draw attack highlights if unit selected
    if (state.selectedTile && state.gameState.units) {
        const selectedUnit = state.gameState.units.find(u => u.col === state.selectedTile.col && u.row === state.selectedTile.row);
        if (selectedUnit && !selectedUnit.moved && selectedUnit.owner === state.gameState.currentPlayer) {
            drawAttackHighlights(ctx, canvas, selectedUnit);
        }
    }
    
    // End performance monitoring
    perfMeasurement.monitorFrameEnd();
    
    // Render performance overlay if needed
    perfMeasurement.renderPerfOverlay(ctx);
}

// Draw visible hexes in a specific region
function drawVisibleHexesInRegion(startX, startY, width, height) {
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight;
    
    // More precise culling calculations for the region
    const minX = Math.max(0, Math.floor((startX - state.offsetX - margin) / (hexSize * 1.5)));
    const maxX = Math.min(state.COLS - 1, Math.ceil((startX + width - state.offsetX + margin) / (hexSize * 1.5)));
    const minY = Math.max(0, Math.floor((startY - state.offsetY - margin) / hexHeight));
    const maxY = Math.min(state.ROWS - 1, Math.ceil((startY + height - state.offsetY + margin) / hexHeight));
    
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
            let x = hexSize * 1.5 * col + state.offsetX + margin;
            let y = hexHeight * row + state.offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            // Fast bounds check for the region
            if (x + hexSize < startX || x - hexSize > startX + width || 
                y + hexSize < startY || y - hexSize > startY + height) continue;
            
            // Track tile and prepare for rendering
            const tileId = `${col},${row}`;
            newVisibleTiles.add(tileId);
            
            const tileType = state.gameState.tiles[col][row].type;
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
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight; // Match the margin used in state.js
    
    // Draw move range
    if (state.moveRange.length > 0) {
        ctx.save();
        ctx.lineWidth = 3;
        ctx.strokeStyle = '#ffffff';
        
        const moveRangeTiles = [];
        for (const {col, row} of state.moveRange) {
            let x = hexSize * 1.5 * col + state.offsetX + margin;
            let y = hexHeight * row + state.offsetY + margin;
            if (col % 2 !== 0) y += hexHeight / 2;
            
            if (isHexVisible(x, y, hexSize, canvas)) {
                moveRangeTiles.push({ x, y });
            }
        }
        
        batchDrawHexOutlines(ctx, moveRangeTiles, hexSize);
        ctx.restore();
    }
    
    // Draw units and buildings
    if (state.gameState.units.length > 0 || state.gameState.buildings.length > 0) {
        // Calculate visible area with correct margins
        const visibleCols = Math.ceil(canvas.width / (hexSize * 1.5)) + 4;
        const visibleRows = Math.ceil(canvas.height / hexHeight) + 4;

        // Calculate bounds using the same logic as drawVisibleHexesInRegion
        const minCol = Math.max(0, Math.floor((-state.offsetX - margin) / (hexSize * 1.5)));
        const maxCol = Math.min(state.COLS - 1, Math.ceil((canvas.width - state.offsetX + margin) / (hexSize * 1.5)));
        const minRow = Math.max(0, Math.floor((-state.offsetY - margin) / hexHeight));
        const maxRow = Math.min(state.ROWS - 1, Math.ceil((canvas.height - state.offsetY + margin) / hexHeight));

        // Debug: Draw culling viewport rectangle
        // ctx.save();
        // ctx.strokeStyle = 'rgba(255, 0, 0, 0.5)';
        // ctx.lineWidth = 2;
        // ctx.beginPath();
        // Convert grid coordinates to screen coordinates
        // const viewportLeft = hexSize * 1.5 * minCol + state.offsetX + margin;
        // const viewportTop = hexHeight * minRow + state.offsetY + margin;
        // const viewportWidth = hexSize * 1.5 * (maxCol - minCol + 1);
        // const viewportHeight = hexHeight * (maxRow - minRow + 1);
        // ctx.rect(viewportLeft, viewportTop, viewportWidth, viewportHeight);
        // ctx.stroke();
        // Add debug text
        // ctx.fillStyle = 'rgba(255, 0, 0, 0.8)';
        // ctx.font = '12px monospace';
        // ctx.fillText(`Viewport: cols ${minCol}-${maxCol}, rows ${minRow}-${maxRow}`, 10, canvas.height - 40);
        // ctx.fillText(`Center: col ${minCol + Math.floor((maxCol - minCol) / 2)}, row ${minRow + Math.floor((maxRow - minRow) / 2)}`, 10, canvas.height - 20);
        // ctx.restore();
        
        // Draw units with same bounds checking
        for (const unit of state.gameState.units) {
            if (unit.col >= minCol && unit.col <= maxCol && unit.row >= minRow && unit.row <= maxRow) {
                let x = hexSize * 1.5 * unit.col + state.offsetX + margin;
                let y = hexHeight * unit.row + state.offsetY + margin;
                if (unit.col % 2 !== 0) y += hexHeight / 2;
                
                if (isHexVisible(x, y, hexSize, canvas)) {
                    drawUnit(x, y, hexSize, unit.moved, unit.owner === state.gameState.currentPlayer, unit.type, unit.owner, unit.tier, ctx);
                }
            }
        }
        
        // Draw buildings with same bounds checking
        for (const building of state.gameState.buildings) {
            if (building.col >= minCol && building.col <= maxCol && building.row >= minRow && building.row <= maxRow) {
                let x = hexSize * 1.5 * building.col + state.offsetX + margin;
                let y = hexHeight * building.row + state.offsetY + margin;
                if (building.col % 2 !== 0) y += hexHeight / 2;
                
                if (isHexVisible(x, y, hexSize, canvas)) {
                    drawBuilding(x, y, hexSize, building.type, building.owner, building.level, ctx);
                }
            }
        }
    }
}

// Helper to check if a hex is visible on screen
function isHexVisible(x, y, size, canvas) {
    // Use 1.5x size to account for the full hex width and height
    const fullSize = size * 1.5;
    return x + fullSize >= 0 && x - fullSize <= canvas.width && 
           y + fullSize >= 0 && y - fullSize <= canvas.height;
}

// Utility functions for faster entity lookup
function findUnitAt(col, row) {
    for (let i = 0; i < state.gameState.units.length; i++) {
        const unit = state.gameState.units[i];
        if (unit.col === col && unit.row === row) return unit;
    }
    return null;
}

function findBuildingAt(col, row) {
    for (let i = 0; i < state.gameState.buildings.length; i++) {
        const building = state.gameState.buildings[i];
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

export function drawUnit(x, y, size, moved, isCurrentPlayer, unitType, owner, tier, ctx) {
    const unitTier = tier || 1; // Default to 1 if undefined
    const scale = 0.8 + unitTier * 0.2; // Larger for higher tiers
    const drawSize = size * scale;
    
    // Get player color
    let playerColor = '#888';
    if (state.gameState.players) {
        const player = state.gameState.players.find(p => p.id === owner);
        if (player) playerColor = player.color;
    }
    
    let unitColor;
    if (moved) {
        unitColor = '#bdbdbd'; // Grey if moved
    } else if (isCurrentPlayer) {
        // Blink for movable units
        const blink = Math.sin(Date.now() / 200) > 0;
        unitColor = blink ? playerColor : '#ffffff';
    } else {
        unitColor = playerColor; // Player color for other players
    }
    
    ctx.fillStyle = unitColor;
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 2;
    
    if (unitType === 'ship') {
        // Draw ship as a boat shape
        ctx.beginPath();
        ctx.moveTo(x - drawSize/2, y + drawSize/3);
        ctx.lineTo(x + drawSize/2, y + drawSize/3);
        ctx.lineTo(x + drawSize/3, y - drawSize/3);
        ctx.lineTo(x - drawSize/3, y - drawSize/3);
        ctx.closePath();
        ctx.fill();
        ctx.stroke();
        
        // Mast
        ctx.beginPath();
        ctx.moveTo(x, y - drawSize/3);
        ctx.lineTo(x, y - drawSize);
        ctx.stroke();
        
        // Sail
        ctx.fillStyle = '#fff';
        ctx.beginPath();
        ctx.moveTo(x, y - drawSize/3);
        ctx.lineTo(x + drawSize/4, y - drawSize/2);
        ctx.lineTo(x, y - drawSize);
        ctx.closePath();
        ctx.fill();
        
    } else {
        // Draw troop as a soldier shape
        ctx.beginPath();
        ctx.arc(x, y - drawSize/4, drawSize/4, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        
        // Body
        ctx.beginPath();
        ctx.rect(x - drawSize/6, y - drawSize/4, drawSize/3, drawSize/2);
        ctx.fill();
        ctx.stroke();
        
        // Legs
        ctx.beginPath();
        ctx.moveTo(x - drawSize/6, y + drawSize/4);
        ctx.lineTo(x - drawSize/4, y + drawSize/2);
        ctx.moveTo(x + drawSize/6, y + drawSize/4);
        ctx.lineTo(x + drawSize/4, y + drawSize/2);
        ctx.stroke();
    }
}

export function drawBuilding(x, y, size, buildingType, owner, level, ctx) {
    const buildingLevel = level || 1; // Default to 1 if undefined
    const scale = 0.8 + buildingLevel * 0.2;
    const drawSize = size * scale;
    
    // Get player color
    let playerColor = '#3949ab';
    if (state.gameState.players) {
        const player = state.gameState.players.find(p => p.id === owner);
        if (player) playerColor = player.color;
    }
    
    ctx.fillStyle = playerColor;
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 2;
    
    if (buildingType === 'city') {
        // Draw city as a castle
        ctx.beginPath();
        ctx.rect(x - drawSize/2, y - drawSize/4, drawSize, drawSize/2);
        ctx.fill();
        ctx.stroke();
        
        // Towers
        ctx.beginPath();
        ctx.rect(x - drawSize/2 - drawSize/6, y - drawSize/2, drawSize/6, drawSize/4);
        ctx.rect(x + drawSize/2, y - drawSize/2, drawSize/6, drawSize/4);
        ctx.fill();
        ctx.stroke();
        
    } else if (buildingType === 'port') {
        // Draw port as a dock
        ctx.beginPath();
        ctx.rect(x - drawSize/2, y - drawSize/4, drawSize, drawSize/4);
        ctx.fill();
        ctx.stroke();
        
        // Pier
        ctx.beginPath();
        ctx.moveTo(x - drawSize/2, y);
        ctx.lineTo(x + drawSize/2, y);
        ctx.lineTo(x + drawSize/3, y + drawSize/4);
        ctx.lineTo(x - drawSize/3, y + drawSize/4);
        ctx.closePath();
        ctx.fill();
        ctx.stroke();
        
    } else if (buildingType === 'fort') {
        // Draw fort as a fortress
        ctx.beginPath();
        ctx.rect(x - drawSize/2, y - drawSize/4, drawSize, drawSize/4);
        ctx.fill();
        ctx.stroke();
        
        // Walls
        ctx.beginPath();
        ctx.moveTo(x - drawSize/2, y - drawSize/4);
        ctx.lineTo(x - drawSize/2, y - drawSize/2);
        ctx.lineTo(x + drawSize/2, y - drawSize/2);
        ctx.lineTo(x + drawSize/2, y - drawSize/4);
        ctx.stroke();
        
        // Turrets
        ctx.beginPath();
        ctx.arc(x - drawSize/3, y - drawSize/2, drawSize/8, 0, Math.PI * 2);
        ctx.arc(x + drawSize/3, y - drawSize/2, drawSize/8, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
    }
    
    // Draw level indicator
    ctx.fillStyle = '#fff';
    ctx.font = `${Math.floor(drawSize / 3)}px Arial`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(`L${buildingLevel}`, x, y);
}

// Force a complete redraw by clearing caches and offscreen canvas
export function invalidateDrawCache() {
    try {
        if (window.offscreenCanvas) {
            const ctx = window.offscreenCanvas.getContext('2d');
            ctx.clearRect(0, 0, window.offscreenCanvas.width, window.offscreenCanvas.height);
        }
        if (window.lastDrawnState) {
            window.lastDrawnState.visibleTiles.clear();
            window.lastDrawnState.lastDrawnTiles.clear();
        }
    } catch (e) {
        // Ignore errors
    }
}

function drawAttackHighlights(ctx, canvas, selectedUnit) {
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = hexHeight;
    
    // Get attack range (same as move range for now)
    const attackRange = state.moveRange;
    
    ctx.save();
    ctx.globalAlpha = 0.5;
    ctx.fillStyle = '#ff0000';
    ctx.strokeStyle = '#ff0000';
    ctx.lineWidth = 3;
    
    // Highlight enemy units in range
    for (const unit of state.gameState.units) {
        if (unit.owner !== selectedUnit.owner) {
            const inRange = attackRange.some(t => t.col === unit.col && t.row === unit.row);
            if (inRange) {
                let x = hexSize * 1.5 * unit.col + state.offsetX + margin;
                let y = hexHeight * unit.row + state.offsetY + margin;
                if (unit.col % 2 !== 0) y += hexHeight / 2;
                
                ctx.beginPath();
                ctx.arc(x, y, hexSize / 2, 0, Math.PI * 2);
                ctx.stroke();
            }
        }
    }
    
    // Highlight enemy buildings in range
    for (const building of state.gameState.buildings) {
        if (building.owner !== selectedUnit.owner) {
            const inRange = attackRange.some(t => t.col === building.col && t.row === building.row);
            if (inRange) {
                let x = hexSize * 1.5 * building.col + state.offsetX + margin;
                let y = hexHeight * building.row + state.offsetY + margin;
                if (building.col % 2 !== 0) y += hexHeight / 2;
                
                ctx.strokeRect(x - hexSize/3, y - hexSize/3, hexSize*2/3, hexSize*2/3);
            }
        }
    }
    
    ctx.restore();
}