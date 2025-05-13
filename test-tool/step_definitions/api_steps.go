package step_definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"test-tool/mock_elsa_server" // Import the mock server package
	"time"

	"github.com/cucumber/godog"
)

// InstructionStatus represents one entry in the status array
type InstructionStatus struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// APIInstructionResponse is the structure for GET /instructions/{TXID}
type APIInstructionResponse struct {
	Href                  string              `json:"href"`
	InstructionType       string              `json:"instructionType"`
	InstructingParty      string              `json:"instructingParty"`
	TxID                  string              `json:"txID"` // This is MitiTXID in the API response
	MovementType          string              `json:"movementType"`
	PaymentType           string              `json:"paymentType"`
	CancellationRequested bool                `json:"cancellationRequested"`
	Status                []InstructionStatus `json:"status"`
	Links                 map[string]string   `json:"links"`
}

func elsaInstructionHasStatus(ctx context.Context, clientTXID string, expectedStatus string) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context")
	}

	pollingInterval := time.Duration(cfg.PollingIntervalSeconds) * time.Second
	timeout := time.Duration(cfg.PollingTimeoutSeconds) * time.Second
	startTime := time.Now()

	apiURL := fmt.Sprintf("%s/instructions/%s", cfg.ElsaAPIBaseURL, clientTXID)

	for {
		if time.Since(startTime) > timeout {
			return ctx, fmt.Errorf("timeout after %s waiting for instruction %s to have status %s. Last URL: %s", timeout, clientTXID, expectedStatus, apiURL)
		}

		resp, err := http.Get(apiURL)
		if err != nil {
			fmt.Printf("API call failed for %s: %v. Retrying in %s...", clientTXID, err, pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Important to close the body
		if err != nil {
			fmt.Printf("Failed to read response body for %s: %v. Retrying in %s...", clientTXID, err, pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("API call for %s returned status %d: %s. Retrying in %s...", clientTXID, resp.StatusCode, string(body), pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		var apiResp APIInstructionResponse
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			fmt.Printf("Failed to unmarshal JSON response for %s: %v. Body: %s. Retrying in %s...", clientTXID, err, string(body), pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		if len(apiResp.Status) > 0 {
			currentStatus := apiResp.Status[0].Name
			fmt.Printf("Polled for %s: current API status is %s. Expected: %s", clientTXID, currentStatus, expectedStatus)
			if currentStatus == expectedStatus {
				fmt.Printf("Success: Instruction %s reached expected status %s", clientTXID, expectedStatus)
				return ctx, nil // Expected status reached
			}
		} else {
			fmt.Printf("Polled for %s: API response has empty status array. Retrying in %s...", clientTXID, pollingInterval)
		}

		time.Sleep(pollingInterval)
	}
}

func elsaInstructionShouldNotHaveStatus(ctx context.Context, clientTXID string, unwantedStatus string) (context.Context, error) {
	cfg, ok := ctx.Value(ConfigKey).(*Config)
	if !ok || cfg == nil {
		return ctx, fmt.Errorf("configuration not found in context")
	}

	pollingInterval := time.Duration(cfg.PollingIntervalSeconds) * time.Second
	timeout := time.Duration(cfg.PollingTimeoutSeconds) * time.Second
	startTime := time.Now()

	apiURL := fmt.Sprintf("%s/instructions/%s", cfg.ElsaAPIBaseURL, clientTXID)
	fmt.Printf("Polling (for NOT status %s): %s every %s for %s\\n", unwantedStatus, apiURL, pollingInterval, timeout)

	for {
		if time.Since(startTime) > timeout {
			fmt.Printf("Success (Timeout): Instruction %s did NOT reach status %s within %s\\n", clientTXID, unwantedStatus, timeout)
			return ctx, nil // Timeout reached without finding unwanted status, which is success for this step
		}

		resp, err := http.Get(apiURL)
		if err != nil {
			fmt.Printf("API call failed for %s (when checking for NOT status): %v. Retrying in %s...\\n", clientTXID, err, pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Failed to read response body for %s (when checking for NOT status): %v. Retrying in %s...\\n", clientTXID, err, pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			// If the API returns 404 or other non-200, it might mean the instruction doesn't exist or an error occurred.
			// For a "should NOT have status" check, a 404 could be a valid intermediate state before timeout.
			// However, if the mock server is robust, it should return a valid structure even if the status isn't the one we're avoiding.
			fmt.Printf("API call for %s returned status %d (when checking for NOT status): %s. Retrying in %s...\\n", clientTXID, resp.StatusCode, string(body), pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		var apiResp APIInstructionResponse
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			fmt.Printf("Failed to unmarshal JSON response for %s (when checking for NOT status): %v. Body: %s. Retrying in %s...\\n", clientTXID, err, string(body), pollingInterval)
			time.Sleep(pollingInterval)
			continue
		}

		if len(apiResp.Status) > 0 {
			currentStatus := apiResp.Status[0].Name
			fmt.Printf("Polled for %s (checking for NOT status): current API status is %s. Unwanted: %s\\n", clientTXID, currentStatus, unwantedStatus)
			if currentStatus == unwantedStatus {
				return ctx, fmt.Errorf("failure: instruction %s reached unwanted status %s", clientTXID, unwantedStatus)
			}
		} else {
			fmt.Printf("Polled for %s (checking for NOT status): API response has empty status array. Retrying in %s...\\n", clientTXID, pollingInterval)
		}

		time.Sleep(pollingInterval)
	}
}

func theStepShouldFailDueToTimeout(ctx context.Context) (context.Context, error) {
	// This step is a placeholder. The actual failure will occur in the polling step
	// if the timeout is reached. If this step is reached, it means the polling step
	// did NOT timeout as expected, which is a test failure.
	return ctx, fmt.Errorf("expected a timeout in the previous step, but it seems to have passed")
}

func theTestIsSuccessful(ctx context.Context) (context.Context, error) {
	// This step simply indicates the scenario passed as expected.
	fmt.Println("Test scenario completed successfully.")
	return ctx, nil
}

// New step to control mock server state
func mockElsaAPIWillSetInstructionStatusTo(ctx context.Context, clientTXID, targetStatus string, timing string) (context.Context, error) {
	mockServer, ok := ctx.Value(MockServerKey).(*mock_elsa_server.MockElsaAPIServer)
	if !ok || mockServer == nil {
		return ctx, fmt.Errorf("mock ELSA API server not found in context")
	}

	// For "after the initial message", we assume the message sending step has already occurred.
	// The mock server will be designed to transition states based on "received" messages or direct calls like this.
	// For this PoC, this step directly tells the mock server the desired end-state for a TXID.
	// In a more complex mock, this might trigger a delayed state change.

	// The 'timing' parameter ("after the initial message" or "keep ... as") helps guide the mock's behavior.
	// If "keep ... as", the mock server should ensure it stays in that state or is set to it.
	// If "after the initial message", it implies a transition.

	fmt.Printf("MockServerControl: Setting instruction %s to eventually have status %s (timing: %s)", clientTXID, targetStatus, timing)
	mockServer.SetInstructionStatus(clientTXID, targetStatus) // Direct state set for PoC simplicity

	return ctx, nil
}

func InitializeAPISteps(s *godog.ScenarioContext) {
	s.Step(`^ELSA instruction "([^"]*)" should have status "([^"]*)" within configured polling limits$`, elsaInstructionHasStatus)
	s.Step(`^ELSA instruction "([^"]*)" should NOT have status "([^"]*)" within configured polling limits$`, elsaInstructionShouldNotHaveStatus)
	// s.Step(\`^ELSA instruction "([^"]*)" has status "([^"]*)"$\`, elsaInstructionHasStatus) // Old step, replaced by the one above
	s.Step(`^the step should fail due to timeout$`, theStepShouldFailDueToTimeout)
	s.Step(`^the test is successful$`, theTestIsSuccessful)
	s.Step(`^the mock ELSA API will set instruction "([^"]*)" status to "([^"]*)" after the initial message$`, func(ctx context.Context, clientTXID, targetStatus string) (context.Context, error) {
		return mockElsaAPIWillSetInstructionStatusTo(ctx, clientTXID, targetStatus, "after initial message")
	})
	s.Step(`^the mock ELSA API will keep instruction "([^"]*)" status as "([^"]*)"$`, func(ctx context.Context, clientTXID, targetStatus string) (context.Context, error) {
		return mockElsaAPIWillSetInstructionStatusTo(ctx, clientTXID, targetStatus, "keep as")
	})
}
