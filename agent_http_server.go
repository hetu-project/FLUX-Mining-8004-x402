// Agent HTTP Server for TEE Validator Interaction
//
// This HTTP server exposes the agent's VLC capabilities to the TEE validator.
// The TEE validator will interact with this server to perform validation tests.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hetu-project/FLUX-Mining-8004-x402/subnet"
	"github.com/hetu-project/FLUX-Mining-8004-x402/subnet/demo"
)

// Global agent instance (set by validation mode)
var globalMiner *subnet.CoreMiner

// VLCStateResponse represents the agent's current VLC state
type VLCStateResponse struct {
	Clock  map[uint64]uint64 `json:"clock"`
	Events []string          `json:"events"`
}

// ProcessTaskRequest represents a task processing request from the validator
type ProcessTaskRequest struct {
	Task      string `json:"task"`
	NodeID    int    `json:"nodeId"`
	RequestID string `json:"requestId"`
}

// ProcessAdditionalInfoRequest represents additional info request
type ProcessAdditionalInfoRequest struct {
	OriginalTask   string `json:"originalTask"`
	AdditionalInfo string `json:"additionalInfo"`
	NodeID         int    `json:"nodeId"`
	RequestID      string `json:"requestId"`
}

// AgentResponse represents the agent's response to a task
type AgentResponse struct {
	OutputType  string            `json:"outputType"`
	Output      string            `json:"output,omitempty"`
	InfoRequest string            `json:"infoRequest,omitempty"`
	VLCClock    map[uint64]uint64 `json:"vlcClock"`
}

// Handler: Get current VLC state
func handleVLCState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if globalMiner == nil {
		http.Error(w, "Agent not initialized", http.StatusInternalServerError)
		return
	}

	currentClock := globalMiner.GetCurrentClock()

	response := VLCStateResponse{
		Clock:  currentClock.Values,
		Events: []string{}, // Events are tracked elsewhere
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler: Process task
func handleProcessTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if globalMiner == nil {
		http.Error(w, "Agent not initialized", http.StatusInternalServerError)
		return
	}

	var req ProcessTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process the task through the miner
	minerResponse := globalMiner.ProcessInput(req.Task, req.NodeID, req.RequestID)

	// Convert to HTTP response format
	response := AgentResponse{
		OutputType:  string(minerResponse.OutputType),
		Output:      minerResponse.Output,
		InfoRequest: minerResponse.InfoRequest,
		VLCClock:    minerResponse.VLCClock.Values,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler: Process additional info
func handleProcessAdditionalInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if globalMiner == nil {
		http.Error(w, "Agent not initialized", http.StatusInternalServerError)
		return
	}

	var req ProcessAdditionalInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process additional info through the miner
	minerResponse := globalMiner.ProcessAdditionalInfo(
		req.OriginalTask,
		req.AdditionalInfo,
		req.NodeID,
		req.RequestID,
	)

	// Convert to HTTP response format
	response := AgentResponse{
		OutputType:  string(minerResponse.OutputType),
		Output:      minerResponse.Output,
		InfoRequest: minerResponse.InfoRequest,
		VLCClock:    minerResponse.VLCClock.Values,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler: Health check
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "flux-mining-agent",
	})
}

// StartAgentHTTPServer starts the HTTP server for TEE validator interaction
func StartAgentHTTPServer(miner *subnet.CoreMiner, port string) error {
	globalMiner = miner

	http.HandleFunc("/vlc-state", handleVLCState)
	http.HandleFunc("/process-task", handleProcessTask)
	http.HandleFunc("/process-additional-info", handleProcessAdditionalInfo)
	http.HandleFunc("/health", handleHealth)

	fmt.Printf("\n🌐 Agent HTTP Server Starting...\n")
	fmt.Printf("   Port: %s\n", port)
	fmt.Printf("   Endpoints:\n")
	fmt.Printf("   - GET  /vlc-state\n")
	fmt.Printf("   - POST /process-task\n")
	fmt.Printf("   - POST /process-additional-info\n")
	fmt.Printf("   - GET  /health\n")
	fmt.Printf("\n")

	return http.ListenAndServe(":"+port, nil)
}

// RunAgentServerForTEEValidation runs the agent in server mode for TEE validation
func RunAgentServerForTEEValidation() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          AGENT HTTP SERVER FOR TEE VALIDATION               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create a miner instance for validation with demo task processor
	// The demo task processor properly handles the "Calculate the optimal route" test
	miner := subnet.NewCoreMiner("1", "Agent-1")
	miner.SetTaskProcessor(demo.NewDemoTaskProcessor())

	// Get port from environment or use default
	port := os.Getenv("AGENT_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	// Start HTTP server (this blocks)
	fmt.Printf("🚀 Starting agent HTTP server on port %s...\n", port)
	fmt.Printf("   TEE Validator will connect to: http://localhost:%s\n", port)
	fmt.Println()

	if err := StartAgentHTTPServer(miner, port); err != nil {
		fmt.Printf("❌ Failed to start HTTP server: %v\n", err)
		os.Exit(1)
	}
}
