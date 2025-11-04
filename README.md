# FLUX Mining with ERC-8004 & x402 Payment Escrow

A trustless AI agent infrastructure combining **ERC-8004 decentralized identity**, **VLC protocol validation**, **x402 payment protocol**, **reputation feedback system**, and **Proof-of-Causal-Work consensus** to mine soulbound FLUX tokens through verified intelligence work. Features mandatory on-chain VLC validation before subnet registration, smart contract escrow payments with on-chain verification, epoch-based reputation tracking with agent-signed FeedbackAuth, Byzantine Fault Tolerant consensus, and Vector Logical Clock causal ordering for permissionless AI marketplaces.

## Overview

This system demonstrates a **permissionless AI marketplace** where agents earn soulbound FLUX tokens through verified intelligence work and receive stablecoin payments (USDC/AIUSD) via trustless escrow. Combines five key innovations:

- **🆔 ERC-8004 Identity**: Decentralized agent registry with cryptographic verification and NFT-based identity
- **🔐 VLC Protocol Validation**: Mandatory on-chain validation ensuring agents correctly implement Vector Logical Clock causal consistency (≥70/100 score required)
- **💳 x402 Payment Protocol**: Trustless escrow with on-chain payment verification - agents verify funds before processing
- **⭐ Reputation Feedback**: Epoch-based on-chain reputation tracking with agent-signed FeedbackAuth for trustless quality assessment
- **⛏️ Proof-of-Causal-Work**: Byzantine Fault Tolerant consensus using Vector Logical Clocks for causal ordering

The **FLUX token** represents verifiable intelligence contributions and is non-transferable (soulbound) but redeemable, while **stablecoins (USDC by default, configurable to AIUSD)** enable instant, cryptographically-secured payments between clients and agents without trusted intermediaries.

### Key Features

- 🧠 **Intelligence Mining**: Earn FLUX tokens through actual AI task completion
- 🔐 **VLC Protocol Validation**: Mandatory on-chain validation before subnet registration
- 💳 **x402 Escrow Payments**: Trustless USDC payments with BFT consensus-based release
- ⭐ **Reputation Feedback System**: Epoch-based on-chain reputation tracking with cryptographic FeedbackAuth
- 🔗 **Vector Logical Clocks**: Causal ordering of distributed consensus events
- 🏛️ **Byzantine Fault Tolerant**: 4-validator consensus with quality assessment
- 💎 **Soulbound Tokens**: Non-transferable but redeemable FLUX tokens
- 🆔 **ERC-8004 Identity**: Trustless agent identity with NFT-based verification
- ✅ **On-Chain Validation Registry**: Permanent record of agent validation scores
- 📊 **Real-time Visualization**: VLC event graph via Dgraph
- ⛓️ **Blockchain Integration**: Smart contracts on Anvil/Ethereum
- 🔒 **Smart Contract Escrow**: Automatic payment release/refund based on validator consensus
- 💵 **Configurable Payment Token**: Use USDC (default) or AIUSD via `--payment-token` flag

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
         │             └──────────────────┘    │ • ERC-8004              │
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

**Purpose**: Complete PoCW system with FLUX mining, VLC validation, x402 escrow payments, reputation feedback, and ERC-8004 identity

**What it does**:
- ✅ Everything from subnet-only mode PLUS:
- ✅ Deploys smart contracts (FLUX, USDC, x402PaymentEscrow, ERC-8004 Identity, ReputationRegistry, etc.)
- ✅ Real-time FLUX mining per epoch (every 3 rounds)
- ✅ **VLC Protocol Validation**: Mandatory on-chain validation (≥70/100 score) before subnet registration
- ✅ **x402 Trustless Escrow**: USDC payments with BFT consensus-based release/refund
- ✅ **ERC-8004 Agent Identity**: NFT-based trustless agent verification
- ✅ **Reputation Feedback System**: Epoch-based batch submission with agent-signed FeedbackAuth
- ✅ Blockchain transactions with verified rewards
- ✅ Bridge service for epoch submission
- ✅ Complete FLUX and USDC balance tracking
- ✅ Demonstrates 5 successful payments + 2 refunds via escrow
- ✅ On-chain reputation tracking with 6 feedback submissions across 2 epochs

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
- **VLC Protocol Validation**: Before subnet registration, agent undergoes VLC validation
  ```
  📋 VLC Protocol Validation

  ✅ VLC VALIDATION PASSED
  Agent: miner-1

     📋 Validator-1 submitting validation...
        ✅ Validator-1: Score 100/100 recorded
     📋 Validator-2 submitting validation...
        ✅ Validator-2: Score 100/100 recorded
     ... [All 4 validators]

     📊 Validation Summary:
        Validators: 4
        Average Score: 100/100
        Status: ✅ PASSED

  🔐 Registering subnet with Agent ID 0...
  ```
- **x402 Payment Requests**: Agent generates payment requests with task details
  ```
  📋 Agent sends x402 Payment Request to Client:
     Task ID: req-subnet-001-1
     Amount: 10000000000000000000 wei (10 USDC)
     Escrow Contract: 0x0165878A594ca255338adfa4d48449f69242Eb8F
  ```
- **Payment Deposits**: Client deposits USDC to escrow (10 USDC per task)
- **Agent Verification**: Agent verifies payment on-chain before processing
  ```
  ✅ Payment verified for task req-subnet-001-1:
     Amount: 10.00 USDC (locked in escrow)
     Agent: 0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc
  🔐 Miner: Payment verified on-chain - proceeding with task
  ```
- **FLUX Mining**: 400 FLUX to miner, 80 FLUX to validators per epoch
- **BFT Consensus**: Validators decide payment release or refund
- **Payment Outcomes**: 5 payments released (50 USDC), 2 refunded (20 USDC)
- **Reputation Feedback**: Epoch-based batch submission after every 3 tasks
  ```
  ╔══════════════════════════════════════════════════════════════╗
  ║           SUBMITTING EPOCH FEEDBACK TO BLOCKCHAIN          ║
  ╚══════════════════════════════════════════════════════════════╝

  📝 Task 1 (req-per-epoch-subnet-001-1): ✅ Success (TX: 0x47ed...)
  📝 Task 2 (req-per-epoch-subnet-001-2): ✅ Success (TX: 0x7889...)
  📝 Task 3 (req-per-epoch-subnet-001-3): ✅ Success (TX: 0x72fe...)

  ╔══════════════════════════════════════════════════════════════╗
  ║        ✅ EPOCH FEEDBACK BATCH SUBMITTED SUCCESSFULLY       ║
  ║  Agent ID: 0 | Total Feedbacks: 3                           ║
  ╚══════════════════════════════════════════════════════════════╝
  ```
- **Final Reputation Summary**: Retrieved from blockchain at end of demo
  ```
  📊 Agent ID 0 Reputation on Blockchain:
    📝 Total Feedbacks Received: 6
    ⭐ Average Score: 70/100 (Good Performance ✅)
    📊 Score Visual: [███████████████████████░░░░░░░░░] 70%
  ```
- **Trustless Operation**: Complete audit trail of all payments and reputation on-chain

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
# Complete PoCW system with blockchain integration (uses USDC by default)
sudo ./run-flux-mining.sh

# Or use AIUSD stablecoin instead
sudo ./run-flux-mining.sh --payment-token AIUSD

# Watch FLUX tokens being mined in real-time
# Explore blockchain inspector at http://localhost:3000/pocw-inspector.html
# Bridge stays active for continued mining
# Press Ctrl+C when done
```

**💡 Payment Token Configuration:**
- **Default**: Uses USDC stablecoin (6 decimals, industry standard)
- **Alternative**: Use `--payment-token AIUSD` flag for custom AI stablecoin (18 decimals, EIP-3009 support)
- Both tokens work identically with the x402 escrow system

## Smart Contracts

| Contract | Purpose | Features |
|----------|---------|----------|
| **FLUXToken** | Soulbound intelligence tokens | Non-transferable, 21M max supply |
| **HETUToken** | Staking for subnet registration | ERC20, 1M total supply |
| **USDC/AIUSD** | Payment stablecoin | ERC20, USD-pegged, for x402 payments (configurable) |
| **x402PaymentEscrow** | Trustless payment escrow | BFT consensus-based release/refund, reentrancy protection |
| **SubnetRegistry** | Manages subnet participants with identity | ERC-8004 identity verification, deposit requirements |
| **PoCWVerifier** | Consensus verification & mining | Per-epoch FLUX distribution, validator authorization |
| **IdentityRegistry** | ERC-8004 Trustless Agents identity | NFT-based agent IDs, ownership verification |
| **ValidationRegistry** | Agent validation requests | Quality assessments, reputation tracking |
| **ReputationRegistry** | Agent reputation feedback | Client feedback with FeedbackAuth, score tracking, reputation queries |

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
- Verify ValidationRegistry scores and VLC compliance

## 🔐 VLC Protocol Validation System

Before any agent can register a subnet and participate in FLUX mining, it **must pass VLC protocol validation** - a rigorous on-chain test ensuring proper implementation of Vector Logical Clock causal consistency.

### Why VLC Validation Matters

**Preventing Byzantine Failures**
- Agents with incorrect VLC implementations can break causal ordering
- Invalid clock increments corrupt the consensus mechanism
- Malformed event sequences compromise network integrity
- Validation ensures only protocol-compliant agents participate

### Validation Process

**📋 Pre-Registration Testing**

The system performs VLC validation BEFORE subnet registration:

```
Flow:
1. Agent registers identity NFT (ERC-8004)
2. 🔐 VALIDATION CHECKPOINT: VLC Protocol Test
   ├─ 4 validators send ambiguous tasks to agent
   ├─ Agent must correctly implement NeedMoreInfo flow
   ├─ VLC clock must increment properly (+2 per message exchange)
   └─ Each validator submits score (0-100) to ValidationRegistry
3. Validators submit scores to blockchain (permanent record)
4. Smart contract verifies: getSummary(agentId) >= 70/100
5. ✅ Only if validation passes → Subnet registration allowed
```

**⛓️ On-Chain Score Recording**

All validation scores are permanently stored in the ValidationRegistry smart contract:

```solidity
// Each validator submits their assessment
function validationResponse(
    bytes32 requestHash,
    uint8 response,        // Score: 0-100
    string calldata responseUri,
    bytes32 responseHash,
    bytes32 tag           // "VLC_PROTOCOL" tag
) external

// Smart contract retrieves summary for subnet registration
function getSummary(
    uint256 agentId,
    address[] calldata validatorAddresses,
    bytes32 tag          // Filter by "VLC_PROTOCOL"
) external view returns (
    uint64 count,        // Number of validations
    uint8 avgResponse    // Average score
)
```

**✅ Pass/Fail Criteria**

```
SubnetRegistry.sol validation check:

(uint64 totalValidations, uint8 avgScore) = validationRegistry.getSummary(
    minerAgentId,
    emptyValidators,  // Get all validators
    VLC_PROTOCOL_TAG
);

require(totalValidations > 0, "Agent has no VLC validation scores");
require(avgScore >= 70, "Agent validation score too low (min 70 required)");

✅ PASSED:  avgScore >= 70 → Subnet registration proceeds
❌ FAILED:  avgScore < 70  → Subnet registration blocked
```

### Validation Outcomes

**Demo Results:**
```bash
📋 VLC Protocol Validation

   📋 Validator-1 submitting validation...
      ✅ Validator-1: Score 100/100 recorded
         ✓ Response confirmed on-chain

   📋 Validator-2 submitting validation...
      ✅ Validator-2: Score 100/100 recorded
         ✓ Response confirmed on-chain

   📋 Validator-3 submitting validation...
      ✅ Validator-3: Score 100/100 recorded
         ✓ Response confirmed on-chain

   📋 Validator-4 submitting validation...
      ✅ Validator-4: Score 100/100 recorded
         ✓ Response confirmed on-chain

   📊 Validation Summary:
      Agent ID: #0
      🔍 Calling ValidationRegistry.getSummary...
      📝 Raw getSummary response: 4
100
      📊 Parsed values: count=4, avgScore=100
      Validators: 4
      Average Score: 100/100
      Status: ✅ PASSED

🔐 Registering subnet with Agent ID 0...
   The SubnetRegistry contract will verify:
   ✓ Agent owns the identity token
   ✓ Agent has passed VLC validation (score >= 70)
```

### Security Benefits

**🛡️ Network Protection**
- **Protocol Enforcement**: Only VLC-compliant agents can participate
- **Permanent Record**: Validation scores stored on-chain forever
- **No Bypass**: Smart contract enforces validation before registration
- **Sybil Resistance**: Each agent ID validated independently

**🔒 Trust Minimization**
- **On-Chain Verification**: No off-chain trust required
- **Validator Consensus**: Multiple independent validators assess agent
- **Cryptographic Proof**: Blockchain provides immutable validation record
- **Permissionless**: Any validator can assess any agent

**📊 Inspector Integration**

The blockchain inspector displays validation status:
- View all validation requests for each agent
- Check individual validator scores
- Verify VLC_PROTOCOL tag compliance
- Monitor average scores and pass/fail status

## 💳 x402 Payment System with Smart Escrow

The system now integrates **x402 protocol** for trustless, escrow-based stablecoin payments (USDC by default) between clients and AI agents, leveraging permissionless validator consensus.

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
     ├──── approve(USDC) ──────────▶│                        │
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
     │                              │  ✓ Agent receives USDC │
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
  "amount": "10000000000000000000",  // 10 USDC in wei (or AIUSD if configured)
  "asset": {
    "symbol": "USDC",
    "contract": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    "decimals": 6
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

// Client deposits payment token (USDC/AIUSD) for a task (via coordinator)
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

### Payment Token: Stablecoin (USDC/AIUSD)

**💵 Configurable Stablecoin Payments**

The system supports configurable stablecoin payments for AI services:

**Default: USDC**
- Industry-standard USD stablecoin
- 6 decimals (matches real USDC)
- Standard ERC-20 transferability
- Production-ready interface

**Alternative: AIUSD** (use `--payment-token AIUSD` flag)
- Custom AI services stablecoin
- 18 decimals
- EIP-3009 gasless transfer support
- Advanced features for special use cases

**Client Payment Workflow:**
```bash
1. Client receives payment tokens (1000 USDC in demo)
2. Client sends task request to agent
3. Agent generates x402 payment request with details:
   - Task ID, amount (10 USDC), escrow address, agent address, deadline
4. Client approves escrow contract to spend USDC
5. Client deposits payment to escrow contract
6. Agent verifies payment on-chain before processing
   - Queries blockchain: payments[taskId].status == DEPOSITED
   - Verifies: correct agent, sufficient amount, deadline valid
7. Agent processes task only after payment verification
8. USDC locked in escrow until consensus decision (release/refund)
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
  "IdentityRegistry": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
  "ValidationRegistry": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
  "ReputationRegistry": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
  "HETUToken": "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
  "FLUXToken": "0x5FC8d32690cc91D4c39d9d3abcBD16989F875707",
  "USDC": "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9",
  "SubnetRegistry": "0x0165878A594ca255338adfa4d48449f69242Eb8F",
  "PoCWVerifier": "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853",
  "x402PaymentEscrow": "0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"
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

## ⭐ Reputation Feedback System

The system implements a comprehensive **ERC-8004 reputation feedback mechanism** that allows clients to rate agent performance after task completion. This creates an immutable, on-chain record of agent quality over time.

### Overview

**🎯 Purpose**: Enable trustless, decentralized reputation tracking for AI agents based on actual task performance.

**Key Features**:
- ✅ **Off-Chain FeedbackAuth Generation**: Agent signs authorization, no blockchain transaction required
- ✅ **On-Chain Reputation Storage**: Client submits feedback to ReputationRegistry smart contract
- ✅ **Epoch-Based Batching**: Feedback submitted every 3 tasks
- ✅ **Cryptographic Authorization**: Agent signs FeedbackAuth (289 bytes: 224 data + 65 signature)
- ✅ **Score Tracking**: Performance scores recorded permanently on blockchain

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│              Reputation Feedback Flow (Per Task)                 │
└─────────────────────────────────────────────────────────────────┘

1️⃣ TASK COMPLETION PHASE
   Agent                    Client                  Blockchain
     │                        │                          │
     ├─ Process Task ─────────▶                         │
     │                        │                          │
     ├─ Generate FeedbackAuth │                         │
     │  (Sign off-chain)      │                          │
     │                        │                          │
     ├─ Send FeedbackAuth ───▶│                         │
     │  (289 bytes)           │  ✓ Store locally        │
     │                        │    (no gas cost)         │

2️⃣ EPOCH ACCUMULATION (Tasks 1-3)
     │                        │                          │
     Task 1: FeedbackAuth generated & stored locally     │
     Task 2: FeedbackAuth generated & stored locally     │
     Task 3: FeedbackAuth generated & stored locally     │
     │                        │                          │

3️⃣ BATCH SUBMISSION PHASE (After 3 Tasks)
     │                        │                          │
     │                        ├─ Submit Batch ──────────▶│
     │                        │   (3 feedbacks)          │
     │                        │                          │ ReputationRegistry
     │                        │                          ├─ Verify Signatures
     │                        │                          ├─ Record Scores
     │                        │                          └─ Emit Events
     │                        │                          │
     │                        │ ◄── Confirmation ────────┤
     │                        │   (TX receipts)          │

4️⃣ VERIFICATION PHASE
     │                        │                          │
     │ ◄─── Query Reputation ──────────────────────────┤
     │      getSummary()      │                         │
     │      (count, avgScore) │                         │
```

### FeedbackAuth Structure

**🔐 Cryptographic Authorization (289 bytes)**

The agent generates a signed authorization that allows the client to submit feedback:

```go
type FeedbackAuthData struct {
    AgentId          *big.Int       // Agent ID (ERC-8004 identity)
    ClientAddress    common.Address // Authorized client address
    IndexLimit       uint64         // Progressive index (1, 2, 3...)
    Expiry           *big.Int       // Authorization expiry timestamp
    ChainId          *big.Int       // Network chain ID
    IdentityRegistry common.Address // IdentityRegistry contract
    SignerAddress    common.Address // Agent owner's address
}

// Signed using Ethereum message format:
// signature = sign(keccak256("\x19Ethereum Signed Message:\n32" + messageHash))
```

**Authorization Process**:
1. Agent encodes FeedbackAuth struct (224 bytes)
2. Agent signs with private key (65 byte signature)
3. Agent sends full 289-byte authorization to client
4. Client stores locally (no gas cost)
5. Client submits to blockchain when epoch completes

### Scoring System

**📊 Performance-Based Scores**

```go
func CalculateFeedbackScore(success bool) uint8 {
    if success {
        return 85  // Successful task completion
    }
    return 40  // Failed or rejected task
}
```

**Task Success Criteria**:
- ✅ **Success (85/100)**: Validator consensus accepts + User accepts output
- ❌ **Failure (40/100)**: Validator consensus rejects OR User rejects output

**Tags for Classification**:
```go
tag1 = keccak256("TASK_SUCCESS")  // or "TASK_FAILED"
tag2 = keccak256("COMPUTE")       // Task type
```

### Epoch-Based Batch Submission

**⚡ Gas Optimization Through Batching**

Instead of submitting feedback after each task (expensive), the system batches 3 feedbacks per epoch:

```
Epoch 1: Tasks 1, 2, 3
  ├─ Task 1 completes → Generate FeedbackAuth → Store locally
  ├─ Task 2 completes → Generate FeedbackAuth → Store locally
  └─ Task 3 completes → Generate FeedbackAuth → Submit all 3 to blockchain

Epoch 2: Tasks 4, 5, 6
  ├─ Task 4 completes → Generate FeedbackAuth → Store locally
  ├─ Task 5 completes → Generate FeedbackAuth → Store locally
  └─ Task 6 completes → Generate FeedbackAuth → Submit all 3 to blockchain
```

**Submission Timing**:
1. ✅ Epoch data submitted to mainnet
2. ✅ Reputation feedback batch submitted
3. ✅ Next epoch begins

### Smart Contract Integration

**📝 ReputationRegistry.sol**

```solidity
// Client submits feedback with agent-signed authorization
function giveFeedback(
    uint256 agentId,
    uint8 score,              // 0-100 performance score
    bytes32 tag1,             // Primary classification tag
    bytes32 tag2,             // Secondary classification tag
    string calldata feedbackUri,
    bytes32 feedbackHash,
    bytes calldata feedbackAuth  // 289-byte signed authorization
) external

// Query agent's reputation summary
function getSummary(
    uint256 agentId,
    address[] calldata clientAddresses,  // Empty = all clients
    bytes32 tag1,                         // 0x0 = all tags
    bytes32 tag2
) external view returns (
    uint64 count,           // Total feedback count
    uint8 averageScore      // Average score (0-100)
)
```

### Demo Output

**📊 Epoch 1 Feedback Submission**:
```bash
╔══════════════════════════════════════════════════════════════╗
║           SUBMITTING EPOCH FEEDBACK TO BLOCKCHAIN          ║
╚══════════════════════════════════════════════════════════════╝

📝 Task 1 (req-per-epoch-subnet-001-1): ✅ Success (TX: 0x47ed...)
📝 Task 2 (req-per-epoch-subnet-001-2): ✅ Success (TX: 0x7889...)
📝 Task 3 (req-per-epoch-subnet-001-3): ✅ Success (TX: 0x72fe...)

╔══════════════════════════════════════════════════════════════╗
║        ✅ EPOCH FEEDBACK BATCH SUBMITTED SUCCESSFULLY       ║
║                                                              ║
║  Agent ID: 0                                                 ║
║  Total Feedbacks: 3                                          ║
║  All feedback recorded on-chain in ReputationRegistry       ║
╚══════════════════════════════════════════════════════════════╝
```

**🔄 Starting Epoch 2**

**📊 Final Reputation Summary**:
```bash
╔══════════════════════════════════════════════════════════════╗
║        🌟 FINAL AGENT REPUTATION SUMMARY                    ║
║           (Read from ReputationRegistry)                    ║
╚══════════════════════════════════════════════════════════════╝

📊 Agent ID 0 Reputation on Blockchain:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  📝 Total Feedbacks Received: 6
  ⭐ Average Score: 70/100 (Good Performance ✅)
  📊 Score Visual: [███████████████████████████████████░░░░░░░░░░░░░░░] 70%
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Reputation data successfully retrieved from blockchain!
```

**Score Breakdown** (from demo with 7 tasks):
- Task 1: 85 (Success)
- Task 2: 85 (Success)
- Task 3: 85 (Success)
- Task 4: 40 (Validator rejection)
- Task 5: 85 (Success)
- Task 6: 40 (User rejection)
- Task 7: 85 (Success)
- **Average**: (85+85+85+40+85+40+85)/7 ≈ 72/100

### Integration with FLUX Mining

**🔗 Parallel Operation**

The reputation system operates alongside FLUX mining and x402 payments:

```
Task Completion:
  ├─ x402 Payment: USDC released from escrow (immediate economic reward)
  ├─ FLUX Tokens: Mined at epoch boundary (long-term reputation token)
  └─ Reputation: Feedback recorded on-chain (permanent quality record)
```

**Why All Three?**
- **x402 Payments**: Immediate compensation for work performed
- **FLUX Tokens**: Soulbound reputation, cannot be transferred
- **Reputation Scores**: Public quality metrics for client decision-making

### Benefits

**For Clients**:
- ✅ Verify agent quality before engaging
- ✅ Transparent performance history
- ✅ Protection against low-quality agents

**For Agents**:
- ✅ Build verifiable reputation over time
- ✅ Differentiate through quality metrics
- ✅ Earn trust through demonstrated performance

**For the Network**:
- ✅ Quality-based agent discovery
- ✅ Market-driven quality improvement
- ✅ Trustless reputation without central authority

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
        ┌─────────────┬───────────┼───────────┬─────────────┐
        │             │           │           │             │
        ▼             ▼           ▼           ▼             ▼
┌──────────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐
│ ERC-8004     │ │ VLC      │ │ x402    │ │ FLUX     │ │ USDC    │
│ Registries   │ │ Valid.   │ │ Payment │ │ Token    │ │ Stable  │
│              │ │ Registry │ │ Escrow  │ │ Mining   │ │ coin    │
│ • Identity   │ │          │ │         │ │          │ │         │
│ • Validation │ │ • Score  │ │ • USDC  │ │ • Soul-  │ │ • Pay   │
│ • Reputation │ │ • Pass/  │ │   Lock  │ │   bound  │ │   ments │
│              │ │   Fail   │ │ • BFT   │ │ • Epoch  │ │ • Comp  │
│              │ │ • ≥70    │ │   Vote  │ │   Reward │ │   ensat │
└──────────────┘ └──────────┘ └─────────┘ └──────────┘ └─────────┘
        │             │           │           │             │
        └─────────────┴───────────┴───────────┴─────────────┘

📊 Complete Agent Lifecycle Flow:
1. Agent registers identity NFT (ERC-8004 IdentityRegistry)
2. Agent passes VLC validation (ValidationRegistry, score ≥70/100)
3. Agent registers subnet and begins mining
4. Agent generates x402 payment request for each task
5. Client deposits USDC to escrow contract
6. Agent verifies payment on-chain (trustless)
7. Agent processes task after verification
8. BFT validators reach consensus on quality
9. Escrow releases payment OR refunds client
10. Agent generates FeedbackAuth for client (off-chain signature)
11. Client submits reputation feedback batch (ReputationRegistry)
12. FLUX tokens mined based on epoch completion
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
│   ├── vlc_validation.go     # VLC protocol validation implementation
│   ├── graph_adapter.go      # VLC graph & HTTP bridge integration
│   ├── reputation_feedback.go # Reputation feedback auth & batch submission
│   ├── payment_coordinator.go # x402 payment system integration
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
