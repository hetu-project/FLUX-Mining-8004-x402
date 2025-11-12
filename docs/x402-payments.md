# x402 Payment System

## Overview

The x402 protocol enables trustless, HTTP-based payments between clients and AI agents using USDC stablecoin. The system supports two modes: escrow-based payments with BFT consensus and direct payments following the HTTP 402 Payment Required standard.

## Payment Modes

### Escrow Mode (Default)

Funds are locked in a smart contract and released/refunded based on validator consensus.

#### Flow Diagram
```
┌─────────────────────────────────────────────────────────────────────┐
│                    x402 Payment Escrow Lifecycle                     │
└─────────────────────────────────────────────────────────────────────┘

1️⃣ PAYMENT REQUEST → Client receives payment details from agent
2️⃣ DEPOSIT PHASE → Client deposits USDC to escrow contract
3️⃣ VERIFICATION → Agent verifies payment on-chain
4️⃣ PROCESSING → Agent processes task with validators
5️⃣ CONSENSUS → BFT vote on quality (threshold: 0.5)
6️⃣ FINALIZATION → Release payment or refund based on outcome
```

#### Key Features
- **Trustless Operation**: Smart contract manages funds
- **BFT Consensus**: Validators determine payment outcome
- **User Veto Power**: Client can reject even if validators approve
- **Automatic Refund**: Failed tasks trigger automatic refunds

### Direct Payment Mode

Client signs transactions that are broadcast by the facilitator after task completion.

#### Flow Diagram
```
┌─────────────────────────────────────────────────────────────────────┐
│                    x402 Direct Payment Lifecycle                     │
└─────────────────────────────────────────────────────────────────────┘

1️⃣ Task Request → Client sends HTTP request for AI task
2️⃣ 402 Response → Agent returns payment requirements
3️⃣ Sign Payment → Client signs USDC transfer locally
4️⃣ X-PAYMENT Header → Client resubmits with signed transaction
5️⃣ Verification → Facilitator validates signature
6️⃣ Task Processing → Agent processes with payment commitment
7️⃣ Settlement → Facilitator broadcasts after completion
8️⃣ Delivery → Result with X-PAYMENT-RESPONSE header
```

#### Key Features
- **Client Control**: Transactions signed locally
- **HTTP Standard**: Follows HTTP 402 Payment Required
- **Fast Processing**: No blockchain wait before task starts
- **Post-Task Settlement**: Payment settles after completion

## Smart Contracts

### x402PaymentEscrow.sol

Core functions for escrow mode:

```solidity
// Deposit payment for a task (via coordinator)
function depositPayment(
    bytes32 taskId,
    address client,
    address agent,
    uint256 amount,
    uint256 deadline
) external onlyCoordinator

// Release payment to agent after consensus
function releasePayment(bytes32 taskId) external onlyCoordinator

// Refund payment to client if rejected
function refundPayment(bytes32 taskId) external onlyCoordinator

// Check payment status
mapping(bytes32 => TaskPayment) public payments;
```

### Payment States
- `NONE`: No payment exists
- `DEPOSITED`: Funds locked in escrow
- `RELEASED`: Payment sent to agent
- `REFUNDED`: Payment returned to client

## USDC Token Details

- **Symbol**: USDC
- **Decimals**: 6 (matching official USDC specification)
- **Standard**: ERC-20
- **Amount Format**: 10 USDC = 10000000 (10 * 10^6)

## Payment Request Format

```json
{
  "taskId": "req-subnet-001-1",
  "amount": "10000000",  // 10 USDC (with 6 decimals)
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

## Consensus Decision Logic

```go
if qualityScore > 0.5 && userAccepted {
    // ✅ Release payment from escrow to agent
    releasePayment(taskId)
} else {
    // ↩️ Refund payment from escrow to client
    refundPayment(taskId)
}
```

## Demonstrated Scenarios

The system handles 7 test scenarios:

### Successful Payments (Tasks 1, 2, 3, 5, 7)
- Quality Score: > 0.5
- User Feedback: Accepted
- Result: 10 USDC released to agent

### Refunded Payments
- **Task 4**: Quality < 0.5 (validator rejection)
- **Task 6**: Quality > 0.5 but user rejected

### Final Balances
```
Client:  1000 → 950 USDC (-50 paid, +20 refunded)
Agent:   0 → 50 USDC (5 successful tasks)
```

## Security Features

### For Clients
- ✅ Funds locked in escrow or signed commitment
- ✅ Automatic refund if quality too low
- ✅ Veto power even if validators approve
- ✅ Time-based deadline protection

### For Agents
- ✅ Guaranteed payment for quality work
- ✅ Objective quality assessment by validators
- ✅ Protection against malicious client rejection
- ✅ Cryptographic proof of task completion

### For the Network
- ✅ No trusted intermediary needed
- ✅ Validators earn rewards for honest consensus
- ✅ Complete payment audit trail on-chain
- ✅ Reentrancy protection in contracts

## Contract Addresses

### Anvil (Local Blockchain)
- Generated dynamically at deployment
- Check `contract_addresses.json` after running

### Ethereum-Sepolia Testnet
- x402PaymentEscrow: `0xB07f985E44fF4c7EA0ad7baeeaE95982ECb0AA57`
- USDC: `0x736C14F6873E54c9A1a215c534f32CF4e010B47b`

## Integration with FLUX Mining

The x402 payment system operates in parallel with FLUX token mining:

```
Successful Task Completion:
  ├─ FLUX Tokens: Mined for verifiable intelligence work (soulbound)
  └─ USDC Payment: Released from escrow or settled directly
```

This dual-token model ensures agents build reputation (FLUX) while earning liquid compensation (USDC).

## Related Documentation

- [Architecture](architecture.md) - System architecture overview
- [Smart Contracts](contracts.md) - Complete contract details
- [API Reference](api.md) - HTTP bridge interface