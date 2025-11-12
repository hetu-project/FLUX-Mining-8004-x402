# Smart Contracts

## Contract Overview

| Contract | Purpose | Key Features |
|----------|---------|--------------|
| **FLUXToken** | Soulbound intelligence tokens | Non-transferable, 21M max supply |
| **HETUToken** | Staking for subnet registration | ERC20, 1M total supply |
| **USDC** | Payment stablecoin | 6 decimals, USD-pegged |
| **x402PaymentEscrow** | Trustless payment escrow | BFT consensus-based release/refund |
| **SubnetRegistry** | Manages subnet participants | ERC-8004 identity verification |
| **PoCWVerifier** | Consensus verification & mining | Per-epoch FLUX distribution |
| **IdentityRegistry** | ERC-8004 Trustless Agents | NFT-based agent IDs |
| **ValidationRegistry** | Agent validation | VLC protocol compliance (≥70 score) |
| **ReputationRegistry** | Agent reputation | Client feedback with FeedbackAuth |

## Contract Addresses

### Anvil (Local Blockchain)

Contracts are **deployed dynamically at runtime** when you run `./run-flux-mining.sh`. Addresses are generated deterministically based on the deployer nonce.

To find current addresses after deployment:
```bash
# Check the generated addresses file
cat contract_addresses.json
```

Note: Addresses will be different each time you restart Anvil since the deployment nonce resets.

### Ethereum-Sepolia Testnet

Pre-deployed contracts on Sepolia:

```json
{
  "IdentityRegistry": "0x8004a6090Cd10A7288092483047B097295Fb8847",
  "ReputationRegistry": "0x8004B8FD1A363aa02fDC07635C0c5F94f6Af5B7E",
  "ValidationRegistry": "0x8004CB39f29c09145F24Ad9dDe2A108C1A2cdfC5",
  "USDC": "0x736C14F6873E54c9A1a215c534f32CF4e010B47b",
  "FLUXToken": "0x8E3B2a73CDcB914bc4DE3316aDA73344F03339d5",
  "HETUToken": "0x3a55eb85486EB637215d2191079B99c47Cf845BC",
  "PoCWVerifier": "0x9C834BD459273183DEE1296D803BbCf128576ea8",
  "SubnetRegistry": "0xcd7Ce71893a679DE8F1480E7284896DDFcc2245b",
  "x402PaymentEscrow": "0xB07f985E44fF4c7EA0ad7baeeaE95982ECb0AA57"
}
```

## Key Contract Interfaces

### FLUXToken.sol
```solidity
// Soulbound - cannot be transferred
function transfer(address, uint256) public pure override returns (bool) {
    revert("FLUX tokens are soulbound");
}

// Mining function
function mine(address recipient, uint256 amount) external onlyMiner
```

### x402PaymentEscrow.sol
```solidity
struct TaskPayment {
    address client;
    address agent;
    uint256 amount;
    uint256 deadline;
    PaymentStatus status;
}

function depositPayment(
    bytes32 taskId,
    address client,
    address agent,
    uint256 amount,
    uint256 deadline
) external onlyCoordinator

function releasePayment(bytes32 taskId) external onlyCoordinator
function refundPayment(bytes32 taskId) external onlyCoordinator
```

### SubnetRegistry.sol
```solidity
function registerSubnet(
    uint256 minerAgentId,
    uint256 depositAmount
) external

function finalizeEpoch(
    uint256 subnetId,
    uint256 epochNumber,
    bytes32 merkleRoot,
    uint256 totalReward
) external
```

### IdentityRegistry.sol
```solidity
function mintIdentity(
    address to,
    string memory metadata
) external returns (uint256)

function ownerOf(uint256 agentId) external view returns (address)
```

### ValidationRegistry.sol
```solidity
function validationResponse(
    bytes32 requestHash,
    uint8 response,
    string calldata responseUri,
    bytes32 responseHash,
    bytes32 tag
) external

function getSummary(
    uint256 agentId,
    address[] calldata validatorAddresses,
    bytes32 tag
) external view returns (uint64 count, uint8 avgResponse)
```

### ReputationRegistry.sol
```solidity
function giveFeedback(
    uint256 agentId,
    uint8 score,
    bytes32 tag1,
    bytes32 tag2,
    string calldata feedbackUri,
    bytes32 feedbackHash,
    bytes calldata feedbackAuth
) external
```

## Deployment Process

### Local Deployment (Anvil)

1. Start Anvil: `anvil`
2. Run deployment script: `./run-flux-mining.sh`
3. Contracts deployed via Foundry forge
4. Addresses saved to `contract_addresses.json`

### Sepolia Deployment

Contracts are pre-deployed. To interact:

1. Set RPC_URL in `.env.sepolia`
2. Fund accounts with Sepolia ETH
3. Authorize facilitator if needed
4. Run with Sepolia configuration

## Gas Optimization

- Batch operations where possible
- Minimal storage updates
- Event emission for off-chain indexing
- Efficient data structures

## Security Features

- Reentrancy guards on payment functions
- Access control (onlyCoordinator, onlyOwner)
- Time-based deadlines
- Signature verification
- Validation score requirements

## Verification

### Etherscan Verification (Sepolia)
```bash
forge verify-contract \
  --chain-id 11155111 \
  --num-of-optimizations 200 \
  --compiler-version v0.8.19 \
  CONTRACT_ADDRESS \
  CONTRACT_NAME
```

### Local Testing
```bash
forge test -vvv
```

## Related Documentation

- [Architecture](architecture.md) - System overview
- [x402 Payments](x402-payments.md) - Payment details
- [Development Guide](development.md) - Building contracts