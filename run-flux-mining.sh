#!/bin/bash

# PoCW FLUX Mining Script with ERC-8004 Identity Integration
# This script runs the complete PoCW system with FLUX token mining:
# - ERC-8004 Identity Registry for miner authentication
# - Miners must have Agent ID to register subnets
# - Real-time epoch submission where each completed epoch (3 rounds)
#   triggers immediate mainnet submission and FLUX mining.

echo "💰 PoCW FLUX MINING SYSTEM WITH ERC-8004 IDENTITY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Architecture: FLUX mining with ERC-8004 identity verification"
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

# === Deploy ERC-8004 Identity Registry FIRST ===
echo "🆔 Deploying ERC-8004 IdentityRegistry..."
IDENTITY_RESULT=$($FORGE_PATH create contracts/8004/IdentityRegistry.sol:IdentityRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
IDENTITY_ADDRESS=$(echo "$IDENTITY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "   IdentityRegistry: $IDENTITY_ADDRESS"

# === Deploy AIUSD Token for x402 Payments ===
echo "💵 Deploying AIUSD Token (x402 payment stablecoin)..."
AIUSD_RESULT=$($FORGE_PATH create contracts/AIUSD.sol:AIUSD \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
AIUSD_ADDRESS=$(echo "$AIUSD_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

# Debug: Show if AIUSD deployment failed
if [ -z "$AIUSD_ADDRESS" ]; then
    echo "   ❌ ERROR: AIUSD Token deployment failed!"
    echo "$AIUSD_RESULT" | head -20
fi

echo "   AIUSD Token: $AIUSD_ADDRESS"

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

# Deploy contracts
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

# Ensure AIUSD_ADDRESS is set before deploying escrow
if [ -z "$AIUSD_ADDRESS" ]; then
    echo "   ❌ ERROR: AIUSD_ADDRESS is not set! Cannot deploy escrow."
    ESCROW_ADDRESS=""
else
    # IMPORTANT: --private-key, --rpc-url, and --broadcast must come BEFORE --constructor-args
    # Otherwise forge parses them as constructor arguments!
    ESCROW_RESULT=$($FORGE_PATH create contracts/x402PaymentEscrow.sol:x402PaymentEscrow \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast \
        --constructor-args "$AIUSD_ADDRESS" 2>&1)
    ESCROW_ADDRESS=$(echo "$ESCROW_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
fi

echo "   x402PaymentEscrow: $ESCROW_ADDRESS"

# Initialize contracts (SubnetRegistry needs both HETU and Identity addresses)
echo "Initializing contracts..."
timeout 3 $CAST_PATH send $REGISTRY_ADDRESS "initialize(address,address)" $HETU_ADDRESS $IDENTITY_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

timeout 3 $CAST_PATH send $VERIFIER_ADDRESS "initialize(address,address)" $FLUX_ADDRESS $REGISTRY_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

timeout 3 $CAST_PATH send $FLUX_ADDRESS "setPoCWVerifier(address)" $VERIFIER_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true

# Authorize V1 (Validator1) as coordinator in x402PaymentEscrow
echo "Authorizing V1 as x402 payment coordinator..."
timeout 3 $CAST_PATH send $ESCROW_ADDRESS "authorizeCoordinator(address)" $VALIDATOR1 \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL 2>&1
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

# Bootstrap client with HETU and AIUSD for x402 payments
echo "💵 Bootstrapping client with ETH, AIUSD and HETU..."
# First, send some ETH to client for gas fees
echo "   Sending 10 ETH to client..."
timeout 3 $CAST_PATH send $CLIENT --value 10ether \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ Client funded with 10 ETH"
else
    echo "   ⚠️  Failed to send ETH to client"
fi
timeout 3 $CAST_PATH send $HETU_ADDRESS "transfer(address,uint256)" $CLIENT $($CAST_PATH --to-wei 1000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
timeout 3 $CAST_PATH send $AIUSD_ADDRESS "mint(address,uint256)" $CLIENT $($CAST_PATH --to-wei 1000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
echo "   Client ($CLIENT) setup complete:"
echo "   - ETH for gas (see above)"
echo "   - 1000 HETU tokens"
echo "   - 1000 AIUSD tokens for task payments"

echo "💵 Bootstrapping V1 Coordinator with AIUSD for demo payments..."
timeout 3 $CAST_PATH send $AIUSD_ADDRESS "mint(address,uint256)" $VALIDATOR1 $($CAST_PATH --to-wei 100) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1 || true
echo "   V1 Coordinator ($VALIDATOR1) bootstrapped with:"
echo "   - 100 AIUSD tokens for demo payment distribution"

echo "🔓 Approving escrow to spend client's AIUSD..."
# Client approves escrow contract to spend AIUSD for payments (unlimited approval)
# Client is VALIDATOR2 (account #2): 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
CLIENT_KEY="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
timeout 3 $CAST_PATH send $AIUSD_ADDRESS "approve(address,uint256)" $ESCROW_ADDRESS "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" \
    --private-key $CLIENT_KEY --rpc-url $RPC_URL 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ Client approved escrow to spend AIUSD"
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

# Register subnet WITH AGENT ID
SUBNET_ID="per-epoch-subnet-001"
echo "🔐 Registering subnet with Agent ID $AGENT_ID_DEC..."
echo "   Subnet: $SUBNET_ID"
echo "   Miner: $MINER"
echo "   Validators: [$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]"

# Register subnet (without hiding errors)
$CAST_PATH send $REGISTRY_ADDRESS "registerSubnet(string,uint256,address,address[4])" \
    "$SUBNET_ID" \
    "$AGENT_ID_DEC" \
    "$MINER" \
    "[$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]" \
    --private-key $MINER_KEY --rpc-url $RPC_URL

if [ $? -eq 0 ]; then
    echo "✅ Subnet registered successfully"
else
    echo "❌ Subnet registration failed"
    exit 1
fi

# Generate contract addresses JSON for bridge and inspector (name -> address format)
cat > contract_addresses.json << EOF
{
  "IdentityRegistry": "$IDENTITY_ADDRESS",
  "HETUToken": "$HETU_ADDRESS",
  "FLUXToken": "$FLUX_ADDRESS",
  "SubnetRegistry": "$REGISTRY_ADDRESS",
  "PoCWVerifier": "$VERIFIER_ADDRESS",
  "AIUSD": "$AIUSD_ADDRESS",
  "x402PaymentEscrow": "$ESCROW_ADDRESS",
  "Client": "$CLIENT",
  "Agent": "$MINER",
  "V1Coordinator": "$VALIDATOR1"
}
EOF

echo ""
echo "📄 Contract addresses saved to contract_addresses.json"
echo "   View in inspector at: http://localhost:3000/pocw-inspector.html"

echo "✅ Mainnet contracts deployed and configured with ERC-8004 Identity + x402 Payments"
echo "   🆔 Identity Registry: $IDENTITY_ADDRESS"
echo "   🆔 Miner Agent ID: $AGENT_ID_DEC"
echo "   💰 HETU Token: $HETU_ADDRESS"
echo "   ⚡ FLUX Token: $FLUX_ADDRESS"
echo "   📋 PoCW Verifier: $VERIFIER_ADDRESS"
echo "   💵 AIUSD Token: $AIUSD_ADDRESS"
echo "   🔒 x402 Escrow: $ESCROW_ADDRESS"
echo "   👤 Client Address: $CLIENT"

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

# === AIUSD TOKEN BALANCES (x402 Payment System) ===
echo "💵 AIUSD Token Balances (x402 Payment System)"
echo "────────────────────────────────────────────────"
echo "📊 Client ($CLIENT):"
CLIENT_AIUSD=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $CLIENT --rpc-url $RPC_URL)
CLIENT_AIUSD_FORMATTED=$(format_flux_balance $CLIENT_AIUSD)
echo "   Balance: $CLIENT_AIUSD_FORMATTED AIUSD"

echo "📊 Miner/Agent ($MINER):"
MINER_AIUSD=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_AIUSD_FORMATTED=$(format_flux_balance $MINER_AIUSD)
echo "   Balance: $MINER_AIUSD_FORMATTED AIUSD"

echo "📊 V1 Coordinator ($VALIDATOR1):"
V1_AIUSD=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_AIUSD_FORMATTED=$(format_flux_balance $V1_AIUSD)
echo "   Balance: $V1_AIUSD_FORMATTED AIUSD"
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

# Run the per-epoch subnet demo with real HTTP bridge integration
echo "🚀 Starting Go subnet with per-epoch submission..."
echo "   This will process 7 inputs across multiple epochs"
echo "   Each epoch (3 rounds) will be submitted to blockchain immediately"
echo ""
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
echo "💵 Final AIUSD Token Balances (x402 Payment System)"
echo "────────────────────────────────────────────────"
echo "📊 Client ($CLIENT):"
CLIENT_AIUSD_FINAL=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $CLIENT --rpc-url $RPC_URL)
CLIENT_AIUSD_FINAL_FORMATTED=$(format_flux_balance $CLIENT_AIUSD_FINAL)
echo "   Balance: $CLIENT_AIUSD_FINAL_FORMATTED AIUSD"

echo "📊 Miner/Agent ($MINER):"
MINER_AIUSD_FINAL=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_AIUSD_FINAL_FORMATTED=$(format_flux_balance $MINER_AIUSD_FINAL)
echo "   Balance: $MINER_AIUSD_FINAL_FORMATTED AIUSD"

echo "📊 V1 Coordinator ($VALIDATOR1):"
V1_AIUSD_FINAL=$($CAST_PATH call $AIUSD_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_AIUSD_FINAL_FORMATTED=$(format_flux_balance $V1_AIUSD_FINAL)
echo "   Balance: $V1_AIUSD_FINAL_FORMATTED AIUSD"

echo ""
echo "🔍 What was demonstrated:"
echo "  1. ✅ Infrastructure setup (Anvil + Dgraph + Contracts)"
echo "  2. ✅ Real-time FLUX mining per epoch completion"  
echo "  3. ✅ Subnet processing with VLC consistency"
echo "  4. ✅ Blockchain integration with verified transactions"
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