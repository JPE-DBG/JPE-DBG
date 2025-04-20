import { setupInputHandlers } from './input.js';
import { drawGrid } from './rendering.js';
import * as state from './state.js'; // Import all state

const canvas = document.getElementById('hexCanvas');
const ctx = canvas.getContext('2d');

function scheduleDrawGrid() {
    // Use requestAnimationFrame for smoother rendering
    requestAnimationFrame(() => drawGrid(ctx, canvas));
}

// --- Add back resize and centering logic ---
function resizeCanvas(skipDraw = false) {
    const bar = document.getElementById('bottom-bar');
    const barRect = bar ? bar.getBoundingClientRect() : { height: 56, top: window.innerHeight - 56 }; // Estimate if bar not found
    canvas.width = window.innerWidth;
    // Ensure height calculation is correct, considering the bar's position
    canvas.height = bar ? barRect.top : (window.innerHeight - barRect.height);

    // Remove centering logic from here - it will be called explicitly after fetch
    // if (!state.mapCenteredOnce) {
    //     centerMapView();
    //     state.setMapCenteredOnce(true);
    // }

    // Schedule a draw unless explicitly skipped (like during initial load)
    if (!skipDraw) scheduleDrawGrid();
}

function centerMapView() {
    const hexSize = 30 * state.zoom; // Use state.zoom
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    // Center of map in pixels
    const mapCenterX = (state.COLS * 1.5 * hexSize) / 2 + margin;
    const mapCenterY = (state.ROWS * hexHeight) / 2 + margin;
    // Center of canvas in pixels
    const canvasCenterX = canvas.width / 2;
    const canvasCenterY = canvas.height / 2;
    state.setOffset(canvasCenterX - mapCenterX, canvasCenterY - mapCenterY);
}
// -----------------------------------------

async function initGame() {
    // Fetch game state first to get dimensions
    await state.fetchGame(false, scheduleDrawGrid);
    // Now resize the canvas (sets width/height)
    resizeCanvas(true); // Skip draw, but sets canvas size
    // Explicitly center the map now that we have dimensions
    centerMapView();
    state.setMapCenteredOnce(true);
    // Schedule the first draw
    scheduleDrawGrid();
}

setupInputHandlers(canvas, ctx, scheduleDrawGrid);
window.addEventListener('resize', () => resizeCanvas(false)); // Add resize listener back
initGame();