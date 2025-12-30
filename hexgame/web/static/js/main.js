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

// Start animation loop for blinking
function startAnimationLoop() {
    function animate() {
        scheduleDrawGrid();
        requestAnimationFrame(animate);
    }
    animate();
}

// Expose the draw function globally so performance testing can access it
window.scheduleDrawGrid = scheduleDrawGrid;

function resizeCanvas(skipDraw = false) {
    const topHeader = document.getElementById('top-header');
    const bottomBar = document.getElementById('bottom-bar');
    
    const topHeight = topHeader ? topHeader.offsetHeight : 40;
    const bottomHeight = bottomBar ? bottomBar.offsetHeight : 56;
    
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight - topHeight - bottomHeight;

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

    // Start animation loop
    startAnimationLoop();
}

// Player identification
window.currentPlayerId = null;

function showPlayerSetup() {
    document.getElementById('player-setup').style.display = 'block';
}

function hidePlayerSetup() {
    document.getElementById('player-setup').style.display = 'none';
}

function setupPlayerSelection() {
    document.getElementById('joinPlayer1').addEventListener('click', async () => {
        const name = document.getElementById('playerName').value || 'Player 1';
        window.currentPlayerId = 1;
        updatePlayerDisplay(name, 1);
        hidePlayerSetup();
        
        // Send join request to server
        await fetch('/api/join', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ playerId: 1, name: name })
        });
        
        initGame();
    });
    
    document.getElementById('joinPlayer2').addEventListener('click', async () => {
        const name = document.getElementById('playerName').value || 'Player 2';
        window.currentPlayerId = 2;
        updatePlayerDisplay(name, 2);
        hidePlayerSetup();
        
        // Send join request to server
        await fetch('/api/join', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ playerId: 2, name: name })
        });
        
        initGame();
    });
}

function updatePlayerDisplay(name, playerId) {
    const playerInfo = document.getElementById('playerInfo');
    const resourcesInfo = document.getElementById('resourcesInfo');
    const playerNameDisplay = document.getElementById('player-name-display');
    
    if (playerInfo) {
        playerInfo.textContent = `You are ${name} (Player ${playerId})`;
    }
    if (resourcesInfo) {
        resourcesInfo.id = `resourcesInfo-${playerId}`;
    }
    if (playerNameDisplay) {
        playerNameDisplay.textContent = `${name} (Player ${playerId})`;
        playerNameDisplay.style.color = playerId === 1 ? '#ff0000' : '#0000ff'; // Player colors
    }
    
    // Keep original tab title
    document.title = 'Hex Island Conquest';
}

// Set up game
setupInputHandlers(canvas, ctx, scheduleDrawGrid);
setupPlayerSelection();
window.addEventListener('resize', () => resizeCanvas(false));

// Show player setup instead of starting game immediately
showPlayerSetup();