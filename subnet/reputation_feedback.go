// Package subnet implements reputation feedback system for ERC-8004 agents
//
// ERC-8004 v1.0 Flow (simplified - no FeedbackAuth):
//   1. Agent completes tasks → Results tracked in memory
//   2. After all rounds complete → Client submits feedback directly
//   3. ReputationRegistry stores feedback on-chain
package subnet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TaskResult tracks a single task's outcome (simplified - no FeedbackAuth)
type TaskResult struct {
	TaskID       string    // Request ID (used as feedbackURI)
	TaskNumber   int       // Task number (1-7)
	Success      bool      // Whether task was successful
	QualityScore float64   // Quality score from validator consensus (0.0-1.0)
	Timestamp    time.Time // When task completed
}

// ReputationFeedbackManager manages task results and feedback submission
type ReputationFeedbackManager struct {
	AgentID          *big.Int       // Agent's identity ID
	ClientAddress    common.Address // Client receiving services
	IdentityRegistry common.Address // Contract address
	ChainID          *big.Int       // Network chain ID
	Endpoint         string         // Service endpoint (e.g., "hetu.subnet1.org/flux-mining")

	// Task tracking
	TaskResults []TaskResult // All task results
	CurrentEpoch int         // Current epoch number (for display)
}

// NewReputationFeedbackManager creates a new feedback manager
func NewReputationFeedbackManager(
	agentID uint64,
	agentPrivateKeyHex string, // Kept for backward compatibility, not used for signing anymore
	clientAddress common.Address,
	identityRegistryAddr common.Address,
	chainID uint64,
) (*ReputationFeedbackManager, error) {
	// Endpoint is the service URI where the agent can be reached
	endpoint := os.Getenv("AGENT_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://hetu.subnet1.org/flux-mining"
	}

	return &ReputationFeedbackManager{
		AgentID:          big.NewInt(int64(agentID)),
		ClientAddress:    clientAddress,
		IdentityRegistry: identityRegistryAddr,
		ChainID:          big.NewInt(int64(chainID)),
		Endpoint:         endpoint,
		TaskResults:      make([]TaskResult, 0, 7),
		CurrentEpoch:     1,
	}, nil
}

// InitializeFromBlockchain queries the blockchain for current state
// Simplified - just validates connection, no FeedbackAuth index needed
func (rfm *ReputationFeedbackManager) InitializeFromBlockchain(
	rpcURL string,
	reputationRegistryAddr common.Address,
) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	// Query current feedback count for this agent/client pair
	lastIndex, err := queryLastIndex(client, reputationRegistryAddr, rfm.AgentID, rfm.ClientAddress)
	if err != nil {
		// Not fatal - might be first time
		fmt.Printf("⚠️  Could not query lastIndex (might be first run): %v\n", err)
		return nil
	}

	if lastIndex > 0 {
		fmt.Printf("📊 Existing feedback count for agent: %d\n", lastIndex)
	}

	return nil
}

// queryLastIndex queries the ReputationRegistry contract for the current lastIndex
func queryLastIndex(
	client *ethclient.Client,
	reputationRegistry common.Address,
	agentID *big.Int,
	clientAddress common.Address,
) (uint64, error) {
	getLastIndexABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "address", "name": "clientAddress", "type": "address"}
		],
		"name": "getLastIndex",
		"outputs": [
			{"internalType": "uint64", "name": "", "type": "uint64"}
		],
		"stateMutability": "view",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(getLastIndexABI))
	if err != nil {
		return 0, fmt.Errorf("failed to parse ABI: %w", err)
	}

	data, err := parsedABI.Pack("getLastIndex", agentID, clientAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to pack function call: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &reputationRegistry,
		Data: data,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %w", err)
	}

	var lastIndex uint64
	err = parsedABI.UnpackIntoInterface(&lastIndex, "getLastIndex", result)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack result: %w", err)
	}

	return lastIndex, nil
}

// RecordTaskResult records a task result (replaces GenerateFeedbackAuth)
func (rfm *ReputationFeedbackManager) RecordTaskResult(
	taskID string,
	taskNumber int,
	success bool,
	qualityScore float64,
) {
	result := TaskResult{
		TaskID:       taskID,
		TaskNumber:   taskNumber,
		Success:      success,
		QualityScore: qualityScore,
		Timestamp:    time.Now(),
	}

	rfm.TaskResults = append(rfm.TaskResults, result)

	status := "✅"
	if !success {
		status = "❌"
	}
	fmt.Printf("📝 Task %d recorded: %s (Quality: %.2f)\n", taskNumber, status, qualityScore)
}

// GenerateFeedbackAuth - DEPRECATED, kept for backward compatibility
// Now just calls RecordTaskResult
func (rfm *ReputationFeedbackManager) GenerateFeedbackAuth(
	taskID string,
	taskNumber int,
	success bool,
) ([]byte, error) {
	// Quality score defaults based on success
	qualityScore := 1.0
	if !success {
		qualityScore = 0.0
	}

	rfm.RecordTaskResult(taskID, taskNumber, success, qualityScore)

	// Return empty bytes - no FeedbackAuth in v1.0
	return []byte{}, nil
}

// IsEpochComplete checks if current epoch has 3 tasks
func (rfm *ReputationFeedbackManager) IsEpochComplete() bool {
	tasksInCurrentEpoch := len(rfm.TaskResults) - ((rfm.CurrentEpoch - 1) * 3)
	return tasksInCurrentEpoch >= 3
}

// StartNextEpoch advances to the next epoch
func (rfm *ReputationFeedbackManager) StartNextEpoch() {
	rfm.CurrentEpoch++
	fmt.Printf("\n🔄 Starting Epoch %d\n", rfm.CurrentEpoch)
}

// GetCurrentEpochFeedbacks returns task results for current epoch
// Returns TaskFeedbackRecord for backward compatibility
func (rfm *ReputationFeedbackManager) GetCurrentEpochFeedbacks() []TaskFeedbackRecord {
	startIdx := (rfm.CurrentEpoch - 1) * 3
	endIdx := startIdx + 3
	if endIdx > len(rfm.TaskResults) {
		endIdx = len(rfm.TaskResults)
	}

	records := make([]TaskFeedbackRecord, 0)
	for i := startIdx; i < endIdx; i++ {
		result := rfm.TaskResults[i]
		records = append(records, TaskFeedbackRecord{
			TaskID:     result.TaskID,
			TaskNumber: result.TaskNumber,
			Success:    result.Success,
			Timestamp:  result.Timestamp,
		})
	}
	return records
}

// TaskFeedbackRecord for backward compatibility
type TaskFeedbackRecord struct {
	TaskID     string
	TaskNumber int
	Success    bool
	Timestamp  time.Time
}

// GetAllTaskResults returns all recorded task results
func (rfm *ReputationFeedbackManager) GetAllTaskResults() []TaskResult {
	return rfm.TaskResults
}

// PrintEpochSummary displays summary of current epoch's tasks
func (rfm *ReputationFeedbackManager) PrintEpochSummary(epochNum int) {
	startIdx := (epochNum - 1) * 3
	endIdx := startIdx + 3
	if endIdx > len(rfm.TaskResults) {
		endIdx = len(rfm.TaskResults)
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║              EPOCH %d TASK SUMMARY                           ║\n", epochNum)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	for i := startIdx; i < endIdx; i++ {
		task := rfm.TaskResults[i]
		status := "✅ Success"
		if !task.Success {
			status = "❌ Failed"
		}
		fmt.Printf("  Task %d: %s (Quality: %.2f)\n", task.TaskNumber, status, task.QualityScore)
		fmt.Printf("    Task ID: %s\n", task.TaskID)
		fmt.Println()
	}

	fmt.Printf("  Total Tasks in Epoch: %d\n\n", endIdx-startIdx)
}

// CalculateFeedbackScore determines the score based on task success and quality
func CalculateFeedbackScore(success bool, qualityScore float64) uint8 {
	if success {
		// Scale quality (0.0-1.0) to score (70-100)
		return uint8(70 + (qualityScore * 30))
	}
	// Failed tasks get lower scores
	return uint8(qualityScore * 40)
}

// ReputationBatchSubmitter handles batch submission of feedback to ReputationRegistry
type ReputationBatchSubmitter struct {
	client             *ethclient.Client
	auth               *bind.TransactOpts
	reputationRegistry common.Address
	clientPrivateKey   *ecdsa.PrivateKey
	chainID            *big.Int
}

// NewReputationBatchSubmitter creates a new batch submitter
func NewReputationBatchSubmitter(
	rpcURL string,
	reputationRegistryAddr common.Address,
	clientPrivateKeyHex string,
	chainID uint64,
) (*ReputationBatchSubmitter, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	keyHex := clientPrivateKeyHex
	if strings.HasPrefix(keyHex, "0x") {
		keyHex = keyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(int64(chainID)))
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	return &ReputationBatchSubmitter{
		client:             client,
		auth:               auth,
		reputationRegistry: reputationRegistryAddr,
		clientPrivateKey:   privateKey,
		chainID:            big.NewInt(int64(chainID)),
	}, nil
}

// SubmitAllFeedback submits feedback for all tasks at once (new v1.0 flow)
func (rbs *ReputationBatchSubmitter) SubmitAllFeedback(
	agentID *big.Int,
	tasks []TaskResult,
	endpoint string,
) error {
	successCount := 0
	for i, task := range tasks {
		fmt.Printf("📝 Task %d (%s): ", task.TaskNumber, task.TaskID[:20]+"...")

		score := CalculateFeedbackScore(task.Success, task.QualityScore)
		tag1 := "flux-mining"
		tag2 := "compute"
		if !task.Success {
			tag2 = "failed"
		}

		// Use intent causal graph SVG as feedbackURI
		feedbackURI := "https://coffee-defiant-raccoon-829.mypinata.cloud/ipfs/bafkreid4ud4ihbwgsxtnc7hkivef6whnqbrzzpncul3r3wxsvi4onjyl64"

		txHash, err := rbs.submitSingleFeedback(
			agentID,
			score,
			tag1,
			tag2,
			endpoint,
			feedbackURI,
		)
		if err != nil {
			fmt.Printf("❌ Failed - %v\n", err)
			continue
		}

		fmt.Printf("✅ Score: %d (TX: %s...)\n", score, txHash[:16])
		successCount++

		// Small delay between submissions
		if i < len(tasks)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

// SubmitEpochFeedback submits feedbacks for an epoch (backward compatibility)
func (rbs *ReputationBatchSubmitter) SubmitEpochFeedback(
	agentID *big.Int,
	tasks []TaskFeedbackRecord,
) error {
	// Convert to TaskResult and call new method
	results := make([]TaskResult, len(tasks))
	for i, t := range tasks {
		qualityScore := 1.0
		if !t.Success {
			qualityScore = 0.0
		}
		results[i] = TaskResult{
			TaskID:       t.TaskID,
			TaskNumber:   t.TaskNumber,
			Success:      t.Success,
			QualityScore: qualityScore,
			Timestamp:    t.Timestamp,
		}
	}
	// Endpoint is the service URI where the agent can be reached
	endpoint := os.Getenv("AGENT_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://hetu.subnet1.org/flux-mining"
	}
	return rbs.SubmitAllFeedback(agentID, results, endpoint)
}

// submitSingleFeedback submits a single feedback transaction (v2.0 - int256 value with decimals)
func (rbs *ReputationBatchSubmitter) submitSingleFeedback(
	agentID *big.Int,
	score uint8,
	tag1, tag2 string,
	endpoint string,
	feedbackURI string,
) (string, error) {
	// ABI for v2.0 (int256 value, uint8 valueDecimals)
	reputationABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "int256", "name": "value", "type": "int256"},
			{"internalType": "uint8", "name": "valueDecimals", "type": "uint8"},
			{"internalType": "string", "name": "tag1", "type": "string"},
			{"internalType": "string", "name": "tag2", "type": "string"},
			{"internalType": "string", "name": "endpoint", "type": "string"},
			{"internalType": "string", "name": "feedbackURI", "type": "string"},
			{"internalType": "bytes32", "name": "feedbackHash", "type": "bytes32"}
		],
		"name": "giveFeedback",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(reputationABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Generate feedbackHash from taskID
	feedbackHash := crypto.Keccak256Hash([]byte(feedbackURI))

	// Convert score (0-100) to int256 value with 0 decimals
	// Score 90 -> value 90, decimals 0
	value := big.NewInt(int64(score))
	valueDecimals := uint8(0)

	data, err := parsedABI.Pack(
		"giveFeedback",
		agentID,
		value,
		valueDecimals,
		tag1,
		tag2,
		endpoint,
		feedbackURI,
		feedbackHash,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	nonce, err := rbs.client.PendingNonceAt(context.Background(), rbs.auth.From)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := rbs.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(
		nonce,
		rbs.reputationRegistry,
		big.NewInt(0),
		300000,
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(rbs.chainID), rbs.clientPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = rbs.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	receipt, err := bind.WaitMined(context.Background(), rbs.client, signedTx)
	if err != nil {
		return txHash, fmt.Errorf("transaction failed: %w", err)
	}

	if receipt.Status != 1 {
		return txHash, fmt.Errorf("transaction reverted")
	}

	return txHash, nil
}

// GetAgentReputationSummary reads agent's reputation from blockchain
func (rbs *ReputationBatchSubmitter) GetAgentReputationSummary(agentID *big.Int) error {
	// Updated ABI for v2.0 (int256 summaryValue, uint8 summaryValueDecimals)
	reputationABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "address[]", "name": "clientAddresses", "type": "address[]"},
			{"internalType": "string", "name": "tag1", "type": "string"},
			{"internalType": "string", "name": "tag2", "type": "string"}
		],
		"name": "getSummary",
		"outputs": [
			{"internalType": "uint64", "name": "count", "type": "uint64"},
			{"internalType": "int256", "name": "summaryValue", "type": "int256"},
			{"internalType": "uint8", "name": "summaryValueDecimals", "type": "uint8"}
		],
		"stateMutability": "view",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(reputationABI))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %w", err)
	}

	emptyAddresses := []common.Address{}
	data, err := parsedABI.Pack("getSummary", agentID, emptyAddresses, "", "")
	if err != nil {
		return fmt.Errorf("failed to pack function call: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &rbs.reputationRegistry,
		Data: data,
	}

	result, err := rbs.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call contract: %w", err)
	}

	results, err := parsedABI.Unpack("getSummary", result)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %w", err)
	}

	var count uint64
	var summaryValue *big.Int
	var summaryValueDecimals uint8
	if len(results) >= 3 {
		count = results[0].(uint64)
		summaryValue = results[1].(*big.Int)
		summaryValueDecimals = results[2].(uint8)
	}

	// Calculate average score from summaryValue
	// If decimals=0, summaryValue is the sum of all scores
	// Average = summaryValue / count
	var averageScore int64 = 0
	if count > 0 && summaryValue != nil {
		avgBig := new(big.Int).Div(summaryValue, big.NewInt(int64(count)))
		averageScore = avgBig.Int64()
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║        🌟 AGENT REPUTATION SUMMARY                          ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("📊 Agent ID %s:\n", agentID.String())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if count > 0 {
		fmt.Printf("  📝 Total Feedbacks: %d\n", count)
		fmt.Printf("  📈 Total Score: %s (decimals: %d)\n", summaryValue.String(), summaryValueDecimals)
		fmt.Printf("  ⭐ Average Score: %d/100", averageScore)

		if averageScore >= 80 {
			fmt.Printf(" (Excellent 🏆)\n")
		} else if averageScore >= 60 {
			fmt.Printf(" (Good ✅)\n")
		} else if averageScore >= 40 {
			fmt.Printf(" (Needs Improvement ⚠️)\n")
		} else {
			fmt.Printf(" (Poor ❌)\n")
		}

		barLength := 50
		filledLength := int(averageScore) * barLength / 100
		if filledLength < 0 {
			filledLength = 0
		}
		if filledLength > barLength {
			filledLength = barLength
		}
		bar := strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)
		fmt.Printf("  📊 [%s] %d%%\n", bar, averageScore)
	} else {
		fmt.Printf("  ❌ No feedback recorded yet\n")
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	return nil
}
