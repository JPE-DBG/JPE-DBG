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

// For improved measurement
let stabilizationFrames = 0;
const STABILIZATION_FRAME_COUNT = 10; // Wait for this many frames before starting to record
let actualFrameTimes = []; // Store actual frame times for accurate statistics
let frameTileCounts = []; // Store tile counts per frame
let outliersRemoved = 0;

// Auto-scroll testing
let isAutoScrolling = false;
let autoScrollSpeed = 8; // pixels per frame
let autoScrollPosition = 0;
let autoScrollMaxPosition = 0;
let originalMapPosition = { x: 0, y: 0 };
let autoScrollAnimationId = null;

// Performance metrics during scroll/zoom operations
export let scrollMetrics = {
    avgFrameTime: 0,
    minFrameTime: Number.MAX_VALUE,
    maxFrameTime: 0,
    avgTilesPerFrame: 0,
    totalFrames: 0,
    totalTiles: 0,
    medianFrameTime: 0,
    outliersRemoved: 0,
    stabilizationFrames: STABILIZATION_FRAME_COUNT
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

// Calculate the median value of an array
function calculateMedian(values) {
    if (values.length === 0) return 0;
    
    // Create a copy of the array and sort it
    const sorted = [...values].sort((a, b) => a - b);
    const middle = Math.floor(sorted.length / 2);
    
    if (sorted.length % 2 === 0) {
        return (sorted[middle - 1] + sorted[middle]) / 2;
    }
    
    return sorted[middle];
}

// Remove outliers using IQR method
function removeOutliers(values) {
    if (values.length < 8) return values; // Need enough data for statistical validity
    
    // Sort the array
    const sorted = [...values].sort((a, b) => a - b);
    
    // Calculate Q1 and Q3
    const q1Index = Math.floor(sorted.length / 4);
    const q3Index = Math.floor(sorted.length * 3 / 4);
    const q1 = sorted[q1Index];
    const q3 = sorted[q3Index];
    
    // Calculate IQR and bounds
    const iqr = q3 - q1;
    const lowerBound = q1 - (iqr * 1.5);
    const upperBound = q3 + (iqr * 1.5);
    
    // Filter out the outliers
    return sorted.filter(x => x >= lowerBound && x <= upperBound);
}

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

// Start automatic scrolling for performance testing
export function startAutoScroll(gameState, setOffset, setZoom, offsetX, offsetY, zoom, cols, rows) {
    if (isAutoScrolling) return;
    
    console.log("Starting auto-scroll with state:", { offsetX, offsetY, zoom, cols, rows });
    isAutoScrolling = true;
    
    // Save original position to restore later if needed
    originalMapPosition = { x: offsetX, y: offsetY };
    
    // Set zoom to minimum (max zoomed out)
    const minZoom = 0.2;
    setZoom(minZoom);
    
    // Calculate map dimensions
    const hexSize = 30 * minZoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    const mapWidth = cols * 1.5 * hexSize + margin * 2;
    
    // Position at the left side and vertical center
    const canvas = document.getElementById('hexCanvas');
    if (!canvas) {
        console.error("Canvas element not found");
        isAutoScrolling = false;
        return;
    }
    
    const verticalCenter = rows * hexHeight / 2;
    
    // Calculate the initial position - left edge with vertical center
    const initialX = canvas.width / 2;
    const initialY = canvas.height / 2 - verticalCenter;
    
    // Set starting position - left edge with vertical center
    setOffset(initialX, initialY);
    
    // Calculate the scrolling distance
    autoScrollPosition = 0;
    autoScrollMaxPosition = mapWidth + 100; // Add more margin to ensure we cover the full map
    
    console.log("Starting auto-scroll test with:", {
        mapWidth: mapWidth + "px",
        initialX,
        initialY,
        maxPosition: autoScrollMaxPosition
    });
    
    // Get reference to redraw function
    let redrawCallback = window.scheduleDrawGrid;
    if (!redrawCallback) {
        console.warn("No global redraw function found, animations may not be visible");
        // Try to find the function in window scope
        for (const key in window) {
            if (typeof window[key] === 'function' && key.toLowerCase().includes('draw')) {
                redrawCallback = window[key];
                console.log("Found possible redraw function:", key);
                break;
            }
        }
    }
    
    // Start the scrolling animation with redraw callback
    startAutoScrollAnimation(setOffset, initialX, initialY, redrawCallback);
    
    return true;
}

// Run the automatic scrolling animation
function startAutoScrollAnimation(setOffset, initialX, initialY, redrawCallback) {
    if (autoScrollAnimationId) {
        cancelAnimationFrame(autoScrollAnimationId);
    }
    
    const animate = () => {
        if (!isAutoScrolling) return;
        
        // Update position
        autoScrollPosition += autoScrollSpeed;
        
        // Move map - use initial position as base
        setOffset(initialX - autoScrollPosition, initialY);
        
        // Trigger redraw explicitly
        if (typeof redrawCallback === 'function') {
            redrawCallback();
        }
        
        // Check if we've reached the end
        if (autoScrollPosition >= autoScrollMaxPosition) {
            console.log("Auto-scroll reached end, stopping");
            stopAutoScroll();
            // Also stop recording 
            if (isPerfTracking) {
                stopPerfTracking();
            }
            return;
        }
        
        // Continue animation
        autoScrollAnimationId = requestAnimationFrame(animate);
    };
    
    // Start animation
    autoScrollAnimationId = requestAnimationFrame(animate);
}

// Stop automatic scrolling
export function stopAutoScroll() {
    isAutoScrolling = false;
    if (autoScrollAnimationId) {
        cancelAnimationFrame(autoScrollAnimationId);
        autoScrollAnimationId = null;
    }
}

// Start tracking a scrolling/zooming session
export function startPerfTracking() {
    if (isPerfTracking) return;
    
    isPerfTracking = true;
    scrollTrackingActive = false; // Start with stabilization period first
    stabilizationFrames = 0;
    resetPerfMetrics();
    
    // Reset tracking arrays
    actualFrameTimes = [];
    frameTileCounts = [];
    outliersRemoved = 0;
    
    console.log("Performance tracking started with stabilization period");
    return true;
}

// Stop tracking the current scrolling/zooming session
export function stopPerfTracking() {
    if (!isPerfTracking) return false;
    
    isPerfTracking = false;
    scrollTrackingActive = false;
    
    // Also stop auto-scrolling if it's active
    if (isAutoScrolling) {
        stopAutoScroll();
    }
    
    // Remove outliers for more stable measurements
    const cleanedFrameTimes = removeOutliers(actualFrameTimes);
    outliersRemoved = actualFrameTimes.length - cleanedFrameTimes.length;
    
    // Calculate final metrics using cleaned data
    if (cleanedFrameTimes.length > 0) {
        const sum = cleanedFrameTimes.reduce((a, b) => a + b, 0);
        scrollMetrics.avgFrameTime = sum / cleanedFrameTimes.length;
        scrollMetrics.minFrameTime = Math.min(...cleanedFrameTimes);
        scrollMetrics.maxFrameTime = Math.max(...cleanedFrameTimes);
        scrollMetrics.medianFrameTime = calculateMedian(cleanedFrameTimes);
    }
    
    // Calculate tile averages
    if (frameTileCounts.length > 0) {
        const tileSum = frameTileCounts.reduce((a, b) => a + b, 0);
        scrollMetrics.avgTilesPerFrame = tileSum / frameTileCounts.length;
        scrollMetrics.totalTiles = tileSum;
    }
    
    scrollMetrics.totalFrames = actualFrameTimes.length;
    scrollMetrics.outliersRemoved = outliersRemoved;
    
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
    actualFrameTimes = [];
    frameTileCounts = [];
    outliersRemoved = 0;
    
    scrollMetrics = {
        avgFrameTime: 0,
        minFrameTime: Number.MAX_VALUE,
        maxFrameTime: 0,
        avgTilesPerFrame: 0,
        totalFrames: 0,
        totalTiles: 0,
        medianFrameTime: 0,
        outliersRemoved: 0,
        stabilizationFrames: STABILIZATION_FRAME_COUNT
    };
}

// Monitor a frame for performance
export function monitorFrameStart() {
    renderStart = performance.now();
    
    // Calculate frame time for this frame
    const now = performance.now();
    const frameTime = now - lastFrameTime;
    lastFrameTime = now;
    
    // Handle stabilization period before actual measurements
    if (isPerfTracking) {
        if (!scrollTrackingActive) {
            stabilizationFrames++;
            
            // After stabilization period, start actual tracking
            if (stabilizationFrames >= STABILIZATION_FRAME_COUNT) {
                scrollTrackingActive = true;
                console.log(`Stabilization complete (${STABILIZATION_FRAME_COUNT} frames), starting measurement`);
            }
            
            // Reset tile counter during stabilization too
            lastTileCount = 0;
            return frameTime;
        }
        
        // Only track metrics after stabilization
        if (frameTime > 0) { // Avoid zero values which might happen if function is called twice
            actualFrameTimes.push(frameTime);
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
    
    // Store tile count for this frame if we're tracking
    if (scrollTrackingActive && lastTileCount > 0) {
        frameTileCounts.push(lastTileCount);
    }
    
    return renderTime;
}

// Track a tile being rendered
export function trackTileRender() {
    lastTileCount++;
    
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
        inStabilization: isPerfTracking && !scrollTrackingActive,
        stabilizationFrames,
        scrollMetrics,
        isAutoScrolling,
        autoScrollPosition,
        autoScrollMaxPosition
    };
}

// Check if auto-scroll is active
export function isAutoScrollActive() {
    return isAutoScrolling;
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
    if (actualFrameTimes.length === 0) {
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
    if (actualFrameTimes.length === 0) {
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
    return `Frame Time (avg): ${metrics.avgFrameTime.toFixed(2)}ms
Frame Time (median): ${metrics.medianFrameTime ? metrics.medianFrameTime.toFixed(2) : 'N/A'}ms
Min/Max Frame Time: ${metrics.minFrameTime.toFixed(2)}ms / ${metrics.maxFrameTime.toFixed(2)}ms
Avg. Tiles per Frame: ${Math.round(metrics.avgTilesPerFrame)}
Total Frames: ${metrics.totalFrames}
Total Tiles Rendered: ${metrics.totalTiles}
Outliers Removed: ${metrics.outliersRemoved || 0}`;
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
            
            // Calculate and display differences (prefer median if available)
            const baselineValue = baselineResults.medianFrameTime || baselineResults.avgFrameTime;
            const optValue = opt.medianFrameTime || opt.avgFrameTime;
            const frameTimeDiff = ((baselineValue - optValue) / baselineValue * 100).toFixed(2);
            
            const sign = frameTimeDiff >= 0 ? "+" : "";
            comparisonText += `Frame Time Change: ${sign}${frameTimeDiff}%\n`;
            
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
    ctx.fillRect(10, 10, 220, 150);
    
    ctx.font = '14px monospace';
    ctx.fillStyle = '#fff';
    
    // Basic metrics
    ctx.fillText(`Frame time: ${renderTime.toFixed(2)}ms`, 20, 30);
    ctx.fillText(`Tiles rendered: ${lastTileCount}`, 20, 50);
    
    // Recording status
    if (isPerfTracking) {
        if (!scrollTrackingActive) {
            ctx.fillStyle = '#FFA500'; // Orange for stabilization
            ctx.fillText(`STABILIZING (${stabilizationFrames}/${STABILIZATION_FRAME_COUNT})`, 20, 70);
        } else {
            ctx.fillStyle = '#4CAF50'; // Green for active recording
            ctx.fillText(`RECORDING (${actualFrameTimes.length} frames)`, 20, 70);
        }
    } else if (actualFrameTimes.length > 0) {
        const usedMetric = scrollMetrics.medianFrameTime || scrollMetrics.avgFrameTime;
        ctx.fillText(`Last test: ${usedMetric.toFixed(2)}ms`, 20, 90);
        ctx.fillText(`Frames: ${scrollMetrics.totalFrames}`, 20, 110);
    }
    
    // Auto-scroll status if active
    if (isAutoScrolling) {
        const progress = (autoScrollPosition / autoScrollMaxPosition * 100).toFixed(1);
        ctx.fillStyle = '#2196F3'; // Blue for auto-scroll
        ctx.fillText(`AUTO-SCROLL: ${progress}%`, 20, 130);
    }
    
    ctx.restore();
}