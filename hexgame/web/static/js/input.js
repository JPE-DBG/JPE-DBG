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
        const cols = parseInt(document.getElementById('mapCols').value, 10) || 100;
        const rows = parseInt(document.getElementById('mapRows').value, 10) || 100;
        await fetch(`/api/game?regen=1&cols=${cols}&rows=${rows}`);
        debouncedGameUpdate();
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