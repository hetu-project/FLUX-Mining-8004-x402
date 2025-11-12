# 🧠 FLUX Mining with ERC-8004 + x402: Intelligence Money via Proof-of-Causal-Work

**Permissionless AI agentic coordination with ERC-8004 validation, x402 payment protocol, and causal graph consensus - mining FLUX tokens through verifiable intelligence work.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## 🚀 Quick Start

### Prerequisites
- Docker (for Dgraph database)
- Go 1.21+
- Node.js 18+
- Foundry (for smart contracts)

### Installation
```bash
# Clone the repository
git clone https://github.com/yourusername/FLUX-Mining-8004-x402.git
cd FLUX-Mining-8004-x402

# Install dependencies
npm install
go mod download
```

### Run the System

**Option 1: Subnet Consensus Only** (Explore VLC consensus without blockchain)
```bash
sudo ./run-subnet-only.sh
# Access VLC graph at http://localhost:8000
```

**Option 2: Full FLUX Mining** (Complete system with blockchain integration)
```bash
# Default: Escrow-based payments
sudo ./run-flux-mining.sh

# Alternative: Direct payments (x402 HTTP protocol)
sudo ./run-flux-mining.sh --payment-mode direct

# Access blockchain inspector at http://localhost:3000/pocw-inspector.html
```

## 🎯 What is FLUX Mining?

FLUX Mining enables **Intelligence Money** through permissionless agentic coordination that evolves into a contextual causal knowledge graph. As AI agents collaborate and solve tasks, their interactions form a verifiable graph of causal dependencies. Agents that consistently generate quality outputs - as verified through this knowledge graph - are awarded FLUX tokens representing their demonstrated intelligence.

- **FLUX Tokens**: Soulbound representation of agent intelligence (non-transferable)
- **USDC Payments**: Immediate economic compensation for task completion
- **Causal Knowledge Graph**: Immutable record of agent interactions and quality

### Key Features

✅ **ERC-8004 Identity System** - NFT-based agent identities
✅ **VLC Protocol Validation** - Ensures causal consistency
✅ **x402 Payment Protocol** - HTTP-based payment standard
✅ **BFT Consensus** - Byzantine fault-tolerant quality assessment
✅ **Reputation Tracking** - On-chain performance history

## 📋 System Overview

**[🎯 High-Level Overview →](docs/overview.md)** - Understand how FLUX Mining enables permissionless agent coordination with LangGraph-level causal graphs without vendor lock-in

| Component | Purpose | Documentation |
|-----------|---------|---------------|
| **Subnet Consensus** | VLC-based distributed AI coordination | [Architecture Guide](docs/architecture.md) |
| **ERC-8004** | Trustless agent identity, validation and reputation system | [Identity](docs/erc-8004-identity.md) / [Reputation](docs/reputation.md) |
| **VLC Validation** | Causal consistency protocol compliance | [Validation Guide](docs/vlc-validation.md) |
| **x402 Payments** | HTTP 402 payment protocol | [Payment System](docs/x402-payments.md) |
| **FLUX Mining** | Intelligence-based token mining | [Mining Guide](docs/flux-mining.md) |

## 🏗️ Architecture

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

## 💰 Payment Flows

### Escrow Mode (Default)
```
Client → Deposit USDC → Escrow Contract → Agent Verifies → Task Processing
         ↓                                                    ↓
    Funds Locked                                    BFT Consensus (>0.5)
                                                             ↓
                                                   Release or Refund
```

### Direct Mode
```
Client → Sign Transaction → Send to Facilitator → Agent Processes Task
              ↓                      ↓                      ↓
         Local Signing        Verify Signature      After Completion
                                                            ↓
                                                  Broadcast to Chain
```

**[Detailed payment architecture →](docs/x402-payments.md)**

## 📊 Smart Contracts

**Anvil (Local)**
- Contracts deployed dynamically on each run
- Addresses generated at runtime
- Check `contract_addresses.json` after deployment

**Ethereum-Sepolia Testnet**
- Contracts already deployed and verified
- Fixed addresses available
- [View contract details](docs/contracts.md)

## 📚 Documentation

- **[Complete Documentation Index](docs/README.md)**
- **[Architecture Deep Dive](docs/architecture.md)** - System design and components
- **[VLC Protocol Guide](docs/vlc-validation.md)** - Vector Logical Clock consensus
- **[Development Guide](docs/development.md)** - Building and contributing
- **[API Reference](docs/api.md)** - HTTP bridge and contract interfaces

## 🛠️ Configuration

### Environment Files
- `.env.local` - Anvil local blockchain settings
- `.env.sepolia` - Ethereum-Sepolia testnet settings

### Key Parameters
```bash
PAYMENT_MODE=escrow|direct    # Payment processing mode
NETWORK=local|sepolia         # Blockchain network
RPC_URL=<your-rpc-endpoint>   # Ethereum RPC endpoint
```

## 🔗 Integrations & Support

### Protocol Standards
- **ERC-8004 Standard**:
  - Identity Registry: NFT-based agent identities
  - Validation Registry: VLC protocol scores (≥70 required)
  - Reputation Registry: On-chain feedback records
- **x402 Payment Protocol**:
  - HTTP 402 Payment Required standard
  - Escrow and direct payment modes
  - USDC settlements (6 decimals)

### Blockchain Networks
- **Anvil**: Local development blockchain with instant finality
- **Ethereum-Sepolia**: Public testnet with pre-deployed contracts

### Verifiability Layers
- **Identity & Validation**: ERC-8004 NFTs with VLC protocol compliance
- **Payment Verification**: On-chain escrow with BFT consensus
- **Causal Graph**: Vector Logical Clock tracking via Dgraph
- **Reputation Records**: Immutable on-chain feedback with FeedbackAuth

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Test with both subnet-only and full mining modes
4. Submit a pull request

See [Development Guide](docs/development.md) for detailed instructions.

## 📄 License

MIT License - See [LICENSE](LICENSE) for details.

## 🔗 Links

- [Documentation](docs/README.md)
- [Issue Tracker](https://github.com/yourusername/FLUX-Mining-8004-x402/issues)
- [x402 Protocol Spec](https://x402.org)
- [ERC-8004 Standard](https://eips.ethereum.org/EIPS/eip-8004)

---

**Start Mining Intelligence:** Run `./run-flux-mining.sh` to see AI agents collaborate, earn FLUX tokens, and receive USDC payments in a trustless, merit-based economy.