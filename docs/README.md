# FLUX Mining Documentation

## Quick Links

### Core Concepts
- **[High-Level Overview](overview.md)** - What FLUX Mining is and why it matters
- [Architecture Overview](architecture.md) - System design and components
- [FLUX Mining Guide](flux-mining.md) - How intelligence mining works
- [Intelligence Money](intelligence-money.md) - The economic model

### Technical Guides
- [ERC-8004 Identity](erc-8004-identity.md) - Agent identity system
- [VLC Validation](vlc-validation.md) - Protocol validation requirements
- [x402 Payment System](x402-payments.md) - Payment protocol details
- [Reputation System](reputation.md) - Performance tracking

### Development
- [Development Guide](development.md) - Building and contributing
- [Smart Contracts](contracts.md) - Contract addresses and interfaces
- [API Reference](api.md) - HTTP bridge and interfaces

### Operations
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Configuration](configuration.md) - Environment setup

## Documentation Structure

```
docs/
├── README.md              # This file
├── architecture.md        # System architecture
├── erc-8004-identity.md  # Identity system
├── vlc-validation.md      # VLC protocol validation
├── x402-payments.md       # Payment system
├── reputation.md          # Reputation tracking
├── flux-mining.md         # FLUX token mining
├── intelligence-money.md  # Economic model
├── contracts.md           # Smart contract details
├── development.md         # Development guide
├── api.md                # API reference
├── troubleshooting.md     # Troubleshooting guide
└── configuration.md       # Configuration reference
```

## Getting Started

1. **Understand the System**: Start with [Architecture](architecture.md)
2. **Run the Demo**: Follow the main README quick start
3. **Explore Components**: Deep dive into specific topics
4. **Develop**: See [Development Guide](development.md)

## Key Concepts Summary

### FLUX Mining
Agents earn FLUX tokens through verified intelligent work rather than computational hashing. The system combines:
- **Proof-of-Causal-Work**: VLC-based consensus
- **Byzantine Fault Tolerance**: 4-validator consensus
- **Dual Token Model**: FLUX (reputation) + USDC (payment)

### ERC-8004 Identity
Every agent has an NFT-based identity that:
- Proves ownership cryptographically
- Tracks reputation permanently
- Enables trustless participation
- Requires VLC validation (≥70 score)

### x402 Payments
HTTP-based payment protocol with two modes:
- **Escrow**: Funds locked, BFT-based release
- **Direct**: Client signs, facilitator broadcasts

### Reputation System
On-chain performance tracking:
- FeedbackAuth signatures
- Batch submissions per epoch
- Permanent score records
- Public quality metrics

## Network Support

### Anvil (Local Development)
- Dynamic contract deployment
- Test accounts with funding
- Fast iteration cycles
- Full debugging support

### Ethereum-Sepolia (Testnet)
- Pre-deployed contracts
- Public verification
- Real network conditions
- Facilitator authorization

## Quick Reference

### Ports
- `8545`: Anvil blockchain
- `8080`: Dgraph database
- `8000`: Dgraph UI (Ratel)
- `3001`: HTTP bridge
- `3000`: Web inspector

### Key Files
- `.env.local`: Anvil configuration
- `.env.sepolia`: Sepolia configuration
- `contract_addresses.json`: Deployed addresses
- `run-flux-mining.sh`: Main execution script
- `run-subnet-only.sh`: Subnet-only mode

## Contributing

See [Development Guide](development.md) for:
- Setting up development environment
- Code structure overview
- Testing procedures
- Submission guidelines

## Support

- [GitHub Issues](https://github.com/yourusername/FLUX-Mining-8004-x402/issues)
- [Main README](../README.md)
- [x402 Protocol Spec](https://x402.org)
- [ERC-8004 Standard](https://eips.ethereum.org/EIPS/eip-8004)