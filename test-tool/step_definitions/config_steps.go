package step_definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
)

// Config holds the application configuration
type Config struct {
	ElsaAPIBaseURL            string `json:"elsaApiBaseUrl"`
	T2SClientRequestQueuePath string `json:"t2sClientRequestQueuePath"`
	T2SAcceptanceQueuePath    string `json:"t2sAcceptanceQueuePath"`
	MockMQRootDir             string `json:"mockMqRootDir"`
	PollingIntervalSeconds    int    `json:"pollingIntervalSeconds"`
	PollingTimeoutSeconds     int    `json:"pollingTimeoutSeconds"`
}

// TestContextKey is used as a key for values in context.Context
type TestContextKey string

const ConfigKey TestContextKey = "config"
const PreparedMessageKey TestContextKey = "preparedMessage"
const CurrentTXIDKey TestContextKey = "currentTXID"
const CurrentMitiTXIDKey TestContextKey = "currentMitiTXID" // Added for MitiTXID

// LoadConfig loads configuration from the specified file path
func LoadConfig(filePath string) (*Config, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path for config: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data from %s: %w", absPath, err)
	}
	return &cfg, nil
}

func elsaServicesAreConfiguredFrom(ctx context.Context, configFile string) (context.Context, error) {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return ctx, fmt.Errorf("failed to load configuration: %w", err)
	}
	// Ensure mock MQ directories exist
	if err := os.MkdirAll(filepath.Dir(cfg.T2SClientRequestQueuePath), 0755); err != nil {
		return ctx, fmt.Errorf("failed to create mock MQ dir for T2SClientRequestQueuePath: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.T2SAcceptanceQueuePath), 0755); err != nil {
		return ctx, fmt.Errorf("failed to create mock MQ dir for T2SAcceptanceQueuePath: %w", err)
	}

	return context.WithValue(ctx, ConfigKey, cfg), nil
}

func InitializeConfigSteps(s *godog.ScenarioContext) {
	s.Step(`^the ELSA services are configured from "([^"]*)"$`, elsaServicesAreConfiguredFrom)
}
