package step_definitions

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"test-tool/mock_elsa_server" // Ensure this path is correct
	"time"

	"github.com/cucumber/godog"
)

const MockServerKey TestContextKey = "mockElsaServer"

var serverInstance *mock_elsa_server.MockElsaAPIServer
var httpServer *http.Server

func mockElsaAPIServerIsRunning(ctx context.Context) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context, cannot start mock server")
	}

	if serverInstance == nil {
		serverInstance = mock_elsa_server.NewMockElsaAPIServer()
		// The mock server needs the base URL to correctly parse {TXID} if it's part of the path prefix
		// However, for this PoC, the handler will be simple.
		// The actual listen address comes from the config's ElsaAPIBaseURL port.
		// For simplicity, assuming ElsaAPIBaseURL is like "http://localhost:8080"

		addr := ":8080" // Extract from cfg.ElsaAPIBaseURL if more dynamic
		// This is a simplified way to get the address. A real app might parse the URL.
		if len(cfg.ElsaAPIBaseURL) > 7 { // len("http://")
			hostAndPort := cfg.ElsaAPIBaseURL[7:] // Remove "http://"
			if colonIndex := strings.LastIndex(hostAndPort, ":"); colonIndex != -1 {
				addr = hostAndPort[colonIndex:]
			}
		}

		httpServer = &http.Server{
			Addr:    addr,
			Handler: serverInstance,
		}

		fmt.Printf("Starting mock ELSA API server on %s...", addr)
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "Mock ELSA API server ListenAndServe error: %v", err)
				// In a real test suite, you might want to panic or signal an error channel
			}
		}()
		// Give the server a moment to start. In a real scenario, use health checks.
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Mock ELSA API server started.")
	}
	return context.WithValue(ctx, MockServerKey, serverInstance), nil
}

// BeforeScenarioHook cleans up mock MQ directories and resets mock server state
func BeforeScenarioHook(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	fmt.Println("Executing BeforeScenarioHook...")
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if ok && cfg != nil {
		fmt.Printf("Cleaning up mock MQ root directory: %s", cfg.MockMQRootDir)
		// Clean up specific queue paths
		if cfg.T2SClientRequestQueuePath != "" {
			if err := os.RemoveAll(cfg.T2SClientRequestQueuePath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove mock queue dir %s: %v", cfg.T2SClientRequestQueuePath, err)
			}
			if err := os.MkdirAll(cfg.T2SClientRequestQueuePath, 0755); err != nil {
				return ctx, fmt.Errorf("failed to recreate mock MQ dir %s: %w", cfg.T2SClientRequestQueuePath, err)
			}
		}
		if cfg.T2SAcceptanceQueuePath != "" && cfg.T2SAcceptanceQueuePath != cfg.T2SClientRequestQueuePath { // Avoid double remove/create
			if err := os.RemoveAll(cfg.T2SAcceptanceQueuePath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove mock queue dir %s: %v", cfg.T2SAcceptanceQueuePath, err)
			}
			if err := os.MkdirAll(cfg.T2SAcceptanceQueuePath, 0755); err != nil {
				return ctx, fmt.Errorf("failed to recreate mock MQ dir %s: %w", cfg.T2SAcceptanceQueuePath, err)
			}
		}
	} else {
		fmt.Println("Config not found in BeforeScenarioHook, skipping MQ cleanup.")
	}

	// Reset mock server state if it's running
	if serverInstance != nil {
		fmt.Println("Resetting mock server state.")
		serverInstance.ResetState()
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
			fmt.Fprintf(os.Stderr, "Mock ELSA API server shutdown error: %v", err)
		}
		fmt.Println("Mock ELSA API server shut down.")
		httpServer = nil
		serverInstance = nil
	}
}

func InitializeHookSteps(s *godog.ScenarioContext) {
	s.Step(`^the mock ELSA API server is running$`, mockElsaAPIServerIsRunning)

	s.Before(BeforeScenarioHook) // Godog v0.12.x uses s.Before
}

// Note: For AfterSuite, Godog doesn't have a direct hook in ScenarioContext.
// It's typically handled in TestMain or by registering a function with `testing.M.Run()`.
// For this PoC, we'll call AfterSuiteHook from main_test.go.
