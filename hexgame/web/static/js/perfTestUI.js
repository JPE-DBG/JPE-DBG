/**
 * Performance testing UI module for hex-game
 * 
 * This module provides the UI components for performance measurement.
 * It can be completely removed when optimizations are complete.
 */

import * as perfMeasurement from './perfMeasurement.js';

// Store references to state-setting functions
let stateSetters = null;

// Create and display performance testing UI
export function createPerfTestUI() {
    // Check if measurement system is working properly
    const validationResult = perfMeasurement.validateMeasurementSystem();
    console.log("Performance measurement validation:", validationResult);
    
    // Create the main container
    const perfTestDiv = document.createElement('div');
    perfTestDiv.id = 'perfTestUI';
    perfTestDiv.style.cssText = `
        position: fixed;
        top: 10px;
        right: 10px;
        background-color: rgba(0, 0, 0, 0.7);
        color: white;
        padding: 10px;
        border-radius: 5px;
        z-index: 1000;
        width: 350px;
        font-family: sans-serif;
    `;
    
    perfTestDiv.innerHTML = `
        <h3 style="margin: 0 0 10px 0;">Performance Testing</h3>
        <div style="margin-bottom: 10px;">
            <button id="startPerfTest" style="margin-right: 5px; padding: 5px 10px; background: #4CAF50; border: none; color: white; border-radius: 3px;">Start Auto Test</button>
            <button id="stopPerfTest" style="padding: 5px 10px; background: #F44336; border: none; color: white; border-radius: 3px;">Stop Recording</button>
        </div>
        <div style="margin-bottom: 10px;">
            <button id="saveBaseline" style="margin-right: 5px; padding: 5px 10px; background: #2196F3; border: none; color: white; border-radius: 3px;">Save as Baseline</button>
            <button id="saveOptimization" style="padding: 5px 10px; background: #9C27B0; border: none; color: white; border-radius: 3px;">Save as Optimization</button>
        </div>
        <div style="margin-bottom: 10px;">
            <button id="resetMetrics" style="margin-right: 5px; padding: 5px 10px; background: #607D8B; border: none; color: white; border-radius: 3px;">Reset Metrics</button>
            <button id="clearAllData" style="padding: 5px 10px; background: #FF5722; border: none; color: white; border-radius: 3px;">Clear All Data</button>
        </div>
        <div id="recordingStatus" style="margin-top: 10px; display: none; padding: 5px; background: rgba(255,255,255,0.1); border-radius: 3px;">
            <div class="status-text">Not recording</div>
            <div class="progress-bar" style="height: 5px; margin-top: 5px; background: #333;"><div class="progress" style="width: 0%; height: 100%; background: #4CAF50;"></div></div>
        </div>
        <div style="margin-top: 15px;">
            <strong>Last Test Results:</strong>
            <pre id="perfResults" style="margin: 5px 0; padding: 5px; background: rgba(255,255,255,0.1); max-height: 200px; overflow-y: auto; font-size: 12px;"></pre>
        </div>
        <div id="comparisonResults" style="margin-top: 15px; display: block;">
            <strong>Comparison:</strong>
            <pre id="comparisonData" style="margin: 5px 0; padding: 5px; background: rgba(255,255,255,0.1); max-height: 200px; overflow-y: auto; font-size: 12px;">No comparison data yet.</pre>
        </div>
    `;
    
    document.body.appendChild(perfTestDiv);
    
    // Set up event listeners for the performance testing buttons
    document.getElementById('startPerfTest').addEventListener('click', startPerfTrackingSession);
    document.getElementById('stopPerfTest').addEventListener('click', stopPerfTrackingSession);
    document.getElementById('saveBaseline').addEventListener('click', saveBaselineResults);
    document.getElementById('saveOptimization').addEventListener('click', saveOptimizationResults);
    document.getElementById('resetMetrics').addEventListener('click', resetMetricsClicked);
    document.getElementById('clearAllData').addEventListener('click', clearAllDataClicked);
    
    // Show any existing comparison data
    showComparison();
    
    // Start update loop for recording status
    startStatusUpdateLoop();
}

// Set state management functions needed for auto-scrolling
export function setStateFunctions(stateData) {
    stateSetters = stateData;
    window.stateSetters = stateData;  // Also set on window for access by perfMeasurement functions
    console.log("State functions registered for auto-testing:", 
        Object.keys(stateData).filter(key => stateData[key] !== undefined));
}

// Update recording status in UI
function startStatusUpdateLoop() {
    const updateStatus = () => {
        const metrics = perfMeasurement.getPerfMetrics();
        const statusDiv = document.getElementById('recordingStatus');
        
        if (metrics.isRecording || metrics.isAutoScrolling) {
            statusDiv.style.display = 'block';
            const statusText = statusDiv.querySelector('.status-text');
            const progress = statusDiv.querySelector('.progress');
            
            // Display auto-scroll progress if active
            if (metrics.isAutoScrolling) {
                const percent = (metrics.autoScrollPosition / metrics.autoScrollMaxPosition * 100);
                statusText.textContent = `Auto-scrolling... ${percent.toFixed(1)}%`;
                progress.style.width = `${percent}%`;
                progress.style.background = '#2196F3'; // Blue
            }
            // Otherwise show stabilization/recording status
            else if (metrics.inStabilization) {
                const percent = (metrics.stabilizationFrames / 10) * 100;
                statusText.textContent = `Stabilizing... (${metrics.stabilizationFrames}/10)`;
                progress.style.width = `${percent}%`;
                progress.style.background = '#FFA500'; // Orange
            } else {
                statusText.textContent = `Recording... (${metrics.scrollMetrics.totalFrames} frames)`;
                progress.style.width = '100%';
                progress.style.background = '#4CAF50'; // Green
            }
        } else {
            statusDiv.style.display = 'none';
        }
        
        // Continue the loop
        requestAnimationFrame(updateStatus);
    };
    
    // Start the loop
    updateStatus();
}

// Start performance tracking session with auto-scrolling
function startPerfTrackingSession() {
    if (!stateSetters) {
        alert('Cannot start auto-test: state functions not available');
        console.error("Auto-test failed: state functions not available");
        return;
    }
    
    const { gameState, setOffset, setZoom, offsetX, offsetY, zoom, COLS, ROWS } = stateSetters;
    
    if (!gameState) {
        alert('Cannot start auto-test: game state not available');
        console.error("Auto-test failed: game state not available");
        return;
    }
    
    console.log("Starting performance tracking with auto-scroll...");
    
    // First, start the performance tracking
    perfMeasurement.startPerfTracking();
    
    // Then start auto-scrolling
    const result = perfMeasurement.startAutoScroll(
        gameState, 
        setOffset,
        setZoom,
        offsetX,
        offsetY,
        zoom,
        COLS,
        ROWS
    );
    
    if (!result) {
        console.error("Failed to start auto-scrolling test");
        return;
    }
    
    document.getElementById('perfResults').textContent = 'Auto-test in progress...';
    document.getElementById('recordingStatus').style.display = 'block';
    document.getElementById('startPerfTest').disabled = true;
}

// Stop performance tracking session and display results
function stopPerfTrackingSession() {
    // This will also stop auto-scrolling if active
    perfMeasurement.stopPerfTracking();
    
    // Format and display the results
    const results = perfMeasurement.formatPerfResults(perfMeasurement.scrollMetrics);
    document.getElementById('perfResults').textContent = results;
    document.getElementById('recordingStatus').style.display = 'none';
    document.getElementById('startPerfTest').disabled = false;
    
    // Show comparison if we have baseline data
    showComparison();
}

// Function to update results display (called when auto-test completes)
function updatePerfResultsDisplay() {
    const results = perfMeasurement.formatPerfResults(perfMeasurement.scrollMetrics);
    document.getElementById('perfResults').textContent = results;
    document.getElementById('startPerfTest').disabled = false;
}

// Expose the update function globally for perfMeasurement to call
window.updatePerfResultsDisplay = updatePerfResultsDisplay;

// Save current results as baseline
function saveBaselineResults() {
    if (!perfMeasurement.saveBaselineResults()) {
        alert('No performance data available. Run a test first.');
        return;
    }
    
    alert('Baseline saved! It will be available even if you restart the game.');
    
    // Show comparison section
    showComparison();
}

// Save current results as an optimization
function saveOptimizationResults() {
    if (perfMeasurement.scrollMetrics.totalFrames === 0) {
        alert('No performance data available. Run a test first.');
        return;
    }
    
    if (!perfMeasurement.getBaselineResults()) {
        if (confirm('No baseline has been set. Would you like to save this as the baseline instead?')) {
            saveBaselineResults();
        }
        return;
    }
    
    perfMeasurement.saveOptimizationResults();
    const optimizations = perfMeasurement.getOptimizationResults();
    alert(`Optimization #${optimizations.length} saved! It will be available even if you restart the game.`);
    
    // Update comparison
    showComparison();
}

// Reset metrics
function resetMetricsClicked() {
    perfMeasurement.resetPerfMetrics();
    document.getElementById('perfResults').textContent = 'Metrics reset.';
}

// Clear all saved data
function clearAllDataClicked() {
    if (confirm('Are you sure you want to clear all saved performance data?')) {
        perfMeasurement.clearAllMeasurements();
        document.getElementById('perfResults').textContent = 'All data cleared.';
        document.getElementById('comparisonData').textContent = 'No comparison data available.';
        alert('All performance data has been cleared.');
    }
}

// Show comparison between baseline and optimizations
function showComparison() {
    const baselineResults = perfMeasurement.getBaselineResults();
    
    if (!baselineResults) {
        document.getElementById('comparisonData').textContent = 'No baseline data available yet. Record performance and save as baseline.';
        return;
    }
    
    const comparisonText = perfMeasurement.generateComparisonText();
    if (comparisonText) {
        document.getElementById('comparisonResults').style.display = 'block';
        document.getElementById('comparisonData').textContent = comparisonText;
    }
}

// Remove performance testing UI
export function removePerfTestUI() {
    const ui = document.getElementById('perfTestUI');
    if (ui) ui.remove();
}