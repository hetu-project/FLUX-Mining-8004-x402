// Package subnet implements reputation feedback system for ERC-8004 agents
//
// This file provides the FeedbackAuth generation and management system that allows
// users to provide reputation feedback for agents after completing tasks.
//
// Flow:
//   1. Agent completes task → Generates FeedbackAuth for client
//   2. Client collects FeedbackAuth for each task in epoch
//   3. At epoch end (every 3 tasks) → Client submits batch feedback
//   4. ReputationRegistry stores feedback on-chain
package subnet

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FeedbackAuthData represents the signed authorization for submitting feedback
// This matches the ReputationRegistry.sol FeedbackAuth struct
type FeedbackAuthData struct {
	AgentId          *big.Int       // Agent ID receiving feedback
	ClientAddress    common.Address // User authorized to give feedback
	IndexLimit       uint64         // Progressive index (1, 2, 3, ...)
	Expiry           *big.Int       // Unix timestamp when auth expires
	ChainId          *big.Int       // Chain ID (31337 for local testnet)
	IdentityRegistry common.Address // IdentityRegistry contract address
	SignerAddress    common.Address // Agent owner's address (who signs)
}

// TaskFeedbackRecord tracks a single task's feedback information
type TaskFeedbackRecord struct {
	TaskID        string    // Request ID
	TaskNumber    int       // Task number within epoch
	Success       bool      // Whether task was successful
	FeedbackAuth  []byte    // Signed authorization from agent
	Submitted     bool      // Whether feedback has been submitted
	Timestamp     time.Time // When task completed
}

// EpochFeedbackBatch tracks all feedbacks for a single epoch
type EpochFeedbackBatch struct {
	EpochNumber int                  // Which epoch (1, 2, 3, ...)
	Tasks       []TaskFeedbackRecord // Up to 3 tasks per epoch
	Submitted   bool                 // Whether batch has been submitted
}

// ReputationFeedbackManager manages feedback auth generation and submission
type ReputationFeedbackManager struct {
	AgentID          *big.Int       // Agent's identity ID
	AgentPrivateKey  *ecdsa.PrivateKey // Agent's signing key
	ClientAddress    common.Address // User receiving services
	IdentityRegistry common.Address // Contract address
	ChainID          *big.Int       // Network chain ID

	// Epoch tracking
	CurrentEpoch     int                   // Current epoch number (1-based)
	EpochBatches     []EpochFeedbackBatch  // All epoch batches
	TaskIndexCounter uint64                // Progressive feedback index counter
}

// NewReputationFeedbackManager creates a new feedback manager
func NewReputationFeedbackManager(
	agentID uint64,
	agentPrivateKeyHex string,
	clientAddress common.Address,
	identityRegistryAddr common.Address,
	chainID uint64,
) (*ReputationFeedbackManager, error) {
	// Parse agent's private key
	privateKey, err := crypto.HexToECDSA(agentPrivateKeyHex[2:]) // Remove 0x prefix
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return &ReputationFeedbackManager{
		AgentID:          big.NewInt(int64(agentID)),
		AgentPrivateKey:  privateKey,
		ClientAddress:    clientAddress,
		IdentityRegistry: identityRegistryAddr,
		ChainID:          big.NewInt(int64(chainID)),
		CurrentEpoch:     1,
		EpochBatches:     make([]EpochFeedbackBatch, 0),
		TaskIndexCounter: 0, // Will be initialized from blockchain
	}, nil
}

// InitializeFromBlockchain queries the blockchain to get the current lastIndex
// and initializes TaskIndexCounter appropriately. This prevents IndexLimit errors.
func (rfm *ReputationFeedbackManager) InitializeFromBlockchain(
	rpcURL string,
	reputationRegistryAddr common.Address,
) error {
	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}
	defer client.Close()

	// Query getLastIndex from ReputationRegistry
	lastIndex, err := queryLastIndex(client, reputationRegistryAddr, rfm.AgentID, rfm.ClientAddress)
	if err != nil {
		return fmt.Errorf("failed to query lastIndex: %w", err)
	}

	// Initialize TaskIndexCounter to current blockchain state
	rfm.TaskIndexCounter = lastIndex

	if lastIndex > 0 {
		fmt.Printf("📊 Initialized TaskIndexCounter from blockchain: %d\n", lastIndex)
		fmt.Printf("   Next feedback will use indexLimit: %d\n", lastIndex+1)
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
	// Define ABI for getLastIndex function
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

	// Encode function call
	data, err := parsedABI.Pack("getLastIndex", agentID, clientAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to pack function call: %w", err)
	}

	// Make the call
	msg := ethereum.CallMsg{
		To:   &reputationRegistry,
		Data: data,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	var lastIndex uint64
	err = parsedABI.UnpackIntoInterface(&lastIndex, "getLastIndex", result)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack result: %w", err)
	}

	return lastIndex, nil
}

// GenerateFeedbackAuth creates a signed authorization for user to submit feedback
// This is called by the agent after completing each task
func (rfm *ReputationFeedbackManager) GenerateFeedbackAuth(
	taskID string,
	taskNumber int,
	success bool,
) ([]byte, error) {
	// Increment task index
	rfm.TaskIndexCounter++

	// Create FeedbackAuth struct
	authData := FeedbackAuthData{
		AgentId:          rfm.AgentID,
		ClientAddress:    rfm.ClientAddress,
		IndexLimit:       rfm.TaskIndexCounter,
		Expiry:           big.NewInt(time.Now().Add(7 * 24 * time.Hour).Unix()), // 7 days
		ChainId:          rfm.ChainID,
		IdentityRegistry: rfm.IdentityRegistry,
		SignerAddress:    crypto.PubkeyToAddress(rfm.AgentPrivateKey.PublicKey),
	}

	// Encode the auth data (first 224 bytes)
	encoded, err := encodeFeedbackAuth(authData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode auth: %w", err)
	}

	// Hash the encoded data
	messageHash := crypto.Keccak256Hash(encoded)

	// Add Ethereum signed message prefix
	// The prefix format is: "\x19Ethereum Signed Message:\n32" + messageHash
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	ethSignedHash := crypto.Keccak256Hash(append(prefix, messageHash.Bytes()...))

	// Sign with agent's private key
	signature, err := crypto.Sign(ethSignedHash.Bytes(), rfm.AgentPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign auth: %w", err)
	}

	// Adjust v value for Ethereum compatibility (0/1 -> 27/28)
	// crypto.Sign returns v as 0 or 1, but Ethereum expects 27 or 28
	if len(signature) == 65 {
		signature[64] += 27
	}

	// Concatenate encoded data + signature (224 + 65 = 289 bytes)
	fullAuth := append(encoded, signature...)

	// Store in current epoch batch
	rfm.addTaskToCurrentEpoch(taskID, taskNumber, success, fullAuth)

	return fullAuth, nil
}

// encodeFeedbackAuth encodes the FeedbackAuth struct using Solidity ABI encoding
func encodeFeedbackAuth(auth FeedbackAuthData) ([]byte, error) {
	// Define the ABI types
	uint256Type, _ := abi.NewType("uint256", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	uint64Type, _ := abi.NewType("uint64", "", nil)

	arguments := abi.Arguments{
		{Type: uint256Type}, // agentId
		{Type: addressType}, // clientAddress
		{Type: uint64Type},  // indexLimit
		{Type: uint256Type}, // expiry
		{Type: uint256Type}, // chainId
		{Type: addressType}, // identityRegistry
		{Type: addressType}, // signerAddress
	}

	return arguments.Pack(
		auth.AgentId,
		auth.ClientAddress,
		auth.IndexLimit,
		auth.Expiry,
		auth.ChainId,
		auth.IdentityRegistry,
		auth.SignerAddress,
	)
}

// addTaskToCurrentEpoch adds a task feedback record to the current epoch batch
func (rfm *ReputationFeedbackManager) addTaskToCurrentEpoch(
	taskID string,
	taskNumber int,
	success bool,
	feedbackAuth []byte,
) {
	// Ensure we have a batch for the current epoch
	for len(rfm.EpochBatches) < rfm.CurrentEpoch {
		rfm.EpochBatches = append(rfm.EpochBatches, EpochFeedbackBatch{
			EpochNumber: len(rfm.EpochBatches) + 1,
			Tasks:       make([]TaskFeedbackRecord, 0, 3), // Max 3 tasks per epoch
			Submitted:   false,
		})
	}

	// Add task to current epoch
	currentBatch := &rfm.EpochBatches[rfm.CurrentEpoch-1]
	currentBatch.Tasks = append(currentBatch.Tasks, TaskFeedbackRecord{
		TaskID:       taskID,
		TaskNumber:   taskNumber,
		Success:      success,
		FeedbackAuth: feedbackAuth,
		Submitted:    false,
		Timestamp:    time.Now(),
	})

	fmt.Printf("📝 FeedbackAuth generated for Task %d (Index: %d, Auth: %d bytes)\n",
		taskNumber, rfm.TaskIndexCounter, len(feedbackAuth))
}

// IsEpochComplete checks if current epoch has 3 tasks (ready for feedback)
func (rfm *ReputationFeedbackManager) IsEpochComplete() bool {
	if rfm.CurrentEpoch > len(rfm.EpochBatches) {
		return false
	}
	currentBatch := rfm.EpochBatches[rfm.CurrentEpoch-1]
	return len(currentBatch.Tasks) >= 3
}

// StartNextEpoch advances to the next epoch
func (rfm *ReputationFeedbackManager) StartNextEpoch() {
	rfm.CurrentEpoch++
	fmt.Printf("\n🔄 Starting Epoch %d\n", rfm.CurrentEpoch)
}

// GetCurrentEpochFeedbacks returns the feedback auth for all tasks in current epoch
func (rfm *ReputationFeedbackManager) GetCurrentEpochFeedbacks() []TaskFeedbackRecord {
	if rfm.CurrentEpoch > len(rfm.EpochBatches) {
		return []TaskFeedbackRecord{}
	}
	return rfm.EpochBatches[rfm.CurrentEpoch-1].Tasks
}

// PrintEpochSummary displays summary of current epoch's tasks
func (rfm *ReputationFeedbackManager) PrintEpochSummary(epochNum int) {
	if epochNum > len(rfm.EpochBatches) {
		return
	}

	batch := rfm.EpochBatches[epochNum-1]
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║              EPOCH %d FEEDBACK SUMMARY                       ║\n", epochNum)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	for i, task := range batch.Tasks {
		status := "✅ Success"
		if !task.Success {
			status = "❌ Failed"
		}
		fmt.Printf("  Task %d: %s\n", i+1, status)
		fmt.Printf("    Task ID: %s\n", task.TaskID)
		fmt.Printf("    FeedbackAuth: %s...%s (%d bytes)\n",
			hex.EncodeToString(task.FeedbackAuth[:8]),
			hex.EncodeToString(task.FeedbackAuth[len(task.FeedbackAuth)-8:]),
			len(task.FeedbackAuth))
		fmt.Println()
	}

	fmt.Printf("  Total Tasks: %d\n", len(batch.Tasks))
	fmt.Printf("  Submitted: %v\n\n", batch.Submitted)
}

// CalculateFeedbackScore determines the score based on task success
// Simplified scoring: 85 for success, 40 for failure
func CalculateFeedbackScore(success bool) uint8 {
	if success {
		return 85 // Good performance score
	}
	return 40 // Failed task score
}

// GetFeedbackTag1 returns the primary tag based on task outcome
func GetFeedbackTag1(success bool) [32]byte {
	if success {
		return crypto.Keccak256Hash([]byte("TASK_SUCCESS"))
	}
	return crypto.Keccak256Hash([]byte("TASK_FAILED"))
}

// GetFeedbackTag2 returns the secondary tag (task type)
func GetFeedbackTag2() [32]byte {
	return crypto.Keccak256Hash([]byte("COMPUTE"))
}

// FormatFeedbackAuthForDisplay returns a human-readable representation
func FormatFeedbackAuthForDisplay(auth []byte) string {
	if len(auth) != 289 {
		return fmt.Sprintf("Invalid auth length: %d (expected 289)", len(auth))
	}
	return fmt.Sprintf("Auth[%s...%s]",
		hexutil.Encode(auth[:8]),
		hexutil.Encode(auth[len(auth)-8:]))
}

// ReputationBatchSubmitter handles batch submission of feedback to ReputationRegistry
type ReputationBatchSubmitter struct {
	client              *ethclient.Client
	auth                *bind.TransactOpts
	reputationRegistry  common.Address
	clientPrivateKey    *ecdsa.PrivateKey
	chainID             *big.Int
}

// NewReputationBatchSubmitter creates a new batch submitter
func NewReputationBatchSubmitter(
	rpcURL string,
	reputationRegistryAddr common.Address,
	clientPrivateKeyHex string,
	chainID uint64,
) (*ReputationBatchSubmitter, error) {
	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	// Parse client's private key
	privateKey, err := crypto.HexToECDSA(clientPrivateKeyHex[2:]) // Remove 0x prefix
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Create transaction auth
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

// SubmitEpochFeedback submits all feedbacks for an epoch in batch
func (rbs *ReputationBatchSubmitter) SubmitEpochFeedback(
	agentID *big.Int,
	tasks []TaskFeedbackRecord,
) error {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           SUBMITTING EPOCH FEEDBACK TO BLOCKCHAIN          ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	successCount := 0
	for i, task := range tasks {
		fmt.Printf("📝 Task %d (%s): ", i+1, task.TaskID)

		// Calculate score based on task outcome
		score := CalculateFeedbackScore(task.Success)
		tag1 := GetFeedbackTag1(task.Success)
		tag2 := GetFeedbackTag2()

		// Submit feedback to ReputationRegistry
		txHash, err := rbs.submitSingleFeedback(agentID, score, tag1, tag2, task.FeedbackAuth)
		if err != nil {
			fmt.Printf("❌ Failed - %v\n", err)
			return fmt.Errorf("failed to submit feedback for task %d: %w", i+1, err)
		}

		fmt.Printf("✅ Success (TX: %s)\n", txHash)
		successCount++

		// Small delay between submissions to avoid nonce issues
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║        ✅ EPOCH FEEDBACK BATCH SUBMITTED SUCCESSFULLY       ║\n")
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║  Agent ID: %-50s ║\n", agentID.String())
	fmt.Printf("║  Total Feedbacks: %d                                          ║\n", successCount)
	fmt.Printf("║  All feedback recorded on-chain in ReputationRegistry       ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	return nil
}

// submitSingleFeedback submits a single feedback transaction
func (rbs *ReputationBatchSubmitter) submitSingleFeedback(
	agentID *big.Int,
	score uint8,
	tag1, tag2 [32]byte,
	feedbackAuth []byte,
) (string, error) {
	// Define ReputationRegistry ABI for giveFeedback function
	reputationABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "uint8", "name": "score", "type": "uint8"},
			{"internalType": "bytes32", "name": "tag1", "type": "bytes32"},
			{"internalType": "bytes32", "name": "tag2", "type": "bytes32"},
			{"internalType": "string", "name": "feedbackUri", "type": "string"},
			{"internalType": "bytes32", "name": "feedbackHash", "type": "bytes32"},
			{"internalType": "bytes", "name": "feedbackAuth", "type": "bytes"}
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

	// Encode function call
	feedbackUri := "" // Empty URI for simple feedback
	feedbackHash := [32]byte{} // Empty hash

	data, err := parsedABI.Pack(
		"giveFeedback",
		agentID,
		score,
		tag1,
		tag2,
		feedbackUri,
		feedbackHash,
		feedbackAuth,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Get current nonce
	nonce, err := rbs.client.PendingNonceAt(context.Background(), rbs.auth.From)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := rbs.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		rbs.reputationRegistry,
		big.NewInt(0), // No ETH value
		300000,        // Gas limit
		gasPrice,
		data,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(rbs.chainID), rbs.clientPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = rbs.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	// Wait for transaction receipt
	receipt, err := bind.WaitMined(context.Background(), rbs.client, signedTx)
	if err != nil {
		return txHash, fmt.Errorf("transaction failed: %w", err)
	}

	if receipt.Status != 1 {
		return txHash, fmt.Errorf("transaction reverted - TX: https://sepolia.etherscan.io/tx/%s", txHash)
	}

	return txHash, nil
}

// GetAgentReputationSummary reads and displays the agent's reputation from the blockchain
func (rbs *ReputationBatchSubmitter) GetAgentReputationSummary(agentID *big.Int) error {
	// Define ReputationRegistry ABI for getSummary function
	reputationABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "address[]", "name": "clientAddresses", "type": "address[]"},
			{"internalType": "bytes32", "name": "tag1", "type": "bytes32"},
			{"internalType": "bytes32", "name": "tag2", "type": "bytes32"}
		],
		"name": "getSummary",
		"outputs": [
			{"internalType": "uint64", "name": "count", "type": "uint64"},
			{"internalType": "uint8", "name": "averageScore", "type": "uint8"}
		],
		"stateMutability": "view",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(reputationABI))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Encode function call with empty client addresses array and zero tags
	emptyAddresses := []common.Address{}
	zeroTag := [32]byte{}
	data, err := parsedABI.Pack("getSummary", agentID, emptyAddresses, zeroTag, zeroTag)
	if err != nil {
		return fmt.Errorf("failed to pack function call: %w", err)
	}

	// Make the call
	msg := ethereum.CallMsg{
		To:   &rbs.reputationRegistry,
		Data: data,
	}

	result, err := rbs.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	var count uint64
	var averageScore uint8
	results, err := parsedABI.Unpack("getSummary", result)
	if err != nil {
		return fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(results) >= 2 {
		count = results[0].(uint64)
		averageScore = results[1].(uint8)
	}

	// Display the summary
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║        🌟 FINAL AGENT REPUTATION SUMMARY                    ║\n")
	fmt.Printf("║           (Read from ReputationRegistry)                    ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("📊 Agent ID %s Reputation on Blockchain:\n", agentID.String())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if count > 0 {
		fmt.Printf("  📝 Total Feedbacks Received: %d\n", count)
		fmt.Printf("  ⭐ Average Score: %d/100", averageScore)

		// Add performance indicator
		if averageScore >= 80 {
			fmt.Printf(" (Excellent Performance 🏆)\n")
		} else if averageScore >= 60 {
			fmt.Printf(" (Good Performance ✅)\n")
		} else if averageScore >= 40 {
			fmt.Printf(" (Needs Improvement ⚠️)\n")
		} else {
			fmt.Printf(" (Poor Performance ❌)\n")
		}

		// Display visual score bar
		barLength := 50
		filledLength := int(averageScore) * barLength / 100
		bar := strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)

		fmt.Printf("  📊 Score Visual: [%s] %d%%\n", bar, averageScore)
	} else {
		fmt.Printf("  ❌ No reputation feedback recorded yet\n")
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ Reputation data successfully retrieved from blockchain!\n\n")

	return nil
}
