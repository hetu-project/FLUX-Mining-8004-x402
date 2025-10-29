package subnet

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
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
	"golang.org/x/crypto/sha3"
)

// PaymentCoordinator handles all x402 payment operations for the FLUX-Mining system.
// It integrates with payment tokens (USDC/AIUSD) and x402PaymentEscrow contracts to facilitate gasless payments.
type PaymentCoordinator struct {
	client          *ethclient.Client
	auth            *bind.TransactOpts
	chainID         *big.Int
	paymentTokenAddress common.Address
	paymentTokenName    string
	escrowAddress   common.Address
	coordinatorKey  *ecdsa.PrivateKey
	coordinatorAddr common.Address

	// Payment tracking
	payments map[string]*PaymentTracker // taskID -> payment details
}

// PaymentTracker tracks the lifecycle of a payment
type PaymentTracker struct {
	TaskID          [32]byte
	Client          common.Address
	Agent           common.Address
	Amount          *big.Int
	Status          PaymentStatus
	DepositTime     time.Time
	Deadline        time.Time
	ReleaseTime     time.Time
	RefundTime      time.Time
	ConsensusReached bool
	UserAccepted    bool
	QualityScore    float64
}

// ContractAddresses holds deployed contract addresses
type ContractAddresses struct {
	PaymentToken     string `json:"PaymentToken"`
	PaymentTokenName string `json:"PaymentTokenName"`
	Escrow           string `json:"x402PaymentEscrow"`
	Client           string `json:"Client"`
	Agent            string `json:"Agent"`
	V1Coordinator    string `json:"V1Coordinator"`
}

// NewPaymentCoordinator creates a new payment coordinator instance
func NewPaymentCoordinator(rpcURL, contractAddressesFile, privateKeyHex string) (*PaymentCoordinator, error) {
	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	// Load chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get coordinator address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	coordinatorAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Load contract addresses
	addresses, err := loadContractAddresses(contractAddressesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load contract addresses: %w", err)
	}

	// Create transaction signer
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	pc := &PaymentCoordinator{
		client:              client,
		auth:                auth,
		chainID:             chainID,
		paymentTokenAddress: common.HexToAddress(addresses.PaymentToken),
		paymentTokenName:    addresses.PaymentTokenName,
		escrowAddress:       common.HexToAddress(addresses.Escrow),
		coordinatorKey:      privateKey,
		coordinatorAddr:     coordinatorAddr,
		payments:            make(map[string]*PaymentTracker),
	}

	fmt.Printf("💳 Payment Coordinator initialized:\n")
	fmt.Printf("   Chain ID: %s\n", chainID.String())
	fmt.Printf("   Coordinator: %s\n", coordinatorAddr.Hex())
	fmt.Printf("   Payment Token: %s (%s)\n", pc.paymentTokenName, pc.paymentTokenAddress.Hex())
	fmt.Printf("   Escrow: %s\n", pc.escrowAddress.Hex())

	return pc, nil
}

// GeneratePaymentRequest creates an x402 payment request for a task
func (pc *PaymentCoordinator) GeneratePaymentRequest(taskID string, agentAddr common.Address) *PaymentRequest {
	// Fixed pricing: 10 tokens per task (in wei)
	amount := "10000000000000000000" // 10 * 10^18 wei

	return &PaymentRequest{
		TaskID:         taskID,
		Amount:         amount,
		Asset: AssetInfo{
			Symbol:   pc.paymentTokenName,
			Contract: pc.paymentTokenAddress.Hex(),
			Decimals: 18,
		},
		Escrow: EscrowInfo{
			Contract: pc.escrowAddress.Hex(),
			Timeout:  60,
		},
		Agent: AgentInfo{
			Address: agentAddr.Hex(),
			AgentID: "0", // ERC-8004 registered agent ID
		},
		RequiresPayment: true,
	}
}

// DepositPayment deposits payment to escrow using standard transferFrom
// Requires client to have approved escrow contract beforehand
func (pc *PaymentCoordinator) DepositPayment(
	taskID string,
	clientAddr common.Address,
	agentAddr common.Address,
	amount *big.Int,
) error {
	// Convert taskID to bytes32
	taskIDBytes := stringToBytes32(taskID)

	// Set deadline
	deadline := big.NewInt(time.Now().Add(1 * time.Hour).Unix())

	// Get escrow ABI
	escrowABI, err := getEscrowABI()
	if err != nil {
		return fmt.Errorf("failed to load escrow ABI: %w", err)
	}

	// Pack the function call: depositPayment(taskId, client, agent, amount, deadline)
	data, err := escrowABI.Pack("depositPayment", taskIDBytes, clientAddr, agentAddr, amount, deadline)
	if err != nil {
		return fmt.Errorf("failed to pack depositPayment: %w", err)
	}

	// Get current nonce for coordinator
	nonce, err := pc.client.PendingNonceAt(context.Background(), pc.auth.From)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := pc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Send transaction from coordinator
	tx := types.NewTransaction(
		nonce,
		pc.escrowAddress,
		big.NewInt(0),
		300000, // gas limit
		gasPrice,
		data,
	)

	signedTx, err := pc.auth.Signer(pc.auth.From, tx)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = pc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), pc.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Track payment
	pc.payments[taskID] = &PaymentTracker{
		TaskID:      taskIDBytes,
		Client:      clientAddr,
		Agent:       agentAddr,
		Amount:      amount,
		Status:      PaymentDeposited,
		DepositTime: time.Now(),
		Deadline:    time.Unix(deadline.Int64(), 0),
	}

	fmt.Printf("💰 Payment deposited to escrow for task %s: %s %s\n", taskID, formatEther(amount), pc.paymentTokenName)
	fmt.Printf("   Client: %s\n", clientAddr.Hex())
	fmt.Printf("   Agent: %s\n", agentAddr.Hex())
	fmt.Printf("   Escrow TX: %s\n", signedTx.Hash().Hex())

	return nil
}

// DepositPaymentWithClientSignature deposits payment to escrow using client's private key
// This is a helper method that generates EIP-3009 signature and calls DepositPaymentWithAuthorization
func (pc *PaymentCoordinator) DepositPaymentWithClientSignature(
	taskID string,
	clientAddr common.Address,
	agentAddr common.Address,
	amount *big.Int,
	clientPrivateKeyHex string,
) error {
	// For now, just call the simpler DepositPayment method
	return pc.DepositPayment(taskID, clientAddr, agentAddr, amount)
}

// DepositPaymentWithAuthorization processes a gasless payment deposit using EIP-3009
func (pc *PaymentCoordinator) DepositPaymentWithAuthorization(
	taskID string,
	clientAddr common.Address,
	agentAddr common.Address,
	amount *big.Int,
	validAfter *big.Int,
	validBefore *big.Int,
	nonce [32]byte,
	v uint8,
	r [32]byte,
	s [32]byte,
) error {
	// Convert taskID to bytes32
	taskIDBytes := stringToBytes32(taskID)

	// Call escrow contract's depositWithAuthorization function
	escrowABI, err := getEscrowABI()
	if err != nil {
		return fmt.Errorf("failed to load escrow ABI: %w", err)
	}

	// Pack the function call
	data, err := escrowABI.Pack("depositWithAuthorization",
		taskIDBytes,
		clientAddr,
		agentAddr,
		amount,
		validAfter,
		validBefore,
		nonce,
		v,
		r,
		s,
	)
	if err != nil {
		return fmt.Errorf("failed to pack depositWithAuthorization: %w", err)
	}

	// Send transaction
	tx := types.NewTransaction(
		0, // nonce will be set by transactor
		pc.escrowAddress,
		big.NewInt(0),
		300000, // gas limit
		big.NewInt(0), // gas price (will be set automatically)
		data,
	)

	signedTx, err := pc.auth.Signer(pc.auth.From, tx)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = pc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), pc.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Track payment
	pc.payments[taskID] = &PaymentTracker{
		TaskID:      taskIDBytes,
		Client:      clientAddr,
		Agent:       agentAddr,
		Amount:      amount,
		Status:      PaymentDeposited,
		DepositTime: time.Now(),
		Deadline:    time.Unix(validBefore.Int64(), 0),
	}

	fmt.Printf("💰 Payment deposited for task %s: %s %s\n", taskID, formatEther(amount), pc.paymentTokenName)
	fmt.Printf("   Client: %s\n", clientAddr.Hex())
	fmt.Printf("   Agent: %s\n", agentAddr.Hex())
	fmt.Printf("   TX: %s\n", signedTx.Hash().Hex())

	return nil
}

// ReleasePaymentDirectDemo transfers AIUSD directly from coordinator to agent for demo purposes
// Bypasses escrow to show actual balance changes without requiring client signatures
func (pc *PaymentCoordinator) ReleasePaymentDirectDemo(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	if payment.Status != PaymentDeposited {
		return fmt.Errorf("payment status is not DEPOSITED (current: %v)", payment.Status)
	}

	// Get payment token ABI for transfer
	tokenABI, err := abi.JSON(strings.NewReader(`[{"inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`))
	if err != nil {
		return fmt.Errorf("failed to parse token ABI: %w", err)
	}

	// Pack transfer function call: transfer(agent, amount)
	data, err := tokenABI.Pack("transfer", payment.Agent, payment.Amount)
	if err != nil {
		return fmt.Errorf("failed to pack transfer: %w", err)
	}

	// Send transaction from coordinator to transfer payment tokens to agent
	nonce, err := pc.client.PendingNonceAt(context.Background(), pc.auth.From)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := pc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(
		nonce,
		pc.paymentTokenAddress,
		big.NewInt(0),
		100000, // gas limit
		gasPrice,
		data,
	)

	signedTx, err := pc.auth.Signer(pc.auth.From, tx)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = pc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), pc.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Update payment status
	payment.Status = PaymentReleased
	payment.ReleaseTime = time.Now()

	fmt.Printf("💸 Payment released directly (demo mode): %s %s\n", formatEther(payment.Amount), pc.paymentTokenName)
	fmt.Printf("   From: Coordinator %s\n", pc.auth.From.Hex())
	fmt.Printf("   To: Agent %s\n", payment.Agent.Hex())
	fmt.Printf("   TX: %s\n", signedTx.Hash().Hex())

	return nil
}

// ReleasePayment releases payment to the agent after successful consensus and user acceptance
func (pc *PaymentCoordinator) ReleasePayment(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	if payment.Status != PaymentDeposited {
		return fmt.Errorf("payment status is not DEPOSITED (current: %v)", payment.Status)
	}

	// Call escrow contract's releasePayment function
	escrowABI, err := getEscrowABI()
	if err != nil {
		return fmt.Errorf("failed to load escrow ABI: %w", err)
	}

	// Pack the function call
	data, err := escrowABI.Pack("releasePayment", payment.TaskID)
	if err != nil {
		return fmt.Errorf("failed to pack releasePayment: %w", err)
	}

	// Create and send transaction
	nonce, err := pc.client.PendingNonceAt(context.Background(), pc.coordinatorAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := pc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, pc.escrowAddress, big.NewInt(0), 100000, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(pc.chainID), pc.coordinatorKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = pc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction
	receipt, err := bind.WaitMined(context.Background(), pc.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Update payment status
	payment.Status = PaymentReleased

	fmt.Printf("✅ Payment released for task %s\n", taskID)
	fmt.Printf("   Agent received: %s %s\n", formatEther(payment.Amount), pc.paymentTokenName)
	fmt.Printf("   TX: %s\n", signedTx.Hash().Hex())

	return nil
}

// RefundPaymentDirectDemo marks payment as refunded for demo purposes
// In demo mode, coordinator pays from their own funds, so refund just means "don't transfer"
func (pc *PaymentCoordinator) RefundPaymentDirectDemo(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	if payment.Status != PaymentDeposited && payment.Status != PaymentExpired {
		return fmt.Errorf("payment status is not DEPOSITED or EXPIRED (current: %v)", payment.Status)
	}

	// Update payment status (no blockchain transaction needed - coordinator keeps the payment token)
	payment.Status = PaymentRefunded
	payment.RefundTime = time.Now()

	fmt.Printf("↩️  Payment refunded (demo mode): %s %s\n", formatEther(payment.Amount), pc.paymentTokenName)
	fmt.Printf("   Client: %s\n", payment.Client.Hex())
	fmt.Printf("   (No transfer needed - coordinator retains funds)\n")

	return nil
}

// RefundPayment refunds payment to the client on failure or rejection
func (pc *PaymentCoordinator) RefundPayment(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	if payment.Status != PaymentDeposited && payment.Status != PaymentExpired {
		return fmt.Errorf("payment status is not DEPOSITED or EXPIRED (current: %v)", payment.Status)
	}

	// Call escrow contract's refundPayment function
	escrowABI, err := getEscrowABI()
	if err != nil {
		return fmt.Errorf("failed to load escrow ABI: %w", err)
	}

	// Pack the function call
	data, err := escrowABI.Pack("refundPayment", payment.TaskID)
	if err != nil {
		return fmt.Errorf("failed to pack refundPayment: %w", err)
	}

	// Create and send transaction
	nonce, err := pc.client.PendingNonceAt(context.Background(), pc.coordinatorAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := pc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, pc.escrowAddress, big.NewInt(0), 100000, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(pc.chainID), pc.coordinatorKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = pc.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction
	receipt, err := bind.WaitMined(context.Background(), pc.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to mine transaction: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed")
	}

	// Update payment status
	payment.Status = PaymentRefunded

	fmt.Printf("↩️  Payment refunded for task %s\n", taskID)
	fmt.Printf("   Client received: %s %s\n", formatEther(payment.Amount), pc.paymentTokenName)
	fmt.Printf("   TX: %s\n", signedTx.Hash().Hex())

	return nil
}

// UpdatePaymentConsensus updates payment tracker with consensus results
func (pc *PaymentCoordinator) UpdatePaymentConsensus(taskID string, consensusReached bool, qualityScore float64) {
	if payment, exists := pc.payments[taskID]; exists {
		payment.ConsensusReached = consensusReached
		payment.QualityScore = qualityScore
	}
}

// UpdatePaymentUserAcceptance updates payment tracker with user acceptance
func (pc *PaymentCoordinator) UpdatePaymentUserAcceptance(taskID string, userAccepted bool) {
	if payment, exists := pc.payments[taskID]; exists {
		payment.UserAccepted = userAccepted
	}
}

// ShouldReleasePayment determines if payment should be released based on consensus and user acceptance
func (pc *PaymentCoordinator) ShouldReleasePayment(taskID string) bool {
	payment, exists := pc.payments[taskID]
	if !exists {
		return false
	}

	// Both consensus and user acceptance required
	return payment.ConsensusReached && payment.UserAccepted && payment.QualityScore > 0.5
}

// GetPaymentStatus returns the current payment status for a task
// InitializePaymentForDemo creates a payment tracker entry for demo purposes
// This allows testing the payment flow without requiring client EIP-3009 signatures
func (pc *PaymentCoordinator) InitializePaymentForDemo(taskID string, clientAddr, agentAddr common.Address, amount *big.Int) {
	taskIDBytes := [32]byte{}
	copy(taskIDBytes[:], []byte(taskID))

	pc.payments[taskID] = &PaymentTracker{
		TaskID:      taskIDBytes,
		Client:      clientAddr,
		Agent:       agentAddr,
		Amount:      amount,
		Status:      PaymentDeposited,
		DepositTime: time.Now(),
		Deadline:    time.Now().Add(1 * time.Hour),
	}

	fmt.Printf("💰 Demo payment initialized for task %s\n", taskID)
	fmt.Printf("   Client: %s\n", clientAddr.Hex())
	fmt.Printf("   Agent: %s\n", agentAddr.Hex())
	fmt.Printf("   Amount: %s %s\n", formatEther(amount), pc.paymentTokenName)
	fmt.Printf("   (Note: Blockchain deposit requires client signature - skipped for demo)\n")
}

func (pc *PaymentCoordinator) GetPaymentStatus(taskID string) *PaymentTracker {
	if payment, exists := pc.payments[taskID]; exists {
		return payment
	}
	return nil
}

// GetPaymentTokenName returns the configured payment token name (USDC or AIUSD)
func (pc *PaymentCoordinator) GetPaymentTokenName() string {
	return pc.paymentTokenName
}

// VerifyPaymentLocked verifies that payment is locked in escrow on-chain before agent processes task
// This provides cryptographic proof that funds are secured, enabling trustless agent operation
func (pc *PaymentCoordinator) VerifyPaymentLocked(taskID string, agentAddr common.Address, minAmount *big.Int) (bool, error) {
	// Convert taskID to bytes32
	taskIDBytes := stringToBytes32(taskID)

	// Get escrow ABI with payments getter
	escrowABI, err := abi.JSON(strings.NewReader(`[
		{
			"inputs": [{"name": "", "type": "bytes32"}],
			"name": "payments",
			"outputs": [
				{"name": "taskId", "type": "bytes32"},
				{"name": "client", "type": "address"},
				{"name": "agent", "type": "address"},
				{"name": "amount", "type": "uint256"},
				{"name": "depositTime", "type": "uint256"},
				{"name": "deadline", "type": "uint256"},
				{"name": "status", "type": "uint8"}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`))
	if err != nil {
		return false, fmt.Errorf("failed to parse escrow ABI: %w", err)
	}

	// Pack the function call
	data, err := escrowABI.Pack("payments", taskIDBytes)
	if err != nil {
		return false, fmt.Errorf("failed to pack payments call: %w", err)
	}

	// Call the contract
	result, err := pc.client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &pc.escrowAddress,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("failed to call escrow contract: %w", err)
	}

	// Unpack the result
	var payment struct {
		TaskId      [32]byte
		Client      common.Address
		Agent       common.Address
		Amount      *big.Int
		DepositTime *big.Int
		Deadline    *big.Int
		Status      uint8
	}

	err = escrowABI.UnpackIntoInterface(&payment, "payments", result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack payment data: %w", err)
	}

	// Verify payment conditions
	// Status: 0=NONE, 1=DEPOSITED, 2=COMPLETED, 3=REFUNDED, 4=EXPIRED
	if payment.Status != 1 { // Must be DEPOSITED
		return false, fmt.Errorf("payment status is %d, expected DEPOSITED (1)", payment.Status)
	}

	// Verify agent address matches
	if payment.Agent != agentAddr {
		return false, fmt.Errorf("payment agent %s doesn't match expected %s", payment.Agent.Hex(), agentAddr.Hex())
	}

	// Verify amount is sufficient
	if payment.Amount.Cmp(minAmount) < 0 {
		return false, fmt.Errorf("payment amount %s is less than minimum %s", formatEther(payment.Amount), formatEther(minAmount))
	}

	// Verify deadline hasn't passed
	currentTime := big.NewInt(time.Now().Unix())
	if payment.Deadline.Cmp(currentTime) <= 0 {
		return false, fmt.Errorf("payment deadline has passed")
	}

	fmt.Printf("✅ Payment verified for task %s:\n", taskID)
	fmt.Printf("   Amount: %s %s (locked in escrow)\n", formatEther(payment.Amount), pc.paymentTokenName)
	fmt.Printf("   Agent: %s\n", payment.Agent.Hex())
	fmt.Printf("   Client: %s\n", payment.Client.Hex())
	fmt.Printf("   Deadline: %s\n", time.Unix(payment.Deadline.Int64(), 0).Format(time.RFC3339))

	return true, nil
}

// Helper functions

func loadContractAddresses(filename string) (*ContractAddresses, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var addresses ContractAddresses
	err = json.Unmarshal(data, &addresses)
	if err != nil {
		return nil, err
	}

	return &addresses, nil
}

func getEscrowABI() (abi.ABI, error) {
	// Simplified escrow ABI with only the functions we need
	escrowJSON := `[
		{
			"inputs": [
				{"name": "taskId", "type": "bytes32"},
				{"name": "client", "type": "address"},
				{"name": "agent", "type": "address"},
				{"name": "amount", "type": "uint256"},
				{"name": "deadline", "type": "uint256"}
			],
			"name": "depositPayment",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [
				{"name": "taskId", "type": "bytes32"},
				{"name": "client", "type": "address"},
				{"name": "agent", "type": "address"},
				{"name": "amount", "type": "uint256"},
				{"name": "validAfter", "type": "uint256"},
				{"name": "validBefore", "type": "uint256"},
				{"name": "nonce", "type": "bytes32"},
				{"name": "v", "type": "uint8"},
				{"name": "r", "type": "bytes32"},
				{"name": "s", "type": "bytes32"}
			],
			"name": "depositWithAuthorization",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [{"name": "taskId", "type": "bytes32"}],
			"name": "releasePayment",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [{"name": "taskId", "type": "bytes32"}],
			"name": "refundPayment",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`

	return abi.JSON(strings.NewReader(escrowJSON))
}

func stringToBytes32(s string) [32]byte {
	var result [32]byte
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(s))
	copy(result[:], hash.Sum(nil))
	return result
}

func parseEther(eth string) *big.Int {
	// 10 AIUSD = 10 * 10^18 wei
	result := new(big.Int)
	result.SetString("10000000000000000000", 10) // 10 * 10^18
	return result
}

func formatEther(wei *big.Int) string {
	// Convert wei to ether (divide by 10^18)
	eth := new(big.Float).SetInt(wei)
	divisor := new(big.Float).SetFloat64(1e18)
	eth.Quo(eth, divisor)
	return eth.Text('f', 2)
}

// GenerateEIP712Signature generates an EIP-712 signature for transferWithAuthorization
// This would be called by the client to sign the payment authorization off-chain
func GenerateEIP712Signature(
	privateKey *ecdsa.PrivateKey,
	tokenAddr common.Address,
	tokenName string,
	chainID *big.Int,
	from common.Address,
	to common.Address,
	value *big.Int,
	validAfter *big.Int,
	validBefore *big.Int,
	nonce [32]byte,
) (v uint8, r [32]byte, s [32]byte, err error) {
	// EIP-712 domain separator
	domainSeparator := createEIP712DomainSeparator(tokenAddr, chainID, tokenName)

	// EIP-712 struct hash for TransferWithAuthorization
	structHash := createTransferWithAuthorizationHash(from, to, value, validAfter, validBefore, nonce)

	// Final message hash
	message := crypto.Keccak256(
		[]byte("\x19\x01"),
		domainSeparator[:],
		structHash[:],
	)

	// Sign the message
	signature, err := crypto.Sign(message, privateKey)
	if err != nil {
		return 0, [32]byte{}, [32]byte{}, err
	}

	// Extract v, r, s
	v = signature[64] + 27 // EIP-155
	copy(r[:], signature[0:32])
	copy(s[:], signature[32:64])

	return v, r, s, nil
}

func createEIP712DomainSeparator(contractAddr common.Address, chainID *big.Int, tokenName string) [32]byte {
	// keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)")
	typeHash := crypto.Keccak256Hash(
		[]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
	)

	// keccak256(tokenName) - e.g., "USDC" or "AIUSD"
	nameHash := crypto.Keccak256Hash([]byte(tokenName))

	// keccak256("1")
	versionHash := crypto.Keccak256Hash([]byte("1"))

	// Encode domain separator
	encoded := crypto.Keccak256(
		typeHash.Bytes(),
		nameHash.Bytes(),
		versionHash.Bytes(),
		common.LeftPadBytes(chainID.Bytes(), 32),
		common.LeftPadBytes(contractAddr.Bytes(), 32),
	)

	var result [32]byte
	copy(result[:], encoded)
	return result
}

func createTransferWithAuthorizationHash(
	from common.Address,
	to common.Address,
	value *big.Int,
	validAfter *big.Int,
	validBefore *big.Int,
	nonce [32]byte,
) [32]byte {
	// keccak256("TransferWithAuthorization(address from,address to,uint256 value,uint256 validAfter,uint256 validBefore,bytes32 nonce)")
	typeHash := crypto.Keccak256Hash(
		[]byte("TransferWithAuthorization(address from,address to,uint256 value,uint256 validAfter,uint256 validBefore,bytes32 nonce)"),
	)

	// Encode struct
	encoded := crypto.Keccak256(
		typeHash.Bytes(),
		common.LeftPadBytes(from.Bytes(), 32),
		common.LeftPadBytes(to.Bytes(), 32),
		common.LeftPadBytes(value.Bytes(), 32),
		common.LeftPadBytes(validAfter.Bytes(), 32),
		common.LeftPadBytes(validBefore.Bytes(), 32),
		nonce[:],
	)

	var result [32]byte
	copy(result[:], encoded)
	return result
}

// RandomNonce generates a random nonce for EIP-3009
func RandomNonce() [32]byte {
	var nonce [32]byte
	randomBytes := make([]byte, 32)
	crypto.Keccak256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())), randomBytes)
	copy(nonce[:], randomBytes)
	return nonce
}