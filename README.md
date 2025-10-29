# FLUX Mining with ERC-8004 Identity & x402 Payment Escrow

A trustless AI agent infrastructure combining **ERC-8004 decentralized identity**, **x402 payment protocol**, and **Proof-of-Causal-Work consensus** to mine soulbound FLUX tokens through verified intelligence work. Features cryptographic agent verification, smart contract escrow payments with on-chain verification, Byzantine Fault Tolerant consensus, and Vector Logical Clock causal ordering for permissionless AI marketplaces.

## Overview

This system demonstrates a **permissionless AI marketplace** where agents earn soulbound FLUX tokens through verified intelligence work and receive AIUSD payments via trustless escrow. Combines three key innovations:

- **🆔 ERC-8004 Identity**: Decentralized agent registry with cryptographic verification and NFT-based identity
- **💳 x402 Payment Protocol**: Trustless escrow with on-chain payment verification - agents verify funds before processing
- **⛏️ Proof-of-Causal-Work**: Byzantine Fault Tolerant consensus using Vector Logical Clocks for causal ordering

The **FLUX token** represents verifiable intelligence contributions and is non-transferable (soulbound) but redeemable, while **AIUSD stablecoin** enables instant, cryptographically-secured payments between clients and agents without trusted intermediaries.

### Key Features

- 🧠 **Intelligence Mining**: Earn FLUX tokens through actual AI task completion
- 💳 **x402 Escrow Payments**: Trustless AIUSD payments with BFT consensus-based release
- 🔗 **Vector Logical Clocks**: Causal ordering of distributed consensus events
- 🏛️ **Byzantine Fault Tolerant**: 4-validator consensus with quality assessment
- 💎 **Soulbound Tokens**: Non-transferable but redeemable FLUX tokens
- 🆔 **ERC-8004 Identity**: Trustless agent identity with NFT-based verification
- 📊 **Real-time Visualization**: VLC event graph via Dgraph
- ⛓️ **Blockchain Integration**: Smart contracts on Anvil/Ethereum
- 🔒 **Smart Contract Escrow**: Automatic payment release/refund based on validator consensus

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────────┐
│   Subnet        │    │   Bridge         │    │   Blockchain            │
│                 │    │                  │    │                         │
│ Miner(8004) +   │◄──►│ HTTP Server      │◄──►│ Smart Contracts         │
│ 4 Validators    │    │ (Port 3001)      │    │ (Anvil)                 │
│                 │    │                  │    │                         │
│ VLC Consensus   │    │ Per-epoch        │    │ • FLUX Mining            │
└─────────────────┘    │ Submission       │    │ • Token Rewards         │
         │             └──────────────────┘    │ • ERC-8004 Identity     │
         ▼                                     │ • x402 Escrow Payments  │
┌─────────────────┐                            └─────────────────────────┘
│   Dgraph        │                                        │
│   VLC Graph     │              ┌─────────────────────────┘
│   Visualization │              ▼
└─────────────────┘    ┌──────────────────────┐
                       │  x402 Payment Escrow │
┌─────────────────┐    │                      │
│   Client        │───▶│ • AIUSD Deposits     │───▶┌─────────────┐
│                 │    │ • BFT Consensus      │    │   Agent     │
│ • Initiates     │    │ • Release/Refund     │    │             │
│   Tasks         │◄───│ • Trustless Flow     │◄───│ • Performs  │
│ • Funds AIUSD   │    └──────────────────────┘    │   Work      │
└─────────────────┘                                 │ • Gets Paid │
                                                    └─────────────┘
```

## Two Execution Modes

### 1. Subnet-Only Mode 🔹

**Purpose**: Pure subnet consensus demonstration with VLC visualization (no blockchain integration)

**What it does**:
- ✅ Runs distributed consensus with 1 miner(8004 Agent) + 4 validators
- ✅ Processes 7 rounds of AI tasks with quality assessment  
- ✅ Generates VLC event graph for causal analysis
- ✅ Provides interactive exploration of consensus behavior
- ❌ No blockchain integration or FLUX token mining

**Run Command**:
```bash
./run-subnet-only.sh
```

**Access Points**:
- 📊 VLC Graph: `http://localhost:8000` (Dgraph Ratel UI)
- 📋 Event Query: `http://localhost:8080/graphql`

### 2. FLUX Mining Mode 💰

**Purpose**: Complete PoCW system with FLUX mining, x402 escrow payments, and ERC-8004 identity

**What it does**:
- ✅ Everything from subnet-only mode PLUS:
- ✅ Deploys smart contracts (FLUX, AIUSD, x402PaymentEscrow, ERC-8004 Identity, etc.)
- ✅ Real-time FLUX mining per epoch (every 3 rounds)
- ✅ **x402 Trustless Escrow**: AIUSD payments with BFT consensus-based release/refund
- ✅ **ERC-8004 Agent Identity**: NFT-based trustless agent verification
- ✅ Blockchain transactions with verified rewards
- ✅ Bridge service for epoch submission
- ✅ Complete FLUX and AIUSD balance tracking
- ✅ Demonstrates 5 successful payments + 2 refunds via escrow

**Run Command**:
```bash
./run-flux-mining.sh
```

**Access Points**:
- 📊 VLC Graph: `http://localhost:8000`
- 🔍 Blockchain Inspector: `http://localhost:3000/pocw-inspector.html`
- ⛓️ Blockchain RPC: `http://localhost:8545`
- 🌐 Bridge API: `http://localhost:3001`

**What You'll See**:
- **x402 Payment Requests**: Agent generates payment requests with task details
  ```
  📋 Agent sends x402 Payment Request to Client:
     Task ID: req-subnet-001-1
     Amount: 10000000000000000000 wei (10 AIUSD)
     Escrow Contract: 0x0165878A594ca255338adfa4d48449f69242Eb8F
  ```
- **Payment Deposits**: Client deposits AIUSD to escrow (10 AIUSD per task)
- **Agent Verification**: Agent verifies payment on-chain before processing
  ```
  ✅ Payment verified for task req-subnet-001-1:
     Amount: 10.00 AIUSD (locked in escrow)
     Agent: 0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc
  🔐 Miner: Payment verified on-chain - proceeding with task
  ```
- **FLUX Mining**: 400 FLUX to miner, 80 FLUX to validators per epoch
- **BFT Consensus**: Validators decide payment release or refund
- **Payment Outcomes**: 5 payments released (50 AIUSD), 2 refunded (20 AIUSD)
- **Trustless Operation**: Complete audit trail of all payments on-chain

## Prerequisites

### Required Software
```bash
# Install Foundry (for Anvil and Cast)
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Install Go >= 1.21
# Install Node.js >= 22
# Install Docker (for Dgraph)
# Install bc (for calculations)
sudo apt install bc
```

### System Requirements
- **Ports**: 3000, 3001, 8000, 8080, 8545, 9080 (must be available)
- **Docker**: Required for Dgraph container
- **Sudo Access**: Required for Docker operations

## Quick Start

### Option 1: Subnet-Only Demo
```bash
# Clean run of subnet consensus with VLC visualization
sudo ./run-subnet-only.sh

# Access VLC graph at http://localhost:8000
# Press Ctrl+C when done exploring
```

### Option 2: Full FLUX Mining Demo  
```bash
# Complete PoCW system with blockchain integration
sudo ./run-flux-mining.sh

# Watch FLUX tokens being mined in real-time
# Explore blockchain inspector at http://localhost:3000/pocw-inspector.html
# Bridge stays active for continued mining
# Press Ctrl+C when done
```

## Smart Contracts

| Contract | Purpose | Features |
|----------|---------|----------|
| **FLUXToken** | Soulbound intelligence tokens | Non-transferable, 21M max supply |
| **HETUToken** | Staking for subnet registration | ERC20, 1M total supply |
| **AIUSD** | AI services stablecoin | ERC20, USD-pegged, for x402 payments |
| **x402PaymentEscrow** | Trustless payment escrow | BFT consensus-based release/refund, reentrancy protection |
| **SubnetRegistry** | Manages subnet participants with identity | ERC-8004 identity verification, deposit requirements |
| **PoCWVerifier** | Consensus verification & mining | Per-epoch FLUX distribution, validator authorization |
| **IdentityRegistry** | ERC-8004 Trustless Agents identity | NFT-based agent IDs, ownership verification |
| **ValidationRegistry** | Agent validation requests | Quality assessments, reputation tracking |
| **ReputationRegistry** | Agent reputation system | Score tracking, signed attestations |

## ERC-8004 Identity Integration

This system implements the **ERC-8004 Trustless Agents** standard, providing verifiable identity for AI agents participating in the PoCW network.

### Identity System

**🆔 Agent Identity NFTs**
- Each AI agent is represented by a unique NFT token ID
- Agent ownership is cryptographically verified on-chain
- Agents must prove ownership to participate in subnet mining
- Identity registry tracks agent metadata and ownership history

**🔐 Subnet Registration with Identity**
```
Registration Requirements:
1. Agent owner mints identity NFT from IdentityRegistry
2. Owner registers subnet with their agent ID
3. Smart contract verifies: identityRegistry.ownerOf(agentId) == minerAddress
4. Only verified agents can participate in consensus and mining
```

**✅ Benefits**
- **Trustless Verification**: No central authority needed for agent authentication
- **Ownership Proof**: Cryptographic proof of agent control
- **Reputation Tracking**: Identity persists across epochs and subnets
- **Sybil Resistance**: One agent ID per subnet, preventing duplicate participation

**📊 Blockchain Inspector**

The web inspector at `http://localhost:3000/pocw-inspector.html` provides:
- View all registered agent identities
- Check agent ID ownership and metadata
- Track which agents are active in subnets
- Monitor identity-verified FLUX mining rewards

## 💳 x402 Payment System with Smart Escrow

The system now integrates **x402 protocol** for trustless, escrow-based AIUSD stablecoin payments between clients and AI agents, leveraging permissionless validator consensus.

### Revolutionary Payment Architecture

**🔒 Trustless Escrow-Based Payments**
```
Traditional AI Services:  Client → Platform → Agent (Platform controls funds)
x402 Escrow System:      Client → Smart Contract → Agent (Code controls funds)
```

The x402 payment system eliminates trusted intermediaries by using smart contract escrow combined with BFT validator consensus for payment release decisions.

### Payment Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    x402 Payment Escrow Lifecycle                     │
└─────────────────────────────────────────────────────────────────────┘

1️⃣ PAYMENT REQUEST PHASE
   Client                    Escrow Contract              Agent
     │                              │                        │
     ├──── Task Request ───────────────────────────────────▶│
     │                              │                        │
     │ ◄──── x402 Payment Request ──────────────────────────┤
     │         {taskId, amount,     │                        │
     │          escrow, agent,      │                        │
     │          token, deadline}    │                        │

2️⃣ DEPOSIT PHASE
     │                              │                        │
     ├──── approve(AIUSD) ─────────▶│                        │
     ├──── depositPayment() ────────▶│                        │
     │                              │ ✓ Funds locked         │
     │                              │                        │

3️⃣ VERIFICATION & PROCESSING PHASE
     │                              │                        │
     │                              │ ◄─ Agent Verifies ─────┤
     │                              │    (blockchain query)  │
     │                              │    ✓ Payment confirmed │
     │                         [Validator Network]           │
     │                              │                        │
     │                         │ ─── Forward Task ──────────▶│
     │                         │ ◄── Agent Processes ────────┤
     │                         │   (generates output/result) │

4️⃣ CONSENSUS PHASE
     │                              │                        │
     │                         [Validator Network]           │
     │                         Quality > 0.5? ────▶ BFT Vote │
     │ ◄──── Result Preview ────────┤                        │
     ├──── User Accept/Reject ──────▶│                        │

5️⃣ FINALIZATION PHASE
     │                              │                        │
     │    If Quality > 0.5 AND User Accepts:                 │
     │                              ├─ releasePayment() ─────▶│
     │                              │  ✓ Agent receives AIUSD │
     │                              │                        │
     │    If Quality ≤ 0.5 OR User Rejects:                  │
     ├◄──── refundPayment() ────────┤                        │
     │  ✓ Client receives refund    │                        │
```

### Smart Contract Escrow System

**🔐 x402PaymentEscrow Contract**

The escrow contract manages the entire payment lifecycle with cryptographic guarantees:

**x402 Payment Request (Agent → Client):**
```json
{
  "taskId": "req-subnet-001-1",
  "amount": "10000000000000000000",  // 10 AIUSD in wei
  "asset": {
    "symbol": "AIUSD",
    "contract": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    "decimals": 18
  },
  "escrow": {
    "contract": "0x0165878A594ca255338adfa4d48449f69242Eb8F",
    "timeout": 60  // seconds
  },
  "agent": {
    "address": "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
    "agentId": "0"
  }
}
```

**Key Functions:**
```solidity
// Public view function - Agent verifies payment before processing
mapping(bytes32 => TaskPayment) public payments;

// Client deposits AIUSD for a task (via coordinator)
function depositPayment(
    bytes32 taskId,
    address client,
    address agent,
    uint256 amount,
    uint256 deadline
) external onlyCoordinator

// Release payment to agent after consensus approval
function releasePayment(bytes32 taskId) external onlyCoordinator

// Refund payment to client if rejected
function refundPayment(bytes32 taskId) external onlyCoordinator
```

**Security Features:**
- ✅ **Payment Verification**: Agent queries blockchain for cryptographic proof before processing
- ✅ **Trustless Operation**: Agent doesn't trust validator - verifies payment on-chain
- ✅ **Reentrancy Protection**: Guards against reentrancy attacks
- ✅ **Authorization Control**: Only authorized validators can finalize payments
- ✅ **Payment States**: Tracks NONE → DEPOSITED → RELEASED/REFUNDED
- ✅ **Deadline Enforcement**: Time-based payment safety with agent verification
- ✅ **Event Emission**: Complete audit trail of all payment operations

### Permissionless Validator Consensus

**🏛️ BFT-Based Payment Decisions**

Payment finalization is determined by the same Byzantine Fault Tolerant consensus used for task validation:

**Validator Network:**
- 4 independent validator nodes
- 0.25 voting weight each (total = 1.0)
- Tolerates up to 1 Byzantine (malicious) validator
- Quality threshold: Tasks must score > 0.5 to qualify for payment

**Consensus Decision Logic:**
```go
if qualityScore > 0.5 && userAccepted {
    // ✅ Release payment from escrow to agent
    releasePayment(taskId)
} else {
    // ↩️ Refund payment from escrow to client
    refundPayment(taskId)
}
```

**Why This Works:**
- **Objective Quality Metrics**: Validators assess task completion quality independently
- **User Final Authority**: Client has ultimate veto power even if validators approve
- **Permissionless Operation**: Any validator can join with stake, no platform approval needed
- **Economic Alignment**: Validators earn rewards for honest consensus participation

### Payment Token: AIUSD Stablecoin

**💵 AIUSD - AI Services Stablecoin**

The system uses AIUSD, an ERC-20 stablecoin specifically designed for AI service payments:

**Properties:**
- Pegged to USD for price stability
- Standard ERC-20 transferability
- Minted for testing/demonstration purposes
- Used for all client-to-agent payments

**Client Payment Workflow:**
```bash
1. Client receives AIUSD tokens (1000 AIUSD in demo)
2. Client sends task request to agent
3. Agent generates x402 payment request with details:
   - Task ID, amount (10 AIUSD), escrow address, agent address, deadline
4. Client approves escrow contract to spend AIUSD
5. Client deposits payment to escrow contract
6. Agent verifies payment on-chain before processing
   - Queries blockchain: payments[taskId].status == DEPOSITED
   - Verifies: correct agent, sufficient amount, deadline valid
7. Agent processes task only after payment verification
8. AIUSD locked in escrow until consensus decision (release/refund)
```

### Demonstrated Payment Scenarios

The demo processes **7 payment scenarios** showing the full range of outcomes:

**✅ Successful Payments (5 tasks)**
```
Task 1, 2, 3, 5, 7:
- Quality Score: > 0.5
- User Feedback: Accepted
- Result: 10 AIUSD released to agent per task
- Total: 50 AIUSD paid to agent
```

**↩️ Refunded Payments (2 tasks)**
```
Task 4:
- Quality Score: < 0.5 (validator rejection)
- Result: 10 AIUSD refunded to client

Task 6:
- Quality Score: > 0.5 (validators approved)
- User Feedback: Rejected
- Result: 10 AIUSD refunded to client
```

**Final Balances:**
```
Client:  1000 → 950 AIUSD (-50 paid, +20 refunded)
Agent:   0 → 50 AIUSD (5 successful tasks)
```

### Economic Security Model

**🛡️ Multi-Layer Protection**

**For Clients:**
- ✅ Funds locked in escrow, not controlled by agent
- ✅ Automatic refund if quality too low
- ✅ Veto power even if validators approve
- ✅ Time-based deadline protection

**For Agents:**
- ✅ Guaranteed payment for quality work
- ✅ Objective quality assessment by validators
- ✅ Protection against malicious client rejection via validator approval requirement
- ✅ Cryptographic proof of task completion

**For the Network:**
- ✅ No trusted intermediary needed
- ✅ Validators earn rewards for honest consensus
- ✅ Sybil-resistant through ERC-8004 identity
- ✅ Complete payment audit trail on-chain

### Smart Contract Addresses

After deployment, the system creates:

```json
{
  "AIUSD": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
  "x402PaymentEscrow": "0x0165878A594ca255338adfa4d48449f69242Eb8F",
  "IdentityRegistry": "0x5FbDB2315678afecb367f032d93F642f64180aa3"
}
```

### Integration with FLUX Mining

The x402 payment system operates **in parallel** with FLUX token mining:

**Dual Reward Structure:**
```
Successful Task Completion:
  ├─ FLUX Tokens: Mined for verifiable intelligence work (soulbound)
  └─ AIUSD Payment: Released from escrow for service rendered (transferable)
```

**Why Both?**
- **FLUX**: Represents reputation and long-term capability (non-transferable)
- **AIUSD**: Provides immediate economic compensation (transferable stablecoin)
- **Combined**: Creates sustainable economic model for AI agents

This dual-token model ensures agents build reputation (FLUX) while earning liquid compensation (AIUSD), creating the world's first **merit-based AI economy** with trustless payments.

## Expected Output

### Subnet-Only Mode
```
🔹 PoCW SUBNET-ONLY DEMONSTRATION
Architecture: Pure subnet consensus with VLC graph visualization

✅ 7 rounds processed with Byzantine consensus
✅ VLC events committed to Dgraph
🔄 Interactive mode - explore at http://localhost:8000
```

### FLUX Mining Mode
```
💰 PoCW FLUX MINING SYSTEM WITH x402 PAYMENTS
Architecture: Complete FLUX mining with blockchain integration + escrow payments

💰 Initial FLUX Token Balances (Before Mining)
📊 Miner: 0.000000 FLUX
📊 Validator-1: 0.000000 FLUX
...

💵 Initial AIUSD Token Balances (x402 Payment System)
📊 Client: 1000.000000 AIUSD
📊 Agent: 0 AIUSD
📊 Escrow: 0 AIUSD

[CONSENSUS & MINING HAPPENS WITH ESCROW PAYMENTS]

💰 Final FLUX Token Balances (After Mining)
📊 Miner: 400.000000 FLUX (+400.000000 FLUX mined)
📊 Validator-1: 80.000000 FLUX (+80.000000 FLUX mined)
📊 Total Supply: 480.000000 FLUX (+480.000000 FLUX total mined)

💵 Final AIUSD Token Balances (x402 Payment System)
📊 Client: 950.000000 AIUSD (-50 paid, +20 refunded from escrow)
📊 Agent: 50.000000 AIUSD (5 tasks × 10 AIUSD released from escrow)
📊 Demonstrated: 5 successful payments + 2 refunds via BFT consensus

🎉 Bridge stays running for continued FLUX mining!
✨ Escrow payment system active for trustless client-agent transactions!
```

## Key Concepts

### Epochs & Rounds
- **Round**: Single AI task with miner output + validator consensus
- **Epoch**: 3 consecutive rounds
- **Mining Trigger**: Each completed epoch triggers FLUX token mining

### VLC (Vector Logical Clocks)
- Ensures causal ordering of distributed events
- Tracks happened-before relationships
- Prevents Byzantine inconsistencies
- Visualized as directed graphs in Dgraph

### FLUX Token Economics
- **Mining Rate**: Based on successful task completion
- **Distribution**: 80% to miner, 20% split among validators
- **Soulbound**: Cannot be transferred, but can be redeemed
- **Max Supply**: 21 million FLUX (like Bitcoin)

## Troubleshooting

### Common Issues

**Port Conflicts**:
```bash
# Check what's using required ports
netstat -tlnp | grep -E ":(3000|3001|8000|8080|8545|9080)"
# Kill conflicting processes if needed
```

**Docker Permissions**:
```bash
# Run the scripts with sudo to handle Docker operations
sudo ./run-subnet-only.sh
sudo ./run-flux-mining.sh
```

**Bridge Connection Errors**:
- Ensure Node.js dependencies are installed: `npm install`
- Bridge starts after contract deployment in flux-mining mode
- Check bridge health: `curl http://localhost:3001/health`

**Dgraph Issues**:
- Wait 30+ seconds for Dgraph container to fully start
- Try accessing Ratel UI directly: `http://localhost:8000`
- Check container: `docker ps | grep dgraph`

### Clean Restart
```bash
# If anything gets stuck, clean everything:
pkill anvil
sudo docker stop dgraph-standalone
sudo docker rm dgraph-standalone  
sudo rm -rf ./dgraph-data
rm -f contract_addresses.json *.log *.pid
```

## 🏗️ Architecture Overview

### Core Components

**🤖 CoreMiner**
- AI agents that process user tasks and requests
- Maintain Vector Logical Clock (VLC) consistency
- Generate responses and request additional context when needed
- Process 7 different test scenarios demonstrating various interaction patterns

**🛡️ CoreValidator** 
- Quality assessment nodes that vote on miner outputs
- Two specialized roles:
  - **UserInterfaceValidator**: Handles user interaction and info requests
  - **ConsensusValidator**: Performs distributed quality voting
- Byzantine Fault Tolerant consensus with 0.25 weight per validator
- Validates VLC sequences and ensures causal consistency

**⏰ Vector Logical Clocks (VLC)**
- Ensures causal ordering of all operations
- Tracks dependencies between events across the network
- Prevents out-of-order execution and maintains consistency
- Critical for distributed consensus and event validation

**📊 Graph Visualization**
- Real-time VLC event tracking via Dgraph database
- Interactive visualization of causal relationships
- Event categorization: UserInput, MinerOutput, InfoRequest, RoundSuccess, etc.
- Complete audit trail of all network interactions

## 🎯 Demonstration Scenarios

The subnet demo processes **7 test scenarios** that showcase all aspects of the PoCW protocol:

### Standard Processing (Scenarios 1, 2, 5, 7)
- **User Input**: "Analyze market trends for Q4"
- **Miner Response**: Direct solution generation
- **Validator Assessment**: Quality voting and consensus
- **User Feedback**: Acceptance confirmation
- **Result**: `OUTPUT DELIVERED TO USER`

### Information Request Flow (Scenarios 3, 6)  
- **User Input**: "Create optimization strategy for resource allocation"
- **Miner Behavior**: Requests additional context
- **Validator Mediation**: Facilitates user-miner communication
- **Enhanced Processing**: Solution with additional context
- **Advanced Validation**: Quality assessment of refined output

### Rejection Scenarios (Scenario 4)
- **Validator Rejection**: Low-quality output rejected by consensus (0.45 quality score)
- **Result**: `OUTPUT REJECTED BY VALIDATORS`

### User Override (Scenario 6)
- **Validator Acceptance**: Output passes validator consensus
- **User Rejection**: User rejects despite validator approval
- **Result**: `OUTPUT REJECTED BY USER (despite validator acceptance)`

## 📊 Visualizing Event Graphs

After running the demonstration, visualize the complete VLC event graph:

### Access Steps
1. **Open Ratel UI**: http://localhost:8000
2. **Verify Connection**: Ensure connection shows `localhost:8080`
3. **Query Events**: Use this DQL query to view all subnet events:

```graphql
{
  events(func: has(id)) {
    uid
    id
    name
    clock
    depth
    value
    key
    node
    parent {
      uid
      id
      name
    }
  }
}
```

### Event Types in Subnet Demo
- **🎯 UserInput**: User task submissions (7 scenarios)
- **🤖 MinerOutput**: AI agent responses and solutions
- **❓ InfoRequest**: Miner requests for additional context  
- **💬 InfoResponse**: User-provided clarifications
- **✅ RoundSuccess**: Successful consensus rounds
- **❌ RoundFailed**: Failed validation or user rejection
- **🏁 EpochFinalized**: Subnet epoch completion markers
- **⭐ GenesisState**: Initial subnet state

### Understanding VLC Relationships
- **Parent Links**: Show causal dependencies between events
- **VLC Clocks**: Demonstrate proper ordering (format: `{miner:X, validator:Y}`)

The graph provides a complete audit trail showing how each user request flows through the subnet, demonstrating the causal consistency guarantees of the PoCW protocol.

## 🧠 Intelligence Money & PoCW Protocol

### Proof-of-Causal-Work Consensus

PoCW extends traditional blockchain consensus by focusing on **causal relationships** rather than just computational work:

**🔗 Causal Consistency**
- Every event must reference its causal dependencies
- Vector Logical Clocks ensure proper ordering across distributed nodes
- Invalid causal relationships are automatically rejected
- Creates immutable audit trail of decision-making processes

**🏛️ Byzantine Fault Tolerant Consensus**  
- 4 validators with 0.25 weight each (total weight = 1.0)
- Requires majority consensus for output acceptance
- Handles up to 1 Byzantine (malicious) validator
- Quality threshold-based voting (accept if quality > 0.5)

**⚡ Event-Driven Architecture**
- Real-time processing of user requests
- Dynamic info request/response cycles
- Asynchronous validator consensus
- Complete traceability of all interactions 

### Blockchain Integration Architecture

The system integrates multiple on-chain components for trustless operation:

```
🏗️ Complete Blockchain Architecture:

┌─────────────────────────────────────────────────────────────────┐
│                         Ethereum Blockchain                      │
│                         (Anvil Local Testnet)                   │
└─────────────────────────────────────────────────────────────────┘
                                  │
        ┌─────────────────────────┼─────────────────────────┐
        │                         │                         │
        ▼                         ▼                         ▼
┌───────────────┐      ┌──────────────────┐      ┌─────────────────┐
│ ERC-8004      │      │ x402 Payment     │      │ FLUX Token      │
│ Agent Registry│      │ Escrow Contract  │      │ Mining Contract │
│               │      │                  │      │                 │
│ • Agent ID: 0 │      │ • AIUSD Deposits │      │ • Soulbound NFT │
│ • Identity    │◄─────│ • Payment Verify │      │ • Epoch Rewards │
│ • Metadata    │      │ • Release/Refund │      │ • VLC Proof     │
└───────────────┘      └──────────────────┘      └─────────────────┘
        │                         │                         │
        │                         │                         │
        └─────────────────────────┼─────────────────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────┐
                    │   AIUSD Stablecoin      │
                    │   (ERC-20 Token)        │
                    │                         │
                    │ • Client Payments       │
                    │ • Agent Compensation    │
                    │ • Escrow Transfers      │
                    └─────────────────────────┘

📊 Payment & Mining Flow:
1. Agent generates x402 payment request
2. Client deposits AIUSD to escrow contract
3. Agent verifies payment on-chain (trustless)
4. Agent processes task after verification
5. BFT validators reach consensus on quality
6. Escrow releases payment OR refunds client
7. FLUX tokens mined based on epoch completion
```

## 💰 Intelligence Money (FLUX Tokens)

### Revolutionary Digital Asset Class

Intelligence Money represents **verifiable units of intelligent work** - the first digital asset derived from provable AI contributions rather than energy consumption.

**🎯 Core Principles**
```
Traditional Crypto:  Energy → Computational Work → Token Value
Intelligence Money:   AI Work → Verified Contribution → FLUX Value
```

### Mining Through Value Creation

**🏗️ The Mining Process**
1. **Query Initiation**: User submits complex problem to subnet
2. **AI Collaboration**: Miners and validators work together via PoCW
3. **Quality Validation**: BFT consensus ensures solution quality  
4. **Value Attribution**: Successful contributions mine new FLUX tokens
5. **Cryptographic Proof**: VLC graph provides immutable work evidence

**⚡ Real-Time Mining**
- Every accepted user solution mines new tokens
- Quality multipliers affect mining rewards
- Validator consensus participation earns rewards
- Failed outputs generate no tokens (merit-based system)

### Soulbound Token Economics

**🔒 Non-Transferable Design**
- **Soulbound**: FLUX tokens cannot be transferred between addresses
- **Reputation-Based**: Tokens represent earned capability and track record
- **Anti-Speculation**: Prevents market manipulation and speculation bubbles
- **Cryptographic Resume**: Immutable proof of AI agent competence

**💵 Liquidity Bridge**
- **Redemption Pool**: One-way bridge to stablecoins (USDC/USDT)
- **Burn Mechanism**: FLUX tokens are destroyed when redeemed
- **Market Valuation**: Exchange rate determined by supply/demand
- **Utility Preservation**: Core reputation asset remains non-transferable

### Economic Model

```
🔄 VALUE FLOW:
User Problem ──▶ AI Solution ──▶ Quality Validation ──▶ FLUX Mining ──▶ Stablecoin Redemption
     ▲                                      │                              │
     └─────── Economic Feedback Loop ───────┴──────────────────────────────┘
```

This creates a **merit-only economy** where value flows directly from problem-solving capability to economic rewards, eliminating speculative intermediaries and ensuring AI agents are compensated based purely on their verifiable contributions to human knowledge and productivity.

## 🛠️ Development & Contributing

### Project Structure
```
Intelligence-FLUX-Mining/
├── main.go                     # Entry point - subnet demonstration
├── go.mod                      # Go module dependencies
├── run-subnet-only.sh          # Subnet consensus only mode
├── run-flux-mining.sh           # Full FLUX mining mode with ERC-8004 identity
├── serve-dashboard.go          # Web UI server (integrated into scripts)
├── mainnet-bridge-per-epoch.js # HTTP bridge service (port 3001)
├── pocw-inspector.html         # Blockchain inspector UI with identity
├── contracts/                  # Solidity smart contracts
│   ├── FLUXToken.sol           # Soulbound FLUX tokens
│   ├── HETUToken.sol          # Staking token
│   ├── AIUSD.sol              # AI services stablecoin for x402 payments
│   ├── x402PaymentEscrow.sol  # Trustless escrow for client-agent payments
│   ├── SubnetRegistry.sol     # Subnet management with ERC-8004 identity
│   ├── PoCWVerifier.sol       # Consensus verification & FLUX mining
│   └── 8004/                  # ERC-8004 Trustless Agents standard
│       ├── IdentityRegistry.sol    # Agent identity NFTs
│       ├── ValidationRegistry.sol  # Agent validation system
│       └── ReputationRegistry.sol  # Agent reputation tracking
├── subnet/                    # Go consensus implementation
│   ├── core_miner.go         # AI miner agents
│   ├── core_validator.go     # BFT validators
│   ├── graph_adapter.go      # VLC graph & HTTP bridge integration
│   ├── messages.go           # Protocol message definitions
│   └── demo/                 # Demo scenarios & coordination
│       ├── demo_coordinator.go      # Demo orchestration
│       ├── demo_task_processor.go   # Task processing logic
│       ├── demo_quality_assessor.go # Quality assessment
│       └── demo_user_interaction.go # User interface simulation
├── vlc/                      # Vector Logical Clock library
│   └── vlc.go               # VLC implementation
├── dgraph/                   # Graph database integration
│   ├── connection.go        # Dgraph client connection
│   └── init.go              # Dgraph initialization
├── models/                   # Data models
│   └── event.go             # Event structure definitions
├── scripts/                  # Deployment scripts
│   └── deploy-identity-phase1.js  # Standalone identity deployment
└── tests/                    # Test utilities
```

### Adding New Features
1. **Subnet Logic**: Modify files in `subnet/` directory
2. **Smart Contracts**: Update contracts in `contracts/` directory  
3. **Bridge Logic**: Enhance `mainnet-bridge-per-epoch.js`
4. **UI Components**: Update `pocw-inspector.html`

## License

MIT License - See [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Test with both `./run-subnet-only.sh` and `./run-flux-mining.sh`
4. Submit a pull request

---

🌟 **Start with `./run-subnet-only.sh` to understand the consensus, then try `./run-flux-mining.sh` for the full FLUX mining experience!**

**Intelligence Money represents the next evolution of digital assets - from energy-based mining to intelligence-based value creation. This system shows how AI agents can collaborate, compete, and be fairly compensated in a trustless, merit-based economy.**
