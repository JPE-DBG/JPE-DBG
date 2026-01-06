import { setupInputHandlers } from './input.js';
import { drawGrid } from './rendering.js';
import * as state from './state.js'; 
import * as perfTestUI from './perfTestUI.js';

// Game state machine
export const GAME_STATES = {
    LOADING: 'loading',
    MENU: 'menu',
    PLAYING: 'playing',
    PAUSED: 'paused'
};

let currentGameState = GAME_STATES.LOADING;

// Asset management
const assets = {
    images: {},
    sounds: {}
};

async function preloadAssets() {
    const imagePromises = [];
    // Add image preloading here if needed, e.g., sprites
    // For now, placeholder
    
    const soundPromises = [];
    // Placeholder for sound loading, e.g., loadSound('click', 'sounds/click.wav')
    
    return Promise.all([...imagePromises, ...soundPromises]);
}

// Web Audio API setup
let audioContext;
function initAudio() {
    try {
        audioContext = new (window.AudioContext || window.webkitAudioContext)();
    } catch (e) {
        console.error('Web Audio API not supported:', e);
    }
}

// Simple sound manager
const sounds = {};
function loadSound(name, url) {
    return fetch(url)
        .then(response => response.arrayBuffer())
        .then(arrayBuffer => audioContext.decodeAudioData(arrayBuffer))
        .then(audioBuffer => {
            sounds[name] = audioBuffer;
        });
}

function playSound(name) {
    if (!audioContext || !sounds[name]) return;
    const source = audioContext.createBufferSource();
    source.buffer = sounds[name];
    source.connect(audioContext.destination);
    source.start();
}

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
    try {
        // Set loading state
        currentGameState = GAME_STATES.LOADING;
        
        // Preload assets
        await preloadAssets();
        
        // Initialize audio
        initAudio();
        
        // Load saved state
        state.loadGameState();
        
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
        
        // Create the performance testing UI
        perfTestUI.createPerfTestUI();
        
        // Connect to WebSocket
        const ws = new WebSocket('ws://localhost:8080/ws');
        ws.onmessage = (event) => {
            try {
                const newGameState = JSON.parse(event.data);
                state.setGameState(newGameState);
                scheduleDrawGrid();
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
        ws.onclose = () => {
            console.log('WebSocket closed');
        };
        
        // Schedule the first draw
        scheduleDrawGrid();
        
        // Initialize the performance testing UI
        perfTestUI.createPerfTestUI();

        // Start animation loop
        startAnimationLoop();
        
        // Transition to playing state
        currentGameState = GAME_STATES.PLAYING;
        
        // Save state periodically
        setInterval(() => {
            state.saveGameState();
        }, 30000); // Every 30 seconds
    } catch (error) {
        console.error('Error initializing game:', error);
        currentGameState = GAME_STATES.MENU; // Fallback
    }
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
    document.getElementById('joinGame').addEventListener('click', async () => {
        const name = document.getElementById('playerName').value.trim() || 'Player';
        
        try {
            const response = await fetch('/api/join', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });
            
            if (!response.ok) {
                throw new Error('Failed to join game');
            }
            
            const data = await response.json();
            
            window.currentPlayerId = data.playerId;
            
            // Use the game state from the join response to immediately update the UI
            if (data.gameState) {
                state.setGameState(data.gameState);
                scheduleDrawGrid();
            }
            
            updatePlayerDisplay(name, data.playerId, data.color);
            hidePlayerSetup();
            
            // Do the post-join setup without fetching game state again
            resizeCanvas(true); // Skip draw, but sets canvas size
            
            // Center the view on the player's capital
            if (data.capital) {
                centerOnCapital(data.capital[0], data.capital[1]);
            } else {
                centerMapView(); // fallback
            }
            
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
            
            // Create the performance testing UI
            try {
                perfTestUI.createPerfTestUI();
            } catch (error) {
                console.error('Failed to create performance testing UI:', error);
            }
            
            // Connect to WebSocket
            const ws = new WebSocket('ws://localhost:8080/ws');
            ws.onmessage = (event) => {
                const newGameState = JSON.parse(event.data);
                state.setGameState(newGameState);
                scheduleDrawGrid();
            };
            
            // Start animation loop
            startAnimationLoop();
        } catch (error) {
            alert('Error joining game: ' + error.message);
        }
    });
}

function updatePlayerDisplay(name, playerId, playerColor) {
    const playerInfo = document.getElementById('playerInfo');
    const playerNameDisplay = document.getElementById('player-name-display');
    
    if (playerInfo) {
        playerInfo.textContent = `You are ${name} (Player ${playerId})`;
    }
    if (playerNameDisplay) {
        playerNameDisplay.textContent = `${name} (Player ${playerId})`;
        playerNameDisplay.style.color = playerColor;
    }
    
    // Keep original tab title
    document.title = 'Hex Island Conquest';
}

// Center the map view on the player's capital coordinates
function centerOnCapital(capitalCol, capitalRow) {
    const hexSize = 30 * state.zoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    
    // Calculate the pixel position of the capital
    let capitalX = hexSize * 1.5 * capitalCol + margin;
    let capitalY = hexHeight * capitalRow + margin;
    if (capitalCol % 2 !== 0) capitalY += hexHeight / 2;
    
    // Center the view on the capital
    const canvasCenterX = canvas.width / 2;
    const canvasCenterY = canvas.height / 2;
    state.setOffset(canvasCenterX - capitalX, canvasCenterY - capitalY);
}

// Set up game
setupInputHandlers(canvas, ctx, scheduleDrawGrid);
setupPlayerSelection();
window.addEventListener('resize', () => resizeCanvas(false));

// Show player setup instead of starting game immediately
showPlayerSetup();