// Package subnet implements ERC-8004 agent wallet binding
//
// This file provides EIP-712 signature generation for setAgentWallet
// which cryptographically binds a wallet address to an agent identity.
package subnet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// WalletBindingManager handles agent wallet binding operations
type WalletBindingManager struct {
	client           *ethclient.Client
	identityRegistry common.Address
	chainID          *big.Int
}

// NewWalletBindingManager creates a new wallet binding manager
func NewWalletBindingManager(
	rpcURL string,
	identityRegistryAddr common.Address,
	chainID uint64,
) (*WalletBindingManager, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	return &WalletBindingManager{
		client:           client,
		identityRegistry: identityRegistryAddr,
		chainID:          big.NewInt(int64(chainID)),
	}, nil
}

// GenerateWalletBindingSignature creates EIP-712 signature for setAgentWallet
// The wallet (newWallet) must sign to consent to being bound to the agent
func GenerateWalletBindingSignature(
	agentID *big.Int,
	newWallet common.Address,
	owner common.Address,
	deadline *big.Int,
	walletPrivateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	identityRegistry common.Address,
) ([]byte, error) {
	// EIP-712 Domain Separator
	// keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)")
	domainTypeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))

	// Domain values
	nameHash := crypto.Keccak256Hash([]byte("ERC8004IdentityRegistry"))
	versionHash := crypto.Keccak256Hash([]byte("1"))

	// Encode domain separator
	domainSeparator := crypto.Keccak256Hash(
		append(append(append(append(
			domainTypeHash.Bytes(),
			nameHash.Bytes()...),
			versionHash.Bytes()...),
			math.U256Bytes(chainID)...),
			common.LeftPadBytes(identityRegistry.Bytes(), 32)...),
	)

	// EIP-712 Type Hash for AgentWalletSet
	// keccak256("AgentWalletSet(uint256 agentId,address newWallet,address owner,uint256 deadline)")
	typeHash := crypto.Keccak256Hash([]byte("AgentWalletSet(uint256 agentId,address newWallet,address owner,uint256 deadline)"))

	// Encode struct hash
	structHash := crypto.Keccak256Hash(
		append(append(append(append(
			typeHash.Bytes(),
			math.U256Bytes(agentID)...),
			common.LeftPadBytes(newWallet.Bytes(), 32)...),
			common.LeftPadBytes(owner.Bytes(), 32)...),
			math.U256Bytes(deadline)...),
	)

	// EIP-712 digest: keccak256("\x19\x01" || domainSeparator || structHash)
	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator.Bytes()...)
	rawData = append(rawData, structHash.Bytes()...)
	digest := crypto.Keccak256Hash(rawData)

	// Sign the digest
	signature, err := crypto.Sign(digest.Bytes(), walletPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Adjust v value for Ethereum (0/1 -> 27/28)
	if len(signature) == 65 {
		signature[64] += 27
	}

	return signature, nil
}

// BindAgentWallet binds a wallet to an agent identity
// This is called by the owner after the wallet has signed consent
func (wbm *WalletBindingManager) BindAgentWallet(
	agentID *big.Int,
	newWallet common.Address,
	walletPrivateKeyHex string, // Wallet signs consent
	ownerPrivateKeyHex string,  // Owner submits transaction
) (string, error) {
	// Parse private keys
	walletKeyHex := walletPrivateKeyHex
	if strings.HasPrefix(walletKeyHex, "0x") {
		walletKeyHex = walletKeyHex[2:]
	}
	walletKey, err := crypto.HexToECDSA(walletKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid wallet private key: %w", err)
	}

	ownerKeyHex := ownerPrivateKeyHex
	if strings.HasPrefix(ownerKeyHex, "0x") {
		ownerKeyHex = ownerKeyHex[2:]
	}
	ownerKey, err := crypto.HexToECDSA(ownerKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid owner private key: %w", err)
	}

	owner := crypto.PubkeyToAddress(ownerKey.PublicKey)

	// Deadline: 5 minutes from now (max allowed by contract)
	deadline := big.NewInt(time.Now().Add(5 * time.Minute).Unix())

	fmt.Printf("🔐 Generating wallet binding signature...\n")
	fmt.Printf("   Agent ID: %s\n", agentID.String())
	fmt.Printf("   Wallet: %s\n", newWallet.Hex())
	fmt.Printf("   Owner: %s\n", owner.Hex())
	fmt.Printf("   Deadline: %s\n", deadline.String())

	// Generate EIP-712 signature from wallet
	signature, err := GenerateWalletBindingSignature(
		agentID,
		newWallet,
		owner,
		deadline,
		walletKey,
		wbm.chainID,
		wbm.identityRegistry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate signature: %w", err)
	}

	fmt.Printf("   Signature: 0x%x...%x\n", signature[:4], signature[len(signature)-4:])

	// Submit setAgentWallet transaction
	txHash, err := wbm.submitSetAgentWallet(agentID, newWallet, deadline, signature, ownerKey)
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction: %w", err)
	}

	fmt.Printf("✅ Wallet binding submitted\n")
	fmt.Printf("   Transaction: %s\n", txHash)

	return txHash, nil
}

// submitSetAgentWallet submits the setAgentWallet transaction
func (wbm *WalletBindingManager) submitSetAgentWallet(
	agentID *big.Int,
	newWallet common.Address,
	deadline *big.Int,
	signature []byte,
	ownerKey *ecdsa.PrivateKey,
) (string, error) {
	// ABI for setAgentWallet
	setAgentWalletABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"},
			{"internalType": "address", "name": "newWallet", "type": "address"},
			{"internalType": "uint256", "name": "deadline", "type": "uint256"},
			{"internalType": "bytes", "name": "signature", "type": "bytes"}
		],
		"name": "setAgentWallet",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(setAgentWalletABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	data, err := parsedABI.Pack("setAgentWallet", agentID, newWallet, deadline, signature)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create transactor
	auth, err := bind.NewKeyedTransactorWithChainID(ownerKey, wbm.chainID)
	if err != nil {
		return "", fmt.Errorf("failed to create transactor: %w", err)
	}

	nonce, err := wbm.client.PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := wbm.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(
		nonce,
		wbm.identityRegistry,
		big.NewInt(0),
		200000,
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(wbm.chainID), ownerKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = wbm.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	// Wait for receipt
	receipt, err := bind.WaitMined(context.Background(), wbm.client, signedTx)
	if err != nil {
		return txHash, fmt.Errorf("transaction failed: %w", err)
	}

	if receipt.Status != 1 {
		return txHash, fmt.Errorf("transaction reverted")
	}

	return txHash, nil
}

// GetAgentWallet queries the bound wallet for an agent
func (wbm *WalletBindingManager) GetAgentWallet(agentID *big.Int) (common.Address, error) {
	getAgentWalletABI := `[{
		"inputs": [
			{"internalType": "uint256", "name": "agentId", "type": "uint256"}
		],
		"name": "getAgentWallet",
		"outputs": [
			{"internalType": "address", "name": "", "type": "address"}
		],
		"stateMutability": "view",
		"type": "function"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(getAgentWalletABI))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to parse ABI: %w", err)
	}

	data, err := parsedABI.Pack("getAgentWallet", agentID)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack function call: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &wbm.identityRegistry,
		Data: data,
	}

	result, err := wbm.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to call contract: %w", err)
	}

	var wallet common.Address
	err = parsedABI.UnpackIntoInterface(&wallet, "getAgentWallet", result)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to unpack result: %w", err)
	}

	return wallet, nil
}

// Close closes the client connection
func (wbm *WalletBindingManager) Close() {
	if wbm.client != nil {
		wbm.client.Close()
	}
}
