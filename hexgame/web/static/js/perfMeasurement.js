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
let tilesPerFrame = 0;
let isPerfTracking = false;
let scrollTrackingActive = false;
let lastTileCount = 0;
let maxTileCount = 0;
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
let autoScrollInitialX = 0; // Store initial X position
let autoScrollAnimationId = null;

// Auto-test phases
let currentTestPhase = 'idle'; // 'idle', 'scroll_min_zoom', 'zoom_test', 'scroll_max_zoom'
let testPhases = [];

// Phase-specific performance metrics
let phaseMetrics = {
    scroll_min_zoom: {
        avgFrameTime: 0,
        minFrameTime: Number.MAX_VALUE,
        maxFrameTime: 0,
        avgTilesPerFrame: 0,
        totalFrames: 0,
        totalTiles: 0,
        frameTimeHistory: [],
        actualFrameTimes: [],
        frameTileCounts: [],
        outliersRemoved: 0
    },
    zoom_test: {
        avgFrameTime: 0,
        minFrameTime: Number.MAX_VALUE,
        maxFrameTime: 0,
        avgTilesPerFrame: 0,
        totalFrames: 0,
        totalTiles: 0,
        frameTimeHistory: [],
        actualFrameTimes: [],
        frameTileCounts: [],
        outliersRemoved: 0
    }
};

// Performance metrics during scroll/zoom operations (legacy - kept for compatibility)
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
    if (isAutoScrolling) return false;
    
    console.log("Starting comprehensive auto-test with scrolling and zoom phases...");
    isAutoScrolling = true;
    
    // Reset phase metrics for new test run
    resetPhaseMetrics();
    
    // Define test phases
    testPhases = [
        { name: 'scroll_min_zoom', description: 'Scroll test at minimum zoom (0.25x)' },
        { name: 'zoom_test', description: 'Zoom performance test (0.25x ↔ 2.5x)' }
    ];
    
    currentTestPhase = 'scroll_min_zoom';
    
    // Start with minimum zoom scroll test
    startScrollTestAtZoom(gameState, setOffset, setZoom, 0.25, cols, rows);
    
    return true;
}

// Run the automatic scrolling animation
function startAutoScrollAnimation(setOffset, setZoom, initialX, initialY, redrawCallback) {
    autoScrollInitialX = initialX; // Store for viewport detection
    
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
        
        // Check if map has left the viewport
        const canvas = document.getElementById('hexCanvas');
        if (canvas) {
            // Calculate if any part of the map is still visible
            const currentOffsetX = initialX - autoScrollPosition;
            const currentOffsetY = initialY; // Y position stays the same during scroll
            const mapVisible = isMapVisibleInViewport(currentOffsetX, currentOffsetY, canvas);
            
            console.log(`Scroll position: ${autoScrollPosition}, offset: (${currentOffsetX}, ${currentOffsetY}), map visible: ${mapVisible}`);
            
            // For debugging, also check if we've scrolled far enough
            // const minScrollDistance = 1000; // Minimum distance to ensure map is off-screen
            // if (autoScrollPosition >= minScrollDistance) {
            //     console.log(`Reached minimum scroll distance ${minScrollDistance}, ending ${currentTestPhase} (debug mode)`);
            //     moveToNextTestPhase(setZoom, redrawCallback);
            //     return;
            // }
            
            if (!mapVisible) {
                console.log(`Map no longer visible in viewport at position ${autoScrollPosition}, ending ${currentTestPhase}`);
                moveToNextTestPhase(setZoom, redrawCallback);
                return;
            }
        }
        
        // Safety check - don't scroll forever
        if (autoScrollPosition >= autoScrollMaxPosition * 2) {
            console.log(`Safety timeout reached, ending ${currentTestPhase}`);
            moveToNextTestPhase(setZoom, redrawCallback);
            return;
        }
        
        // Continue animation
        autoScrollAnimationId = requestAnimationFrame(animate);
    };
    
    // Start animation
    autoScrollAnimationId = requestAnimationFrame(animate);
}

// Check if any part of the map is visible in the viewport
function isMapVisibleInViewport(offsetX, offsetY, canvas) {
    // Check how far we've scrolled from the initial position
    const scrollDistance = Math.abs(offsetX - autoScrollInitialX);
    const minScrollToHide = 6000; // Conservative estimate for map width + viewport
    
    if (scrollDistance > minScrollToHide) {
        console.log(`Map scrolled far enough (${scrollDistance.toFixed(0)}px > ${minScrollToHide}px), considering invisible`);
        return false;
    }
    
    // Fallback to basic check
    const zoom = (window.stateSetters && window.stateSetters.zoom) || 1;
    const cols = (window.stateSetters && window.stateSetters.COLS) || 500;
    const hexSize = 30 * zoom;
    const mapWidth = cols * 1.5 * hexSize + 40;
    const mapRight = offsetX + mapWidth;
    
    const visible = mapRight > 0;
    console.log(`Viewport check: offsetX=${offsetX.toFixed(0)}, mapRight=${mapRight.toFixed(0)}, scrollDistance=${scrollDistance.toFixed(0)}, visible=${visible}`);
    
    return visible;
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
            
            // Also store in current phase metrics if we're in a test phase
            if (currentTestPhase !== 'idle' && phaseMetrics[currentTestPhase]) {
                phaseMetrics[currentTestPhase].actualFrameTimes.push(frameTime);
            }
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
        
        // Also store in current phase metrics if we're in a test phase
        if (currentTestPhase !== 'idle' && phaseMetrics[currentTestPhase]) {
            phaseMetrics[currentTestPhase].frameTileCounts.push(lastTileCount);
        }
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

// Create formatted results for all test phases
export function formatPerfResults(metrics) {
    // If we have phase metrics, show individual phase results
    const phaseData = getPhaseMetrics();
    const hasPhaseData = phaseData.scroll_min_zoom.totalFrames > 0 || 
                        phaseData.zoom_test.totalFrames > 0;
    
    if (hasPhaseData) {
        let result = "PHASE-SPECIFIC PERFORMANCE RESULTS:\n\n";
        
        // Scroll at minimum zoom
        if (phaseData.scroll_min_zoom.totalFrames > 0) {
            result += "📊 SCROLL at 0.25x zoom:\n";
            result += formatSinglePhaseResults(phaseData.scroll_min_zoom);
            result += "\n";
        }
        
        // Zoom test
        if (phaseData.zoom_test.totalFrames > 0) {
            result += "🔍 ZOOM transitions (0.25x ↔ 2.5x):\n";
            result += formatSinglePhaseResults(phaseData.zoom_test);
            result += "\n";
        }
        
        return result.trim();
    }
    
    // Fallback to legacy combined metrics
    return `Frame Time (avg): ${metrics.avgFrameTime.toFixed(2)}ms
Frame Time (median): ${metrics.medianFrameTime ? metrics.medianFrameTime.toFixed(2) : 'N/A'}ms
Min/Max Frame Time: ${metrics.minFrameTime.toFixed(2)}ms / ${metrics.maxFrameTime.toFixed(2)}ms
Avg. Tiles per Frame: ${Math.round(metrics.avgTilesPerFrame)}
Total Frames: ${metrics.totalFrames}
Total Tiles Rendered: ${metrics.totalTiles}
Outliers Removed: ${metrics.outliersRemoved || 0}`;
}

// Format results for a single phase
function formatSinglePhaseResults(phaseMetrics) {
    return `  Frame Time (avg): ${phaseMetrics.avgFrameTime.toFixed(2)}ms
  Frame Time (median): ${phaseMetrics.medianFrameTime ? phaseMetrics.medianFrameTime.toFixed(2) : 'N/A'}ms
  Min/Max Frame Time: ${phaseMetrics.minFrameTime.toFixed(2)}ms / ${phaseMetrics.maxFrameTime.toFixed(2)}ms
  Avg. Tiles per Frame: ${Math.round(phaseMetrics.avgTilesPerFrame)}
  Total Frames: ${phaseMetrics.totalFrames}
  Total Tiles Rendered: ${phaseMetrics.totalTiles}
  Outliers Removed: ${phaseMetrics.outliersRemoved}`;
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

// Start scroll test at specific zoom level
function startScrollTestAtZoom(gameState, setOffset, setZoom, targetZoom, cols, rows) {
    console.log(`Starting ${currentTestPhase}: ${testPhases.find(p => p.name === currentTestPhase)?.description}`);
    
    // Set the target zoom
    setZoom(targetZoom);
    
    // Calculate map dimensions at this zoom
    const hexSize = 30 * targetZoom;
    const hexHeight = Math.sqrt(3) * hexSize;
    const margin = 20;
    
    // Position at the left side and vertical center
    const canvas = document.getElementById('hexCanvas');
    if (!canvas) {
        console.error("Canvas element not found");
        stopAutoScroll();
        return;
    }
    
    const verticalCenter = rows * hexHeight / 2;
    
    // Calculate the initial position - start with map filling the screen
    const initialX = 0; // Start with left edge of map aligned with left edge of screen
    const initialY = canvas.height / 2 - verticalCenter;
    
    // Set starting position
    setOffset(initialX, initialY);
    
    // Calculate the scrolling distance - scroll until map completely leaves viewport
    autoScrollPosition = 0;
    const mapWidth = cols * 1.5 * hexSize + margin * 2;
    // Scroll distance = map width (to move left edge past right edge of viewport)
    autoScrollMaxPosition = mapWidth;
    
    console.log(`Scroll test setup:`, {
        phase: currentTestPhase,
        zoom: targetZoom,
        mapWidth: mapWidth + "px",
        canvasWidth: canvas.width + "px",
        initialX,
        initialY,
        maxPosition: autoScrollMaxPosition
    });
    
    // Get reference to redraw function
    let redrawCallback = window.scheduleDrawGrid;
    if (!redrawCallback) {
        console.warn("No global redraw function found");
    }
    
    // Start the scrolling animation
    startAutoScrollAnimation(setOffset, setZoom, initialX, initialY, redrawCallback);
}

// Start zoom performance test
function startZoomTest(setZoom, redrawCallback) {
    console.log(`🚀 STARTING ZOOM TEST: ${currentTestPhase}`);
    
    // Before starting zoom test, reset map position to be visible
    // Use stateSetters to access setOffset function
    if (window.stateSetters && window.stateSetters.setOffset) {
        const canvas = document.getElementById('hexCanvas');
        if (canvas) {
            // Get current zoom level for accurate positioning
            // Use a reasonable default since stateSetters.zoom might not be current
            const currentZoom = 1.0; // Fixed value for positioning - map should be visible at any zoom
            const cols = window.stateSetters.COLS || 500;
            const rows = window.stateSetters.ROWS || 500;
            
            console.log(`Zoom test setup: canvas=${canvas.width}x${canvas.height}, zoom=${currentZoom}, map=${cols}x${rows}`);
            
            const hexSize = 30 * currentZoom;
            const hexHeight = Math.sqrt(3) * hexSize;
            const margin = 20;
            
            // Calculate map dimensions at current zoom
            const mapWidth = cols * 1.5 * hexSize + margin * 2;
            const mapHeight = Math.ceil(rows / 2) * hexHeight * 2 + margin * 2; // Account for hex staggering
            
            console.log(`Calculated map size: ${mapWidth}x${mapHeight} at zoom ${currentZoom}`);
            
            // Position map so it's visible (not necessarily centered if too big)
            const offsetX = Math.max(0, (canvas.width - mapWidth) / 2);
            const offsetY = Math.max(0, (canvas.height - mapHeight) / 2);
            
            // For debugging, use a simple offset that ensures visibility
            const simpleOffsetX = 0;
            const simpleOffsetY = 0;
            
            console.log(`Setting map offset to: (${simpleOffsetX}, ${simpleOffsetY}) [simple positioning]`);
            window.stateSetters.setOffset(simpleOffsetX, simpleOffsetY);
            
            // Force a redraw to show the repositioned map
            if (redrawCallback) {
                setTimeout(() => {
                    console.log("Forcing redraw after position reset");
                    redrawCallback();
                }, 100);
            }
        } else {
            console.error("Canvas not found for zoom test positioning");
        }
    } else {
        console.error("stateSetters not available for zoom test positioning");
    }
    
    // Perform zoom in/out cycles
    let zoomCycle = 0;
    const maxCycles = 3;
    const zoomLevels = [0.25, 1.0, 2.5, 1.0]; // Cycle through these zoom levels
    
    const performZoomCycle = () => {
        console.log(`🔄 Starting zoom cycle ${zoomCycle + 1}/${maxCycles}`);
        
        if (zoomCycle >= maxCycles) {
            console.log(`✅ All zoom cycles complete, ending zoom test`);
            // Zoom test complete, move to next phase
            moveToNextTestPhase(setZoom, redrawCallback);
            return;
        }
        
        // Perform one complete zoom cycle (min -> max -> min)
        let step = 0;
        const steps = zoomLevels.length;
        
        const zoomStep = () => {
            if (step >= steps) {
                zoomCycle++;
                console.log(`Zoom cycle ${zoomCycle} completed`);
                setTimeout(performZoomCycle, 500); // Brief pause between cycles
                return;
            }
            
            const targetZoom = zoomLevels[step];
            console.log(`Zoom test: cycle ${zoomCycle + 1}/${maxCycles}, step ${step + 1}/${steps}, zooming to ${targetZoom}x`);
            
            try {
                setZoom(targetZoom);
                console.log(`setZoom(${targetZoom}) called successfully`);
            } catch (error) {
                console.error(`Error calling setZoom(${targetZoom}):`, error);
            }
            
            // Ensure redraw happens after zoom change
            if (redrawCallback) {
                setTimeout(() => {
                    console.log(`Redrawing after zoom to ${targetZoom}x`);
                    redrawCallback();
                }, 50);
            } else {
                console.warn("No redraw callback available for zoom test");
            }
            
            step++;
            setTimeout(zoomStep, 800); // Increased delay to 800ms for better visibility
        };
        
        zoomStep();
    };
    
    // Start zoom testing after a brief delay to ensure position reset
    setTimeout(performZoomCycle, 200);
}

// Calculate metrics for a specific test phase
function calculatePhaseMetrics(phaseName) {
    const phase = phaseMetrics[phaseName];
    if (!phase || phase.actualFrameTimes.length === 0) {
        console.warn(`No data for phase: ${phaseName}`);
        return;
    }
    
    // Remove outliers for more stable measurements
    const cleanedFrameTimes = removeOutliers(phase.actualFrameTimes);
    phase.outliersRemoved = phase.actualFrameTimes.length - cleanedFrameTimes.length;
    
    // Calculate final metrics using cleaned data
    if (cleanedFrameTimes.length > 0) {
        const sum = cleanedFrameTimes.reduce((a, b) => a + b, 0);
        phase.avgFrameTime = sum / cleanedFrameTimes.length;
        phase.minFrameTime = Math.min(...cleanedFrameTimes);
        phase.maxFrameTime = Math.max(...cleanedFrameTimes);
        phase.medianFrameTime = calculateMedian(cleanedFrameTimes);
    }
    
    // Calculate tile averages
    if (phase.frameTileCounts.length > 0) {
        const tileSum = phase.frameTileCounts.reduce((a, b) => a + b, 0);
        phase.avgTilesPerFrame = tileSum / phase.frameTileCounts.length;
        phase.totalTiles = tileSum;
    }
    
    phase.totalFrames = phase.actualFrameTimes.length;
    
    console.log(`Phase ${phaseName} metrics:`, phase);
}

// Move to next test phase
function moveToNextTestPhase(setZoom, redrawCallback) {
    console.log(`🔄 PHASE TRANSITION: ${currentTestPhase} → next`);
    
    // Calculate metrics for the completed phase before moving to next
    if (currentTestPhase !== 'idle' && phaseMetrics[currentTestPhase]) {
        calculatePhaseMetrics(currentTestPhase);
        console.log(`📊 Calculated metrics for phase: ${currentTestPhase}`);
    }
    
    const currentIndex = testPhases.findIndex(p => p.name === currentTestPhase);
    console.log(`Current phase index: ${currentIndex}, total phases: ${testPhases.length}`);
    
    if (currentIndex < testPhases.length - 1) {
        // Move to next phase
        const nextPhase = testPhases[currentIndex + 1];
        currentTestPhase = nextPhase.name;
        console.log(`✅ Starting next phase: ${nextPhase.name} - ${nextPhase.description}`);
        
        if (currentTestPhase === 'zoom_test') {
            console.log("Calling startZoomTest");
            startZoomTest(setZoom, redrawCallback);
        }
    } else {
        console.log(`🎯 ALL PHASES COMPLETE: Stopping performance tracking`);
        // Calculate metrics for the final phase
        if (currentTestPhase !== 'idle' && phaseMetrics[currentTestPhase]) {
            calculatePhaseMetrics(currentTestPhase);
            console.log(`📊 Calculated final phase metrics for: ${currentTestPhase}`);
        }
        
        // All phases complete
        console.log("All test phases completed");
        currentTestPhase = 'idle';
        stopAutoScroll();
        if (isPerfTracking) {
            stopPerfTracking();
        }
        
        // Update UI to show completion and results
        if (window.updatePerfResultsDisplay) {
            window.updatePerfResultsDisplay();
        }
    }
}

// Get metrics for all test phases
export function getPhaseMetrics() {
    return {
        scroll_min_zoom: { ...phaseMetrics.scroll_min_zoom },
        zoom_test: { ...phaseMetrics.zoom_test }
    };
}

// Reset phase metrics for a new test run
export function resetPhaseMetrics() {
    Object.keys(phaseMetrics).forEach(phase => {
        phaseMetrics[phase] = {
            avgFrameTime: 0,
            minFrameTime: Number.MAX_VALUE,
            maxFrameTime: 0,
            avgTilesPerFrame: 0,
            totalFrames: 0,
            totalTiles: 0,
            frameTimeHistory: [],
            actualFrameTimes: [],
            frameTileCounts: [],
            outliersRemoved: 0
        };
    });
}