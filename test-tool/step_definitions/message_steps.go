package step_definitions

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cucumber/godog"
)

const messageTemplateDir = "testdata/messages/templates"

func t2sPreparesAMessageWithValues(ctx context.Context, templateFileName string, data *godog.Table) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context")
	}

	templatePath := filepath.Join(messageTemplateDir, templateFileName)
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return ctx, fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	tmpl, err := template.New(templateFileName).Parse(string(tmplContent))
	if err != nil {
		return ctx, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	templateData := make(map[string]interface{})
	currentTXID := ""
	currentMitiTXID := ""

	if len(data.Rows) > 0 { // Check if there are any rows, including header
		header := data.Rows[0]
		if len(header.Cells) != 2 || header.Cells[0].Value != "Field" || header.Cells[1].Value != "Value" {
			return ctx, fmt.Errorf("expected DataTable header to be | Field | Value |, got | %s | %s |", header.Cells[0].Value, header.Cells[1].Value)
		}
		for _, row := range data.Rows[1:] { // Skip header row
			if len(row.Cells) != 2 {
				return ctx, fmt.Errorf("expected 2 cells per row in DataTable, got %d", len(row.Cells))
			}
			fieldName := row.Cells[0].Value
			fieldValue := row.Cells[1].Value
			templateData[fieldName] = fieldValue
			if fieldName == "TXID" {
				currentTXID = fieldValue
			}
			if fieldName == "MitiTXID" { // Store MitiTXID if present
				currentMitiTXID = fieldValue
			}
		}
	}

	// If MitiTXID was not in the table but TXID was, construct MitiTXID
	if currentTXID != "" && currentMitiTXID == "" {
		// Check if the template expects MitiTXID. This is a simple check;
		// a more robust way might involve inspecting template fields.
		if _, ok := templateData["MitiTXID"]; !ok && bytes.Contains(tmplContent, []byte("MitiTXID")) {
			templateData["MitiTXID"] = "miti-" + currentTXID
			currentMitiTXID = "miti-" + currentTXID
		}
	}

	var processedXML bytes.Buffer
	if err := tmpl.Execute(&processedXML, templateData); err != nil {
		return ctx, fmt.Errorf("failed to execute template %s with data %+v: %w", templateFileName, templateData, err)
	}

	newCtx := context.WithValue(ctx, PreparedMessageKey, processedXML.String())
	if currentTXID != "" {
		newCtx = context.WithValue(newCtx, CurrentTXIDKey, currentTXID)
	}
	if currentMitiTXID != "" {
		newCtx = context.WithValue(newCtx, CurrentMitiTXIDKey, currentMitiTXID)
	}
	return newCtx, nil
}

func t2sSendsThePreparedMessageToQueueWithClientTXID(ctx context.Context, queueConfigKey string, clientTXID string) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context")
	}

	payload, ok := ctx.Value(PreparedMessageKey).(string)
	if !ok || payload == "" {
		return ctx, fmt.Errorf("no prepared message found in context to send")
	}

	var queuePath string
	switch queueConfigKey {
	case "T2S client request queue":
		queuePath = cfg.T2SClientRequestQueuePath
	case "T2S acceptance queue":
		queuePath = cfg.T2SAcceptanceQueuePath
	default:
		return ctx, fmt.Errorf("unknown queue configuration key: %s", queueConfigKey)
	}

	// Ensure the directory for the queue exists
	err := os.MkdirAll(queuePath, 0755)
	if err != nil {
		return ctx, fmt.Errorf("failed to create mock MQ directory %s: %w", queuePath, err)
	}

	// Use clientTXID for the filename. Add suffix for acceptance to differentiate if needed.
	fileName := clientTXID + ".xml"
	if queueConfigKey == "T2S acceptance queue" {
		fileName = clientTXID + "_accept.xml"
	}
	filePath := filepath.Join(queuePath, fileName)

	err = os.WriteFile(filePath, []byte(payload), 0644)
	if err != nil {
		return ctx, fmt.Errorf("failed to write message to mock queue file %s: %w", filePath, err)
	}
	fmt.Printf("MockMQ: Message written to %s", filePath)

	// Store the clientTXID from this step, as it's explicitly provided and might be the one to use for API calls.
	return context.WithValue(ctx, CurrentTXIDKey, clientTXID), nil
}

func InitializeMessageSteps(s *godog.ScenarioContext) {
	s.Step(`^T2S prepares an initial client request "([^"]*)" with values:$`, t2sPreparesAMessageWithValues)
	s.Step(`^T2S prepares an acceptance message "([^"]*)" with values:$`, t2sPreparesAMessageWithValues)
	s.Step(`^T2S sends the prepared message to the T2S client request queue with client TXID "([^"]*)"$`, t2sSendsThePreparedMessageToQueueWithClientTXID)
	s.Step(`^T2S sends the prepared acceptance message to the T2S acceptance queue with client TXID "([^"]*)"$`, t2sSendsThePreparedMessageToQueueWithClientTXID)
}
