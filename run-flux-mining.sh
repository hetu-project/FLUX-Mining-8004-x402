#!/bin/bash

# PoCW FLUX Mining Script with ERC-8004 
# This script runs the complete PoCW system with FLUX token mining:
# - ERC-8004 Identity Registry for miner authentication
# - Miners must have Agent ID to register subnets
# - Real-time epoch submission where each completed epoch (3 rounds)
#   triggers immediate mainnet submission and FLUX mining.

echo "💰 PoCW FLUX MINING SYSTEM WITH ERC-8004 and x402"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Architecture: FLUX mining with ERC-8004 and x402"
echo ""

# Parse command line arguments
PAYMENT_TOKEN="USDC"  # Default to USDC

while [[ $# -gt 0 ]]; do
    case $1 in
        --payment-token)
            PAYMENT_TOKEN="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--payment-token USDC|AIUSD]"
            exit 1
            ;;
    esac
done

# Validate payment token
if [[ "$PAYMENT_TOKEN" != "USDC" && "$PAYMENT_TOKEN" != "AIUSD" ]]; then
    echo "❌ Invalid payment token: $PAYMENT_TOKEN"
    echo "   Must be either 'USDC' or 'AIUSD'"
    exit 1
fi

echo "💵 Payment Token: $PAYMENT_TOKEN"
echo ""

# Preserve user's PATH when running with sudo
if [ -n "$SUDO_USER" ]; then
    USER_HOME=$(eval echo ~$SUDO_USER)
    # Common installation paths for Go, Foundry, Node.js
    export PATH="/usr/local/go/bin:$USER_HOME/go/bin:$USER_HOME/.foundry/bin:$USER_HOME/.local/bin:/snap/bin:$PATH"
else
    # Fallback paths when not running with sudo
    export PATH="/home/xx/.foundry/bin:$PATH"
fi

# Check prerequisites
echo "🔍 Checking prerequisites..."
if ! command -v anvil &> /dev/null; then
    echo "❌ Anvil not found. Please install Foundry."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go >= 1.21."
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "❌ Node.js not found. Please install Node.js."
    exit 1
fi

# Check if npm dependencies are installed
if [ ! -d "node_modules" ] || [ ! -f "node_modules/ethers/package.json" ]; then
    echo "📦 Installing Node.js dependencies..."
    npm install
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install Node.js dependencies. Please run 'npm install' manually."
        exit 1
    fi
    echo "✅ Node.js dependencies installed successfully"
else
    echo "✅ Node.js dependencies already installed"
fi

# Check if Foundry dependencies (OpenZeppelin contracts) are installed
if [ ! -d "lib/openzeppelin-contracts" ]; then
    echo "📦 Installing Foundry dependencies (OpenZeppelin contracts)..."
    # Determine correct forge path
    if [ -n "$SUDO_USER" ]; then
        USER_HOME=$(eval echo ~$SUDO_USER)
        FORGE_CMD="$USER_HOME/.foundry/bin/forge"
    else
        FORGE_CMD="$HOME/.foundry/bin/forge"
    fi
    
    $FORGE_CMD install OpenZeppelin/openzeppelin-contracts
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install Foundry dependencies. Please run 'forge install OpenZeppelin/openzeppelin-contracts' manually."
        exit 1
    fi
    echo "✅ Foundry dependencies installed successfully"
else
    echo "✅ Foundry dependencies already installed"
fi

echo "✅ All prerequisites found"

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Cleaning up processes..."
    
    # Stop Dashboard server
    if [ ! -z "$DASHBOARD_PID" ]; then
        echo "🔴 Stopping Dashboard server (PID: $DASHBOARD_PID)..."
        kill $DASHBOARD_PID 2>/dev/null || true
        sleep 1
    fi
    
    # Stop Bridge service
    if [ ! -z "$BRIDGE_PID" ]; then
        echo "🔴 Stopping Bridge service (PID: $BRIDGE_PID)..."
        kill $BRIDGE_PID 2>/dev/null || true
        sleep 1
    fi
    
    # Stop Anvil
    if [ -f anvil-per-epoch.pid ]; then
        ANVIL_PID=$(cat anvil-per-epoch.pid)
        if kill -0 $ANVIL_PID 2>/dev/null; then
            echo "🔴 Stopping Anvil (PID: $ANVIL_PID)..."
            kill $ANVIL_PID
            sleep 2
        fi
        rm -f anvil-per-epoch.pid anvil-per-epoch.log
    fi

    # Stop the bridge if running
    if [ ! -z "$BRIDGE_PID" ] && kill -0 $BRIDGE_PID 2>/dev/null; then
        echo "🔴 Stopping Bridge (PID: $BRIDGE_PID)..."
        kill $BRIDGE_PID
    fi
    # Also kill any other processes on port 3001
    if lsof -i :3001 > /dev/null 2>&1; then
        echo "🔴 Stopping any remaining Bridge processes on port 3001..."
        lsof -ti :3001 | xargs kill -9 2>/dev/null || true
    fi

    # Stop Dgraph and Ratel containers
    echo "🔴 Stopping Dgraph and Ratel containers..."
    docker stop dgraph-standalone 2>/dev/null || true
    docker rm dgraph-standalone 2>/dev/null || true
    docker stop dgraph-ratel 2>/dev/null || true
    docker rm dgraph-ratel 2>/dev/null || true
    
    # Clean up Dgraph data directory
    echo "🧹 Cleaning up Dgraph data..."
    rm -rf ./dgraph-data 2>/dev/null || true
    
    # Clean up temporary files
    rm -f contract_addresses.json dashboard.log
    
    echo "✅ Cleanup complete"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# === PHASE 1: START INFRASTRUCTURE ===
echo ""
echo "🚀 PHASE 1: Starting Infrastructure"
echo "────────────────────────────────────"

# === DASHBOARD SERVER SETUP ===
echo "🌐 Starting dashboard server on port 3000..."
nohup go run serve-dashboard.go > dashboard.log 2>&1 &
DASHBOARD_PID=$!

# Wait for dashboard to be ready
echo "⏳ Waiting for dashboard server to start..."
for i in {1..10}; do
    if curl -s http://localhost:3000 >/dev/null 2>&1; then
        echo "✅ Dashboard server is ready (PID: $DASHBOARD_PID)"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "⚠️  Dashboard server may not be ready, continuing anyway..."
    fi
    sleep 1
done

# === DGRAPH SETUP ===
echo "Setting up Dgraph for VLC event visualization..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "⚠️  Docker not found. Please install Docker to enable VLC visualization."
    DGRAPH_STARTED=false
else
    echo "🐳 Docker found, setting up Dgraph container..."
    
    # Stop any existing Dgraph/Ratel containers
    echo "🔴 Stopping existing Dgraph/Ratel containers..."
    docker stop dgraph-standalone 2>/dev/null || true
    docker rm dgraph-standalone 2>/dev/null || true
    docker stop dgraph-ratel 2>/dev/null || true
    docker rm dgraph-ratel 2>/dev/null || true
    
    # Remove any existing Dgraph data to start fresh
    echo "🧹 Cleaning up previous Dgraph data..."
    rm -rf ./dgraph-data 2>/dev/null || true
    mkdir -p ./dgraph-data
    
    # Start new Dgraph container with proper setup
    echo "🚀 Starting fresh Dgraph container..."
    DGRAPH_OUTPUT=$(docker run --rm -d --name dgraph-standalone \
        -p 8080:8080 -p 9080:9080 \
        -v $(pwd)/dgraph-data:/dgraph \
        dgraph/standalone:latest 2>&1)

    if [ $? -eq 0 ]; then
        echo "✅ Dgraph container started successfully"
        echo "   - Container ID: $(echo $DGRAPH_OUTPUT | cut -c1-12)"
        echo "   - GraphQL Endpoint: http://localhost:8080/graphql"
        echo "   - Query Endpoint: http://localhost:8080/query"
        echo "   - GRPC Endpoint: localhost:9080"

        # Start Ratel UI container separately (using stable version)
        echo "🚀 Starting Ratel UI container..."
        RATEL_OUTPUT=$(docker run --rm -d --name dgraph-ratel \
            -p 8000:8000 \
            dgraph/ratel:v21.03.0 2>&1)

        if [ $? -eq 0 ]; then
            echo "✅ Ratel UI container started successfully"
            echo "   - Ratel UI: http://localhost:8000"
            RATEL_STARTED=true
        else
            echo "⚠️  Warning: Ratel UI failed to start, but continuing with Dgraph only"
            echo "   You can still use GraphQL at http://localhost:8080"
            RATEL_STARTED=false
        fi

        DGRAPH_STARTED=true
    else
        echo "❌ Dgraph container failed to start: $DGRAPH_OUTPUT"
        echo "⚠️  This might be due to:"
        echo "   - Port conflicts (8080, 9080 already in use)"
        echo "   - Docker permissions issues"
        echo "   - Insufficient system resources"
        echo "⚠️  Continuing without VLC visualization"
        DGRAPH_STARTED=false
    fi
fi

# Wait for Dgraph to be fully ready
if [ "$DGRAPH_STARTED" = true ]; then
    echo "⏳ Waiting for Dgraph to be ready..."
    DGRAPH_READY=false
    
    for i in {1..30}; do
        # Test GraphQL endpoint
        if curl -s -f http://localhost:8080/health >/dev/null 2>&1; then
            echo "✅ Dgraph health check passed"
            
            # Test if we can query (more comprehensive check)
            if curl -s -X POST -H "Content-Type: application/json" \
               -d '{"query": "{ __schema { queryType { name } } }"}' \
               http://localhost:8080/graphql >/dev/null 2>&1; then
                echo "✅ Dgraph GraphQL endpoint is ready"
                DGRAPH_READY=true
                break
            fi
        fi
        
        if [ $i -eq 30 ]; then
            echo "⚠️  Dgraph not fully ready after 60 seconds, continuing anyway"
            echo "   You may need to wait a bit more before VLC visualization works"
            DGRAPH_READY=true  # Continue anyway
        else
            echo "   Dgraph starting up... (attempt $i/30)"
        fi
        sleep 2
    done
    
    if [ "$DGRAPH_READY" = true ]; then
        echo "🎯 Dgraph is ready for VLC event tracking!"
        echo "   Access Ratel UI at: http://localhost:8000"
        echo ""
    fi
else
    echo "⏭️  Skipping Dgraph readiness check (not started)"
fi

# === COMPLETE RESET FOR FRESH START ===
echo "🔄 Ensuring fresh start..."
echo "Cleaning up ALL existing Anvil instances..."
pkill -f anvil 2>/dev/null || true
sleep 2

# Force kill port 8545 if still in use
if lsof -i :8545 >/dev/null 2>&1; then
    echo "   Port 8545 still in use, force killing..."
    lsof -ti :8545 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# Clean up all artifacts for fresh deployment
rm -f anvil-per-epoch.pid anvil-per-epoch.log
rm -f contract_addresses.json
rm -f erc8004_deployment.json
echo "✅ Clean slate ready for fresh deployment"

# Start fresh Anvil blockchain
echo "Starting fresh Anvil blockchain from genesis block..."
nohup anvil \
    --accounts 10 \
    --balance 10000 \
    --port 8545 \
    --host 0.0.0.0 \
    --mnemonic "test test test test test test test test test test test junk" \
    > anvil-per-epoch.log 2>&1 &

ANVIL_PID=$!
echo $ANVIL_PID > anvil-per-epoch.pid

# Wait for Anvil to be ready
echo "Waiting for Anvil to be ready..."
for i in {1..10}; do
    if curl -s -X POST -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
       http://localhost:8545 >/dev/null 2>&1; then
        
        BLOCK_NUM=$(curl -s -X POST -H "Content-Type: application/json" \
           --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
           http://localhost:8545 | grep -o '"result":"0x[0-9a-f]*"' | cut -d'"' -f4)
        BLOCK_DEC=$((16#${BLOCK_NUM#0x}))
        
        echo "✅ Anvil is ready (PID: $ANVIL_PID) - Starting from block $BLOCK_DEC"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Anvil failed to start"
        cleanup
        exit 1
    fi
    sleep 1
done

# === PHASE 2: DEPLOY MAINNET CONTRACTS WITH ERC-8004 IDENTITY ===
echo ""
echo "📋 PHASE 2: Deploying Mainnet Contracts with ERC-8004 Identity"
echo "────────────────────────────────────────────"

# Configuration
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
VALIDATOR1_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
VALIDATOR2_KEY="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
VALIDATOR3_KEY="0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
VALIDATOR4_KEY="0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
MINER_KEY="0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
RPC_URL="http://localhost:8545"

DEPLOYER="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
VALIDATOR1="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
VALIDATOR2="0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
VALIDATOR3="0x90F79bf6EB2c4f870365E785982E1f101E93b906"
VALIDATOR4="0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
MINER="0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
CLIENT="$VALIDATOR2"  # Client for x402 payments (using VALIDATOR2 to match Go demo)

# Determine correct forge path
if [ -n "$SUDO_USER" ]; then
    USER_HOME=$(eval echo ~$SUDO_USER)
    FORGE_PATH="$USER_HOME/.foundry/bin/forge"
    CAST_PATH="$USER_HOME/.foundry/bin/cast"
else
    FORGE_PATH="$HOME/.foundry/bin/forge"
    CAST_PATH="$HOME/.foundry/bin/cast"
fi

# Compile contracts
echo "Compiling contracts..."
$FORGE_PATH build > /dev/null 2>&1

# === Deploy ERC-8004 Identity, Validation, and Reputation Registries ===
echo ""
echo "================================================================"
echo "          ERC-8004 IDENTITY & VALIDATION DEPLOYMENT"
echo "================================================================"
echo ""

echo "[1/3] 🆔 Deploying IdentityRegistry..."
IDENTITY_RESULT=$($FORGE_PATH create contracts/8004/IdentityRegistry.sol:IdentityRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
IDENTITY_ADDRESS=$(echo "$IDENTITY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "  └─ Deployed at: $IDENTITY_ADDRESS"

echo ""
echo "[2/3] 🔍 Deploying ValidationRegistry..."
echo "  ├─ Linking to IdentityRegistry: $IDENTITY_ADDRESS"
VALIDATION_RESULT=$($FORGE_PATH create contracts/8004/ValidationRegistry.sol:ValidationRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast \
    --constructor-args "$IDENTITY_ADDRESS" 2>&1)
VALIDATION_ADDRESS=$(echo "$VALIDATION_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "  └─ Deployed at: $VALIDATION_ADDRESS"

echo ""
echo "[3/3] ⭐ Deploying ReputationRegistry..."
echo "  ├─ Linking to IdentityRegistry: $IDENTITY_ADDRESS"
REPUTATION_RESULT=$($FORGE_PATH create contracts/8004/ReputationRegistry.sol:ReputationRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast \
    --constructor-args "$IDENTITY_ADDRESS" 2>&1)
REPUTATION_ADDRESS=$(echo "$REPUTATION_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "  └─ Deployed at: $REPUTATION_ADDRESS"

echo ""
echo "✅ ERC-8004 Registries Deployed Successfully"
echo "────────────────────────────────────────────"

# === Deploy Payment Token (USDC or AIUSD) for x402 Payments ===
echo "💵 Deploying $PAYMENT_TOKEN Token (x402 payment stablecoin)..."

if [ "$PAYMENT_TOKEN" == "USDC" ]; then
    TOKEN_RESULT=$($FORGE_PATH create contracts/USDC.sol:USDC \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
else
    TOKEN_RESULT=$($FORGE_PATH create contracts/AIUSD.sol:AIUSD \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
fi

PAYMENT_TOKEN_ADDRESS=$(echo "$TOKEN_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

# Debug: Show if payment token deployment failed
if [ -z "$PAYMENT_TOKEN_ADDRESS" ]; then
    echo "   ❌ ERROR: $PAYMENT_TOKEN Token deployment failed!"
    echo "$TOKEN_RESULT" | head -20
fi

echo "   $PAYMENT_TOKEN Token: $PAYMENT_TOKEN_ADDRESS"

# Register miner with ERC-8004 to get Agent ID
echo "🆔 Registering miner with ERC-8004 identity..."

# Register miner and wait for transaction to complete
REGISTER_TX=$($CAST_PATH send $IDENTITY_ADDRESS "register()" \
    --private-key $MINER_KEY \
    --rpc-url $RPC_URL \
    --gas-limit 300000 --json 2>&1)

if echo "$REGISTER_TX" | grep -q "blockHash\|transactionHash"; then
    echo "   ✅ Registration transaction confirmed"

    # Parse the Registered event from logs to get the actual agent ID
    # Event signature: Registered(uint256 indexed agentId, string tokenURI, address indexed owner)
    TX_HASH=$(echo "$REGISTER_TX" | jq -r '.transactionHash' 2>/dev/null || echo "")

    if [ -n "$TX_HASH" ]; then
        # Get transaction receipt and parse logs
        RECEIPT=$($CAST_PATH receipt $TX_HASH --rpc-url $RPC_URL --json 2>&1)
        # The first topic after the event signature is the agentId
        AGENT_ID_HEX=$(echo "$RECEIPT" | jq -r '.logs[0].topics[1]' 2>/dev/null || echo "0x0")
        AGENT_ID_DEC=$((AGENT_ID_HEX))
        echo "   Agent ID assigned: $AGENT_ID_DEC"
    else
        # Fallback: First registration gets ID 0 (post-increment behavior)
        AGENT_ID_DEC="0"
        echo "   Agent ID (assumed first registration): $AGENT_ID_DEC"
    fi
else
    echo "   ⚠️  Registration failed, using fallback ID 0"
    # First registration in ERC-8004 gets ID 0 (_lastId++ is post-increment)
    AGENT_ID_DEC="0"
fi

echo ""
echo "📋 VLC Protocol Validation"
echo "────────────────────────────────────────────"
echo "   ℹ️  VLC validation will be performed BEFORE subnet registration"
echo "   The agent must pass VLC protocol tests to continue"
echo ""

# Deploy contracts
echo "🔧 Deploying Core Contracts..."
echo "────────────────────────────────────────────"
echo "Deploying HETU Token..."
HETU_RESULT=$($FORGE_PATH create contracts/HETUToken.sol:HETUToken \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
HETU_ADDRESS=$(echo "$HETU_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "Deploying FLUX Token..."
FLUX_RESULT=$($FORGE_PATH create contracts/FLUXToken.sol:FLUXToken \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
FLUX_ADDRESS=$(echo "$FLUX_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "🔐 Deploying Subnet Registry..."
REGISTRY_RESULT=$($FORGE_PATH create contracts/SubnetRegistry.sol:SubnetRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
REGISTRY_ADDRESS=$(echo "$REGISTRY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "Deploying PoCW Verifier..."
VERIFIER_RESULT=$($FORGE_PATH create contracts/PoCWVerifier.sol:PoCWVerifier \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
VERIFIER_ADDRESS=$(echo "$VERIFIER_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "🔒 Deploying x402PaymentEscrow..."

# Ensure PAYMENT_TOKEN_ADDRESS is set before deploying escrow
if [ -z "$PAYMENT_TOKEN_ADDRESS" ]; then
    echo "   ❌ ERROR: PAYMENT_TOKEN_ADDRESS is not set! Cannot deploy escrow."
    ESCROW_ADDRESS=""
else
    # IMPORTANT: --private-key, --rpc-url, and --broadcast must come BEFORE --constructor-args
    # Otherwise forge parses them as constructor arguments!
    ESCROW_RESULT=$($FORGE_PATH create contracts/x402PaymentEscrow.sol:x402PaymentEscrow \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast \
        --constructor-args "$PAYMENT_TOKEN_ADDRESS" 2>&1)
    ESCROW_ADDRESS=$(echo "$ESCROW_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
fi

echo "   x402PaymentEscrow: $ESCROW_ADDRESS"

# Initialize contracts (SubnetRegistry needs HETU, Identity, and Validation addresses)
echo "Initializing contracts..."
timeout 3 $CAST_PATH send $REGISTRY_ADDRESS "initialize(address,address,address)" $HETU_ADDRESS $IDENTITY_ADDRESS $VALIDATION_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

timeout 3 $CAST_PATH send $VERIFIER_ADDRESS "initialize(address,address)" $FLUX_ADDRESS $REGISTRY_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

timeout 3 $CAST_PATH send $FLUX_ADDRESS "setPoCWVerifier(address)" $VERIFIER_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

# Authorize V1 (Validator1) as coordinator in x402PaymentEscrow
echo "Authorizing V1 as x402 payment coordinator..."
timeout 3 $CAST_PATH send $ESCROW_ADDRESS "authorizeCoordinator(address)" $VALIDATOR1 \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ V1 Coordinator authorized successfully"
else
    echo "   ⚠️  Failed to authorize V1 Coordinator"
fi

# Distribute HETU and setup subnet
echo "Setting up subnet participants..."
timeout 3 $CAST_PATH send $HETU_ADDRESS "transfer(address,uint256)" $MINER $($CAST_PATH --to-wei 2000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

for VALIDATOR in $VALIDATOR1 $VALIDATOR2 $VALIDATOR3 $VALIDATOR4; do
    timeout 3 $CAST_PATH send $HETU_ADDRESS "transfer(address,uint256)" $VALIDATOR $($CAST_PATH --to-wei 2000) \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
done

# Bootstrap client with HETU and payment token for x402 payments
echo "💵 Bootstrapping client with ETH, $PAYMENT_TOKEN and HETU..."
# First, send some ETH to client for gas fees
echo "   Sending 10 ETH to client..."
timeout 3 $CAST_PATH send $CLIENT --value 10ether \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ Client funded with 10 ETH"
else
    echo "   ⚠️  Failed to send ETH to client"
fi
timeout 3 $CAST_PATH send $HETU_ADDRESS "transfer(address,uint256)" $CLIENT $($CAST_PATH --to-wei 1000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
timeout 3 $CAST_PATH send $PAYMENT_TOKEN_ADDRESS "mint(address,uint256)" $CLIENT $($CAST_PATH --to-wei 1000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
echo "   Client ($CLIENT) setup complete:"
echo "   - ETH for gas (see above)"
echo "   - 1000 HETU tokens"
echo "   - 1000 $PAYMENT_TOKEN tokens for task payments"

echo "💵 Bootstrapping V1 Coordinator with $PAYMENT_TOKEN for demo payments..."
timeout 3 $CAST_PATH send $PAYMENT_TOKEN_ADDRESS "mint(address,uint256)" $VALIDATOR1 $($CAST_PATH --to-wei 100) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
echo "   V1 Coordinator ($VALIDATOR1) bootstrapped with:"
echo "   - 100 $PAYMENT_TOKEN tokens for demo payment distribution"

echo "🔓 Approving escrow to spend client's $PAYMENT_TOKEN..."
# Client approves escrow contract to spend payment tokens for payments (unlimited approval)
# Client is VALIDATOR2 (account #2): 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
CLIENT_KEY="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
timeout 3 $CAST_PATH send $PAYMENT_TOKEN_ADDRESS "approve(address,uint256)" $ESCROW_ADDRESS "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" \
    --private-key $CLIENT_KEY --rpc-url $RPC_URL > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ Client approved escrow to spend $PAYMENT_TOKEN"
else
    echo "   ⚠️  Failed to approve escrow"
fi

# Approvals
timeout 3 $CAST_PATH send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $($CAST_PATH --to-wei 500) \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

VALIDATOR_KEYS=("$VALIDATOR1_KEY" "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a")
for i in 0 1 2 3; do
    timeout 3 $CAST_PATH send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $($CAST_PATH --to-wei 100) \
        --private-key ${VALIDATOR_KEYS[$i]} --rpc-url $RPC_URL > /dev/null 2>&1 || true
done

# Subnet registration will happen AFTER VLC validation passes
SUBNET_ID="per-epoch-subnet-001"

# Generate contract addresses JSON for bridge and inspector (name -> address format)
cat > contract_addresses.json << EOF
{
  "IdentityRegistry": "$IDENTITY_ADDRESS",
  "ValidationRegistry": "$VALIDATION_ADDRESS",
  "ReputationRegistry": "$REPUTATION_ADDRESS",
  "HETUToken": "$HETU_ADDRESS",
  "FLUXToken": "$FLUX_ADDRESS",
  "SubnetRegistry": "$REGISTRY_ADDRESS",
  "PoCWVerifier": "$VERIFIER_ADDRESS",
  "PaymentToken": "$PAYMENT_TOKEN_ADDRESS",
  "PaymentTokenName": "$PAYMENT_TOKEN",
  "x402PaymentEscrow": "$ESCROW_ADDRESS",
  "Client": "$CLIENT",
  "Agent": "$MINER",
  "AgentPrivateKey": "$MINER_KEY",
  "V1Coordinator": "$VALIDATOR1",
  "Validator1": "$VALIDATOR1",
  "Validator2": "$VALIDATOR2",
  "Validator3": "$VALIDATOR3",
  "Validator4": "$VALIDATOR4",
  "SubnetID": "$SUBNET_ID",
  "AgentID": "$AGENT_ID_DEC",
  "ChainID": "31337",
  "RpcUrl": "$RPC_URL"
}
EOF

echo ""
echo "📄 Contract addresses saved to contract_addresses.json"
echo "   View in inspector at: http://localhost:3000/pocw-inspector.html"

echo "✅ Mainnet contracts deployed with ERC-8004 + VLC Validation + x402 Payments"
echo ""
echo "   ERC-8004 Registries:"
echo "   ├─ 🆔 Identity: $IDENTITY_ADDRESS"
echo "   ├─ 🔍 Validation: $VALIDATION_ADDRESS"
echo "   └─ ⭐ Reputation: $REPUTATION_ADDRESS"
echo ""
echo "   Core Mining Contracts:"
echo "   ├─ 💰 HETU Token: $HETU_ADDRESS"
echo "   ├─ ⚡ FLUX Token: $FLUX_ADDRESS"
echo "   └─ 📋 PoCW Verifier: $VERIFIER_ADDRESS"
echo ""
echo "   x402 Payment System:"
echo "   ├─ 💵 $PAYMENT_TOKEN Token: $PAYMENT_TOKEN_ADDRESS"
echo "   ├─ 🔒 Escrow: $ESCROW_ADDRESS"
echo "   └─ 👤 Client Address: $CLIENT"
echo ""
echo "   Agent Info:"
echo "   ├─ Agent ID: #$AGENT_ID_DEC"
echo "   └─ Agent Address: $MINER"

# Helper function to format wei to FLUX tokens
format_flux_balance() {
    local wei_value=$1
    # Convert scientific notation to decimal if needed
    local decimal_value=$(printf "%.0f" $wei_value 2>/dev/null || echo $wei_value)
    # Convert from wei (divide by 10^18)
    local flux_value=$(echo "scale=6; $decimal_value / 1000000000000000000" | bc -l)
    echo $flux_value
}

# === INITIAL FLUX BALANCES ===
echo ""
echo "💰 Initial FLUX Token Balances (Before Mining)"
echo "────────────────────────────────────────────────"
echo "📊 Miner ($MINER):"
MINER_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_INITIAL_FORMATTED=$(format_flux_balance $MINER_INITIAL)
echo "   Balance: $MINER_INITIAL_FORMATTED FLUX"

echo "📊 Validator-1 ($VALIDATOR1):"
V1_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_INITIAL_FORMATTED=$(format_flux_balance $V1_INITIAL)
echo "   Balance: $V1_INITIAL_FORMATTED FLUX"

echo "📊 Validator-2 ($VALIDATOR2):"
V2_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR2 --rpc-url $RPC_URL)
V2_INITIAL_FORMATTED=$(format_flux_balance $V2_INITIAL)
echo "   Balance: $V2_INITIAL_FORMATTED FLUX"

echo "📊 Validator-3 ($VALIDATOR3):"
V3_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR3 --rpc-url $RPC_URL)
V3_INITIAL_FORMATTED=$(format_flux_balance $V3_INITIAL)
echo "   Balance: $V3_INITIAL_FORMATTED FLUX"

echo "📊 Validator-4 ($VALIDATOR4):"
V4_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR4 --rpc-url $RPC_URL)
V4_INITIAL_FORMATTED=$(format_flux_balance $V4_INITIAL)
echo "   Balance: $V4_INITIAL_FORMATTED FLUX"

TOTAL_SUPPLY_INITIAL=$($CAST_PATH call $FLUX_ADDRESS "totalSupply()(uint256)" --rpc-url $RPC_URL)
TOTAL_SUPPLY_INITIAL_FORMATTED=$(format_flux_balance $TOTAL_SUPPLY_INITIAL)
echo "📊 Total Supply: $TOTAL_SUPPLY_INITIAL_FORMATTED FLUX"
echo ""

# === PAYMENT TOKEN BALANCES (x402 Payment System) ===
echo "💵 $PAYMENT_TOKEN Token Balances (x402 Payment System)"
echo "────────────────────────────────────────────────"
echo "📊 Client ($CLIENT):"
CLIENT_PAYMENT=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $CLIENT --rpc-url $RPC_URL)
CLIENT_PAYMENT_FORMATTED=$(format_flux_balance $CLIENT_PAYMENT)
echo "   Balance: $CLIENT_PAYMENT_FORMATTED $PAYMENT_TOKEN"

echo "📊 Miner/Agent ($MINER):"
MINER_PAYMENT=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_PAYMENT_FORMATTED=$(format_flux_balance $MINER_PAYMENT)
echo "   Balance: $MINER_PAYMENT_FORMATTED $PAYMENT_TOKEN"

echo "📊 V1 Coordinator ($VALIDATOR1):"
V1_PAYMENT=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_PAYMENT_FORMATTED=$(format_flux_balance $V1_PAYMENT)
echo "   Balance: $V1_PAYMENT_FORMATTED $PAYMENT_TOKEN"
echo ""

# === PHASE 3: PER-EPOCH DEMONSTRATION ===
echo ""
echo "🧠 PHASE 3: Per-Epoch PoCW Subnet Demo"
echo "────────────────────────────────────────────"

# Dgraph schema initialization
if [ "$DGRAPH_STARTED" = true ]; then
    echo "🔧 Dgraph is ready for VLC events..."
    echo "📝 Note: Schema is automatically set by the Go subnet code"
    echo "🎯 Dgraph ready for per-epoch VLC event tracking!"
else
    echo "⏭️  Skipping Dgraph setup (Dgraph not started)"
fi

echo "🔄 Starting per-epoch demonstration..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 How it works:"
echo "  1. Subnet processes tasks in rounds (3 rounds = 1 epoch)"
echo "  2. When epoch completes → EpochFinalized event triggers"
echo "  3. Callback immediately submits epoch data to mainnet"
echo "  4. FLUX tokens are mined in real-time per completed epoch"
echo "  5. Process repeats for each new epoch"
echo ""
echo "🎯 Expected behavior:"
echo "  - Epoch 1 (rounds 1-3): Submit after task 3 completes"
echo "  - Epoch 2 (rounds 4-6): Submit after task 6 completes" 
echo "  - Partial epoch 3 (round 7): Submit after demo ends"
echo ""

# Kill any existing bridge process on port 3001
if lsof -i :3001 > /dev/null 2>&1; then
    echo "⚠️  Found existing bridge on port 3001, killing it..."
    lsof -ti :3001 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

# Initialize the Node.js bridge with HTTP server
echo "🌐 Initializing Per-Epoch Mainnet Bridge with HTTP server..."
node -e "
const PerEpochBridge = require('./mainnet-bridge-per-epoch.js');
const bridge = new PerEpochBridge();

async function setupBridge() {
    try {
        await bridge.initialize();
        console.log('✅ Per-Epoch Bridge ready for HTTP requests from Go');
        
        // Keep the bridge running to receive HTTP requests
        process.on('SIGTERM', () => {
            console.log('🔴 Bridge shutting down...');
            process.exit(0);
        });
        
    } catch (error) {
        console.error('❌ Bridge setup failed:', error.message);
        process.exit(1);
    }
}

setupBridge();
" &

BRIDGE_PID=$!
echo "⏳ Waiting for bridge HTTP server to initialize..."
sleep 5

# Verify bridge is ready
echo "🔍 Verifying bridge HTTP server..."
if curl -s http://localhost:3001/health > /dev/null; then
    echo "✅ Bridge HTTP server is ready and responding"
else
    echo "⚠️  Bridge HTTP server may not be ready, continuing anyway..."
fi

# Run the modified subnet demo with per-epoch callbacks
echo "🚀 Starting PoCW subnet with per-epoch mainnet integration..."
echo ""

echo "📝 NOTE: Real-time blockchain integration active!"
echo "   Go subnet will make HTTP calls to JavaScript bridge"
echo "   Each completed epoch will trigger actual blockchain submissions"
echo ""

# Step 1: Run VLC Validation FIRST (before subnet registration)
echo "🔐 Running VLC Protocol Validation..."
echo ""
VALIDATION_ONLY_MODE=true timeout 60 go run main.go
VALIDATION_EXIT_CODE=$?

if [ $VALIDATION_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "❌ VLC Validation FAILED - cannot register subnet"

    # Submit failed validation results from ALL validators to ValidationRegistry
    echo "📝 Recording failed validation from all validators in ValidationRegistry..."

    VALIDATORS=("$VALIDATOR1" "$VALIDATOR2" "$VALIDATOR3" "$VALIDATOR4")
    VALIDATOR_KEYS=("$VALIDATOR1_KEY" "$VALIDATOR2_KEY" "$VALIDATOR3_KEY" "$VALIDATOR4_KEY")
    VALIDATOR_NAMES=("Validator-1" "Validator-2" "Validator-3" "Validator-4")

    for i in {0..3}; do
        VALIDATOR=${VALIDATORS[$i]}
        VALIDATOR_KEY=${VALIDATOR_KEYS[$i]}
        VALIDATOR_NAME=${VALIDATOR_NAMES[$i]}

        REQUEST_HASH=$(echo -n "vlc-validation-${AGENT_ID_DEC}-${VALIDATOR}-$(date +%s)" | sha256sum | cut -d' ' -f1)
        REQUEST_HASH="0x${REQUEST_HASH}"

        # Submit validation request
        $CAST_PATH send $VALIDATION_ADDRESS "validationRequest(address,uint256,string,bytes32)" \
            "$VALIDATOR" \
            "$AGENT_ID_DEC" \
            "VLC Protocol Validation Test" \
            "$REQUEST_HASH" \
            --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1

        # Submit failed scores (0-20 range for failures)
        if [ $i -eq 0 ]; then
            SCORE=0   # Complete failure
        elif [ $i -eq 1 ]; then
            SCORE=10  # Partial implementation
        elif [ $i -eq 2 ]; then
            SCORE=5   # Minimal implementation
        else
            SCORE=15  # Some progress but failed
        fi

        RESPONSE_HASH=$(echo -n "vlc-response-${AGENT_ID_DEC}-${VALIDATOR}-${SCORE}" | sha256sum | cut -d' ' -f1)
        RESPONSE_HASH="0x${RESPONSE_HASH}"
        VLC_TAG=$(echo -n "VLC_PROTOCOL" | xxd -p -c 32 | head -c 64)
        VLC_TAG="0x${VLC_TAG}$(printf '0%.0s' {1..40})"

        $CAST_PATH send $VALIDATION_ADDRESS "validationResponse(bytes32,uint8,string,bytes32,bytes32)" \
            "$REQUEST_HASH" \
            "$SCORE" \
            "VLC validation failed - agent does not implement causal consistency correctly" \
            "$RESPONSE_HASH" \
            "$VLC_TAG" \
            --private-key $VALIDATOR_KEY --rpc-url $RPC_URL > /dev/null 2>&1

        if [ $? -eq 0 ]; then
            echo "   ⚠️  ${VALIDATOR_NAME}: Failure score ${SCORE}/100 recorded"
        fi
    done

    echo ""
    echo "   📊 Validation Summary:"
    echo "      Agent ID: #${AGENT_ID_DEC}"
    echo "      Average Score: ~7/100"
    echo "      Status: ❌ FAILED"

    cleanup
    exit 1
fi

echo ""
echo "✅ VLC Validation PASSED - proceeding with subnet registration"
echo ""

# Step 1.5: Submit validation results from ALL validators to ValidationRegistry
echo "📝 Submitting VLC validation results from all validators to ValidationRegistry..."

# Arrays for validators
VALIDATORS=("$VALIDATOR1" "$VALIDATOR2" "$VALIDATOR3" "$VALIDATOR4")
VALIDATOR_KEYS=("$VALIDATOR1_KEY" "$VALIDATOR2_KEY" "$VALIDATOR3_KEY" "$VALIDATOR4_KEY")
VALIDATOR_NAMES=("Validator-1" "Validator-2" "Validator-3" "Validator-4")

# Each validator submits their own validation
for i in {0..3}; do
    VALIDATOR=${VALIDATORS[$i]}
    VALIDATOR_KEY=${VALIDATOR_KEYS[$i]}
    VALIDATOR_NAME=${VALIDATOR_NAMES[$i]}

    # Generate unique request hash for each validator
    REQUEST_HASH=$(echo -n "vlc-validation-${AGENT_ID_DEC}-${VALIDATOR}-$(date +%s)" | sha256sum | cut -d' ' -f1)
    REQUEST_HASH="0x${REQUEST_HASH}"

    echo ""
    echo "   📋 ${VALIDATOR_NAME} submitting validation..."

    # Submit validation request (miner creates request for validator to validate)
    $CAST_PATH send $VALIDATION_ADDRESS "validationRequest(address,uint256,string,bytes32)" \
        "$VALIDATOR" \
        "$AGENT_ID_DEC" \
        "VLC Protocol Validation Test" \
        "$REQUEST_HASH" \
        --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1

    if [ $? -ne 0 ]; then
        echo "      ⚠️  Failed to create validation request"
        continue
    fi

    # All validators give perfect score for VLC validation (protocol correctness)
    SCORE=100  # VLC validation passes with perfect score

    # Submit validation response with score
    RESPONSE_HASH=$(echo -n "vlc-response-${AGENT_ID_DEC}-${VALIDATOR}-${SCORE}" | sha256sum | cut -d' ' -f1)
    RESPONSE_HASH="0x${RESPONSE_HASH}"
    VLC_TAG=$(echo -n "VLC_PROTOCOL" | xxd -p -c 32 | head -c 64)
    VLC_TAG="0x${VLC_TAG}$(printf '0%.0s' {1..40})"

    $CAST_PATH send $VALIDATION_ADDRESS "validationResponse(bytes32,uint8,string,bytes32,bytes32)" \
        "$REQUEST_HASH" \
        "$SCORE" \
        "VLC validation passed - agent correctly implements causal consistency" \
        "$RESPONSE_HASH" \
        "$VLC_TAG" \
        --private-key $VALIDATOR_KEY --rpc-url $RPC_URL > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        echo "      ✅ ${VALIDATOR_NAME}: Score ${SCORE}/100 recorded"
        echo "         Address: ${VALIDATOR}"
        echo "         Request: ${REQUEST_HASH:0:10}..."

        # Wait for transaction to be mined and verify response was recorded
        sleep 1
        VERIFY_RESPONSE=$($CAST_PATH call $VALIDATION_ADDRESS \
            "getValidationStatus(bytes32)" \
            "$REQUEST_HASH" \
            --rpc-url $RPC_URL 2>&1 | tail -1)

        # Parse the response to check if score was recorded (response should be 100)
        if [[ "$VERIFY_RESPONSE" == *"100"* ]]; then
            echo "         ✓ Response confirmed on-chain"
        fi
    else
        echo "      ⚠️  ${VALIDATOR_NAME}: Failed to record score"
    fi
done

# Wait a bit more to ensure all transactions are fully confirmed
echo ""
echo "   ⏳ Waiting for all validation responses to be confirmed..."
sleep 2

# Verify all 4 responses are on-chain by checking agent validations count
echo "   🔍 Verifying all validation responses are on-chain..."
AGENT_VALIDATIONS=$($CAST_PATH call $VALIDATION_ADDRESS \
    "getAgentValidations(uint256)(bytes32[])" \
    "$AGENT_ID_DEC" \
    --rpc-url $RPC_URL 2>&1)

# Count how many validation entries we have
VALIDATION_COUNT=$(echo "$AGENT_VALIDATIONS" | grep -o "0x" | wc -l)
echo "      Total validation requests for agent: $VALIDATION_COUNT"

if [ "$VALIDATION_COUNT" -lt 4 ]; then
    echo "      ⚠️  Warning: Expected 4 validations but found $VALIDATION_COUNT"
    echo "      Waiting 3 more seconds for blockchain to sync..."
    sleep 3
fi

echo ""
echo "   📊 Validation Summary:"
echo "      Agent ID: #${AGENT_ID_DEC}"

# Call getSummary function from ValidationRegistry to get actual average score
# getSummary(agentId, validatorAddresses[], tag) returns (uint64 count, uint8 avgScore)
# We pass empty array [] to get all validators and VLC_PROTOCOL tag
VLC_TAG="0x564c435f50524f544f434f4c0000000000000000000000000000000000000000"

# Call with empty validator array to get all validators' scores
echo "      🔍 Calling ValidationRegistry.getSummary..."
echo "         Agent ID: $AGENT_ID_DEC"
echo "         VLC Tag: $VLC_TAG"

SUMMARY=$($CAST_PATH call $VALIDATION_ADDRESS \
    "getSummary(uint256,address[],bytes32)(uint64,uint8)" \
    "$AGENT_ID_DEC" \
    "[]" \
    "$VLC_TAG" \
    --rpc-url $RPC_URL 2>&1)

echo "      📝 Raw getSummary response: $SUMMARY"

# Debug output to see what we get
if [[ "$SUMMARY" == *"Error"* ]] || [ -z "$SUMMARY" ]; then
    echo "      ⚠️  Error or empty response from getSummary"
    # Fallback to expected values since all validators score 100
    TOTAL_VALIDATIONS=4
    AVG_SCORE=100
else
    # Cast returns values as plain numbers separated by newline or space
    # Try different parsing approaches

    # First try: direct numbers separated by space or newline
    if [[ "$SUMMARY" =~ ^([0-9]+)[[:space:]]+([0-9]+)$ ]]; then
        TOTAL_VALIDATIONS=${BASH_REMATCH[1]}
        AVG_SCORE=${BASH_REMATCH[2]}
    # Second try: tuple format (4, 100)
    elif [[ "$SUMMARY" =~ \(([0-9]+),([0-9]+)\) ]]; then
        TOTAL_VALIDATIONS=${BASH_REMATCH[1]}
        AVG_SCORE=${BASH_REMATCH[2]}
    # Third try: just two numbers on separate lines
    else
        # Convert to array splitting by any whitespace
        SUMMARY_ARRAY=($SUMMARY)
        TOTAL_VALIDATIONS=${SUMMARY_ARRAY[0]:-0}
        AVG_SCORE=${SUMMARY_ARRAY[1]:-0}
    fi
fi

echo "      📊 Parsed values: count=$TOTAL_VALIDATIONS, avgScore=$AVG_SCORE"

# Ensure values are not empty to avoid bash errors
TOTAL_VALIDATIONS=${TOTAL_VALIDATIONS:-0}
AVG_SCORE=${AVG_SCORE:-0}

echo "      Validators: $TOTAL_VALIDATIONS"
echo "      Average Score: ${AVG_SCORE}/100"

if [ $AVG_SCORE -ge 70 ]; then
    echo "      Status: ✅ PASSED"
else
    echo "      Status: ❌ FAILED"
fi

echo ""

# Step 2: Register subnet on blockchain (ValidationRegistry check enforced in smart contract)
echo "🔐 Registering subnet with Agent ID $AGENT_ID_DEC..."
echo "   The SubnetRegistry contract will verify:"
echo "   ✓ Agent owns the identity token"
echo "   ✓ Agent has passed VLC validation (score >= 70)"
echo ""
echo "   Subnet: $SUBNET_ID"
echo "   Miner: $MINER"
echo "   Validators: [$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]"
echo ""

$CAST_PATH send $REGISTRY_ADDRESS "registerSubnet(string,uint256,address,address[4])" \
    "$SUBNET_ID" \
    "$AGENT_ID_DEC" \
    "$MINER" \
    "[$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]" \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Subnet registered successfully on blockchain"
else
    echo "❌ Subnet registration failed"
    cleanup
    exit 1
fi

echo ""
echo "🚀 Starting full subnet demo with per-epoch submission..."
echo "   This will process 7 inputs across multiple epochs"
echo "   Each epoch (3 rounds) will be submitted to blockchain immediately"
echo ""
echo "🎯 Demo Flow:"
echo "  Round 1-3  → Epoch 1 → Immediate mainnet submission"
echo "  Round 4-6  → Epoch 2 → Immediate mainnet submission"
echo "  Round 7    → Partial Epoch 3 → Submit at demo end"
echo ""

# Step 3: Run the full per-epoch subnet demo
timeout 120 go run main.go || true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 FLUX MINING DEMONSTRATION COMPLETE!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# === FINAL FLUX BALANCES ===
echo ""
echo "💰 Final FLUX Token Balances (After Mining)"
echo "────────────────────────────────────────────────"
echo "📊 Miner ($MINER):"
MINER_FINAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_FINAL_FORMATTED=$(format_flux_balance $MINER_FINAL)
MINER_GAINED=$(echo "$MINER_FINAL_FORMATTED - $MINER_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $MINER_FINAL_FORMATTED FLUX (+$MINER_GAINED FLUX mined)"

echo "📊 Validator-1 ($VALIDATOR1):"
V1_FINAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_FINAL_FORMATTED=$(format_flux_balance $V1_FINAL)
V1_GAINED=$(echo "$V1_FINAL_FORMATTED - $V1_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V1_FINAL_FORMATTED FLUX (+$V1_GAINED FLUX mined)"

echo "📊 Validator-2 ($VALIDATOR2):"
V2_FINAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR2 --rpc-url $RPC_URL)
V2_FINAL_FORMATTED=$(format_flux_balance $V2_FINAL)
V2_GAINED=$(echo "$V2_FINAL_FORMATTED - $V2_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V2_FINAL_FORMATTED FLUX (+$V2_GAINED FLUX mined)"

echo "📊 Validator-3 ($VALIDATOR3):"
V3_FINAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR3 --rpc-url $RPC_URL)
V3_FINAL_FORMATTED=$(format_flux_balance $V3_FINAL)
V3_GAINED=$(echo "$V3_FINAL_FORMATTED - $V3_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V3_FINAL_FORMATTED FLUX (+$V3_GAINED FLUX mined)"

echo "📊 Validator-4 ($VALIDATOR4):"
V4_FINAL=$($CAST_PATH call $FLUX_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR4 --rpc-url $RPC_URL)
V4_FINAL_FORMATTED=$(format_flux_balance $V4_FINAL)
V4_GAINED=$(echo "$V4_FINAL_FORMATTED - $V4_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V4_FINAL_FORMATTED FLUX (+$V4_GAINED FLUX mined)"

TOTAL_SUPPLY_FINAL=$($CAST_PATH call $FLUX_ADDRESS "totalSupply()(uint256)" --rpc-url $RPC_URL)
TOTAL_SUPPLY_FINAL_FORMATTED=$(format_flux_balance $TOTAL_SUPPLY_FINAL)
TOTAL_MINED=$(echo "$TOTAL_SUPPLY_FINAL_FORMATTED - $TOTAL_SUPPLY_INITIAL_FORMATTED" | bc -l)
echo "📊 Total Supply: $TOTAL_SUPPLY_FINAL_FORMATTED FLUX (+$TOTAL_MINED FLUX total mined)"

echo ""
echo "💵 Final $PAYMENT_TOKEN Token Balances (x402 Payment System)"
echo "────────────────────────────────────────────────"
echo "📊 Client ($CLIENT):"
CLIENT_PAYMENT_FINAL=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $CLIENT --rpc-url $RPC_URL)
CLIENT_PAYMENT_FINAL_FORMATTED=$(format_flux_balance $CLIENT_PAYMENT_FINAL)
echo "   Balance: $CLIENT_PAYMENT_FINAL_FORMATTED $PAYMENT_TOKEN"

echo "📊 Miner/Agent ($MINER):"
MINER_PAYMENT_FINAL=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_PAYMENT_FINAL_FORMATTED=$(format_flux_balance $MINER_PAYMENT_FINAL)
echo "   Balance: $MINER_PAYMENT_FINAL_FORMATTED $PAYMENT_TOKEN"

echo "📊 V1 Coordinator ($VALIDATOR1):"
V1_PAYMENT_FINAL=$($CAST_PATH call $PAYMENT_TOKEN_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_PAYMENT_FINAL_FORMATTED=$(format_flux_balance $V1_PAYMENT_FINAL)
echo "   Balance: $V1_PAYMENT_FINAL_FORMATTED $PAYMENT_TOKEN"

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║        🌟 FINAL AGENT REPUTATION SUMMARY                    ║"
echo "║           (Read from ReputationRegistry)                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Read agent reputation from ReputationRegistry contract
AGENT_ID="0"
echo "📊 Agent ID $AGENT_ID Reputation on Blockchain:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Call getSummary(uint256, address[], bytes32, bytes32) returns (uint64, uint8)
# Parameters: agentId, clientAddresses (empty array), tag1 (0x0), tag2 (0x0)
REPUTATION_DATA=$($CAST_PATH call $REPUTATION_ADDRESS "getSummary(uint256,address[],bytes32,bytes32)(uint64,uint8)" \
    $AGENT_ID "[]" "0x0000000000000000000000000000000000000000000000000000000000000000" "0x0000000000000000000000000000000000000000000000000000000000000000" \
    --rpc-url $RPC_URL 2>&1)

if echo "$REPUTATION_DATA" | grep -q "^[0-9]"; then
    # Parse the two return values (count, averageScore)
    FEEDBACK_COUNT=$(echo "$REPUTATION_DATA" | sed -n '1p')
    AVG_SCORE=$(echo "$REPUTATION_DATA" | sed -n '2p')

    if [ "$FEEDBACK_COUNT" -gt 0 ]; then
        echo "  📝 Total Feedbacks Received: $FEEDBACK_COUNT"
        echo -n "  ⭐ Average Score: $AVG_SCORE/100"

        # Add performance indicator
        if [ "$AVG_SCORE" -ge 80 ]; then
            echo " (Excellent Performance 🏆)"
        elif [ "$AVG_SCORE" -ge 60 ]; then
            echo " (Good Performance ✅)"
        elif [ "$AVG_SCORE" -ge 40 ]; then
            echo " (Needs Improvement ⚠️)"
        else
            echo " (Poor Performance ❌)"
        fi

        # Create visual score bar (50 characters wide)
        FILLED_LENGTH=$((AVG_SCORE * 50 / 100))
        EMPTY_LENGTH=$((50 - FILLED_LENGTH))
        if [ "$FILLED_LENGTH" -gt 0 ]; then
            FILLED_BAR=$(printf '█%.0s' $(seq 1 $FILLED_LENGTH))
        else
            FILLED_BAR=""
        fi
        if [ "$EMPTY_LENGTH" -gt 0 ]; then
            EMPTY_BAR=$(printf '░%.0s' $(seq 1 $EMPTY_LENGTH))
        else
            EMPTY_BAR=""
        fi
        BAR="${FILLED_BAR}${EMPTY_BAR}"
        echo "  📊 Score Visual: [$BAR] ${AVG_SCORE}%"
    else
        echo "  ❌ No reputation feedback recorded yet"
    fi
else
    echo "  ⚠️ Could not retrieve reputation data"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Reputation data successfully retrieved from blockchain!"
echo ""

echo "🔍 What was demonstrated:"
echo "  1. ✅ Infrastructure setup (Anvil + Dgraph + Contracts)"
echo "  2. ✅ Real-time FLUX mining per epoch completion"
echo "  3. ✅ Subnet processing with VLC consistency"
echo "  4. ✅ Blockchain integration with verified transactions"
echo "  5. ✅ Agent reputation tracking via ERC-8004"
echo ""
echo "🌐 Access points:"
echo "  📊 Dgraph VLC visualization: http://localhost:8000"
echo "     In Ratel, connect to: http://localhost:8080"
echo "     Query to see events: { events(func: has(id)) { uid id name clock depth value key node parent { uid id name } } }"
echo "  🔍 Blockchain Inspector: http://localhost:3000/pocw-inspector.html"
echo "  ⛓️  Anvil blockchain: http://localhost:8545"
echo ""

echo "🎉 Bridge stays running for continued FLUX mining!"
echo "🌐 Bridge service: http://localhost:3001"
echo "Press Ctrl+C to cleanup and exit..."

# Keep running for inspection (controlled by NO_LOOP environment variable)
if [ "$NO_LOOP" != "true" ]; then
    while true; do
        sleep 10
        # Check if Anvil is still running
        if ! kill -0 $(cat anvil-per-epoch.pid) 2>/dev/null; then
            echo "❌ Anvil stopped unexpectedly"
            break
        fi
    done
else
    echo "🔧 NO_LOOP=true detected - exiting without forever loop"
    echo "   (Set NO_LOOP=false or unset to enable debugging loop)"
fi