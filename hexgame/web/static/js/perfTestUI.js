/**
 * Performance testing UI module for hex-game
 * 
 * This module provides the UI components for performance measurement.
 * It can be completely removed when optimizations are complete.
 */

import * as perfMeasurement from './perfMeasurement.js';

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
        width: 300px;
        font-family: sans-serif;
    `;
    
    perfTestDiv.innerHTML = `
        <h3 style="margin: 0 0 10px 0;">Performance Testing</h3>
        <div style="margin-bottom: 10px;">
            <button id="startPerfTest" style="margin-right: 5px; padding: 5px 10px; background: #4CAF50; border: none; color: white; border-radius: 3px;">Start Recording</button>
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
    
    // Add auto-tracking toggle
    createAutoTrackingToggle();
    
    // Show any existing comparison data
    showComparison();
}

// Create auto-tracking toggle button
function createAutoTrackingToggle() {
    // Create toggle button for automatic performance tracking
    const toggleBtn = document.createElement('button');
    toggleBtn.id = 'autoTrackToggle';
    toggleBtn.style.cssText = `
        position: fixed;
        bottom: 70px;
        right: 10px;
        padding: 8px 12px;
        background-color: #FF9800;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-weight: bold;
        z-index: 1000;
    `;
    toggleBtn.textContent = 'Auto-Track: OFF';
    
    // Toggle auto-tracking when clicked
    toggleBtn.addEventListener('click', () => {
        const newState = !perfMeasurement.isAutoTrackingEnabled();
        perfMeasurement.setAutoTracking(newState);
        updateAutoTrackToggleUI();
    });
    
    document.body.appendChild(toggleBtn);
}

// Update auto-tracking toggle UI
export function updateAutoTrackToggleUI() {
    const toggleBtn = document.getElementById('autoTrackToggle');
    if (toggleBtn) {
        const autoTrackingEnabled = perfMeasurement.isAutoTrackingEnabled();
        toggleBtn.textContent = `Auto-Track: ${autoTrackingEnabled ? 'ON' : 'OFF'}`;
        toggleBtn.style.backgroundColor = autoTrackingEnabled ? '#4CAF50' : '#FF9800';
    }
}

// Start performance tracking session
function startPerfTrackingSession() {
    perfMeasurement.startPerfTracking();
    document.getElementById('perfResults').textContent = 'Recording in progress...';
}

// Stop performance tracking session and display results
function stopPerfTrackingSession() {
    perfMeasurement.stopPerfTracking();
    
    // Format and display the results
    const results = perfMeasurement.formatPerfResults(perfMeasurement.scrollMetrics);
    document.getElementById('perfResults').textContent = results;
    
    // Show comparison if we have baseline data
    showComparison();
}

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
    
    const toggle = document.getElementById('autoTrackToggle');
    if (toggle) toggle.remove();
}