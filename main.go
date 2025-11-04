// Proof-of-Causal-Work (PoCW) Per-Epoch Integration
//
// This is the main entry point for the PoCW subnet with real-time per-epoch
// blockchain integration, showcasing a distributed consensus system where
// AI agents (miners) process user tasks and immediately submit verified
// intelligence work to the blockchain for FLUX token mining.
//
// Architecture:
//   - Miners: AI entities that process user requests with VLC consistency
//   - Validators: Quality assessors using Byzantine Fault Tolerant consensus  
//   - VLC: Vector Logical Clocks ensure causal ordering of operations
//   - Per-Epoch Integration: Real-time blockchain submission every 3 rounds
//   - Intelligence Money: Verifiable work tokens based on actual task success

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hetu-project/FLUX-Mining-8004-x402/dgraph"
	"github.com/hetu-project/FLUX-Mining-8004-x402/subnet/demo"
)


// waitForDgraph waits for Dgraph to be fully ready
func waitForDgraph() error {
	maxRetries := 15
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:8080/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		fmt.Printf("Dgraph not ready yet (attempt %d/%d), waiting %v...\n", i+1, maxRetries, retryInterval)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("dgraph not ready after %d attempts", maxRetries)
}

// main demonstrates the per-epoch PoCW integration
func main() {
	// Check if running in subnet-only mode
	subnetOnlyMode := os.Getenv("SUBNET_ONLY_MODE") == "true"
	
	if subnetOnlyMode {
		fmt.Println("=== PoCW Subnet-Only Demo ===")
		fmt.Println("Architecture: Pure subnet consensus with VLC visualization")
		fmt.Println("")
	} else {
		fmt.Println("=== PoCW Per-Epoch Mainnet Integration Demo ===")
		fmt.Println("Architecture: Real-time epoch submission (every 3 rounds)")
		fmt.Println("")
	}

	// Bridge is started by the bash script, so we don't need to start it here
	// The Go code will communicate with the bridge via HTTP on port 3001

	// Try to initialize Dgraph gracefully
	fmt.Println("Waiting for Dgraph to be ready...")
	if err := waitForDgraph(); err != nil {
		fmt.Printf("Dgraph not available: %v\n", err)
		fmt.Println("Running demo without graph visualization...")
	} else {
		fmt.Println("Initializing Dgraph connection...")
		dgraph.InitDgraph("localhost:9080")
		fmt.Println("Dgraph initialized successfully!")
	}

	// Create demo coordinator with per-epoch callback integration  
	coordinator := demo.NewDemoCoordinator("per-epoch-subnet-001")
	
	// Set up HTTP bridge URL only if not in subnet-only mode
	if !subnetOnlyMode && coordinator.GraphAdapter != nil {
		fmt.Println("🔗 Setting up per-epoch HTTP bridge integration...")
		
		// Set the bridge URL for HTTP communication
		coordinator.GraphAdapter.SetBridgeURL("http://localhost:3001")
		
		fmt.Println("✅ Per-epoch HTTP bridge configured successfully")
		fmt.Println("📡 Graph adapter will send HTTP requests to JavaScript bridge")
	} else if subnetOnlyMode {
		fmt.Println("🔹 Running in subnet-only mode - no blockchain integration")
	} else {
		fmt.Println("⚠️  GraphAdapter not available - running standard demo")
	}


	// Check if running in validation-only mode
	validationOnlyMode := os.Getenv("VALIDATION_ONLY_MODE") == "true"
	if validationOnlyMode {
		// Run VLC validation ONLY when in validation mode
		fmt.Println("🔐 Running VLC Protocol Validation...")
		if !coordinator.RunVLCValidation() {
			fmt.Println("❌ Agent failed VLC validation - cannot proceed with demo")
			fmt.Println("Exiting...")
			os.Exit(1)
		}
		fmt.Println("✅ Validation complete - exiting (validation-only mode)")
		fmt.Println("   Bash script will now register subnet on blockchain")
		os.Exit(0)
	}

	// Run the subnet demo (validation already passed in previous run)
	coordinator.RunDemo()

	fmt.Println("")
	if subnetOnlyMode {
		fmt.Println("🎉 Subnet-Only Demo Complete!")
	} else {
		fmt.Println("🎉 Per-Epoch Integration Demo Complete!")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Demonstrated real-time epoch submission architecture")
	fmt.Println("✅ Each completed epoch triggers immediate mainnet posting")
	fmt.Println("✅ FLUX tokens are mined in real-time per epoch")
	fmt.Println("")
	fmt.Println("🔍 Visualization Access:")
	fmt.Println("  - Ratel UI: http://localhost:8000")
	fmt.Println("  - Inspector: http://localhost:3000/pocw-inspector.html")
}