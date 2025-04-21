/**
 * Performance measurement module for hex-game
 * 
 * This module provides tools for measuring and analyzing rendering performance.
 * It can be completely removed when optimizations are complete.
 */

// Performance metrics storage
let lastFrameTime = performance.now();
let renderStart = 0;
let renderTime = 0;
let totalRenderTime = 0;
let tilesPerFrame = 0;
let frameTimeHistory = [];
let isPerfTracking = false;
let scrollTrackingActive = false;
let lastTileCount = 0;
let maxTileCount = 0;
let tileRenderCount = 0;
let minFrameTime = Number.MAX_VALUE;
let maxFrameTime = 0;
let totalScrollFrames = 0;

// Performance metrics during scroll/zoom operations
export let scrollMetrics = {
    avgFrameTime: 0,
    minFrameTime: Number.MAX_VALUE,
    maxFrameTime: 0,
    avgTilesPerFrame: 0,
    totalFrames: 0,
    totalTiles: 0
};

// Test result storage - now loaded from localStorage if available
let baselineResults = null;
let optimizationResults = [];

// Load saved measurements from localStorage on module init
try {
    const savedBaseline = localStorage.getItem('hexGameBaseline');
    if (savedBaseline) {
        baselineResults = JSON.parse(savedBaseline);
        console.log("Loaded baseline from localStorage:", baselineResults);
    }
    
    const savedOptimizations = localStorage.getItem('hexGameOptimizations');
    if (savedOptimizations) {
        optimizationResults = JSON.parse(savedOptimizations);
        console.log("Loaded optimizations from localStorage:", optimizationResults);
    }
} catch (e) {
    console.error("Error loading saved measurements:", e);
}

// Configuration
let autoTrackingEnabled = false;
let lastPanFrameTime = 0;

// Validate if our measurement approach is consistent
export function validateMeasurementSystem() {
    // Perform a quick consistency check to ensure our timing is working
    const testStart = performance.now();
    // Do a microbenchmark
    let sum = 0;
    for (let i = 0; i < 10000; i++) {
        sum += i;
    }
    const elapsed = performance.now() - testStart;
    
    console.log(`Measurement validation: calculation took ${elapsed.toFixed(3)}ms`);
    console.log(`Current timing resolution: ${performance.timeOrigin}`);
    
    return {
        elapsed,
        valid: elapsed > 0,
        sum
    };
}

// Start tracking a scrolling/zooming session
export function startPerfTracking() {
    if (isPerfTracking) return;
    
    isPerfTracking = true;
    scrollTrackingActive = true;
    resetPerfMetrics();
    
    console.log("Performance tracking started");
    return true;
}

// Stop tracking the current scrolling/zooming session
export function stopPerfTracking() {
    if (!isPerfTracking) return false;
    
    isPerfTracking = false;
    scrollTrackingActive = false;
    
    // Calculate final metrics
    scrollMetrics.avgFrameTime = totalScrollFrames > 0 ? totalRenderTime / totalScrollFrames : 0;
    scrollMetrics.avgTilesPerFrame = totalScrollFrames > 0 ? tileRenderCount / totalScrollFrames : 0;
    scrollMetrics.minFrameTime = minFrameTime === Number.MAX_VALUE ? 0 : minFrameTime;
    scrollMetrics.maxFrameTime = maxFrameTime;
    scrollMetrics.totalFrames = totalScrollFrames;
    scrollMetrics.totalTiles = tileRenderCount;
    
    console.log("Performance tracking stopped", scrollMetrics);
    return true;
}

// Reset all performance metrics
export function resetPerfMetrics() {
    totalRenderTime = 0;
    tileRenderCount = 0;
    frameTimeHistory = [];
    minFrameTime = Number.MAX_VALUE;
    maxFrameTime = 0;
    totalScrollFrames = 0;
    scrollMetrics = {
        avgFrameTime: 0,
        minFrameTime: Number.MAX_VALUE,
        maxFrameTime: 0,
        avgTilesPerFrame: 0,
        totalFrames: 0,
        totalTiles: 0
    };
}

// Monitor a frame for performance
export function monitorFrameStart() {
    renderStart = performance.now();
    
    // Calculate frame time for this frame
    const frameTime = performance.now() - lastFrameTime;
    lastFrameTime = performance.now();
    
    // Track performance if we're in a scrolling/zooming session
    if (scrollTrackingActive) {
        totalRenderTime += frameTime;
        totalScrollFrames++;
        
        // Update min/max frame times
        if (frameTime < minFrameTime) minFrameTime = frameTime;
        if (frameTime > maxFrameTime) maxFrameTime = frameTime;
        
        // Keep the last 60 frame times for analysis
        frameTimeHistory.push(frameTime);
        if (frameTimeHistory.length > 60) {
            frameTimeHistory.shift();
        }
    }
    
    // Reset tile counter for this frame
    lastTileCount = 0;
    
    return frameTime;
}

// Record the end of a frame
export function monitorFrameEnd() {
    const endRenderTime = performance.now();
    renderTime = endRenderTime - renderStart;
    return renderTime;
}

// Track a tile being rendered
export function trackTileRender() {
    lastTileCount++;
    if (scrollTrackingActive) {
        tileRenderCount++;
    }
    
    // Track maximum tiles rendered in a single frame
    if (lastTileCount > maxTileCount) {
        maxTileCount = lastTileCount;
    }
}

// Get current performance metrics
export function getPerfMetrics() {
    return {
        renderTime,
        tilesPerFrame,
        visibleTiles: lastTileCount,
        isRecording: isPerfTracking,
        scrollMetrics
    };
}

// Get auto-tracking status
export function isAutoTrackingEnabled() {
    return autoTrackingEnabled;
}

// Set auto-tracking status
export function setAutoTracking(enabled) {
    autoTrackingEnabled = enabled;
    return autoTrackingEnabled;
}

// Update last pan frame time for auto-tracking
export function updatePanFrameTime() {
    lastPanFrameTime = performance.now();
    return lastPanFrameTime;
}

// Get last pan frame time
export function getLastPanFrameTime() {
    return lastPanFrameTime;
}

// Save baseline results
export function saveBaselineResults() {
    if (scrollMetrics.totalFrames === 0) {
        return false;
    }
    
    baselineResults = { 
        ...scrollMetrics,
        timestamp: new Date().toISOString(),
        name: "Baseline"
    };
    
    // Save to localStorage
    try {
        localStorage.setItem('hexGameBaseline', JSON.stringify(baselineResults));
        console.log("Saved baseline to localStorage");
    } catch (e) {
        console.error("Failed to save baseline to localStorage:", e);
    }
    
    // Clear any existing optimization comparisons
    optimizationResults = [];
    localStorage.removeItem('hexGameOptimizations');
    
    return true;
}

// Save optimization results
export function saveOptimizationResults() {
    if (scrollMetrics.totalFrames === 0) {
        return false;
    }
    
    const newOptimization = { 
        ...scrollMetrics, 
        timestamp: new Date().toISOString(),
        name: `Optimization #${optimizationResults.length + 1}` 
    };
    
    optimizationResults.push(newOptimization);
    
    // Save to localStorage
    try {
        localStorage.setItem('hexGameOptimizations', JSON.stringify(optimizationResults));
        console.log("Saved optimization to localStorage");
    } catch (e) {
        console.error("Failed to save optimization to localStorage:", e);
    }
    
    return true;
}

// Get baseline results
export function getBaselineResults() {
    return baselineResults;
}

// Get optimization results
export function getOptimizationResults() {
    return optimizationResults;
}

// Format performance results into a readable string
export function formatPerfResults(metrics) {
    return `Frame Time: ${metrics.avgFrameTime.toFixed(2)}ms
Min/Max Frame Time: ${metrics.minFrameTime.toFixed(2)}ms / ${metrics.maxFrameTime.toFixed(2)}ms
Avg. Tiles per Frame: ${Math.round(metrics.avgTilesPerFrame)}
Total Frames: ${metrics.totalFrames}
Total Tiles Rendered: ${metrics.totalTiles}`;
}

// Create comparison text between baseline and optimizations
export function generateComparisonText() {
    if (!baselineResults) return null;
    
    let comparisonText = `== ${baselineResults.name || "BASELINE"} ==\n`;
    comparisonText += formatPerfResults(baselineResults) + '\n';
    if (baselineResults.timestamp) {
        comparisonText += `Recorded: ${new Date(baselineResults.timestamp).toLocaleString()}\n\n`;
    } else {
        comparisonText += '\n';
    }
    
    // Add optimization results if any
    if (optimizationResults.length > 0) {
        optimizationResults.forEach((opt) => {
            comparisonText += `== ${opt.name} ==\n`;
            comparisonText += formatPerfResults(opt) + '\n';
            
            // Calculate and display differences
            const frameTimeDiff = ((baselineResults.avgFrameTime - opt.avgFrameTime) / baselineResults.avgFrameTime * 100).toFixed(2);
            comparisonText += `Frame Time Change: ${frameTimeDiff}%\n`;
            
            if (opt.timestamp) {
                comparisonText += `Recorded: ${new Date(opt.timestamp).toLocaleString()}\n\n`;
            } else {
                comparisonText += '\n';
            }
        });
    }
    
    return comparisonText;
}

// Clear all saved measurements
export function clearAllMeasurements() {
    baselineResults = null;
    optimizationResults = [];
    
    // Clear from localStorage
    localStorage.removeItem('hexGameBaseline');
    localStorage.removeItem('hexGameOptimizations');
    
    return true;
}

// Render performance metrics overlay on canvas
export function renderPerfOverlay(ctx) {
    if (!ctx) return;
    
    ctx.save();
    
    // Background for metrics panel
    ctx.fillStyle = 'rgba(0,0,0,0.7)';
    ctx.fillRect(10, 10, 220, 120);
    
    ctx.font = '14px monospace';
    ctx.fillStyle = '#fff';
    
    // Basic metrics
    ctx.fillText(`Frame time: ${renderTime.toFixed(2)}ms`, 20, 30);
    ctx.fillText(`Tiles/frame: ${tilesPerFrame}`, 20, 50);
    ctx.fillText(`Visible tiles: ${lastTileCount}`, 20, 70);
    
    // Scroll session metrics
    if (isPerfTracking) {
        ctx.fillStyle = '#4CAF50'; // Green to indicate active tracking
        ctx.fillText('RECORDING PERFORMANCE', 20, 90);
    } else if (scrollMetrics.totalFrames > 0) {
        ctx.fillText(`Avg move: ${scrollMetrics.avgFrameTime.toFixed(2)}ms`, 20, 90);
        ctx.fillText(`Min/Max: ${scrollMetrics.minFrameTime.toFixed(1)}/${scrollMetrics.maxFrameTime.toFixed(1)}ms`, 20, 110);
    }
    
    ctx.restore();
}