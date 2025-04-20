import { setupInputHandlers } from './input.js';
import { drawGrid } from './rendering.js'; // Removed renderOffscreenMap import
import * as state from './state.js';

const canvas = document.getElementById('hexCanvas');
const ctx = canvas.getContext('2d');

function scheduleDrawGrid() {
    requestAnimationFrame(() => drawGrid(ctx, canvas));
}

function resizeCanvas(skipDraw = false) {
    const bar = document.getElementById('bottom-bar');
    const barRect = bar ? bar.getBoundingClientRect() : { height: 56, top: window.innerHeight - 56 };
    canvas.width = window.innerWidth;
    canvas.height = bar ? barRect.top : (window.innerHeight - barRect.height);

    if (!skipDraw) {
        scheduleDrawGrid();
    }
}

function centerMapView() {
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    const mapCenterX = (state.COLS * 1.5 * hexSize) / 2 + margin;
    const mapCenterY = (state.ROWS * hexHeight) / 2 + margin;
    const canvasCenterX = canvas.width / 2;
    const canvasCenterY = canvas.height / 2;
    state.setOffset(canvasCenterX - mapCenterX, canvasCenterY - mapCenterY);
}

async function initGame() {
    await state.fetchGame(false, scheduleDrawGrid); // Pass scheduleDrawGrid back

    resizeCanvas(true);
    centerMapView();
    state.setMapCenteredOnce(true);
    scheduleDrawGrid();
}

// Regenerate Map Button Handler (Example - needs wiring up if not already)
document.getElementById('regenMapBtn')?.addEventListener('click', async () => {
    await state.fetchGame(false, scheduleDrawGrid); // Refetch game state
    centerMapView(); // Recenter
    scheduleDrawGrid(); // Trigger redraw
});

setupInputHandlers(canvas, ctx, scheduleDrawGrid);
window.addEventListener('resize', () => resizeCanvas(false));
initGame();