package mock_elsa_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	// "elsa-test-tool/step_definitions" // Avoid circular dependency if not strictly needed for types here
)

// InstructionState holds the current status of a transaction
type InstructionState struct {
	TXID          string           `json:"txID"`          // Client's TXID
	MitiTXID      string           `json:"mitiTXID"`      // Miti TXID (used in API response as 'txID')
	StatusHistory []APIStatusEntry `json:"statusHistory"` // Chronological, newest first
	// Add any other relevant fields ELSA might track
}

// APIStatusEntry matches the structure in the API response
type APIStatusEntry struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"` // Link to message API
}

// MockElsaAPIServer simulates the ELSA REST API
type MockElsaAPIServer struct {
	mu sync.RWMutex
	// instructions map[string]*InstructionState // Keyed by ClientTXID
	instructions map[string][]APIStatusEntry // Keyed by ClientTXID, stores status history
	apiBaseURL   string                      // Not strictly needed for handler if paths are fixed, but good for context
}

// NewMockElsaAPIServer creates a new mock server instance
func NewMockElsaAPIServer() *MockElsaAPIServer {
	return &MockElsaAPIServer{
		instructions: make(map[string][]APIStatusEntry),
	}
}

// ResetState clears all stored instruction states.
func (s *MockElsaAPIServer) ResetState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instructions = make(map[string][]APIStatusEntry)
	fmt.Println("MockElsaAPIServer state reset.")
}

// SetInstructionStatus sets or updates the status of an instruction.
// This is how tests can control the mock's behavior.
func (s *MockElsaAPIServer) SetInstructionStatus(clientTXID string, newStatusName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	messageLink := fmt.Sprintf("http://localhost:8080/messages/id/mock-%s-%s", clientTXID, newStatusName) // Example message link

	newEntry := APIStatusEntry{
		Name:      newStatusName,
		Timestamp: timestamp,
		Message:   messageLink,
	}

	// Prepend new status to maintain newest-first order
	s.instructions[clientTXID] = append([]APIStatusEntry{newEntry}, s.instructions[clientTXID]...)
	fmt.Printf("MockServer: Set status for %s to %s. History: %+v", clientTXID, newStatusName, s.instructions[clientTXID])
}

// ServeHTTP handles incoming HTTP requests
func (s *MockElsaAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("MockServer: Received request: %s %s", r.Method, r.URL.Path)
	s.mu.RLock() // Use RLock for read-heavy operations initially

	// Example: /instructions/{TXID}
	if strings.HasPrefix(r.URL.Path, "/instructions/") && r.Method == http.MethodGet {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/instructions/"), "/")
		clientTXID := parts[0]

		statusHistory, exists := s.instructions[clientTXID]
		s.mu.RUnlock() // Unlock before writing response or further processing

		if !exists || len(statusHistory) == 0 {
			fmt.Printf("MockServer: Instruction %s not found or no status history", clientTXID)
			http.Error(w, fmt.Sprintf(`{"error": "Instruction not found", "transactionId": "%s"}`, clientTXID), http.StatusNotFound)
			return
		}

		// Construct the API response based on your provided structure
		// The 'txID' in the main response body is the MitiTXID
		mitiTXID := "miti-" + clientTXID // Construct MitiTXID

		apiResponse := struct { // Using an anonymous struct for the response shape
			Href                  string            `json:"href"`
			InstructionType       string            `json:"instructionType"`
			InstructingParty      string            `json:"instructingParty"`
			TxID                  string            `json:"txID"` // This is MitiTXID
			MovementType          string            `json:"movementType"`
			PaymentType           string            `json:"paymentType"`
			CancellationRequested bool              `json:"cancellationRequested"`
			Status                []APIStatusEntry  `json:"status"`
			Links                 map[string]string `json:"links"`
		}{
			Href:                  fmt.Sprintf("http://localhost:8080/instructions/id/%s/audit", clientTXID), // Example
			InstructionType:       "elsa_to_t2s",                                                             // Example
			InstructingParty:      "MOCKT2SXXX",                                                              // Example
			TxID:                  mitiTXID,
			MovementType:          "RECE", // Example
			PaymentType:           "APMT", // Example
			CancellationRequested: false,  // Example
			Status:                statusHistory,
			Links: map[string]string{
				"instruction": fmt.Sprintf("http://localhost:8080/instructions/id/%s", clientTXID),
				"collection":  fmt.Sprintf("http://localhost:8080/collections/id/mock-%s", clientTXID),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiResponse)
		fmt.Printf("MockServer: Responded for %s with status history: %+v", clientTXID, statusHistory)
		return
	} else {
		s.mu.RUnlock() // Ensure RUnlock is called if not handled by specific path
	}

	http.NotFound(w, r)
}
