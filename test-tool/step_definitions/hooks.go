package step_definitions

import (
	"context"
	"fmt"
	"net/http"
	"net/url" // Added import
	"os"
	"strconv" // Added import
	"strings"
	"sync" // Added for server start synchronization
	"test-tool/mock_elsa_server"
	"time"

	"github.com/cucumber/godog"
)

const MockServerKey TestContextKey = "mockElsaServer"

var serverInstance *mock_elsa_server.MockElsaAPIServer
var httpServer *http.Server
var serverStartOnce sync.Once
var serverErr error

// ensureMockServerIsRunningAndResetState starts the mock server if not already running,
// and resets its state. It's designed to be called before each scenario.
func ensureMockServerIsRunningAndResetState(ctx context.Context) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil { // Attempt to load a default config if not found, for server startup
		defaultCfg, err := LoadConfig("testdata/config/elsa_services.json")
		if err != nil {
			return ctx, fmt.Errorf("configuration not found and default config failed to load: %w", err)
		}
		cfg = defaultCfg
		ctx = context.WithValue(ctx, ConfigKey, cfg) // Add loaded default config to context
		fmt.Println("Loaded default configuration for server startup.")
	}

	serverStartOnce.Do(func() {
		serverInstance = mock_elsa_server.NewMockElsaAPIServer()
		addr := ":8080" // Default port

		if cfg.ElsaAPIBaseURL != "" {
			parsedURL, err := url.Parse(cfg.ElsaAPIBaseURL)
			if err == nil && parsedURL.Port() != "" {
				addr = ":" + parsedURL.Port()
			} else if err == nil && parsedURL.Host != "" && !strings.Contains(parsedURL.Host, ":") {
				// If URL is like http://localhost (no port), default to 80 or 443 based on scheme
				// For this mock, we'll stick to a configurable default or 8080 if not parseable.
				// Or, if host is just a port like ":8081"
				if strings.HasPrefix(parsedURL.Host, ":") {
					addr = parsedURL.Host
				}
			} else if err != nil {
				// Fallback for simple "hostname:port" or ":port" strings if url.Parse fails
				parts := strings.Split(cfg.ElsaAPIBaseURL, ":")
				if len(parts) > 1 {
					potentialPort := parts[len(parts)-1]
					if _, e := strconv.Atoi(potentialPort); e == nil {
						addr = ":" + potentialPort
					}
				}
			}
		}

		httpServer = &http.Server{
			Addr:    addr,
			Handler: serverInstance,
		}

		fmt.Printf("Attempting to start mock ELSA API server on %s...\n", addr)
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr = fmt.Errorf("mock ELSA API server ListenAndServe error: %v", err)
				fmt.Fprintln(os.Stderr, serverErr)
			}
		}()
		// Increased sleep and added a simple check. Replace with a proper health check.
		time.Sleep(250 * time.Millisecond)
		// Basic check (not a real health check)
		if serverErr == nil {
			fmt.Printf("Mock ELSA API server presumed started on %s.\n", addr)
		} else {
			fmt.Printf("Mock ELSA API server may not have started correctly on %s.\n", addr)
		}
	})

	if serverErr != nil {
		return ctx, fmt.Errorf("mock server failed to start previously: %w", serverErr)
	}
	if serverInstance == nil {
		// This should not happen if serverStartOnce.Do completed without error
		return ctx, fmt.Errorf("server instance is nil after startup attempt")
	}

	fmt.Println("Resetting mock server state.")
	serverInstance.ResetState()
	return context.WithValue(ctx, MockServerKey, serverInstance), nil
}

func mockElsaAPIServerIsRunning(ctx context.Context) (context.Context, error) {
	fmt.Println("Step: mock ELSA API server is running")
	if serverInstance == nil || serverErr != nil {
		return ctx, fmt.Errorf("mock server is not running or encountered an error: %v", serverErr)
	}
	fmt.Println("Mock ELSA API server is confirmed running.")
	return ctx, nil
}

// BeforeScenarioHook ensures the mock server is running and state is reset,
// and cleans up mock MQ directories.
func BeforeScenarioHook(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	fmt.Println("Executing BeforeScenarioHook...")
	var err error

	// Ensure server is running and state is reset
	ctx, err = ensureMockServerIsRunningAndResetState(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed in BeforeScenarioHook while ensuring server state: %w", err)
	}

	// MQ Cleanup logic
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if ok && cfg != nil {
		fmt.Printf("Cleaning up mock MQ directories based on config (root: %s)\n", cfg.MockMQRootDir)
		pathsToClean := []string{
			cfg.T2SClientRequestQueuePath,
			cfg.T2SAcceptanceQueuePath,
			cfg.CreationRequestQueuePath,
			cfg.CreationAcceptanceQueuePath,
			// Add any other dynamic queue paths from config here
		}
		cleanedPaths := make(map[string]bool) // To avoid double processing

		for _, qPath := range pathsToClean {
			if qPath != "" && !cleanedPaths[qPath] {
				fmt.Printf("Cleaning and recreating: %s\n", qPath)
				if err := os.RemoveAll(qPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove mock queue dir %s: %v\n", qPath, err)
				}
				if err := os.MkdirAll(qPath, 0755); err != nil {
					// Log error but continue if possible, or return error if critical
					fmt.Fprintf(os.Stderr, "Error: failed to recreate mock MQ dir %s: %v. Continuing cleanup.\n", qPath, err)
					// return ctx, fmt.Errorf("failed to recreate mock MQ dir %s: %w", qPath, err)
				}
				cleanedPaths[qPath] = true
			}
		}
	} else {
		fmt.Println("Config not found in BeforeScenarioHook, attempting to skip MQ cleanup or use defaults if applicable.")
		// Optionally, try to load a default config here too for MQ cleanup if that makes sense for your setup
	}

	return ctx, nil
}

// AfterSuiteHook shuts down the mock server
func AfterSuiteHook() {
	fmt.Println("Executing AfterSuiteHook...")
	if httpServer != nil {
		fmt.Println("Shutting down mock ELSA API server...")
		ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctxShutDown); err != nil {
			fmt.Fprintf(os.Stderr, "Mock ELSA API server shutdown error: %v\n", err)
		} else {
			fmt.Println("Mock ELSA API server shut down successfully.")
		}
		httpServer = nil
		serverInstance = nil
		serverErr = nil               // Reset server error
		serverStartOnce = sync.Once{} // Reset sync.Once for potential re-runs in tests (though typically AfterSuite is final)

	} else {
		fmt.Println("Mock ELSA API server was not running or already shut down.")
	}
}

func InitializeHookSteps(s *godog.ScenarioContext) {
	s.Step(`^the mock ELSA API server is running$`, mockElsaAPIServerIsRunning)

	s.Before(BeforeScenarioHook) // Godog v0.12.x uses s.Before
}

// Note: For AfterSuite, Godog doesn't have a direct hook in ScenarioContext.
// It's typically handled in TestMain or by registering a function with `testing.M.Run()`.
// For this PoC, we'll call AfterSuiteHook from main_test.go.
