# System Architecture

## Overview

FLUX Mining implements a multi-layer architecture combining blockchain smart contracts, distributed consensus, and AI agent coordination.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                   Blockchain Networks                       │
│         (Anvil Local / Ethereum-Sepolia Testnet)           │
└─────────────────────────────────────────────────────────────┘
                            │
    ┌───────┬───────┬───────┼───────┬───────┬──────┐
    ▼       ▼       ▼       ▼       ▼       ▼      ▼
┌────────┐┌────────┐┌───────┐┌──────┐┌──────┐┌────────┐┌────────┐
│ERC-8004││ERC-8004││ x402  ││ FLUX ││ USDC ││ERC-8004││Subnet  │
│Identity││Validatn││Escrow ││Mining││Token ││Reputatn││Registry│
└────────┘└────────┘└───────┘└──────┘└──────┘└────────┘└────────┘
    │         │         │       │       │        │         │
    └─────────┴─────────┴───────┴───────┴────────┴─────────┘
                            │
                    ┌───────┴────────┐
                    │  Subnet Layer  │
                    │   (PoCW BFT)   │
                    └────────────────┘
                            │
                    ┌───────┴────────┐
                    │   VLC Protocol │
                    │  (Causal Order)│
                    └────────────────┘
```

## Core Components

### 🤖 CoreMiner
- Processes user tasks and requests
- Maintains Vector Logical Clock consistency
- Generates AI responses
- Handles 7 test scenarios

### 🛡️ CoreValidator
- Quality assessment nodes
- Byzantine Fault Tolerant voting
- Two specialized roles:
  - **UserInterfaceValidator**: User interaction
  - **ConsensusValidator**: Quality voting

### ⏰ Vector Logical Clocks (VLC)
- Ensures causal ordering
- Tracks event dependencies
- Prevents out-of-order execution
- Critical for consensus

### 📊 Graph Database (Dgraph)
- Real-time VLC event tracking
- Interactive visualization
- Complete audit trail
- Causal relationship mapping

### 🌉 HTTP Bridge
- Port 3001
- Connects subnet to blockchain
- Handles epoch submissions
- Manages payment flows

## Data Flow

### Task Processing
1. User submits task request
2. Miner processes with VLC tracking
3. Validators assess quality (BFT)
4. Consensus determines outcome
5. Payment released/refunded
6. FLUX tokens mined at epoch

### Epoch Management
- **Round**: Single task with consensus
- **Epoch**: 3 consecutive rounds
- **Mining**: Triggered per epoch
- **Reputation**: Batch submitted per epoch

## Network Topology

### Local Development (Anvil)
```
Anvil (port 8545)
  ├── Deploy contracts
  ├── Fund accounts
  └── Process transactions

Dgraph (port 8080)
  └── Store VLC events

HTTP Bridge (port 3001)
  └── Connect subnet to chain

Web Inspector (port 3000)
  └── Blockchain visualization
```

### Sepolia Testnet
- Pre-deployed contracts
- Public RPC endpoints
- Etherscan verification
- Facilitator authorization

## Security Architecture

### Smart Contract Layer
- Reentrancy guards
- Access control
- Time-based deadlines
- Signature verification

### Consensus Layer
- BFT with 4 validators
- 0.25 weight each
- Tolerates 1 Byzantine node
- Quality threshold: 0.5

### Identity Layer
- ERC-8004 NFTs
- VLC validation requirement
- Minimum score: 70/100
- Sybil resistance

## Event Types

### Subnet Events
- `GenesisState`: Initial state
- `UserInput`: Task submissions
- `MinerOutput`: AI responses
- `InfoRequest`: Additional context
- `InfoResponse`: Clarifications
- `RoundSuccess`: Successful consensus
- `RoundFailed`: Failed validation
- `EpochFinalized`: Mining trigger

### Blockchain Events
- `IdentityMinted`: Agent creation
- `ValidationSubmitted`: VLC scores
- `PaymentDeposited`: Escrow locks
- `PaymentReleased`: Successful payment
- `PaymentRefunded`: Failed payment
- `FeedbackSubmitted`: Reputation update
- `TokensMined`: FLUX distribution

## Performance Characteristics

### Throughput
- 7 tasks per epoch cycle
- ~30 second epoch duration
- Parallel validator assessment
- Batch blockchain submissions

### Scalability
- Horizontal validator scaling
- Multiple subnet support
- Dgraph handles millions of events
- Smart contract gas optimization

## Development Stack

### Backend
- **Go**: Subnet consensus implementation
- **Node.js**: HTTP bridge and facilitator
- **Solidity**: Smart contracts
- **Foundry**: Contract deployment

### Frontend
- **HTML/JS**: Blockchain inspector
- **Ratel**: Dgraph visualization

### Infrastructure
- **Docker**: Dgraph container
- **Anvil**: Local blockchain
- **Ethereum**: Production network

## Related Documentation

- [VLC Validation](vlc-validation.md) - Protocol details
- [x402 Payments](x402-payments.md) - Payment system
- [Development Guide](development.md) - Building and contributing