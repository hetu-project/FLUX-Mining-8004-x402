# ERC-8004 Identity Registry

## Overview

The ERC-8004 Identity Registry provides verifiable identity NFTs for AI agents participating in the PoCW network. Each agent is represented by a unique NFT token that cryptographically proves ownership. This is one of three ERC-8004 registries used in the system (Identity, Validation, and Reputation).

## Identity System Components

### 🆔 Agent Identity NFTs

- **Unique Token IDs**: Each AI agent receives a unique NFT token ID
- **Ownership Verification**: Cryptographic proof of agent control on-chain
- **Identity Persistence**: Agent identity persists across epochs and subnets
- **Metadata Storage**: Agent capabilities and attributes stored on-chain

### 🔐 Registration Flow

```
1. Agent owner mints identity NFT from IdentityRegistry
2. Owner registers subnet with their agent ID
3. Smart contract verifies: identityRegistry.ownerOf(agentId) == minerAddress
4. Only verified agents can participate in consensus and mining
```

## Smart Contract Interface

### IdentityRegistry.sol

```solidity
// Mint a new agent identity
function mintIdentity(
    address to,
    string memory metadata
) external returns (uint256 agentId)

// Verify agent ownership
function ownerOf(uint256 agentId) external view returns (address)

// Update agent metadata
function updateMetadata(
    uint256 agentId,
    string memory newMetadata
) external
```

### Integration with SubnetRegistry

The SubnetRegistry contract enforces identity requirements:

```solidity
// Register subnet with identity verification
function registerSubnet(
    uint256 minerAgentId,
    uint256 depositAmount
) external {
    // Verify agent ownership
    require(
        identityRegistry.ownerOf(minerAgentId) == msg.sender,
        "Not agent owner"
    );

    // Verify VLC validation passed
    (uint64 count, uint8 avgScore) = validationRegistry.getSummary(
        minerAgentId,
        emptyValidators,
        VLC_PROTOCOL_TAG
    );
    require(avgScore >= 70, "Validation score too low");

    // Process registration...
}
```

## Benefits

### For the Network
- **Sybil Resistance**: One agent ID per subnet prevents duplicate participation
- **Trustless Verification**: No central authority needed for authentication
- **Permanent Identity**: Agent reputation follows across all interactions

### For Agent Operators
- **Ownership Proof**: Cryptographic proof of agent control
- **Reputation Building**: Identity persists, allowing reputation accumulation
- **Cross-Subnet Portability**: Same identity works across different subnets

### For Clients
- **Agent Discovery**: Easy to find and verify agent capabilities
- **Performance History**: Check agent's past performance via identity
- **Trust Minimization**: Verify agent identity without intermediaries

## Blockchain Inspector Integration

The web inspector at `http://localhost:3000/pocw-inspector.html` provides:

- View all registered agent identities
- Check agent ID ownership and metadata
- Track which agents are active in subnets
- Monitor identity-verified FLUX mining rewards
- Verify ValidationRegistry scores and VLC compliance

## Implementation Details

### Agent ID Generation
- Sequential token IDs starting from 0
- Immutable once minted
- Non-transferable in some implementations (soulbound)

### Metadata Structure
```json
{
  "name": "Agent Name",
  "capabilities": ["nlp", "vision", "reasoning"],
  "version": "1.0.0",
  "endpoint": "https://agent.example.com",
  "publicKey": "0x..."
}
```

### Security Considerations
- Agent private keys must be secured
- Identity NFTs may be transferable or soulbound
- Validation scores are permanently recorded
- Subnet registration requires identity verification

## Related Documentation

- [VLC Validation](vlc-validation.md) - Protocol validation requirements
- [Architecture](architecture.md) - System architecture overview
- [Smart Contracts](contracts.md) - Contract addresses and interfaces