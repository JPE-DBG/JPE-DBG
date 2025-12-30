import { setupInputHandlers } from './input.js';
import { drawGrid } from './rendering.js';
import * as state from './state.js'; 
import * as perfTestUI from './perfTestUI.js';

const canvas = document.getElementById('hexCanvas');
const ctx = canvas.getContext('2d');

function scheduleDrawGrid() {
    // Use requestAnimationFrame for smoother rendering
    requestAnimationFrame(() => drawGrid(ctx, canvas));
}

// Expose the draw function globally so performance testing can access it
window.scheduleDrawGrid = scheduleDrawGrid;

function resizeCanvas(skipDraw = false) {
    const bar = document.getElementById('bottom-bar');
    const barRect = bar ? bar.getBoundingClientRect() : { height: 56, top: window.innerHeight - 56 };
    canvas.width = window.innerWidth;
    canvas.height = bar ? barRect.top : (window.innerHeight - barRect.height);

    if (!skipDraw) scheduleDrawGrid();
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
    // Fetch game state to get dimensions and set input fields
    await state.fetchGame(false, scheduleDrawGrid);
    
    // Now resize the canvas (sets width/height)
    resizeCanvas(true); // Skip draw, but sets canvas size
    
    // Explicitly center the map now that we have dimensions
    centerMapView();
    state.setMapCenteredOnce(true);
    
    // Provide state management functions to the performance testing module
    perfTestUI.setStateFunctions({
        gameState: state.gameState,
        setOffset: state.setOffset,
        setZoom: state.setZoom,
        offsetX: state.offsetX,
        offsetY: state.offsetY,
        zoom: state.zoom,
        COLS: state.COLS,
        ROWS: state.ROWS,
    });
    
    // Connect to WebSocket
    const ws = new WebSocket('ws://localhost:8080/ws');
    ws.onmessage = (event) => {
        const newGameState = JSON.parse(event.data);
        state.setGameState(newGameState);
        scheduleDrawGrid();
    };
    
    // Schedule the first draw
    scheduleDrawGrid();
    
    // Initialize the performance testing UI
    perfTestUI.createPerfTestUI();
}

// Set up game
setupInputHandlers(canvas, ctx, scheduleDrawGrid);
window.addEventListener('resize', () => resizeCanvas(false));
initGame();