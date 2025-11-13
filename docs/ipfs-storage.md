# IPFS Storage for VLC Graph Data

## Overview

The FLUX Mining system supports storing VLC (Vector Logical Clock) graph data on IPFS (InterPlanetary File System) using Pinata as the storage provider. This significantly reduces on-chain storage costs by storing only IPFS URIs on-chain instead of full JSON data.

## Benefits

### Cost Reduction
- **Traditional**: ~2,000-4,000 bytes per epoch (full JSON on-chain)
- **IPFS Mode**: ~60-80 bytes per epoch (IPFS URI only)
- **Savings**: ~95-98% reduction in on-chain storage costs

### Data Availability
- VLC graph data remains permanently accessible via IPFS
- Multiple retrieval methods: IPFS protocol, Pinata gateway, or any IPFS gateway
- Distributed storage ensures data resilience

### Transparency
- All VLC data **publicly accessible** via IPFS (no authentication required)
- Smart contract stores IPFS CID for integrity verification
- Gateway URLs provide easy HTTP access
- Files uploaded with `groupId: null` for public access

## Configuration

### 1. Environment Setup

**IMPORTANT: Use your own credentials!**

The repository includes `.env.pinata.example` as a template. Copy it and add your own Pinata credentials:

```bash
# Copy the example file
cp .env.pinata.example .env.pinata

# Edit with your credentials
nano .env.pinata
```

Get your Pinata credentials from https://app.pinata.cloud/developers/api-keys

Example `.env.pinata` configuration:

```bash
# Enable IPFS storage
USE_PINATA="true"

# Public/Private Access Control
# "true" = files accessible via any IPFS gateway (recommended for transparency)
# "false" = files only accessible via authenticated Pinata gateway
PINATA_PUBLIC="true"

# Pinata API Credentials (get from https://app.pinata.cloud/)
JWT_SECRET_ACCESS="your_jwt_token_here"

# Pinata Gateway (from your Pinata gateway settings)
GATEWAY_PINATA="your-gateway.mypinata.cloud"
```

**Note:** `.env.pinata` is in `.gitignore` to prevent credential leaks.

**🔐 Privacy Control with Pinata v3 Files API**

The `PINATA_PUBLIC` flag provides **TRUE** privacy control using Pinata's v3 Files API:

| Setting | Access Control | Details |
|---------|----------------|---------|
| `true` | ✅ Public IPFS | File uploaded to public IPFS with `network: "public"`<br/>Accessible via any gateway (ipfs.io, cloudflare-ipfs.com, etc.) |
| `false` | 🔒 Private IPFS | File uploaded to private IPFS (no network parameter)<br/>Only accessible via authenticated Pinata gateway<br/>Requires your JWT token to access |

**How It Works**:

- **Public Mode** (`PINATA_PUBLIC="true"`):
  - File uploaded with `network: "public"` parameter
  - Content pinned to public IPFS network
  - Anyone with CID can access from any gateway
  - Use for transparency and maximum availability

- **Private Mode** (`PINATA_PUBLIC="false"`):
  - File uploaded to private IPFS (network parameter omitted)
  - Content stored in Pinata's private storage (NOT on public IPFS)
  - NOT accessible via public IPFS gateways (ipfs.io, cloudflare, etc.)
  - Retrieve via Pinata API with JWT authentication
  - Note: Your dedicated gateway will show ERR_ID:00006 (this is expected)
  - Use for sensitive VLC data

**For Even More Security**:

1. **Encryption** (Maximum Security):
   ```javascript
   const encrypted = await encrypt(vlcGraphData, secretKey);
   await uploadToPinata(encrypted); // Upload encrypted data
   ```

2. **Access Control Lists**:
   - Pinata Enterprise feature
   - Fine-grained access control per file
   - Time-limited access tokens

### 2. Running with IPFS Storage

The system automatically loads Pinata configuration when running:

```bash
# Will automatically load .env.pinata if present
sudo ./run-flux-mining.sh

# For Sepolia testnet
sudo ./run-flux-mining.sh --network sepolia
```

### 3. Disabling IPFS Storage

To use traditional on-chain storage:

```bash
# Option 1: Set USE_PINATA to false
USE_PINATA="false"

# Option 2: Remove or rename .env.pinata
mv .env.pinata .env.pinata.disabled
```

## Data Structure

### IPFS-Stored JSON

Each epoch's VLC graph data is stored as JSON on IPFS:

```json
{
  "epochNumber": 1,
  "subnetId": "subnet-1",
  "vlcClockState": {
    "1": 6,
    "2": 6
  },
  "detailedRounds": [
    {
      "roundNumber": 1,
      "requestId": "req-subnet-1-1-...",
      "userInput": "Analyze market trends for Q4",
      "minerOutput": "Analyzed your request...",
      "consensusResult": "ACCEPTED",
      "userFeedback": "This looks good, thank you!",
      "success": true,
      "vlcClockState": {"1": 2, "2": 2}
    }
    // ... rounds 2 and 3
  ],
  "epochEventId": "epoch_1_event_subnet-1",
  "parentRoundEventId": "round_3_complete_subnet-1",
  "timestamp": 1762910120
}
```

### On-Chain Data

The smart contract stores only the IPFS URI:

```solidity
struct EpochSubmission {
    bytes32 subnetId;
    uint256 epochNumber;
    bytes vlcGraphData;        // "ipfs://Qm..." (60-80 bytes)
    address[] successfulMiners;
    uint256 successfulTasks;
    uint256 failedTasks;
    uint256 timestamp;
    bool verified;
    bool rewardsDistributed;
    address submittingValidator;
}
```

## Accessing VLC Data

### Method 1: IPFS Protocol

```bash
# Using IPFS CLI
ipfs cat Qm...

# Using IPFS gateway
curl https://ipfs.io/ipfs/Qm...
```

### Method 2: Pinata Gateway (Fastest)

```bash
# Direct HTTP access
curl https://coffee-defiant-raccoon-829.mypinata.cloud/ipfs/Qm...

# Browser access
https://coffee-defiant-raccoon-829.mypinata.cloud/ipfs/Qm...
```

### Method 3: Extract from Blockchain

```javascript
const PoCWVerifier = new ethers.Contract(verifierAddress, abi, provider);

// Get epoch submission
const epoch = await PoCWVerifier.epochSubmissions(subnetId, epochNumber);

// Extract IPFS URI from vlcGraphData bytes
const ipfsUri = ethers.toUtf8String(epoch.vlcGraphData);
// Result: "ipfs://Qm..."

// Convert to gateway URL
const cid = ipfsUri.replace('ipfs://', '');
const gatewayUrl = `https://coffee-defiant-raccoon-829.mypinata.cloud/ipfs/${cid}`;

// Fetch the data
const response = await fetch(gatewayUrl);
const vlcData = await response.json();
```

## Pinata Metadata

Each uploaded file includes metadata for easy searching:

```json
{
  "name": "Epoch 1 VLC Graph",
  "keyvalues": {
    "epochNumber": "1",
    "subnetId": "subnet-1",
    "timestamp": "1762910120",
    "type": "vlc-graph-data"
  }
}
```

## Upload Process

The system automatically handles IPFS uploads:

```
1. Epoch completes (3 rounds done)
   │
2. Go subnet sends epoch data to bridge
   │
3. Bridge checks USE_PINATA flag
   │
4. If enabled:
   ├─→ Create JSON with VLC graph data
   ├─→ Upload to Pinata via API
   ├─→ Receive IPFS CID
   ├─→ Convert to IPFS URI (ipfs://Qm...)
   ├─→ Encode URI as bytes
   └─→ Submit to PoCWVerifier contract
   │
5. Smart contract stores IPFS URI
   │
6. FLUX tokens mined and distributed
```

## Gas Cost Comparison

### Epoch 1 Example

**Traditional (full JSON on-chain)**:
- Data size: 2,048 bytes
- Gas cost: ~500,000-600,000 gas
- At 30 gwei: ~0.015-0.018 ETH

**IPFS Mode**:
- Data size: 67 bytes (IPFS URI)
- Gas cost: ~250,000-300,000 gas
- At 30 gwei: ~0.0075-0.009 ETH
- **Savings: ~50% gas reduction**

### 7 Epochs (Full Demo)

**Traditional**:
- Total gas: ~3,500,000-4,200,000
- Cost: ~0.105-0.126 ETH

**IPFS Mode**:
- Total gas: ~1,750,000-2,100,000
- Cost: ~0.053-0.063 ETH
- **Savings: ~50% or 0.05-0.06 ETH**

## Security Considerations

### Data Integrity
- IPFS CIDs are content-addressed (hash of content)
- Any tampering changes the CID
- Smart contract stores the original CID for verification

### Data Availability
- Pinata provides guaranteed storage with SLA
- Data pinned across multiple IPFS nodes
- Can be re-pinned to other IPFS services

### Privacy
- All VLC data is public on IPFS
- Do not include sensitive information in task inputs/outputs
- IPFS is immutable - data cannot be deleted

## Troubleshooting

### Upload Fails

```bash
# Check JWT token
echo $JWT_SECRET_ACCESS

# Verify token not expired (check exp claim)

# Test Pinata API directly
curl -X POST "https://uploads.pinata.cloud/v3/files" \
  -H "Authorization: Bearer $JWT_SECRET_ACCESS" \
  -F "file=@test.json"
```

### ERR_ID:00006 - "Gateway does not have this content pinned"

**This is EXPECTED behavior for private files!**

When `PINATA_PUBLIC="false"`, files are stored in Pinata's private storage and are NOT pinned to the public IPFS network. Your dedicated gateway will show ERR_ID:00006 because it only serves files pinned to your account on public IPFS.

**To verify private files are working correctly:**
```bash
# This should FAIL (returns timeout or 404)
curl https://ipfs.io/ipfs/<PRIVATE_CID>
# ✅ Expected: "no providers found for the CID"

# This should also show ERR_ID:00006
curl https://your-gateway.mypinata.cloud/ipfs/<PRIVATE_CID>
# ✅ Expected: ERR_ID:00006 error message
```

**To access private files:**
- Use Pinata SDK with JWT authentication
- Or use Pinata API directly: `GET https://api.pinata.cloud/data/pinList`
- Or retrieve via Files API endpoint with Authorization header

**If you WANT public access:**
1. Set `PINATA_PUBLIC="true"` in `.env.pinata`
2. Re-run the system
3. Files will be accessible at `https://ipfs.io/ipfs/<CID>`

### Gas Cost Still High

Verify IPFS mode is enabled:
```bash
# Check logs during submission
# Should see: "📌 Pinata mode: Uploading VLC data to IPFS..."
# NOT: "📦 Traditional mode: Encoding full VLC data on-chain..."

# Verify environment variable
echo $USE_PINATA  # Should output: true
```

## API Reference

### Pinata Upload API

**Endpoint**: `https://uploads.pinata.cloud/v3/files`

**Method**: POST

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data
```

**Body** (multipart/form-data):
```
file: <JSON file buffer>
  filename: epoch-1-vlc-graph.json
  contentType: application/json

pinataMetadata: {
  "name": "Epoch 1 VLC Graph",
  "keyvalues": {
    "epochNumber": "1",
    "subnetId": "subnet-1",
    "type": "vlc-graph-data"
  }
}

network: "public"          // Include for public IPFS, omit for private
```

**Response**:
```json
{
  "data": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "cid": "bafkreih5aznjvttude6c3wbvqeebb6rlx5wkbzyppv7garjiubll2ceym4",
    "name": "epoch-1-vlc-graph.json",
    "size": 2048,
    "created_at": "2025-01-13T10:30:00.000Z"
  }
}
```

**Privacy Control**:
- `network: "public"` → File uploaded to public IPFS (accessible via any gateway)
- Network parameter omitted → File uploaded to private IPFS (requires authentication)

## Future Enhancements

- **Compression**: Apply gzip compression before upload
- **Batching**: Upload multiple epochs in single file
- **Caching**: Local cache of recent IPFS data
- **Verification**: On-chain proof of IPFS data integrity
- **Migration**: Tools to migrate old on-chain data to IPFS

## Resources

- [Pinata Documentation](https://docs.pinata.cloud/)
- [IPFS Documentation](https://docs.ipfs.tech/)
- [IPFS Gateway Checker](https://ipfs.github.io/public-gateway-checker/)
- [CID Inspector](https://cid.ipfs.tech/)
