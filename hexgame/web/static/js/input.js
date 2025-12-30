import * as state from './state.js';
import * as perfMeasurement from './perfMeasurement.js';

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
    
    canvas.addEventListener('contextmenu', e => e.preventDefault());
    
    // Batch UI updates
    const debouncedGameUpdate = debounce(async () => {
        await state.fetchGame(true, scheduleDrawGrid);
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
        await fetch('/api/endturn', { method: 'POST' });
        debouncedGameUpdate();
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
        
        if (state.selectedBarType) {
            // Place unit or building
            const type = state.selectedBarType === 'unit' ? 'place_unit' : 'place_building';
            await fetch('/api/move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type, toCol: col, toRow: row })
            });
            // Update game state and clear selection
            state.setSelectedBarType(null);
            document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
            await debouncedGameUpdate();
            scheduleDrawGrid();
        } else if (unit) {
            // Handle unit selection
            state.setSelectedTile({col, row});
            if (!unit.moved && unit.owner === state.gameState.currentPlayer) {
                const res = await fetch('/api/move-range', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ col, row, range: 5 })
                });
                const data = await res.json();
                state.setMoveRange(data.tiles.map(([c, r]) => ({col: c, row: r})));
            } else {
                state.setMoveRange([]);
            }
            scheduleDrawGrid();
            
        } else if (state.selectedTile && state.moveRange.length > 0 && !unit && !building) {
            // Handle unit movement
            const inRange = state.moveRange.some(t => t.col === col && t.row === row);
            if (inRange) {
                await fetch('/api/move', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        type: 'move',
                        fromCol: state.selectedTile.col,
                        fromRow: state.selectedTile.row,
                        toCol: col,
                        toRow: row
                    })
                });
                debouncedGameUpdate();
                state.setSelectedTile({col, row});
            }
            
        } else {
            // Clear selection
            state.setSelectedTile(null);
            state.setMoveRange([]);
            scheduleDrawGrid();
        }
    });
}