package step_definitions

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"test-tool/mock_elsa_server"
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
			templateData[fieldName] = fieldValue // Store with original field name

			// Handle TXID variations for context and template
			if fieldName == "TransactionId" || fieldName == "TXID" {
				currentTXID = fieldValue
				templateData["TXID"] = fieldValue // Ensure template data has "TXID" key
			}
			// Handle MitiTXID variations (if Gherkin might use different names)
			if fieldName == "MitiTransactionId" || fieldName == "MitiTXID" {
				currentMitiTXID = fieldValue
				templateData["MitiTXID"] = fieldValue // Ensure template data has "MitiTXID" key
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

func t2sSendsThePreparedMessageToQueueWithCorrelationID(ctx context.Context, queueIdentifierKey string, correlationID string) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context")
	}

	payload, ok := ctx.Value(PreparedMessageKey).(string)
	if !ok || payload == "" {
		return ctx, fmt.Errorf("no prepared message found in context to send")
	}

	var queuePath string
	switch queueIdentifierKey {
	case "t2sClientRequestQueueName": // Matches the string in the feature file
		queuePath = cfg.T2SClientRequestQueuePath
	case "t2sAcceptanceQueueName": // Matches the string in the feature file
		queuePath = cfg.T2SAcceptanceQueuePath
	default:
		return ctx, fmt.Errorf("unknown queue identifier key: %s. Expected 't2sClientRequestQueueName' or 't2sAcceptanceQueueName'", queueIdentifierKey)
	}

	if queuePath == "" {
		return ctx, fmt.Errorf("queue path is empty for identifier %s, check config elsa_services.json", queueIdentifierKey)
	}

	// Ensure the directory for the queue exists
	// The queuePath from config is expected to be a directory.
	err := os.MkdirAll(queuePath, 0755)
	if err != nil {
		return ctx, fmt.Errorf("failed to create mock MQ directory %s: %w", queuePath, err)
	}

	fileName := correlationID + ".xml"
	if queueIdentifierKey == "t2sAcceptanceQueueName" {
		fileName = correlationID + "_accept.xml"
	}
	filePath := filepath.Join(queuePath, fileName)

	err = os.WriteFile(filePath, []byte(payload), 0644)
	if err != nil {
		return ctx, fmt.Errorf("failed to write message to mock queue file %s: %w", filePath, err)
	}
	fmt.Printf("MockMQ: Message written to %s\\n", filePath)

	// After writing to the mock queue, simulate ELSA receiving the message by setting an initial status
	mockServer, ok := ctx.Value(MockServerKey).(*mock_elsa_server.MockElsaAPIServer)
	if !ok || mockServer == nil {
		return ctx, fmt.Errorf("mock ELSA API server not found in context")
	}

	mockServer.SetInstructionStatus(correlationID, "created") // TODO

	return context.WithValue(ctx, CurrentTXIDKey, correlationID), nil
}

func InitializeMessageSteps(s *godog.ScenarioContext) {
	s.Step(`^T2S prepares an initial client request message using template "([^"]*)" with values:$`, t2sPreparesAMessageWithValues)
	s.Step(`^T2S prepares an acceptance message using template "([^"]*)" with values:$`, t2sPreparesAMessageWithValues)
	s.Step(`^T2S sends the prepared message to the "([^"]*)" queue with correlation ID "([^"]*)"$`, t2sSendsThePreparedMessageToQueueWithCorrelationID)
	// Remove or comment out old step definitions if they are no longer used by this new single step
	// s.Step(\`^T2S sends the prepared message to the T2S client request queue with client TXID "([^"]*)"$\`, t2sSendsThePreparedMessageToQueueWithClientTXID)
	// s.Step(\`^T2S sends the prepared acceptance message to the T2S acceptance queue with client TXID "([^"]*)"$\`, t2sSendsThePreparedMessageToQueueWithClientTXID)
}
