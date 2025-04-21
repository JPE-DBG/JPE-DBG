import * as state from './state.js';
import * as perfMeasurement from './perfMeasurement.js';

export function setupInputHandlers(canvas, ctx, scheduleDrawGrid) {
    // Mouse wheel zoom
    canvas.addEventListener('wheel', (e) => {
        e.preventDefault();
        
        // Auto-start performance tracking on zoom if enabled
        if (perfMeasurement.isAutoTrackingEnabled() && !state.isPanning) {
            perfMeasurement.startPerfTracking();
            perfMeasurement.updatePanFrameTime();
        }
        
        const prevZoom = state.zoom;
        let newZoom = state.zoom;
        if (e.deltaY < 0) newZoom *= 1.1;
        else newZoom /= 1.1;
        newZoom = Math.max(0.25, Math.min(2.5, newZoom));
        state.setZoom(newZoom);
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        state.setOffset((state.offsetX - mx) * (newZoom / prevZoom) + mx, (state.offsetY - my) * (newZoom / prevZoom) + my);
        scheduleDrawGrid();
        
        // Set a timeout to stop tracking after zoom stops
        if (perfMeasurement.isAutoTrackingEnabled()) {
            setTimeout(() => {
                // If no more zoom events happened, stop tracking
                if (performance.now() - perfMeasurement.getLastPanFrameTime() > 300) {
                    perfMeasurement.stopPerfTracking();
                }
            }, 350);
        }
    }, { passive: false });

    // Mouse drag pan
    canvas.addEventListener('mousedown', (e) => {
        if (e.button === 2) {
            state.setPanning(true);
            state.setPanStart({x: e.clientX, y: e.clientY, ox: state.offsetX, oy: state.offsetY});
            
            // Auto-start performance tracking
            if (perfMeasurement.isAutoTrackingEnabled()) {
                perfMeasurement.startPerfTracking();
                perfMeasurement.updatePanFrameTime();
            }
            
            e.preventDefault();
        }
    });
    
    window.addEventListener('mousemove', (e) => {
        if (state.isPanning) {
            // Update timestamp for last pan action
            if (perfMeasurement.isAutoTrackingEnabled()) {
                perfMeasurement.updatePanFrameTime();
            }
            
            state.setOffset(state.panStart.ox + (e.clientX - state.panStart.x), state.panStart.oy + (e.clientY - state.panStart.y));
            scheduleDrawGrid();
        }
    });
    
    window.addEventListener('mouseup', (e) => {
        if (e.button === 2) {
            state.setPanning(false);
            
            // Stop tracking after a short delay
            if (perfMeasurement.isAutoTrackingEnabled()) {
                setTimeout(() => {
                    perfMeasurement.stopPerfTracking();
                }, 200);
            }
        }
    });
    
    canvas.addEventListener('contextmenu', e => e.preventDefault());

    // UI: Bottom Bar Selection
    document.querySelectorAll('.icon-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');
            // Use the setter function instead of direct assignment
            state.setSelectedBarType(btn.dataset.type);
            state.setSelectedTile(null);
            state.setMoveRange([]);
            scheduleDrawGrid();
        });
    });

    document.getElementById('nextTurnBtn').addEventListener('click', async () => {
        await fetch('/api/endturn', { method: 'POST' });
        await state.fetchGame(true, scheduleDrawGrid);
        state.setSelectedTile(null);
        state.setMoveRange([]);
        scheduleDrawGrid();
    });

    document.getElementById('regenMapBtn').addEventListener('click', async () => {
        const cols = parseInt(document.getElementById('mapCols').value, 10) || 100;
        const rows = parseInt(document.getElementById('mapRows').value, 10) || 100;
        await fetch(`/api/game?regen=1&cols=${cols}&rows=${rows}`);
        await state.fetchGame(true, scheduleDrawGrid);
        state.setSelectedTile(null);
        state.setMoveRange([]);
        scheduleDrawGrid();
    });

    // Canvas Click: Place, Select, or Move
    canvas.addEventListener('click', async (e) => {
        if (!state.gameState) return;
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        const tile = state.getHexAt(mx, my);
        if (!tile) return;
        const {col, row} = tile;
        const unit = state.gameState.units.find(u => u.col === col && u.row === row);
        const building = state.gameState.buildings.find(b => b.col === col && b.row === row);
        if (state.selectedBarType) {
            let type = state.selectedBarType === 'unit' ? 'place_unit' : 'place_building';
            await fetch('/api/move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type, toCol: col, toRow: row })
            });
            // Use the setter function to clear the selection
            state.setSelectedBarType(null);
            document.querySelectorAll('.icon-btn').forEach(b => b.classList.remove('selected'));
            await state.fetchGame(true, scheduleDrawGrid);
            state.setSelectedTile({col, row});
            state.setMoveRange([]);
            scheduleDrawGrid();
        } else if (unit) {
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
            const inRange = state.moveRange.some(t => t.col === col && t.row === row);
            if (inRange) {
                await fetch('/api/move', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ type: 'move', fromCol: state.selectedTile.col, fromRow: state.selectedTile.row, toCol: col, toRow: row })
                });
                await state.fetchGame(true, scheduleDrawGrid);
                state.setSelectedTile({col, row});
                state.setMoveRange([]);
                scheduleDrawGrid();
            }
        } else {
            state.setSelectedTile(null);
            state.setMoveRange([]);
            scheduleDrawGrid();
        }
    });
}