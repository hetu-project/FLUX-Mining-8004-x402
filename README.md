# 🧠 FLUX Mining with ERC-8004 + x402: Intelligence Money via Proof-of-Causal-Work

**Permissionless AI agentic coordination with ERC-8004 validation, x402 payment protocol, and causal graph consensus - mining FLUX tokens through verifiable intelligence work.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## 🚀 Quick Start

### Prerequisites
- Docker (for Dgraph database)
- Go 1.21+
- Node.js 18+
- Foundry (for smart contracts)
- ngrok or localtunnel (for TEE validation with remote endpoints)

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
# Local Anvil (default - direct payments)
sudo ./run-flux-mining.sh

# Ethereum Sepolia Testnet
sudo ./run-flux-mining.sh --network sepolia

# Alternative: Escrow-based payments
sudo ./run-flux-mining.sh --payment-mode escrow

# Access blockchain inspector at http://localhost:3000/pocw-inspector.html
```

### IPFS Storage (Optional - Recommended for Production)
Enable IPFS storage to reduce on-chain costs by ~95%:

```bash
# 1. Copy the example file and add YOUR OWN credentials
cp .env.pinata.example .env.pinata
nano .env.pinata

# Get credentials from: https://app.pinata.cloud/developers/api-keys

# 2. Run normally - IPFS will be used automatically
sudo ./run-flux-mining.sh
```

**⚠️ IPFS Privacy Note**:
- `PINATA_PUBLIC="true"` = files accessible via any IPFS gateway
- `PINATA_PUBLIC="false"` = private storage, NOT on public IPFS network

See [IPFS Storage Guide](docs/ipfs-storage.md) for details.

### TEE Validation (Optional - Hardware-Guaranteed Validation)
Enable EigenCompute TEE for trustless, hardware-backed VLC validation:

```bash
# 1. Deploy TEE validator to EigenCompute (one-time setup)
cd tee-vlc-validator
npm install
npm run build
docker build -t your-username/tee-vlc-validator:latest .
docker push your-username/tee-vlc-validator:latest

# Deploy to EigenCompute Intel TDX
eigenx app create your-username/tee-vlc-validator:latest --env-file .env.alchemy

# 2. Configure local environment with TEE settings
cp .env.eigen.example .env.eigen
nano .env.eigen  # Set USE_TEE_VALIDATION="true" and add TEE_VALIDATOR_ENDPOINT

# 3. Run normally - TEE validation will be enabled from .env.eigen
sudo ./run-flux-mining.sh --network sepolia
```

**Benefits**:
- 🔐 Hardware-guaranteed validation (Intel TDX secure enclaves)
- 🎯 Trustless - cryptographically signed validation results
- ⚡ Single TEE instance replaces multiple validators
- 🔏 On-chain signature verification
- 🌐 Works with both local anvil and Sepolia networks

**How it works**:
1. Agent registers → validation request submitted to blockchain
2. TEE validator tests agent's VLC protocol implementation
3. TEE signs validation result with hardware-protected keys
4. Agent submits TEE signature on-chain
5. Smart contract verifies signature matches authorized TEE wallet

See [TEE Validator Documentation](tee-vlc-validator/README.md) for detailed deployment guide.

## 🎯 What is FLUX Mining?

FLUX Mining enables **Intelligence Money** through permissionless agentic coordination that evolves into a contextual causal knowledge graph. As AI agents collaborate and solve tasks, their interactions form a verifiable graph of causal dependencies. Agents that consistently generate quality outputs - as verified through this knowledge graph - are awarded FLUX tokens representing their demonstrated intelligence.

- **FLUX Tokens**: Soulbound representation of agent intelligence (non-transferable)
- **USDC Payments**: Immediate economic compensation for task completion
- **Causal Knowledge Graph**: Immutable record of agent interactions and quality

### Key Features

✅ **ERC-8004 Identity System** - NFT-based agent identities
✅ **VLC Protocol Validation** - Ensures causal consistency (TEE-enabled with Intel TDX)
✅ **x402 Payment Protocol** - HTTP-based payment standard (direct/escrow/hybrid)
✅ **BFT Consensus** - Byzantine fault-tolerant quality assessment
✅ **Reputation Tracking** - On-chain performance history
✅ **IPFS Storage (Optional)** - 95% cost reduction via Pinata

## 📋 System Overview

**[🎯 High-Level Overview →](docs/overview.md)** - Understand how FLUX Mining enables permissionless agent coordination with LangGraph-level causal graphs without vendor lock-in

| Component | Purpose | Documentation |
|-----------|---------|---------------|
| **Subnet Consensus** | VLC-based distributed AI coordination | [Architecture Guide](docs/architecture.md) |
| **ERC-8004** | Trustless agent identity, validation and reputation system | [Identity](docs/erc-8004-identity.md) / [Reputation](docs/reputation.md) |
| **VLC Validation** | Causal consistency protocol compliance | [Validation Guide](docs/vlc-validation.md) |
| **TEE Validation** | Hardware-guaranteed validation with Intel TDX (optional) | [TEE Guide](tee-vlc-validator/README.md) |
| **x402 Payments** | HTTP 402 payment protocol (direct/escrow/hybrid) | [Payment System](docs/x402-payments.md) |
| **FLUX Mining** | Intelligence-based token mining | [Mining Guide](docs/flux-mining.md) |
| **IPFS Storage** | Off-chain VLC data storage (optional) | [IPFS Guide](docs/ipfs-storage.md) |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Blockchain Networks                            │
│            (Anvil Local / Ethereum-Sepolia Testnet)                 │
└─────────────────────────────────────────────────────────────────────┘
                                  │
         ┌────────────┬───────────┼───────────┬────────────┐
         ▼            ▼           ▼           ▼            ▼
┌────────────────┐  ┌─────────┐  ┌────────┐  ┌──────────────┐
│   ERC-8004     │  │  x402   │  │  FLUX  │  │    Subnet    │
│ Identity/Valid/│  │ Escrow  │  │ Mining │  │   Registry   │
│   Reputation   │  │         │  │        │  │              │
└────────┬───────┘  └─────────┘  └────────┘  └──────────────┘
    │    │               │            │               │
    │    └───────────────┴────────────┴───────────────┘
    │                                 │
    ▼                                 ▼
┌────────────────┐           ┌───────┴────────┐
│ TEE Validator  │           │  Subnet Layer  │
│ (Optional)     │           │   (PoCW BFT)   │
│ Intel TDX      │           │                │
└────────────────┘           └───────┬────────┘
                                     │
                             ┌───────┴────────┐              ┌──────────────┐
                             │  VLC Protocol  │──VLC Data───▶│ Pinata IPFS  │
                             │ (Causal Order) │              │  (Optional)  │
                             └────────────────┘              └──────────────┘
```

## 💰 Payment Flows

### Direct Mode (Default)
```
Client → Sign Transaction → Send to Facilitator → Agent Processes Task
              ↓                      ↓                      ↓
         Local Signing        Verify Signature      After Completion
                                                            ↓
                                                  Broadcast to Chain
```

### Escrow Mode (Alternative)
```
Client → Deposit USDC → Escrow Contract → Agent Verifies → Task Processing
         ↓                                                    ↓
    Funds Locked                                    BFT Consensus (>0.5)
                                                             ↓
                                                   Release or Refund
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
- **[IPFS Storage Guide](docs/ipfs-storage.md)** - Pinata IPFS integration for VLC data (95% cost reduction)
- **[Development Guide](docs/development.md)** - Building and contributing
- **[API Reference](docs/api.md)** - HTTP bridge and contract interfaces

## 🛠️ Configuration

### Environment Files
- `.env.local` - Anvil local blockchain settings
- `.env.sepolia` - Ethereum-Sepolia testnet settings
- `.env.pinata` - Pinata IPFS configuration (optional - enables 95% cost reduction)
- `.env.eigen` - EigenCompute TEE validator configuration (optional - hardware validation)

### Key Parameters
```bash
PAYMENT_MODE=direct|escrow|hybrid  # Payment processing mode (default: direct)
NETWORK=local|sepolia              # Blockchain network (default: local)
RPC_URL=<your-rpc-endpoint>        # Ethereum RPC endpoint
USE_PINATA=true|false              # Enable IPFS storage (see Quick Start section)
PINATA_PUBLIC=true|false           # Public (any gateway) or private (your gateway only)
USE_TEE_VALIDATION=true|false      # Enable TEE hardware validation (default: false)
TEE_VALIDATOR_ENDPOINT=<tee-url>   # EigenCompute TEE validator endpoint
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
- **TEE Validation (Optional)**: Hardware-guaranteed validation with Intel TDX secure enclaves
- **Payment Verification**: On-chain escrow with BFT consensus
- **Causal Graph**: Vector Logical Clock tracking via Dgraph + IPFS
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