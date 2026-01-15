package subnet

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
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
// Supports both per-task payments and x402 V2 session-based payments (per-epoch).
type PaymentCoordinator struct {
	client          *ethclient.Client
	auth            *bind.TransactOpts
	chainID         *big.Int
	paymentTokenAddress common.Address
	paymentTokenName    string
	escrowAddress   common.Address
	coordinatorKey  *ecdsa.PrivateKey
	coordinatorAddr common.Address
	clientKey       *ecdsa.PrivateKey  // Client's private key for signing transactions
	clientAddr      common.Address      // Client's address
	facilitatorURL  string // x402 facilitator service URL
	paymentMode     string // direct or session (x402 V2)

	// Payment tracking (per-task)
	payments map[string]*PaymentTracker // taskID -> payment details

	// Session payment tracking (x402 V2)
	useSessionPayments bool                       // Enable session-based payments
	tasksPerSession    int                        // Tasks per session/epoch (default 3)
	sessions           map[int]*SessionPayment    // epochNumber -> session payment

	// Direct payment tracking (for tasks outside sessions)
	directPaymentTotal int64 // Total direct payments in base units (e.g., USDC with 6 decimals)
	directPaymentCount int   // Number of direct payments made
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

// SessionPaymentStatus represents the status of a session payment
type SessionPaymentStatus int

const (
	SessionPending   SessionPaymentStatus = iota // Session created, payment not yet deposited
	SessionActive                                // Payment deposited, tasks running
	SessionCompleted                             // All tasks done, payment released
	SessionRefunded                              // Session failed, payment refunded
)

func (s SessionPaymentStatus) String() string {
	switch s {
	case SessionPending:
		return "PENDING"
	case SessionActive:
		return "ACTIVE"
	case SessionCompleted:
		return "COMPLETED"
	case SessionRefunded:
		return "REFUNDED"
	default:
		return "UNKNOWN"
	}
}

// SessionPayment tracks payment for an entire epoch/session (x402 V2)
type SessionPayment struct {
	SessionID        string               // "session-epoch-1"
	EpochNumber      int                  // Which epoch this session represents
	ClientAddr       common.Address       // Client paying for the session
	AgentAddr        common.Address       // Agent receiving payment
	TotalAmount      *big.Int             // Total payment for session (e.g., 30 USDC)
	AmountPerTask    *big.Int             // Amount per task (e.g., 10 USDC)
	TasksInSession   []string             // Task IDs in this session
	TasksPerEpoch    int                  // Number of tasks per epoch
	Status           SessionPaymentStatus // Current session status
	SessionStart     time.Time            // When session started
	SessionExpiry    time.Time            // When session expires
	AggregateQuality float64              // Weighted average quality score
	TasksCompleted   int                  // Number of tasks completed
	TasksApproved    int                  // Number of tasks approved
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

	// Get facilitator URL from environment or use default
	facilitatorURL := os.Getenv("FACILITATOR_URL")
	if facilitatorURL == "" {
		facilitatorURL = "http://localhost:3002"
	}

	// Get payment mode from environment or use default (x402 V2: direct or session)
	paymentMode := os.Getenv("PAYMENT_MODE")
	if paymentMode == "" {
		paymentMode = "direct" // Default to direct mode
	}

	// Get client private key from environment for direct payment signing
	clientKeyHex := os.Getenv("CLIENT_KEY")
	if clientKeyHex == "" {
		// Fallback to PRIVATE_KEY_CLIENT for backwards compatibility
		clientKeyHex = os.Getenv("PRIVATE_KEY_CLIENT")
		if clientKeyHex == "" {
			return nil, fmt.Errorf("CLIENT_KEY environment variable not set")
		}
	}

	clientKey, err := crypto.HexToECDSA(strings.TrimPrefix(clientKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse client private key: %w", err)
	}

	// Derive client address from private key
	clientPublicKey := clientKey.Public()
	clientPublicKeyECDSA, ok := clientPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting client public key to ECDSA")
	}
	clientAddr := crypto.PubkeyToAddress(*clientPublicKeyECDSA)

	// Session payments enabled when paymentMode == "session"
	tasksPerSession := 3 // Default: 3 tasks per epoch/session

	pc := &PaymentCoordinator{
		client:              client,
		auth:                auth,
		chainID:             chainID,
		paymentTokenAddress: common.HexToAddress(addresses.PaymentToken),
		paymentTokenName:    addresses.PaymentTokenName,
		escrowAddress:       common.HexToAddress(addresses.Escrow),
		coordinatorKey:      privateKey,
		coordinatorAddr:     coordinatorAddr,
		clientKey:           clientKey,
		clientAddr:          clientAddr,
		facilitatorURL:      facilitatorURL,
		paymentMode:         paymentMode,
		payments:            make(map[string]*PaymentTracker),
		useSessionPayments:  paymentMode == "session",
		tasksPerSession:     tasksPerSession,
		sessions:            make(map[int]*SessionPayment),
	}

	modeStr := paymentMode
	if paymentMode == "session" {
		modeStr = fmt.Sprintf("session (%d tasks/session)", tasksPerSession)
	}
	fmt.Printf("💳 Payments: %s via %s (%s)\n", pc.paymentTokenName, modeStr, facilitatorURL)

	return pc, nil
}

// GeneratePaymentRequest creates an x402 payment request for a task
func (pc *PaymentCoordinator) GeneratePaymentRequest(taskID string, agentAddr common.Address) *PaymentRequest {
	// Fixed pricing: 10 tokens per task
	// Amount should be human-readable (e.g., "10" for 10 USDC)
	// The facilitator will use parseUnits to convert to wei based on decimals
	amount := "10" // 10 USDC

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
			AgentID: os.Getenv("AGENT_ID_DEC"), // ERC-8004 registered agent ID from environment
		},
		RequiresPayment: true,
	}
}

// UseFacilitator checks if we should use the facilitator service
func (pc *PaymentCoordinator) UseFacilitator() bool {
	return pc.facilitatorURL != ""
}

// VerifyPaymentWithFacilitator verifies payment through the x402 facilitator
func (pc *PaymentCoordinator) VerifyPaymentWithFacilitator(payment map[string]interface{}, scheme string) (bool, error) {
	if !pc.UseFacilitator() {
		return false, fmt.Errorf("facilitator URL not configured")
	}

	reqBody := map[string]interface{}{
		"payment": payment,
		"scheme":  scheme,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	resp, err := http.Post(pc.facilitatorURL+"/verify", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to contact facilitator: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Valid        bool     `json:"valid"`
		Facilitator  string   `json:"facilitator"`
		Scheme       string   `json:"scheme"`
		Capabilities []string `json:"capabilities"`
		Error        string   `json:"error,omitempty"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, fmt.Errorf("verification failed: %s", result.Error)
	}

	return result.Valid, nil
}

// createSignedPaymentTransaction creates and signs an ERC20 transfer transaction for direct payments
func (pc *PaymentCoordinator) createSignedPaymentTransaction(recipient common.Address, amount string) (string, error) {
	// Parse amount to wei (USDC has 6 decimals)
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse amount: %w", err)
	}
	amountWei := new(big.Int).SetUint64(uint64(amountFloat * 1e6)) // 6 decimals for USDC

	// Get current nonce for client
	nonce, err := pc.client.PendingNonceAt(context.Background(), pc.clientAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := pc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create ERC20 transfer data
	// Function signature: transfer(address,uint256)
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]

	// Pad recipient address to 32 bytes
	paddedRecipient := common.LeftPadBytes(recipient.Bytes(), 32)

	// Pad amount to 32 bytes
	paddedAmount := common.LeftPadBytes(amountWei.Bytes(), 32)

	// Concatenate method ID + padded recipient + padded amount
	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedRecipient...)
	data = append(data, paddedAmount...)

	// Create transaction
	gasLimit := uint64(100000) // Standard ERC20 transfer gas limit
	tx := types.NewTransaction(nonce, pc.paymentTokenAddress, big.NewInt(0), gasLimit, gasPrice, data)

	// Sign transaction with client's private key
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(pc.chainID), pc.clientKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Encode signed transaction to hex string
	txBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	signedTxHex := "0x" + common.Bytes2Hex(txBytes)
	return signedTxHex, nil
}

// SettlePaymentWithFacilitator settles payment through the x402 facilitator
func (pc *PaymentCoordinator) SettlePaymentWithFacilitator(
	taskID string,
	clientAddr common.Address,
	agentAddr common.Address,
	amount string,
	scheme string,
) error {
	if !pc.UseFacilitator() {
		return fmt.Errorf("facilitator URL not configured")
	}

	payment := map[string]interface{}{
		"amount":    amount,
		"recipient": agentAddr.Hex(),
		"client":    clientAddr.Hex(),
		"agent":     agentAddr.Hex(),
		"taskId":    taskID,
	}

	// For direct/exact payments, client must create and sign the transaction
	// Both "direct" and "exact" schemes require pre-signed transactions from the client
	if scheme == "direct" || scheme == "exact" {
		signedTx, err := pc.createSignedPaymentTransaction(agentAddr, amount)
		if err != nil {
			return fmt.Errorf("failed to create signed transaction: %w", err)
		}
		payment["signedTx"] = signedTx
	}

	reqBody := map[string]interface{}{
		"payment": payment,
		"scheme":  scheme,
		"taskId":  taskID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(pc.facilitatorURL+"/settle", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to contact facilitator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("settlement failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		TransactionHash string `json:"transactionHash"`
		BlockNumber     int64  `json:"blockNumber"`
		Status          string `json:"status"`
		Scheme          string `json:"scheme"`
		Amount          string `json:"amount"`
		TaskID          string `json:"taskId,omitempty"`
		Error           string `json:"error,omitempty"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Error != "" {
		return fmt.Errorf("settlement error: %s", result.Error)
	}

	fmt.Printf("✅ Payment: %s (%s)\n", result.Scheme, result.Status)

	// Track the payment in the local map for later release/refund
	taskIDBytes := [32]byte{}
	copy(taskIDBytes[:], []byte(taskID))

	// Parse amount - amount is human-readable (e.g., "10" for 10 USDC)
	// Convert to wei (USDC uses 6 decimals)
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}
	amountWei := uint64(amountFloat * 1000000) // USDC has 6 decimals
	amountBig := new(big.Int).SetUint64(amountWei)

	// Determine payment status based on scheme and result status
	// For escrow: payment is deposited and needs to be released
	// For direct: payment is pending validation (not yet broadcast to blockchain)
	paymentStatus := PaymentDeposited
	if result.Scheme == "direct" || result.Scheme == "exact" {
		if result.Status == "pending_validation" {
			// Direct payment is stored but not yet broadcast
			paymentStatus = PaymentPending
		} else if result.Status == "settled" {
			// Legacy behavior - immediate settlement
			paymentStatus = PaymentCompleted
		}
	}

	pc.payments[taskID] = &PaymentTracker{
		TaskID:           taskIDBytes,
		Client:           clientAddr,
		Agent:            agentAddr,
		Amount:           amountBig,
		DepositTime:      time.Now(),
		Status:           paymentStatus,
		ConsensusReached: false,
		QualityScore:     0,
		UserAccepted:     false,
	}

	return nil
}

// GetPaymentScheme returns the payment scheme (always "direct" for x402 V2)
func (pc *PaymentCoordinator) GetPaymentScheme() (string, error) {
	return "direct", nil
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

	fmt.Printf("💰 Escrow deposit: %s %s (tx: %s)\n", formatEther(amount), pc.paymentTokenName, signedTx.Hash().Hex()[:10]+"...")

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

	fmt.Printf("💰 Payment deposited: %s %s (tx: %s)\n", formatEther(amount), pc.paymentTokenName, signedTx.Hash().Hex()[:10]+"...")

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

	fmt.Printf("💸 Released: %s %s to agent (tx: %s)\n", formatEther(payment.Amount), pc.paymentTokenName, signedTx.Hash().Hex()[:10]+"...")

	return nil
}

// ReleasePayment releases payment to the agent after successful consensus and user acceptance
func (pc *PaymentCoordinator) ReleasePayment(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	// Handle different payment statuses
	if payment.Status == PaymentCompleted {
		fmt.Printf("ℹ️  Payment %s already completed\n", taskID)
		return nil
	}

	// For pending direct payments, finalize the transaction (broadcast to blockchain)
	if payment.Status == PaymentPending {
		if pc.UseFacilitator() {
			finalizeReq := map[string]interface{}{
				"taskId":   taskID,
				"approved": true,
				"validatorApprovals": []string{"validator-1", "validator-2"},
			}

			reqBody, _ := json.Marshal(finalizeReq)
			resp, err := http.Post(
				pc.facilitatorURL+"/direct/finalize",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			if err != nil {
				return fmt.Errorf("failed to finalize direct payment: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("finalization failed: %s", string(body))
			}

			body, _ := io.ReadAll(resp.Body)
			var result struct {
				TransactionHash string `json:"transactionHash"`
				BlockNumber     int64  `json:"blockNumber"`
				Status          string `json:"status"`
			}
			json.Unmarshal(body, &result)
			txShort := "pending"
			if result.TransactionHash != "" && len(result.TransactionHash) > 10 {
				txShort = result.TransactionHash[:10] + "..."
			}
			fmt.Printf("✅ Direct payment finalized (tx: %s)\n", txShort)
			payment.Status = PaymentCompleted
			return nil
		}
	}

	if payment.Status != PaymentDeposited {
		return fmt.Errorf("payment status is not valid for release (current: %v)", payment.Status)
	}

	// Check if we should use facilitator service for release
	if pc.UseFacilitator() {
		validatorApprovals := []string{"validator1-approved"}

		reqBody := map[string]interface{}{
			"taskId":             taskID,
			"validatorApprovals": validatorApprovals,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		resp, err := http.Post(pc.facilitatorURL+"/escrow/release", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to contact facilitator: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("release failed with status %d: %s", resp.StatusCode, body)
		}

		var result struct {
			TransactionHash string `json:"transactionHash"`
			BlockNumber     int64  `json:"blockNumber"`
			Status          string `json:"status"`
			TaskID          string `json:"taskId"`
			Error           string `json:"error,omitempty"`
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}

		if result.Error != "" {
			return fmt.Errorf("release error: %s", result.Error)
		}

		txShort := "pending"
		if result.TransactionHash != "" && len(result.TransactionHash) > 10 {
			txShort = result.TransactionHash[:10] + "..."
		}
		fmt.Printf("✅ Released via facilitator (block %d, tx: %s)\n", result.BlockNumber, txShort)

		payment.Status = PaymentReleased
		payment.ReleaseTime = time.Now()

		return nil
	}

	// Fallback to direct escrow release (old method)
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

	fmt.Printf("✅ Released: %s %s (tx: %s)\n", formatEther(payment.Amount), pc.paymentTokenName, signedTx.Hash().Hex()[:10]+"...")

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

	fmt.Printf("↩️  Refunded (demo): %s %s\n", formatEther(payment.Amount), pc.paymentTokenName)

	return nil
}

// RefundPayment refunds payment to the client on failure or rejection
func (pc *PaymentCoordinator) RefundPayment(taskID string) error {
	payment, exists := pc.payments[taskID]
	if !exists {
		return fmt.Errorf("payment not found for task %s", taskID)
	}

	// Handle different payment statuses
	if payment.Status == PaymentCompleted {
		fmt.Printf("⚠️  Cannot refund %s - already completed\n", taskID)
		return fmt.Errorf("cannot refund completed direct payment")
	}

	// For pending direct payments, discard the transaction (don't broadcast)
	if payment.Status == PaymentPending {
		if pc.UseFacilitator() {
			finalizeReq := map[string]interface{}{
				"taskId":   taskID,
				"approved": false,
				"validatorApprovals": []string{},
			}

			reqBody, _ := json.Marshal(finalizeReq)
			resp, err := http.Post(
				pc.facilitatorURL+"/direct/finalize",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			if err != nil {
				return fmt.Errorf("failed to discard direct payment: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("discard failed: %s", string(body))
			}

			fmt.Printf("✅ Payment discarded\n")
			payment.Status = PaymentRefunded
			return nil
		}
	}

	if payment.Status != PaymentDeposited && payment.Status != PaymentExpired {
		return fmt.Errorf("payment status is not valid for refund (current: %v)", payment.Status)
	}

	// If using facilitator, route through it
	if pc.UseFacilitator() {
		refundRequest := map[string]interface{}{
			"taskId": taskID,
			"reason": "User rejected or low quality",
		}

		reqBody, err := json.Marshal(refundRequest)
		if err != nil {
			return fmt.Errorf("failed to marshal refund request: %w", err)
		}

		resp, err := http.Post(
			pc.facilitatorURL+"/escrow/refund",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		if err != nil {
			return fmt.Errorf("failed to call facilitator refund: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read refund response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("facilitator refund failed (status %d): %s", resp.StatusCode, string(body))
		}

		var refundResp struct {
			TransactionHash string `json:"transactionHash"`
			BlockNumber     int    `json:"blockNumber"`
			Status          string `json:"status"`
		}

		if err := json.Unmarshal(body, &refundResp); err != nil {
			return fmt.Errorf("failed to parse refund response: %w", err)
		}

		txShort := "pending"
		if refundResp.TransactionHash != "" && len(refundResp.TransactionHash) > 10 {
			txShort = refundResp.TransactionHash[:10] + "..."
		}
		fmt.Printf("↩️  Refunded via facilitator (block %d, tx: %s)\n", refundResp.BlockNumber, txShort)

		payment.Status = PaymentRefunded

		return nil
	}

	// Direct escrow refund (fallback when not using facilitator)
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

	fmt.Printf("↩️  Refunded: %s %s (tx: %s)\n", formatEther(payment.Amount), pc.paymentTokenName, signedTx.Hash().Hex()[:10]+"...")

	return nil
}

// ReleasePartialPayment releases payment for approved tasks only (partial session payment)
func (pc *PaymentCoordinator) ReleasePartialPayment(sessionID string, approvedTasks, totalTasks int) error {
	payment, exists := pc.payments[sessionID]
	if !exists {
		// For session payments, try to release via facilitator directly
		if pc.UseFacilitator() {
			fmt.Printf("📡 Releasing partial payment via x402 Facilitator...\n")
			fmt.Printf("   Approved tasks: %d/%d\n", approvedTasks, totalTasks)

			releaseReq := map[string]interface{}{
				"taskId":        sessionID,
				"approvedTasks": approvedTasks,
				"totalTasks":    totalTasks,
				"partial":       true,
			}

			jsonData, err := json.Marshal(releaseReq)
			if err != nil {
				return err
			}

			resp, err := http.Post(pc.facilitatorURL+"/direct/finalize", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				return fmt.Errorf("failed to release partial payment: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("partial release failed: %s", string(body))
			}

			fmt.Printf("✅ Partial payment released for %d/%d approved tasks\n", approvedTasks, totalTasks)
			return nil
		}
		return fmt.Errorf("payment not found for session %s", sessionID)
	}

	// Calculate partial amount
	amountPerTask := new(big.Int).Div(payment.Amount, big.NewInt(int64(totalTasks)))
	partialAmount := new(big.Int).Mul(amountPerTask, big.NewInt(int64(approvedTasks)))

	fmt.Printf("📡 Releasing partial payment: %s %s (%d/%d tasks)\n",
		formatEther(partialAmount), pc.paymentTokenName, approvedTasks, totalTasks)

	// For facilitator-based payments
	if pc.UseFacilitator() && payment.Status == PaymentPending {
		releaseReq := map[string]interface{}{
			"taskId":        sessionID,
			"approved":      true,
			"approvedTasks": approvedTasks,
			"totalTasks":    totalTasks,
			"partial":       true,
		}

		jsonData, _ := json.Marshal(releaseReq)
		resp, err := http.Post(pc.facilitatorURL+"/direct/finalize", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("partial release failed: %s", string(body))
		}

		payment.Status = PaymentCompleted
		fmt.Printf("✅ Partial payment completed\n")
		return nil
	}

	payment.Status = PaymentCompleted
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

	fmt.Printf("💰 Demo payment: %s %s for task %s\n", formatEther(amount), pc.paymentTokenName, taskID)
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

// GetPaymentMode returns the configured payment mode (direct or session)
func (pc *PaymentCoordinator) GetPaymentMode() string {
	return pc.paymentMode
}

// VerifyPaymentLocked verifies that payment is locked in escrow on-chain before agent processes task
// This provides cryptographic proof that funds are secured, enabling trustless agent operation
func (pc *PaymentCoordinator) VerifyPaymentLocked(taskID string, agentAddr common.Address, minAmount *big.Int) (bool, error) {
	// SESSION MODE: Check if task is part of an active session
	if pc.useSessionPayments {
		for _, session := range pc.sessions {
			if session.Status == SessionActive || session.Status == SessionPending {
				for _, t := range session.TasksInSession {
					if t == taskID {
						// Verify agent address matches
						if session.AgentAddr != agentAddr {
							return false, fmt.Errorf("session agent %s doesn't match expected %s", session.AgentAddr.Hex(), agentAddr.Hex())
						}
						return true, nil
					}
				}
			}
		}
		// In session mode but task not in any session - might be standalone, fall through to check payments
	}

	// First check local payment tracker (for direct payments or facilitator payments)
	if trackedPayment, exists := pc.payments[taskID]; exists {
		// Verify the payment status is valid (deposited for escrow, pending/completed for direct)
		if trackedPayment.Status != PaymentDeposited &&
		   trackedPayment.Status != PaymentCompleted &&
		   trackedPayment.Status != PaymentPending {
			return false, fmt.Errorf("payment status is %s, expected deposited, pending, or completed", trackedPayment.Status)
		}

		// Verify agent address matches
		if trackedPayment.Agent != agentAddr {
			return false, fmt.Errorf("payment agent %s doesn't match expected %s", trackedPayment.Agent.Hex(), agentAddr.Hex())
		}

		// Verify amount is sufficient
		if trackedPayment.Amount.Cmp(minAmount) < 0 {
			return false, fmt.Errorf("payment amount %s is less than minimum %s", formatEther(trackedPayment.Amount), formatEther(minAmount))
		}

		return true, nil
	}

	// Fallback to checking escrow contract directly (for non-facilitator escrow payments)
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
	// 10 USDC = 10 * 10^6 wei (USDC has 6 decimals on Sepolia)
	result := new(big.Int)
	result.SetString("10000000", 10) // 10 * 10^6
	return result
}

func formatEther(wei *big.Int) string {
	// Convert USDC smallest units to USDC (divide by 10^6, not 10^18)
	// USDC uses 6 decimals on Sepolia, not 18 like ETH
	usdc := new(big.Float).SetInt(wei)
	divisor := new(big.Float).SetFloat64(1e6)
	usdc.Quo(usdc, divisor)
	return usdc.Text('f', 2)
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

// ============================================================================
// SESSION PAYMENT METHODS (x402 V2)
// ============================================================================

// IsSessionMode returns true if using session-based payments
func (pc *PaymentCoordinator) IsSessionMode() bool {
	return pc.useSessionPayments
}

// GetTasksPerSession returns the number of tasks per session/epoch
func (pc *PaymentCoordinator) GetTasksPerSession() int {
	return pc.tasksPerSession
}

// ShouldStartSession returns true if this task number starts a new session
// Task 1 starts session 1, task 4 starts session 2, etc.
func (pc *PaymentCoordinator) ShouldStartSession(taskNumber int) bool {
	if !pc.useSessionPayments {
		return false
	}
	return (taskNumber-1)%pc.tasksPerSession == 0
}

// ShouldFinalizeSession returns true if this task number ends a session
// Task 3 ends session 1, task 6 ends session 2, etc.
func (pc *PaymentCoordinator) ShouldFinalizeSession(taskNumber int) bool {
	if !pc.useSessionPayments {
		return false
	}
	return taskNumber%pc.tasksPerSession == 0
}

// GetEpochForTask returns the epoch number for a given task (1-indexed)
func (pc *PaymentCoordinator) GetEpochForTask(taskNumber int) int {
	return (taskNumber-1)/pc.tasksPerSession + 1
}

// IsStandaloneTask returns true if this task is outside complete epochs
// e.g., task 7 when we have 7 total tasks (2 complete epochs of 3 + 1 standalone)
func (pc *PaymentCoordinator) IsStandaloneTask(taskNumber, totalTasks int) bool {
	if !pc.useSessionPayments {
		return true // All tasks are standalone in per-task mode
	}
	completeEpochs := totalTasks / pc.tasksPerSession
	tasksInCompleteEpochs := completeEpochs * pc.tasksPerSession
	return taskNumber > tasksInCompleteEpochs
}

// StartSession initiates a session payment for an epoch
func (pc *PaymentCoordinator) StartSession(epochNumber int, clientAddr, agentAddr common.Address) error {
	sessionID := fmt.Sprintf("session-epoch-%d", epochNumber)

	// Calculate payment: 10 USDC per task × tasks per session
	amountPerTask := big.NewInt(10_000_000) // 10 USDC (6 decimals)
	totalAmount := new(big.Int).Mul(amountPerTask, big.NewInt(int64(pc.tasksPerSession)))

	session := &SessionPayment{
		SessionID:      sessionID,
		EpochNumber:    epochNumber,
		ClientAddr:     clientAddr,
		AgentAddr:      agentAddr,
		TotalAmount:    totalAmount,
		AmountPerTask:  amountPerTask,
		TasksInSession: make([]string, 0, pc.tasksPerSession),
		TasksPerEpoch:  pc.tasksPerSession,
		Status:         SessionPending,
		SessionStart:   time.Now(),
		SessionExpiry:  time.Now().Add(1 * time.Hour),
	}

	fmt.Printf("\n💰 Session %d starting: %d %s for %d tasks\n", epochNumber, pc.tasksPerSession*10, pc.paymentTokenName, pc.tasksPerSession)

	// Deposit session payment using direct/exact scheme (session payments don't use escrow)
	if pc.UseFacilitator() {
		amountHuman := fmt.Sprintf("%d", totalAmount.Int64()/1_000_000)
		// Session payments use "exact" scheme - direct payment, no escrow
		err := pc.SettlePaymentWithFacilitator(sessionID, clientAddr, agentAddr, amountHuman, "exact")
		if err != nil {
			return fmt.Errorf("failed to deposit session payment: %w", err)
		}
	} else {
		// Fallback: direct deposit to escrow contract
		err := pc.DepositPayment(sessionID, clientAddr, agentAddr, totalAmount)
		if err != nil {
			return fmt.Errorf("failed to deposit session payment: %w", err)
		}
	}

	session.Status = SessionActive
	pc.sessions[epochNumber] = session

	return nil
}

// AddTaskToSession tracks a task within the current session
func (pc *PaymentCoordinator) AddTaskToSession(epochNumber int, taskID string) {
	session, exists := pc.sessions[epochNumber]
	if !exists || session.Status != SessionActive {
		return
	}
	session.TasksInSession = append(session.TasksInSession, taskID)
}

// UpdateSessionTaskResult updates session with task quality result
func (pc *PaymentCoordinator) UpdateSessionTaskResult(epochNumber int, approved bool, qualityScore float64) {
	session, exists := pc.sessions[epochNumber]
	if !exists {
		return
	}
	session.TasksCompleted++
	if approved {
		session.TasksApproved++
	}
	// Running weighted average
	if session.TasksCompleted == 1 {
		session.AggregateQuality = qualityScore
	} else {
		session.AggregateQuality = (session.AggregateQuality*float64(session.TasksCompleted-1) + qualityScore) / float64(session.TasksCompleted)
	}
}

// FinalizeSession settles payment based on approved tasks (partial payment)
func (pc *PaymentCoordinator) FinalizeSession(epochNumber int) error {
	session, exists := pc.sessions[epochNumber]
	if !exists {
		return fmt.Errorf("session %d not found", epochNumber)
	}
	if session.Status != SessionActive {
		return fmt.Errorf("session %d not active (status: %s)", epochNumber, session.Status)
	}

	// Calculate partial payment: pay only for approved tasks
	amountPerTask := session.AmountPerTask.Int64() / 1_000_000 // Convert to USDC units
	approvedPayment := amountPerTask * int64(session.TasksApproved)
	refundAmount := amountPerTask * int64(session.TasksCompleted-session.TasksApproved)

	fmt.Printf("\n📊 Session %d: %d/%d approved, %d %s paid, %d %s refund\n",
		epochNumber, session.TasksApproved, session.TasksCompleted, approvedPayment, pc.paymentTokenName, refundAmount, pc.paymentTokenName)

	if session.TasksApproved > 0 {
		// Partial release: pay for approved tasks only
		err := pc.ReleasePartialPayment(session.SessionID, session.TasksApproved, session.TasksCompleted)
		if err != nil {
			return err
		}
		session.Status = SessionCompleted
	} else {
		// Full refund: no approved tasks
		err := pc.RefundPayment(session.SessionID)
		if err != nil {
			return err
		}
		session.Status = SessionRefunded
	}

	return nil
}

// GetSession returns the session for a given epoch
func (pc *PaymentCoordinator) GetSession(epochNumber int) *SessionPayment {
	return pc.sessions[epochNumber]
}

// PrintSessionSummary prints a summary of all sessions
func (pc *PaymentCoordinator) PrintSessionSummary() {
	if !pc.useSessionPayments && pc.directPaymentCount == 0 {
		return
	}

	fmt.Printf("\n💰 Payment Summary:\n")

	var totalPaid, totalRefunded int64

	// Session payments
	for i := 1; i <= len(pc.sessions); i++ {
		session := pc.sessions[i]
		if session == nil {
			continue
		}
		amountPerTask := session.AmountPerTask.Int64() / 1_000_000
		paidAmount := amountPerTask * int64(session.TasksApproved)
		refundedAmount := amountPerTask * int64(session.TasksCompleted-session.TasksApproved)

		switch session.Status {
		case SessionCompleted:
			totalPaid += paidAmount
			totalRefunded += refundedAmount
			fmt.Printf("   Session %d: %d %s (%d/%d approved)\n",
				i, paidAmount, pc.paymentTokenName, session.TasksApproved, session.TasksCompleted)
		case SessionRefunded:
			totalRefunded += session.TotalAmount.Int64() / 1_000_000
			fmt.Printf("   Session %d: REFUNDED\n", i)
		case SessionActive:
			fmt.Printf("   Session %d: ACTIVE\n", i)
		}
	}

	// Direct payments
	if pc.directPaymentCount > 0 {
		directTotal := pc.directPaymentTotal / 1_000_000
		totalPaid += directTotal
		fmt.Printf("   Direct: %d %s (%d tasks)\n",
			directTotal, pc.paymentTokenName, pc.directPaymentCount)
	}

	fmt.Printf("   ─────────────────────────────\n")
	fmt.Printf("   Total: %d %s paid, %d %s refunded\n",
		totalPaid, pc.paymentTokenName, totalRefunded, pc.paymentTokenName)
}