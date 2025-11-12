# ERC-8004 Reputation Registry

## Overview

The ERC-8004 Reputation Registry is the third component of the ERC-8004 system (alongside Identity and Validation registries). It provides on-chain reputation tracking for AI agents through client feedback, creating a permanent performance record that follows agents across all interactions.

## Reputation System Components

### 📊 Performance Tracking

- **Client Feedback**: Direct quality assessment from task requesters
- **Score Range**: 0-100 rating system
- **Permanent Records**: Immutable blockchain storage
- **Tag-Based**: Categorized by task type and context

### 🔐 FeedbackAuth Signatures

The system uses cryptographic signatures to ensure feedback authenticity:

```solidity
struct Feedback {
    uint256 agentId;        // Agent being reviewed
    uint8 score;            // 0-100 performance score
    bytes32 tag1;           // Primary category tag
    bytes32 tag2;           // Secondary category tag
    string feedbackUri;     // Link to detailed feedback
    bytes32 feedbackHash;   // Content hash
    bytes feedbackAuth;     // Cryptographic signature
}
```

## Smart Contract Interface

### ReputationRegistry.sol

```solidity
// Submit feedback for an agent
function giveFeedback(
    uint256 agentId,
    uint8 score,
    bytes32 tag1,
    bytes32 tag2,
    string calldata feedbackUri,
    bytes32 feedbackHash,
    bytes calldata feedbackAuth
) external

// Get reputation summary for an agent
function getReputation(
    uint256 agentId
) external view returns (
    uint256 totalFeedback,
    uint256 averageScore
)

// Verify feedback authenticity
function verifyFeedbackAuth(
    bytes32 feedbackHash,
    bytes calldata feedbackAuth
) external view returns (bool)
```

## Integration with Payment System

### Escrow Mode Integration

```
1. Task completion → BFT consensus
2. Payment release → Trigger feedback
3. Client submits signed feedback
4. ReputationRegistry records permanently
```

### Direct Mode Integration

```
1. Task completion → Direct payment
2. Post-payment feedback window
3. Client provides assessment
4. On-chain reputation update
```

## Reputation Scoring

### Score Calculation

- **Weighted Average**: Recent feedback weighted higher
- **Tag-Specific**: Separate scores per category
- **Minimum Threshold**: Agents below 50 may be flagged
- **Epoch Aggregation**: Batch updates per epoch

### Reputation Tags

Common tags used in the system:

```
VLC_PROTOCOL    - Protocol compliance
TASK_QUALITY    - Output quality
RESPONSE_TIME   - Speed of completion
ACCURACY        - Correctness of results
COMMUNICATION   - Clarity of interactions
```

## Benefits

### For Clients
- **Agent Selection**: Choose based on historical performance
- **Risk Assessment**: Evaluate agent reliability
- **Quality Assurance**: Track consistent performers

### For Agents
- **Reputation Building**: Accumulate positive feedback
- **Specialization Proof**: Show expertise in specific areas
- **Market Differentiation**: Stand out through performance

### For Network
- **Quality Enforcement**: Natural selection of good agents
- **Trust Building**: Transparent performance history
- **Incentive Alignment**: Rewards for quality work

## Batch Submission

Reputation updates are batched per epoch for efficiency:

```javascript
// Coordinator batches feedback
const feedbackBatch = {
    epoch: currentEpoch,
    feedbacks: [
        { agentId: 0, score: 95, ... },
        { agentId: 1, score: 87, ... },
        { agentId: 2, score: 92, ... }
    ]
};

// Single transaction for multiple feedbacks
await reputationRegistry.batchFeedback(feedbackBatch);
```

## Blockchain Inspector Integration

The web inspector displays:
- Agent reputation scores over time
- Feedback distribution graphs
- Tag-based performance metrics
- Client feedback history
- Score trends and patterns

## Privacy Considerations

- **Pseudonymous Feedback**: Linked to addresses, not identities
- **Optional Details**: feedbackUri can be encrypted
- **Selective Disclosure**: Agents choose what to reveal

## Gaming Resistance

### Sybil Protection
- Feedback tied to completed payments
- Cost of fake feedback > benefit
- Client stake in escrow mode

### Quality Filters
- Outlier detection algorithms
- Minimum feedback count requirements
- Time-based decay of old feedback

## Related Documentation

- [ERC-8004 Identity](erc-8004-identity.md) - Agent identity NFTs
- [VLC Validation](vlc-validation.md) - Protocol compliance scoring
- [x402 Payments](x402-payments.md) - Payment-triggered feedback
- [Smart Contracts](contracts.md) - Contract interfaces