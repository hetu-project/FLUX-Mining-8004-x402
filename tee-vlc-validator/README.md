# TEE VLC Validator

Hardware-backed VLC validation service for FLUX Mining using EigenCompute Trusted Execution Environment (Intel TDX).

## Overview

This service validates VLC (Vector Logical Clock) sequences for AI agents in a tamper-proof TEE, ensuring:
- **Verifiable Execution**: Intel TDX hardware attestation
- **Trustless Validation**: No need to trust validators
- **Causal Consistency**: Validates clock increments and event ordering
- **On-chain Recording**: Submits validation scores to ValidationRegistry

## Quick Start

### Prerequisites
- EigenX CLI installed
- Docker logged in
- Sepolia ETH for transactions

### 1. Install EigenX CLI

```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash
```

### 2. Authenticate

```bash
# Generate new wallet or use existing
eigenx auth generate --store

# Check wallet address
eigenx auth whoami

# Subscribe to EigenCompute
eigenx billing subscribe
```

### 3. Configure Environment

```bash
cp .env.example .env
nano .env
```

Update:
- `RPC_URL`: Your Sepolia RPC endpoint
- `VALIDATION_REGISTRY_ADDRESS`: Your deployed ValidationRegistry address

### 4. Deploy to TEE

```bash
eigenx app deploy
```

This will:
1. Build Docker image for linux/amd64
2. Deploy to Intel TDX TEE
3. Return instance IP and app ID

### 5. Test the Deployment

```bash
# Get instance info
eigenx app info

# Check health
curl http://YOUR_INSTANCE_IP:3000/health

# View logs
eigenx app logs
```

## API Endpoints

### POST /validate-agent

Validates agent's VLC state and submits score to ValidationRegistry.

**Request:**
```json
{
  "agentId": "123",
  "agentAddress": "0x...",
  "previousState": {
    "clock": { "123": 0, "456": 0 },
    "events": []
  },
  "currentState": {
    "clock": { "123": 1, "456": 0 },
    "events": ["event1"]
  }
}
```

**Response:**
```json
{
  "success": true,
  "agentId": "123",
  "validation": {
    "valid": true,
    "score": 90,
    "feedback": "VLC validation passed - causal consistency maintained"
  },
  "blockchain": {
    "requestTx": "0x...",
    "responseTx": "0x..."
  },
  "tee": {
    "wallet": "0x...",
    "attestation": "verified-by-eigencompute-tee"
  }
}
```

### GET /health

Health check endpoint.

### GET /info

Service information and capabilities.

## Local Development

```bash
# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Run locally (for testing only - not in TEE)
npm run dev
```

## Update Deployment

After making changes:

```bash
eigenx app upgrade
```

## Monitoring

```bash
# View real-time logs
eigenx app logs

# Check app status
eigenx app info
```

## Security

- **Private Key**: Managed by EigenX KMS, never exposed
- **TEE Isolation**: Code runs in Intel TDX secure enclave
- **Hardware Attestation**: Every validation includes TEE proof
- **Immutable Execution**: Deployed code cannot be tampered with

## Integration with FLUX Mining

The FLUX Mining system calls this TEE validator via `.env.eigen` configuration:

```bash
USE_TEE_VALIDATION="true"
TEE_VALIDATOR_ENDPOINT="http://your-instance-ip:3000"
```

When enabled, `run-flux-mining.sh` sends VLC states to this TEE for validation instead of local validation.
