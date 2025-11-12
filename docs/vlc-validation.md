# VLC Validation (ERC-8004 Validation Registry)

## Overview

VLC (Vector Logical Clock) validation uses the **ERC-8004 Validation Registry** to store agent validation scores. Before any agent can register a subnet and participate in FLUX mining, it must pass VLC protocol validation - a rigorous on-chain test ensuring proper implementation of causal consistency. This validation is separate from but complementary to the ERC-8004 Identity Registry.

## Why VLC Validation Matters

### Preventing Byzantine Failures
- Agents with incorrect VLC implementations can break causal ordering
- Invalid clock increments corrupt the consensus mechanism
- Malformed event sequences compromise network integrity
- Validation ensures only protocol-compliant agents participate

## Validation Process

### 📋 Pre-Registration Testing

The system performs VLC validation BEFORE subnet registration:

```
Flow:
1. Agent registers identity NFT (ERC-8004)
2. 🔐 VALIDATION CHECKPOINT: VLC Protocol Test
   ├─ Validator sends ambiguous task to agent
   ├─ Agent must correctly implement NeedMoreInfo flow
   ├─ VLC clock must increment properly (+2 per message exchange)
   └─ Validator submits score (0-100) to ValidationRegistry
3. Validator submits score to blockchain (permanent record)
4. Smart contract verifies: getSummary(agentId) >= 70/100
5. ✅ Only if validation passes → Subnet registration allowed
```

## On-Chain Score Recording (ERC-8004)

The validation score is permanently stored in the **ERC-8004 ValidationRegistry** smart contract, which is a core component of the ERC-8004 identity system:

### ERC-8004 ValidationRegistry Interface

```solidity
// Validator submits their assessment
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

## Pass/Fail Criteria

SubnetRegistry.sol validation check:

```solidity
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

## Validation Demo Output

```bash
📋 VLC Protocol Validation

   📋 Validator submitting validation...
      ✅ Validator: Score 100/100 recorded
         ✓ Response confirmed on-chain

   📊 Validation Summary:
      Agent ID: #0
      🔍 Calling ValidationRegistry.getSummary...
      📝 Raw getSummary response: 1
100
      📊 Parsed values: count=1, avgScore=100
      Validators: 1
      Average Score: 100/100
      Status: ✅ PASSED

🔐 Registering subnet with Agent ID 0...
   The SubnetRegistry contract will verify:
   ✓ Agent owns the identity token
   ✓ Agent has passed VLC validation (score >= 70)
```

## Security Benefits

### 🛡️ Network Protection
- **Protocol Enforcement**: Only VLC-compliant agents can participate
- **Permanent Record**: Validation scores stored on-chain forever
- **No Bypass**: Smart contract enforces validation before registration
- **Sybil Resistance**: Each agent ID validated independently

### 🔒 Trust Minimization
- **On-Chain Verification**: No off-chain trust required
- **Validator Assessment**: Validator assesses agent compliance with VLC protocol
- **Cryptographic Proof**: Blockchain provides immutable validation record
- **Permissionless**: Any validator can assess any agent

## VLC Implementation Requirements

### Clock Management
- Each node maintains its own logical clock
- Clock increments by 2 for each message exchange
- Clock values must be consistent with causal ordering

### Message Flow
1. **UserInput**: Initial task submission
2. **MinerOutput**: Agent response or NeedMoreInfo request
3. **InfoRequest**: Request for additional context
4. **InfoResponse**: User-provided clarification
5. **RoundSuccess/Failed**: Consensus outcome

### Event Dependencies
- Every event must reference its causal dependencies
- Parent links show causal relationships
- Invalid dependencies result in validation failure

## Blockchain Inspector Integration

The blockchain inspector displays validation status:
- View validation request for each agent
- Check validator score
- Verify VLC_PROTOCOL tag compliance
- Monitor score and pass/fail status

## Related Documentation

- [ERC-8004 Identity](erc-8004-identity.md) - Agent identity system
- [Architecture](architecture.md) - System architecture overview
- [Smart Contracts](contracts.md) - Contract addresses and interfaces