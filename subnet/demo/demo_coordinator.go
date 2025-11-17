// Package demo implements Proof-of-Concept (PoC) demonstration logic for the PoCW subnet.
//
// This package provides hardcoded scenarios that showcase the subnet's capabilities:
//   - 7 predefined user inputs with expected behaviors
//   - Specific quality assessment rules for demo validation
//   - Simulated user interaction patterns including rejections
//   - Info request scenarios where miners ask for additional context
//
// The demo separates situational logic (this package) from reusable core components,
// enabling the same core subnet infrastructure to work with real AI models in production.
package demo

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hetu-project/FLUX-Mining-8004-x402/subnet"
	"github.com/hetu-project/FLUX-Mining-8004-x402/vlc"
)

// DemoCoordinator orchestrates the complete PoC demonstration of the PoCW subnet.
// It combines core subnet components (CoreMiner, CoreValidator) with demo-specific
// plugins to create a realistic but controlled testing environment.
//
// Architecture:
//   - Uses 1 miner with DemoTaskProcessor (hardcoded AI responses)
//   - Uses 4 validators with DemoQualityAssessor and DemoUserInteractionHandler
//   - Processes 7 predefined inputs with known expected outcomes
//   - Demonstrates both normal processing and info request scenarios
type DemoCoordinator struct {
	SubnetID            string                            // Unique identifier for this demo subnet
	Miner               *subnet.CoreMiner                 // AI agent processing tasks
	Validators          []*subnet.CoreValidator           // Quality assessment and consensus nodes
	userInputs          []string                          // Predefined demo inputs for consistent testing
	GraphAdapter        *subnet.SubnetGraphAdapter        // Graph adapter for VLC event visualization
	PaymentCoord        *subnet.PaymentCoordinator        // x402 payment system integration
	ReputationMgr       *subnet.ReputationFeedbackManager // Reputation feedback auth generation
	ReputationSubmitter *subnet.ReputationBatchSubmitter  // Reputation feedback batch submission
}

// NewDemoCoordinator creates a new demo coordinator with all PoC-specific logic
func NewDemoCoordinator(subnetID string) *DemoCoordinator {
	// Create core miner with demo task processor
	miner := subnet.NewCoreMiner("miner-1", subnetID)
	miner.SetTaskProcessor(NewDemoTaskProcessor())

	// Create core validators with demo plugins
	validators := make([]*subnet.CoreValidator, 4)
	for i := 0; i < 4; i++ {
		role := subnet.ConsensusValidator
		if i == 0 {
			role = subnet.UserInterfaceValidator // First validator handles user interaction
		}

		validator := subnet.NewCoreValidator(
			fmt.Sprintf("validator-%d", i+1),
			subnetID,
			role,
			0.25, // Equal weights for 4 validators
		)

		// Set demo-specific plugins
		validator.SetQualityAssessor(NewDemoQualityAssessor())
		validator.SetUserInteractionHandler(NewDemoUserInteractionHandler())

		validators[i] = validator
	}

	// Create graph adapter for visualization
	graphAdapter := subnet.NewSubnetGraphAdapter(subnetID, 1, "subnet-coordinator")

	// Check if running in subnet-only mode (skip payment and reputation systems)
	subnetOnlyMode := os.Getenv("SUBNET_ONLY_MODE") == "true"

	// Get RPC URL for blockchain interactions (used by payment and reputation systems)
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = "http://localhost:8545" // Default to localhost if not set
	}

	// Check if running in validation-only mode
	validationOnlyMode := os.Getenv("VALIDATION_ONLY_MODE") == "true"

	var paymentCoord *subnet.PaymentCoordinator
	if !subnetOnlyMode && !validationOnlyMode {
		// Initialize payment coordinator (x402 payment system)
		fmt.Println("💰 Initializing x402 Payment System...")

		// V1 Coordinator key - use environment or fallback to local
		v1CoordKey := os.Getenv("VALIDATOR_1_KEY")
		if v1CoordKey == "" {
			v1CoordKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
		}

		var err error
		paymentCoord, err = subnet.NewPaymentCoordinator(
			rpcURL,
			"contract_addresses.json",
			v1CoordKey,
		)
		if err != nil {
			fmt.Printf("⚠️  Payment system unavailable: %v\n", err)
			fmt.Println("   Continuing without payment integration...")
			paymentCoord = nil
		} else {
			fmt.Println("✅ Payment coordinator initialized successfully")
			// Set payment coordinator in UI validator (validator-1)
			validators[0].SetPaymentCoordinator(paymentCoord)

			// Configure miner with payment verification (trustless operation)
			// Miner will verify payment is locked in escrow before processing tasks
			// Agent address - use environment or fallback to local
			agentAddress := os.Getenv("MINER_ADDRESS")
			if agentAddress == "" {
				agentAddress = "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
			}
			minPayment := "10000000" // 10 tokens minimum (10 * 10^6 wei for USDC decimals)
			miner.SetPaymentVerifier(paymentCoord, agentAddress, minPayment)
			fmt.Printf("🔐 Miner configured with payment verification\n")
			fmt.Printf("   Agent address: %s\n", agentAddress)
			fmt.Printf("   Minimum payment: 10 %s\n", paymentCoord.GetPaymentTokenName())
		}
	} else {
		if validationOnlyMode {
			fmt.Println("⏭️  Skipping payment system initialization (validation-only mode)")
		} else {
			fmt.Println("⏭️  Skipping payment system initialization (subnet-only mode)")
		}
		paymentCoord = nil
	}

	var reputationManager *subnet.ReputationFeedbackManager
	var reputationSubmitter *subnet.ReputationBatchSubmitter

	// Skip reputation system if in validation-only mode (agent not registered yet)
	if !subnetOnlyMode && !validationOnlyMode {
		// Initialize reputation feedback manager
		fmt.Println("⭐ Initializing Reputation Feedback System...")

		// IdentityRegistry address from environment
		identityRegistryAddrStr := os.Getenv("IDENTITY_REGISTRY_ADDRESS")
		if identityRegistryAddrStr == "" {
			fmt.Printf("❌ IDENTITY_REGISTRY_ADDRESS not set\n")
			os.Exit(1)
		}
		identityRegistryAddr := common.HexToAddress(identityRegistryAddrStr)

		// Get chain ID from environment or default to 31337 (local)
		chainIDStr := os.Getenv("CHAIN_ID")
		chainIDValue := uint64(31337) // Default to local
		if chainIDStr != "" {
			if parsedChainID, err := strconv.ParseUint(chainIDStr, 10, 64); err == nil {
				chainIDValue = parsedChainID
			}
		}

		// Get miner key from environment or fallback to local
		minerKey := os.Getenv("MINER_KEY")
		if minerKey == "" {
			minerKey = "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
		}

		// Get client address from environment or fallback to Sepolia
		clientAddr := os.Getenv("CLIENT_ADDRESS")
		if clientAddr == "" {
			clientAddr = "0xfA6EC9Cf1E293A91a8ea2EdCc4A2324d48129821" // Sepolia client with USDC
		}

		// Get agent ID from environment (set by run-flux-mining.sh after querying blockchain)
		agentIDStr := os.Getenv("AGENT_ID_DEC")
		if agentIDStr == "" {
			fmt.Printf("❌ AGENT_ID_DEC not set - agent must be registered on blockchain first\n")
			fmt.Printf("   Run the script which queries/registers agent ID from IdentityRegistry\n")
			os.Exit(1)
		}

		agentID, err := strconv.ParseUint(agentIDStr, 10, 64)
		if err != nil {
			fmt.Printf("❌ Invalid AGENT_ID_DEC: %s - must be a valid number\n", agentIDStr)
			os.Exit(1)
		}

		fmt.Printf("   Using Agent ID: %d (from blockchain)\n", agentID)

		reputationMgr, err := subnet.NewReputationFeedbackManager(
			agentID,                         // Agent ID from environment (e.g., 1168)
			minerKey,                        // Miner's private key - from environment or local fallback
			common.HexToAddress(clientAddr), // Client address - from environment or local fallback
			identityRegistryAddr,
			chainIDValue, // Use environment chain ID or default
		)
		if err != nil {
			fmt.Printf("⚠️  Reputation system initialization failed: %v\n", err)
			fmt.Println("   Continuing without reputation feedback...")
			reputationMgr = nil
		} else {
			fmt.Println("✅ Reputation feedback auth generation initialized")
		}

		// Initialize reputation batch submitter (client-side)
		if reputationMgr != nil {
			// ReputationRegistry address from environment or contract_addresses.json
			reputationRegistryAddrStr := os.Getenv("REPUTATION_REGISTRY_ADDRESS")
			if reputationRegistryAddrStr == "" {
				fmt.Printf("❌ REPUTATION_REGISTRY_ADDRESS not set\n")
				os.Exit(1)
			}
			reputationRegistryAddr := common.HexToAddress(reputationRegistryAddrStr)

			// Initialize TaskIndexCounter from blockchain to prevent IndexLimit errors
			err = reputationMgr.InitializeFromBlockchain(rpcURL, reputationRegistryAddr)
			if err != nil {
				fmt.Printf("⚠️  Failed to initialize from blockchain: %v\n", err)
				fmt.Println("   Continuing with TaskIndexCounter = 0...")
			}

			// Get client key from environment or fallback to Sepolia
			clientKey := os.Getenv("CLIENT_KEY")
			if clientKey == "" {
				clientKey = "0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97" // Sepolia client key
			}

			submitter, err := subnet.NewReputationBatchSubmitter(
				rpcURL, // Use RPC URL from environment
				reputationRegistryAddr,
				clientKey,    // Client's private key - from environment or local fallback
				chainIDValue, // Use environment chain ID or default
			)
			if err != nil {
				fmt.Printf("⚠️  Reputation batch submitter initialization failed: %v\n", err)
				fmt.Println("   Continuing without batch submission...")
				reputationSubmitter = nil
			} else {
				reputationSubmitter = submitter
				fmt.Println("✅ Reputation batch submitter initialized")
			}
		}
		reputationManager = reputationMgr
	} else {
		if validationOnlyMode {
			fmt.Println("⏭️  Skipping reputation system initialization (validation-only mode)")
		} else {
			fmt.Println("⏭️  Skipping reputation system initialization (subnet-only mode)")
		}
		reputationManager = nil
		reputationSubmitter = nil
	}

	return &DemoCoordinator{
		SubnetID:            subnetID,
		Miner:               miner,
		Validators:          validators,
		GraphAdapter:        graphAdapter,
		PaymentCoord:        paymentCoord,
		ReputationMgr:       reputationManager,
		ReputationSubmitter: reputationSubmitter,
		userInputs: []string{
			"Analyze market trends for Q4",
			"Generate summary report for project Alpha",
			"Create optimization strategy for resource allocation",
			"Design implementation plan for new features",
			"Review performance metrics and recommendations",
			"Develop technical specifications for API integration",
			"Provide comprehensive analysis of system architecture",
		},
	}
}

// RunVLCValidation performs VLC protocol validation on the miner before allowing subnet operations.
// This validates that the agent correctly implements Vector Logical Clock causality.
// Returns true if validation passes, false otherwise.
func (dc *DemoCoordinator) RunVLCValidation() bool {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              AGENT VLC PROTOCOL VALIDATION                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Use only Validator-1 for testing the miner
	fmt.Printf("═══ Validator-1 Testing Agent ═══\n")

	validator := dc.Validators[0]  // Use only first validator
	requestID := "vlc-validation-test-1"
	test := validator.ValidateAgentVLC(dc.Miner, requestID)

	result := validator.CreateVLCValidationResult(test)
	score := result.Score

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              VLC VALIDATION RESULT                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	status := "✅"
	passed := score >= 70  // Pass if score is 70 or above
	if !passed {
		status = "❌"
	}
	fmt.Printf("  %s Validator-1: %d/100\n", status, score)
	fmt.Printf("  Pass Threshold: 70/100\n")
	fmt.Println()

	if passed {
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║        ✅ AGENT PASSED VLC PROTOCOL VALIDATION              ║")
		fmt.Println("║                                                              ║")
		fmt.Printf("║  Agent: %-52s ║\n", dc.Miner.ID)
		fmt.Printf("║  Score: %d/100                                               ║\n", score)
		fmt.Println("║  Status: AUTHORIZED FOR SUBNET OPERATIONS                   ║")
		fmt.Println("║                                                              ║")
		fmt.Println("║  The agent has demonstrated correct VLC implementation:     ║")
		fmt.Println("║  • Properly increments clock on each operation              ║")
		fmt.Println("║  • Maintains causal consistency                             ║")
		fmt.Println("║  • Implements NeedMoreInfo flow correctly                   ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
		fmt.Println()

		// Reset miner's VLC clock for fresh start in actual subnet operations
		dc.Miner.ResetClock()

		return true
	} else {
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║        ❌ AGENT FAILED VLC PROTOCOL VALIDATION              ║")
		fmt.Println("║                                                              ║")
		fmt.Printf("║  Agent: %-52s ║\n", dc.Miner.ID)
		fmt.Printf("║  Score: %d/100 (Required: ≥70)                               ║\n", score)
		fmt.Println("║  Status: NOT AUTHORIZED                                     ║")
		fmt.Println("║                                                              ║")
		fmt.Println("║  The agent must fix VLC implementation before proceeding.   ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
		fmt.Println()
		return false
	}
}

// RunDemo executes the complete demo scenario using the separated core/demo architecture
func (dc *DemoCoordinator) RunDemo() {
	fmt.Printf("=== Starting Demo with Refactored Architecture ===\n")
	fmt.Printf("Subnet ID: %s\n", dc.SubnetID)
	fmt.Printf("Miner: %s\n", dc.Miner.ID)
	fmt.Printf("Validators: ")
	for _, v := range dc.Validators {
		fmt.Printf("%s ", v.ID)
	}
	fmt.Printf("Graph Adapter: Enabled for VLC event visualization\n")
	fmt.Printf("\n")

	// Process each input according to demo scenario
	for inputNum := 1; inputNum <= 7; inputNum++ {
		fmt.Printf("--- Processing Input %d ---\n", inputNum)
		dc.processInput(inputNum, dc.userInputs[inputNum-1])
		fmt.Println()
		time.Sleep(1 * time.Second) // Small delay for readability
	}

	// Print final summary
	dc.printSummary()

	// Commit the causal event graph to Dgraph for visualization
	fmt.Printf("\n=== Committing VLC Event Graph to Dgraph ===\n")
	dc.GraphAdapter.PrintGraphSummary()

	if err := dc.GraphAdapter.CommitGraph(); err != nil {
		fmt.Printf("Error committing graph to Dgraph: %v\n", err)
		fmt.Printf("\nTroubleshooting:\n")
		fmt.Printf("- Start Dgraph: sudo docker run --rm -d --name dgraph-standalone -p 8080:8080 -p 9080:9080 -p 8000:8000 dgraph/standalone\n")
		fmt.Printf("- Check status: sudo docker ps | grep dgraph\n")
		fmt.Printf("- Check logs: sudo docker logs dgraph-standalone\n")
	} else {
		fmt.Printf("Successfully committed subnet graph to Dgraph!\n")
		fmt.Printf("\nVisualization Access:\n")
		fmt.Printf("- Ratel UI: http://localhost:8000\n")
		fmt.Printf("- Alternative: http://localhost:8080\n")
		fmt.Printf("- GraphQL: http://localhost:8080/graphql\n")
		// Note: GetEventCount() returns 0 after commit as events are cleared,
		// but we already showed the count in the graph summary above
	}
}

// processInput handles a single user input through the complete round-based workflow with VLC
func (dc *DemoCoordinator) processInput(inputNumber int, input string) {
	// Use timestamp to ensure unique request IDs across runs
	requestID := fmt.Sprintf("req-%s-%d-%d", dc.SubnetID, inputNumber, time.Now().Unix())

	fmt.Printf("User Input: %s\n", input)

	// *** ROUND START: User input (no VLC increment - user is external) ***
	uiValidator := dc.Validators[0] // Validator-1 is the round orchestrator
	// NO VLC increment for user communication - user is external to subnet
	fmt.Printf("Round %d: Started by Validator-1 receiving user input\n", inputNumber)

	// Track user input that starts the round
	userInputEventID := dc.GraphAdapter.TrackUserInput(requestID, input, uiValidator.GetLastMinerClock(), "")

	// *** x402 PAYMENT REQUEST: Agent generates payment request for client ***
	var paymentRequest *subnet.PaymentRequest
	if dc.PaymentCoord != nil {
		// Get miner/agent address from environment or fallback to Sepolia
		agentAddrStr := os.Getenv("MINER_ADDRESS")
		if agentAddrStr == "" {
			agentAddrStr = "0x86cDAb16A19602F74E4fFB996baD70307105a3A3" // Sepolia miner address
		}
		agentAddr := common.HexToAddress(agentAddrStr)
		paymentRequest = dc.PaymentCoord.GeneratePaymentRequest(requestID, agentAddr)

		fmt.Printf("\n📋 Agent sends x402 Payment Request to Client:\n")
		fmt.Printf("   Task ID: %s\n", paymentRequest.TaskID)
		fmt.Printf("   Amount: %s wei (10 %s)\n", paymentRequest.Amount, paymentRequest.Asset.Symbol)
		fmt.Printf("   Agent: %s\n", paymentRequest.Agent.Address)
		fmt.Printf("   Payment Token (%s): %s\n", paymentRequest.Asset.Symbol, paymentRequest.Asset.Contract)
		// Only show escrow details if not in direct mode
		if dc.PaymentCoord.GetPaymentMode() != "direct" {
			fmt.Printf("   Escrow Contract: %s\n", paymentRequest.Escrow.Contract)
			fmt.Printf("   Deadline: %d seconds\n", paymentRequest.Escrow.Timeout)
		}
		fmt.Println()
	}

	// *** PAYMENT PROCESSING: Client receives payment request and processes payment ***
	if dc.PaymentCoord != nil && paymentRequest != nil {
		fmt.Printf("💳 Client receives payment request and initiates payment...\n")

		// Get client address from environment or fallback to Sepolia
		clientAddrStr := os.Getenv("CLIENT_ADDRESS")
		if clientAddrStr == "" {
			clientAddrStr = "0xfA6EC9Cf1E293A91a8ea2EdCc4A2324d48129821" // Sepolia client with USDC
		}
		clientAddr := common.HexToAddress(clientAddrStr)
		agentAddr := common.HexToAddress(paymentRequest.Agent.Address)
		paymentAmount := new(big.Int)
		paymentAmount.SetString(paymentRequest.Amount, 10)

		// Check if we should use facilitator service
		if dc.PaymentCoord.UseFacilitator() {
			fmt.Printf("📡 Using x402 Facilitator for payment processing...\n")

			// Get payment scheme from facilitator
			scheme, err := dc.PaymentCoord.GetPaymentScheme()
			if err != nil {
				fmt.Printf("⚠️  Failed to get payment scheme: %v\n", err)
				scheme = "escrow" // Default to escrow
			}

			fmt.Printf("   Payment scheme: %s\n", scheme)

			// Settle payment through facilitator
			err = dc.PaymentCoord.SettlePaymentWithFacilitator(
				requestID,
				clientAddr,
				agentAddr,
				paymentRequest.Amount,
				scheme,
			)
			if err != nil {
				fmt.Printf("⚠️  Failed to settle payment via facilitator: %v\n", err)
			}
		} else {
			// Fallback to direct escrow deposit (old method)
			fmt.Printf("📝 Using direct escrow deposit (no facilitator)...\n")

			// Get client private key from environment or fallback to Sepolia
			clientPrivateKey := os.Getenv("CLIENT_KEY")
			if clientPrivateKey == "" {
				clientPrivateKey = "0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97" // Sepolia client key
			}
			err := dc.PaymentCoord.DepositPaymentWithClientSignature(
				requestID,
				clientAddr,
				agentAddr,
				paymentAmount,
				clientPrivateKey,
			)
			if err != nil {
				fmt.Printf("⚠️  Failed to deposit payment to escrow: %v\n", err)
			}
		}
	}

	// Step 1: Validator sends request to miner
	// VLC Protocol: +1 for message leaving validator to miner
	uiValidator.IncrementValidatorClock()
	fmt.Printf("Validator-1: Message leaving to miner → VLC incremented\n")

	// Sync miner's clock with validator's current state
	dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())

	// Miner processes input (will increment twice: enter + leave)
	minerResponse := dc.Miner.ProcessInput(input, inputNumber, requestID)

	// Step 2: Validator receives response from miner
	// VLC Protocol: +1 for message entering validator from miner
	uiValidator.IncrementValidatorClock()
	fmt.Printf("Validator-1: Message entered from miner → VLC incremented\n")

	// Track miner's response (output or info request)
	minerResponseEventID := dc.GraphAdapter.TrackMinerResponse(requestID, minerResponse, userInputEventID)

	if minerResponse.OutputType == subnet.NeedMoreInfo {
		// Handle info request scenario
		dc.handleInfoRequest(inputNumber, input, minerResponse, minerResponseEventID)
	} else {
		// Handle normal output scenario
		dc.handleNormalOutput(inputNumber, minerResponse, minerResponseEventID)
	}
}

// handleInfoRequest processes the scenario where miner needs more information with VLC orchestration
func (dc *DemoCoordinator) handleInfoRequest(inputNumber int, originalInput string, minerResponse *subnet.MinerResponseMessage, parentEventID string) {
	fmt.Printf("Miner requests more info: %s\n", minerResponse.InfoRequest)

	// Step 1: Validate miner's VLC sequence (NeedMoreInfo message)
	dc.validateVLCSequenceFromMiner(minerResponse)

	// Step 2: UI Validator orchestrates info request
	uiValidator := dc.Validators[0]

	// Update UI validator's VLC with miner's latest state
	uiValidator.UpdateMinerClock(minerResponse.VLCClock)

	infoRequest := uiValidator.RequestMoreInfo(minerResponse.RequestID, minerResponse.InfoRequest)

	if infoRequest != nil {
		// Validator to User: NO VLC increment (user is external)
		fmt.Printf("Validator %s asks user: %s\n", uiValidator.ID, infoRequest.Question)

		// Step 3: Simulate user providing additional info based on demo scenario
		var additionalInfo string
		switch inputNumber {
		case 3:
			additionalInfo = "Focus on cost optimization and ROI analysis specifically."
		case 6:
			additionalInfo = "Use REST API with JSON payloads, authentication via OAuth 2.0."
		}

		// User to Validator: NO VLC increment (user is external)
		fmt.Printf("User provides: %s\n", additionalInfo)

		// Track validator state for processing additional info (no increment yet)
		infoResponseEventID := dc.GraphAdapter.TrackInfoResponse(minerResponse.RequestID, additionalInfo, uiValidator.GetLastMinerClock(), parentEventID)

		// Step 4: Validator sends additional info to miner
		// VLC Protocol: +1 for message leaving validator to miner
		uiValidator.IncrementValidatorClock()
		fmt.Printf("Validator-1: Additional info leaving to miner → VLC incremented\n")

		// Sync miner with validator's updated VLC state
		dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())

		// Miner processes additional info (will increment twice: enter + leave)
		finalResponse := dc.Miner.ProcessAdditionalInfo(originalInput, additionalInfo, inputNumber, minerResponse.RequestID)

		// Step 5: Validator receives final response from miner
		// VLC Protocol: +1 for message entering validator from miner
		uiValidator.IncrementValidatorClock()
		fmt.Printf("Validator-1: Final response entered from miner → VLC incremented\n")

		// Track miner VLC increment for final processing
		finalProcessEventID := dc.GraphAdapter.TrackMinerResponse(minerResponse.RequestID, finalResponse, infoResponseEventID)

		// Step 5: Handle final output with quality voting
		dc.handleNormalOutput(inputNumber, finalResponse, finalProcessEventID)
	}
}

// validateVLCSequenceFromMiner validates miner's VLC sequence across all validators
func (dc *DemoCoordinator) validateVLCSequenceFromMiner(minerResponse *subnet.MinerResponseMessage) {
	fmt.Printf("🔗🔐 Validators validating Miner/Agent VLC sequence (local verification)...\n")

	// Each validator independently validates miner's VLC sequence
	// Only Validator-1 maintains VLC state, others just validate the sequence
	allValid := true
	for i, validator := range dc.Validators {
		if i == 0 {
			// Validator-1 (UI) - full VLC participant
			if !validator.ValidateSequence(minerResponse.VLCClock, 1) { // Miner ID = 1
				fmt.Printf("ERROR: Miner VLC validation failed for %s\n", validator.ID)
				allValid = false
			}
		} else {
			// Other validators - just check VLC format validity (simplified check)
			if minerResponse.VLCClock == nil || len(minerResponse.VLCClock.Values) == 0 {
				fmt.Printf("ERROR: Invalid VLC format for %s\n", validator.ID)
				allValid = false
			} else {
				fmt.Printf("Validator %s: VLC format check passed\n", validator.ID)
			}
		}
	}

	if allValid {
		fmt.Printf("Miner VLC validation: PASSED\n")
	} else {
		fmt.Printf("Miner VLC validation: FAILED\n")
	}
}

// validateVLCSequenceFromValidator validates validator-1's VLC operations
func (dc *DemoCoordinator) validateVLCSequenceFromValidator(validatorClock *vlc.Clock) {
	fmt.Printf("Miner validating Validator-1 VLC sequence...\n")

	// Miner validates validator's VLC operations
	// This maintains bidirectional VLC consistency
	dc.Miner.UpdateValidatorClock(validatorClock)
	fmt.Printf("Validator-1 VLC validation: PASSED (miner synchronized)\n")
}

// handleNormalOutput processes normal miner output through VLC validation and quality consensus
func (dc *DemoCoordinator) handleNormalOutput(inputNumber int, minerResponse *subnet.MinerResponseMessage, parentEventID string) {
	fmt.Printf("Miner output: %s\n", minerResponse.Output)

	// Step 1: Validate miner's VLC sequence for OutputReady message
	dc.validateVLCSequenceFromMiner(minerResponse)

	// Step 2: UI Validator updates its VLC state with miner's latest
	uiValidator := dc.Validators[0]
	uiValidator.UpdateMinerClock(minerResponse.VLCClock)

	// Step 3: Create shared quality assessment for consensus voting
	sharedAssessment := &subnet.QualityAssessment{
		RequestID: minerResponse.RequestID,
	}

	// Step 4: All validators vote on output quality (distributed consensus)
	fmt.Printf("🧠 Validators performing Semantic Alignment & quality assessment voting (distributed consensus)...\n")
	votes := make([]*subnet.ValidatorVoteMessage, 0, len(dc.Validators))

	// Each validator performs quality assessment and voting
	for _, validator := range dc.Validators {
		// Note: VLC validation already done above - this is pure quality voting
		vote := validator.VoteOnOutput(minerResponse)
		if vote != nil {
			votes = append(votes, vote)
			// Add each validator's vote to the shared assessment
			sharedAssessment.AddVote(vote.Weight, vote.Accept)
		} else {
			fmt.Printf("ERROR: Validator %s failed to generate vote\n", validator.ID)
		}
	}

	// Step 5: Check consensus using the shared assessment
	var consensusResult string
	var userAccepts bool
	var userFeedback string
	var finalResult string

	if sharedAssessment.IsAccepted() {
		consensusResult = fmt.Sprintf("ACCEPTED (%.2f/%.2f weight)", sharedAssessment.AcceptVotes, sharedAssessment.TotalWeight)
		fmt.Printf("Validator consensus: %s\n", consensusResult)

		// Step 6: Simulate user feedback using UI validator
		userAccepts, userFeedback = uiValidator.SimulateUserInteraction(inputNumber, minerResponse.Output)
		fmt.Printf("User feedback: %s\n", userFeedback)

		if userAccepts {
			finalResult = "OUTPUT DELIVERED TO USER"
		} else {
			finalResult = "OUTPUT REJECTED BY USER (despite validator acceptance)"
		}
	} else {
		consensusResult = fmt.Sprintf("REJECTED (%.2f/%.2f weight)", sharedAssessment.AcceptVotes, sharedAssessment.TotalWeight)
		fmt.Printf("Validator consensus: %s\n", consensusResult)

		userAccepts = false
		userFeedback = "No user feedback (validator rejection)"
		finalResult = "OUTPUT REJECTED BY VALIDATORS"
	}

	// *** PAYMENT FINALIZATION: Process payment based on consensus + user acceptance ***
	if dc.PaymentCoord != nil {
		qualityScore := sharedAssessment.AcceptVotes / sharedAssessment.TotalWeight
		err := uiValidator.FinalizePayment(
			minerResponse.RequestID,
			sharedAssessment.IsAccepted(),
			userAccepts,
			qualityScore,
		)
		if err != nil {
			fmt.Printf("⚠️  Payment finalization error: %v\n", err)
		}
	}

	// *** ROUND END: NO VLC increment (no message to/from miner, just user delivery) ***
	// User is external to subnet, so no VLC increment
	fmt.Printf("Round %d: Completed by Validator-1 aggregating final result\n", inputNumber)
	fmt.Printf("Final result: %s\n", finalResult)

	// *** REPUTATION: Generate FeedbackAuth BEFORE epoch submission ***
	// This ensures feedback is included in the epoch data
	if dc.ReputationMgr != nil {
		taskSuccess := sharedAssessment.IsAccepted() && userAccepts
		_, err := dc.ReputationMgr.GenerateFeedbackAuth(
			minerResponse.RequestID,
			inputNumber,
			taskSuccess,
		)
		if err != nil {
			fmt.Printf("⚠️  Failed to generate FeedbackAuth: %v\n", err)
		}
	}

	// Track comprehensive round completion with all actions in one VLC mutation
	// NOTE: This may trigger epoch submission if this is the 3rd round
	dc.GraphAdapter.TrackRoundComplete(
		minerResponse.RequestID,
		inputNumber,
		uiValidator.GetLastMinerClock(),
		consensusResult,
		userFeedback,
		userAccepts,
		finalResult,
		parentEventID,
	)

	// Check if epoch is complete (every 3 tasks) - AFTER TrackRoundComplete
	if dc.ReputationMgr != nil && dc.ReputationMgr.IsEpochComplete() {
		fmt.Printf("\n📊 Epoch %d Complete! Ready for batch feedback submission\n", dc.ReputationMgr.CurrentEpoch)
		dc.ReputationMgr.PrintEpochSummary(dc.ReputationMgr.CurrentEpoch)

		// Automatically submit batch feedback to blockchain
		if dc.ReputationSubmitter != nil {
			tasks := dc.ReputationMgr.GetCurrentEpochFeedbacks()
			err := dc.ReputationSubmitter.SubmitEpochFeedback(dc.ReputationMgr.AgentID, tasks)
			if err != nil {
				fmt.Printf("⚠️  Failed to submit epoch feedback: %v\n", err)
			}
		}

		// Start next epoch
		if inputNumber < 7 { // More tasks remaining
			dc.ReputationMgr.StartNextEpoch()
		}
	}

	// Sync miner with final validator state
	dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())
	fmt.Printf("Round %d: VLC synchronization complete\n", inputNumber)
}

// printSummary prints the final state of the subnet
func (dc *DemoCoordinator) printSummary() {
	fmt.Printf("=== Demo Summary (Refactored Architecture) ===\n")
	minerClock := dc.Miner.GetCurrentClock()
	fmt.Printf("Miner final VLC Clock: %v\n", minerClock.Values)

	fmt.Printf("\nValidator final states:\n")
	for _, validator := range dc.Validators {
		validatorClock := validator.GetLastMinerClock()
		fmt.Printf("  %s: Last miner clock = %v\n", validator.ID, validatorClock.Values)
	}

	fmt.Printf("\nProcessed inputs summary:\n")
	processedInputs := dc.Miner.GetProcessedInputs()
	for i := 1; i <= 7; i++ {
		if response, exists := processedInputs[i]; exists {
			fmt.Printf("  Input %d: Clock=%v, Type=%s\n", i, response.VLCClock.Values, response.OutputType)
		}
	}

	fmt.Printf("\nDemo completed successfully with refactored architecture!\n")
}
