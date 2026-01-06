import * as state from './state.js';
import * as perfMeasurement from './perfMeasurement.js';
import { invalidateDrawCache } from './rendering.js';

// Debounce helper for performance optimization
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Optimize panning with requestAnimationFrame
let panAnimationFrame = null;
function handlePanUpdate(e, scheduleDrawGrid) {
    if (state.isPanning) {
        state.setOffset(
            state.panStart.ox + (e.clientX - state.panStart.x),
            state.panStart.oy + (e.clientY - state.panStart.y)
        );
        
        if (panAnimationFrame) cancelAnimationFrame(panAnimationFrame);
        panAnimationFrame = requestAnimationFrame(scheduleDrawGrid);
    }
}

export function setupInputHandlers(canvas, ctx, scheduleDrawGrid) {
    // Keyboard input handling
    document.addEventListener('keydown', (e) => {
        if (!e.key) return;
        const panSpeed = 20;
        switch (e.key.toLowerCase()) {
            case 'w':
            case 'arrowup':
                state.setOffset(state.offsetX, state.offsetY + panSpeed);
                scheduleDrawGrid();
                break;
            case 's':
            case 'arrowdown':
                state.setOffset(state.offsetX, state.offsetY - panSpeed);
                scheduleDrawGrid();
                break;
            case 'a':
            case 'arrowleft':
                state.setOffset(state.offsetX + panSpeed, state.offsetY);
                scheduleDrawGrid();
                break;
            case 'd':
            case 'arrowright':
                state.setOffset(state.offsetX - panSpeed, state.offsetY);
                scheduleDrawGrid();
                break;
            case 'q':
                state.setZoom(Math.max(0.25, state.zoom * 0.9), canvas.width / 2, canvas.height / 2);
                scheduleDrawGrid();
                break;
            case 'e':
                state.setZoom(Math.min(2.5, state.zoom * 1.1), canvas.width / 2, canvas.height / 2);
                scheduleDrawGrid();
                break;
        }
    });

    // Optimized wheel zoom handler
    const handleWheel = (e) => {
        e.preventDefault();
        
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        
        // Calculate new zoom based on wheel delta
        const prevZoom = state.zoom;
        let newZoom = state.zoom;
        if (e.deltaY < 0) newZoom *= 1.1;
        else newZoom /= 1.1;
        newZoom = Math.max(0.25, Math.min(2.5, newZoom));
        
        // Set zoom while maintaining the mouse position as the center point
        state.setZoom(newZoom, mx, my);
        
        // Schedule a redraw
        scheduleDrawGrid();
    };
    
    // Optimized mouse handlers for panning
    canvas.addEventListener('wheel', handleWheel, { passive: false });
    
    canvas.addEventListener('mousedown', (e) => {
        if (e.button === 2) {
            state.setPanning(true);
            state.setPanStart({
                x: e.clientX,
                y: e.clientY,
                ox: state.offsetX,
                oy: state.offsetY
            });
            e.preventDefault();
        }
    });
    
    // Throttle mousemove for better performance
    window.addEventListener('mousemove', (e) => {
        handlePanUpdate(e, scheduleDrawGrid);
    });
    
    window.addEventListener('mouseup', (e) => {
        if (e.button === 2) {
            state.setPanning(false);
            if (panAnimationFrame) {
                cancelAnimationFrame(panAnimationFrame);
                panAnimationFrame = null;
            }
        }
    });
    
    canvas.addEventListener('contextmenu', async (e) => {
        e.preventDefault();
        
        if (!state.gameState || !state.selectedTile) {
            return;
        }
        
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        
        const tile = state.getHexAt(mx, my);
        if (!tile) {
            return;
        }
        
        const { col, row } = tile;
        const unit = state.gameState.units.find(u => u.col === col && u.row === row);
        const building = state.gameState.buildings.find(b => b.col === col && b.row === row);
        
        // Check if there's an enemy unit or building to attack
        if ((unit && unit.owner !== state.gameState.currentPlayer) || (building && building.owner !== state.gameState.currentPlayer)) {
            // Check if the target is in range (attack range = 5, same as movement)
            const distance = state.hexDistance(state.selectedTile.col, state.selectedTile.row, col, row);
            if (distance <= 5) {
                await fetch('/api/move', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        type: 'attack',
                        fromCol: state.selectedTile.col,
                        fromRow: state.selectedTile.row,
                        toCol: col,
                        toRow: row,
                        playerId: window.currentPlayerId
                    })
                });
                debouncedGameUpdate();
                state.setSelectedTile(null);
                state.setMoveRange([]);
            }
        }
    });
    
    // Batch UI updates
    const debouncedGameUpdate = debounce(async () => {
        await state.fetchGame(false, scheduleDrawGrid);
        state.setSelectedTile(null);
        state.setMoveRange([]);
    }, 250);

    // UI Event Handlers
    document.querySelectorAll('.icon-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');
            state.setSelectedBarType(btn.dataset.type);
            state.setSelectedTile(null);
            state.setMoveRange([]);
            scheduleDrawGrid();
        });
    });

    document.getElementById('nextTurnBtn')?.addEventListener('click', async () => {
        try {
            await fetch('/api/endturn', { 
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ playerId: window.currentPlayerId })
            });
            debouncedGameUpdate();
        } catch (error) {
            console.error('Error ending turn:', error);
        }
    });

    document.getElementById('regenMapBtn')?.addEventListener('click', async () => {
        // Store the user's edited values in variables
        const userColsValue = document.getElementById('mapCols').value;
        const userRowsValue = document.getElementById('mapRows').value;
        const cols = parseInt(userColsValue, 10) || 100;
        const rows = parseInt(userRowsValue, 10) || 100;
        
        // Generate a unique parameter to prevent caching
        const timestamp = Date.now();
        
        try {
            console.log(`Regenerating map with dimensions: ${cols}x${rows}`);
            
            // Make the API call with the user's values
            const response = await fetch(`/api/game?regen=1&cols=${cols}&rows=${rows}&_=${timestamp}`);
            if (!response.ok) {
                throw new Error(`Server responded with ${response.status}`);
            }
            
            // Get the game state directly from this response without making another call
            const newGameState = await response.json();
            console.log("Received new game state from server:", 
                       `${newGameState.cols}x${newGameState.rows} map, ` + 
                       `${newGameState.tiles.length} column arrays`);
            
            // Make sure we properly update the game state using exported setters
            state.setGameState(newGameState);
            state.setMapSize(newGameState.cols, newGameState.rows);
            state.setMapData({ tiles: newGameState.tiles });
            
            // Force the input fields to keep the user's entered values
            document.getElementById('mapCols').value = userColsValue;
            document.getElementById('mapRows').value = userRowsValue;
            
            // Reset selection state
            state.setSelectedTile(null);
            state.setMoveRange([]);
            
            // Clear any cached positions
            state.clearHexPositionCache();

            // Invalidate rendering caches and offscreen canvas so a full redraw will occur
            try { invalidateDrawCache(); } catch (e) { /* ignore */ }
            
            // Reset the lastDrawnState to force a complete redraw
            if (window.lastDrawnState) {
                window.lastDrawnState = {
                    offsetX: 0,
                    offsetY: 0,
                    zoom: 0, // Different from current zoom to force redraw
                    visibleTiles: new Set(),
                    lastDrawnTiles: new Map()
                };
            }
            
            // Recenter the map with the new dimensions
            const hexSize = 30 * state.zoom;
            const hexHeight = Math.sqrt(3) * hexSize;
            const margin = 20;
            const mapCenterX = (state.COLS * 1.5 * hexSize) / 2 + margin;
            const mapCenterY = (state.ROWS * hexHeight) / 2 + margin;
            const canvasCenterX = canvas.width / 2;
            const canvasCenterY = canvas.height / 2;
            
            // Center the map and make sure it's visible
            state.setOffset(canvasCenterX - mapCenterX, canvasCenterY - mapCenterY);
            
            // Force a complete redraw by clearing the offscreen canvas
            if (window.offscreenCanvas) {
                const offscreenCtx = window.offscreenCanvas.getContext('2d');
                offscreenCtx.clearRect(0, 0, window.offscreenCanvas.width, window.offscreenCanvas.height);
                console.log("Cleared offscreen canvas for complete redraw");
            }
            
            // Schedule a redraw with a slight delay to ensure state is updated
            setTimeout(() => {
                scheduleDrawGrid();
                console.log("Map regenerated and redrawn");
            }, 50);
            
        } catch (error) {
            console.error("Error regenerating map:", error);
            alert("Error regenerating map: " + error.message);
        }
    });

    // Optimized click handler with efficient hit detection
    canvas.addEventListener('click', async (e) => {
        if (!state.gameState) return;
        
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        
        const tile = state.getHexAt(mx, my);
        if (!tile) return;
        
        const { col, row } = tile;
        const unit = state.gameState.units.find(u => u.col === col && u.row === row);
        const building = state.gameState.buildings.find(b => b.col === col && b.row === row);
        
        // Don't interact with other players' units or buildings
        if ((unit && unit.owner !== state.gameState.currentPlayer) || 
            (building && building.owner !== state.gameState.currentPlayer)) {
            return;
        }
        
        if (state.selectedBarType) {
            // Place unit or building
            let type = '';
            if (state.selectedBarType === 'troop') type = 'place_troop';
            else if (state.selectedBarType === 'ship') type = 'place_ship';
            else if (state.selectedBarType === 'advanced_troop') type = 'place_advanced_troop';
            else if (state.selectedBarType === 'advanced_ship') type = 'place_advanced_ship';
            else if (state.selectedBarType === 'city') type = 'place_city';
            else if (state.selectedBarType === 'port') type = 'place_port';
            else if (state.selectedBarType === 'fort') type = 'place_fort';
            if (type) {
                try {
                    const url = type === 'move' ? '/api/move' : '/api/place';
                    await fetch(url, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ type, toCol: col, toRow: row, playerId: window.currentPlayerId })
                    });
                    // Update game state and clear selection
                    state.setSelectedBarType(null);
                    document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
                    await debouncedGameUpdate();
                    scheduleDrawGrid();
                } catch (error) {
                    console.error('Error placing unit/building:', error);
                }
            }
        } else if (unit) {
            // Handle unit selection
            state.setSelectedTile({col, row});
            if (!unit.moved && unit.owner === state.gameState.currentPlayer) {
                try {
                    const res = await fetch('/api/move-range', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ col, row, range: 5, unitType: unit.type, playerId: window.currentPlayerId })
                    });
                    const data = await res.json();
                    state.setMoveRange(data.tiles.map(([c, r]) => ({col: c, row: r})));
                } catch (error) {
                    console.error('Error fetching move range:', error);
                    state.setMoveRange([]);
                }
            } else {
                state.setMoveRange([]);
            }
            scheduleDrawGrid();
            
        } else if (state.selectedTile && state.moveRange.length > 0 && !unit && !building) {
            // Handle unit movement
            const inRange = state.moveRange.some(t => t.col === col && t.row === row);
            if (inRange) {
                try {
                    await fetch('/api/move', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            type: 'move',
                            fromCol: state.selectedTile.col,
                            fromRow: state.selectedTile.row,
                            toCol: col,
                            toRow: row,
                            playerId: window.currentPlayerId
                        })
                    });
                    debouncedGameUpdate();
                    state.setSelectedTile({col, row});
                } catch (error) {
                    console.error('Error moving unit:', error);
                }
            }
            
        } else {
            // Clear selection
            state.setSelectedTile(null);
            state.setMoveRange([]);
            scheduleDrawGrid();
        }
    });

    // Right-click handler for attacks
    canvas.addEventListener('contextmenu', async (e) => {
        e.preventDefault();
        
        if (!state.gameState || !state.selectedTile) {
            return;
        }
        
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        
        const tile = state.getHexAt(mx, my);
        if (!tile) {
            return;
        }
        
        const { col, row } = tile;
        const unit = state.gameState.units.find(u => u.col === col && u.row === row);
        const building = state.gameState.buildings.find(b => b.col === col && b.row === row);
        
        // Check if there's an enemy unit or building to attack
        if ((unit && unit.owner !== state.gameState.currentPlayer) || (building && building.owner !== state.gameState.currentPlayer)) {
            // Check if the target is in range (attack range = 5, same as movement)
            const distance = state.hexDistance(state.selectedTile.col, state.selectedTile.row, col, row);
            if (distance <= 5) {
                try {
                    await fetch('/api/move', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            type: 'attack',
                            fromCol: state.selectedTile.col,
                            fromRow: state.selectedTile.row,
                            toCol: col,
                            toRow: row
                        })
                    });
                    debouncedGameUpdate();
                    state.setSelectedTile(null);
                    state.setMoveRange([]);
                } catch (error) {
                    console.error('Error attacking:', error);
                }
            }
        }
    });

    // Touch support for mobile
    let touchStartX = 0, touchStartY = 0;
    canvas.addEventListener('touchstart', (e) => {
        e.preventDefault();
        const touch = e.touches[0];
        touchStartX = touch.clientX;
        touchStartY = touch.clientY;
        state.isPanning = true;
        state.panStart = { x: touch.clientX, y: touch.clientY, ox: state.offsetX, oy: state.offsetY };
    });

    canvas.addEventListener('touchmove', (e) => {
        e.preventDefault();
        if (state.isPanning && e.touches.length === 1) {
            const touch = e.touches[0];
            state.setOffset(
                state.panStart.ox + (touch.clientX - state.panStart.x),
                state.panStart.oy + (touch.clientY - state.panStart.y)
            );
            scheduleDrawGrid();
        }
    });

    canvas.addEventListener('touchend', (e) => {
        e.preventDefault();
        state.isPanning = false;
    });

    // Pinch zoom for touch
    let initialDistance = 0;
    canvas.addEventListener('touchstart', (e) => {
        if (e.touches.length === 2) {
            const dx = e.touches[0].clientX - e.touches[1].clientX;
            const dy = e.touches[0].clientY - e.touches[1].clientY;
            initialDistance = Math.sqrt(dx * dx + dy * dy);
        }
    });

    canvas.addEventListener('touchmove', (e) => {
        if (e.touches.length === 2) {
            const dx = e.touches[0].clientX - e.touches[1].clientX;
            const dy = e.touches[0].clientY - e.touches[1].clientY;
            const distance = Math.sqrt(dx * dx + dy * dy);
            const scale = distance / initialDistance;
            const newZoom = Math.max(0.25, Math.min(2.5, state.zoom * scale));
            state.setZoom(newZoom, canvas.width / 2, canvas.height / 2);
            initialDistance = distance;
            scheduleDrawGrid();
        }
    });
}