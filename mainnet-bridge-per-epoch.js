#!/usr/bin/env node

/**
 * Enhanced PoCW Subnet to Mainnet Bridge - Per-Epoch Submission
 * 
 * This module handles real-time epoch submission to mainnet as each epoch
 * completes (every 3 rounds), rather than batching all epochs at the end.
 * 
 * Architecture:
 * 1. Integrates with the Go subnet system via callback mechanism
 * 2. Submits each epoch immediately when EpochFinalized event occurs
 * 3. Provides real-time FLUX token mining per completed epoch
 * 4. Maintains epoch tracking and statistics
 */

const { ethers } = require('ethers');
const { spawn } = require('child_process');
const fs = require('fs').promises;
const path = require('path');
const http = require('http');
const url = require('url');
const FormData = require('form-data');
const fetch = require('node-fetch');

class PerEpochMainnetBridge {
    constructor() {
        this.provider = null;
        this.contracts = {};
        this.wallets = {};
        this.subnetProcess = null;
        this.epochSubmissions = new Map(); // Track submitted epochs
        this.httpServer = null;

        // Network configuration - use environment variable or default to localhost
        this.RPC_URL = process.env.RPC_URL || "http://localhost:8545";
        this.DGRAPH_URL = "http://localhost:8080";

        // Pinata IPFS configuration
        this.USE_PINATA = process.env.USE_PINATA === "true";
        this.PINATA_PUBLIC = process.env.PINATA_PUBLIC !== "false"; // Default to public
        this.PINATA_JWT = process.env.JWT_SECRET_ACCESS;
        this.PINATA_GATEWAY = process.env.GATEWAY_PINATA || "coffee-defiant-raccoon-829.mypinata.cloud";
        this.PINATA_API_URL = "https://uploads.pinata.cloud/v3/files";

        // Account configuration - use environment variables or defaults for local Anvil
        this.accounts = {
            deployer: {
                address: process.env.DEPLOYER_ADDRESS || "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
                privateKey: process.env.DEPLOYER_KEY || "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
            },
            validator1: {
                address: process.env.VALIDATOR_1_ADDRESS || "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
                privateKey: process.env.VALIDATOR_1_KEY || "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
            },
            miner: {
                address: process.env.MINER_ADDRESS || "0x86cDAb16A19602F74E4fFB996baD70307105a3A3", // Sepolia miner
                privateKey: process.env.MINER_KEY || "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
            }
        };
    }

    async initialize() {
        console.log("🌐 Per-Epoch PoCW Mainnet Bridge Initializing...");
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");

        // Initialize ethers provider
        this.provider = new ethers.JsonRpcProvider(this.RPC_URL);

        // Create wallets
        this.wallets.validator1 = new ethers.Wallet(this.accounts.validator1.privateKey, this.provider);
        this.wallets.miner = new ethers.Wallet(this.accounts.miner.privateKey, this.provider);

        // Load contract addresses and ABIs
        await this.loadContracts();

        // Start HTTP server for Go integration
        await this.startHttpServer();

        console.log("✅ Bridge initialized successfully!");
        console.log(`📍 Validator-1: ${this.accounts.validator1.address}`);
        console.log(`⛏️  Miner: ${this.accounts.miner.address}`);

        // Log IPFS configuration
        if (this.USE_PINATA) {
            console.log(`📌 Pinata IPFS: ENABLED`);
            console.log(`🌐 Gateway: https://${this.PINATA_GATEWAY}/ipfs/`);
            console.log(`🔓 Access Mode: ${this.PINATA_PUBLIC ? 'PUBLIC' : 'PRIVATE'}`);
        } else {
            console.log(`📌 Pinata IPFS: DISABLED (storing full data on-chain)`);
        }
    }

    /**
     * Upload VLC graph data to Pinata IPFS
     * @param {Object} vlcGraphData - The VLC graph data to upload
     * @param {number} epochNumber - Epoch number for naming
     * @returns {Promise<{cid: string, ipfsUri: string, gatewayUrl: string}>}
     */
    async uploadToPinata(vlcGraphData, epochNumber) {
        try {
            console.log(`📤 Uploading Epoch ${epochNumber} data to Pinata IPFS...`);

            if (!this.PINATA_JWT) {
                throw new Error('PINATA_JWT not configured. Please set JWT_SECRET_ACCESS in environment.');
            }

            // Convert VLC graph data to JSON string
            const jsonContent = JSON.stringify(vlcGraphData, null, 2);
            const jsonBuffer = Buffer.from(jsonContent, 'utf-8');

            // Create form data with file
            const FormData = require('form-data');
            const formData = new FormData();
            formData.append('file', jsonBuffer, {
                filename: `epoch-${epochNumber}-vlc-graph.json`,
                contentType: 'application/json'
            });

            // Add metadata
            const metadata = JSON.stringify({
                name: `Epoch ${epochNumber} VLC Graph`,
                keyvalues: {
                    epochNumber: epochNumber.toString(),
                    subnetId: vlcGraphData.subnetId || 'subnet-1',
                    timestamp: vlcGraphData.timestamp?.toString() || Date.now().toString(),
                    type: 'vlc-graph-data'
                }
            });
            formData.append('pinataMetadata', metadata);

            // Add network parameter to control public/private access
            // "public" = accessible via any IPFS gateway
            // omitted or "private" = only accessible via authenticated Pinata gateway
            if (this.PINATA_PUBLIC) {
                formData.append('network', 'public');
            }
            // If PINATA_PUBLIC is false, don't add network parameter (defaults to private)

            // Upload to Pinata v3 Files API
            const response = await fetch(this.PINATA_API_URL, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.PINATA_JWT}`,
                    ...formData.getHeaders()
                },
                body: formData
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Pinata upload failed: ${response.status} ${errorText}`);
            }

            const result = await response.json();
            const cid = result.data.cid;
            const ipfsUri = `ipfs://${cid}`;
            const gatewayUrl = `https://${this.PINATA_GATEWAY}/ipfs/${cid}`;
            const jsonSize = jsonContent.length;

            console.log(`✅ Uploaded to Pinata v3 Files API successfully!`);
            console.log(`   CID: ${cid}`);
            console.log(`   IPFS URI: ${ipfsUri}`);
            console.log(`   Gateway URL: ${gatewayUrl}`);

            // v3 Files API with network parameter provides TRUE private/public distinction
            if (this.PINATA_PUBLIC) {
                console.log(`   🔓 PUBLIC: https://ipfs.io/ipfs/${cid}`);
            } else {
                console.log(`   🔒 PRIVATE: NOT on public IPFS (ERR_ID:00006 is expected)`);
            }

            return {
                cid,
                ipfsUri,
                gatewayUrl,
                size: jsonSize
            };

        } catch (error) {
            console.error(`❌ Failed to upload to Pinata: ${error.message}`);
            throw error;
        }
    }

    async loadContracts() {
        try {
            // Load contract addresses
            const addressesPath = path.join(__dirname, 'contract_addresses.json');
            const addressesData = await fs.readFile(addressesPath, 'utf8');
            const addresses = JSON.parse(addressesData);

            // Find contract addresses - handle both old (address->name) and new (name->address) formats
            let hetuAddress, fluxAddress, registryAddress, verifierAddress;

            // Check if this is the new format (keys are contract names)
            if (addresses.HETUToken || addresses.FLUXToken) {
                // New format: name -> address
                hetuAddress = addresses.HETUToken;
                fluxAddress = addresses.FLUXToken;
                registryAddress = addresses.SubnetRegistry;
                verifierAddress = addresses.PoCWVerifier;
            } else {
                // Old format: address -> name
                for (const [address, name] of Object.entries(addresses)) {
                    if (name.includes('HETU Token')) hetuAddress = address;
                    else if (name.includes('Intelligence Token') || name.includes('FLUX')) fluxAddress = address;
                    else if (name.includes('Subnet Registry')) registryAddress = address;
                    else if (name.includes('Enhanced PoCW') || name.includes('Verifier')) verifierAddress = address;
                }
            }

            if (!hetuAddress || !fluxAddress || !registryAddress || !verifierAddress) {
                console.error('Available addresses:', addresses);
                throw new Error('Could not find all required contract addresses');
            }

            // Load ABIs
            const verifierABI = [
                "function submitAndDistributeEpoch(string memory subnetId, bytes memory vlcGraphData, address[] memory successfulMiners, uint256 successfulTasks, uint256 failedTasks) external",
                "function getMinerStats(address miner) external view returns (tuple(address owner, uint256 successfulTasks, uint256 totalTasks, uint256 totalIntelligenceMined, uint256 reputationScore, uint256 lastActiveEpoch, uint256 joinedTimestamp, bool isActive))",
                "function subnetIdToHash(string memory subnetId) external view returns (bytes32)"
            ];

            const fluxABI = [
                "function balanceOf(address account) external view returns (uint256)",
                "function totalSupply() external view returns (uint256)"
            ];

            // Create contract instances
            this.contracts.verifier = new ethers.Contract(verifierAddress, verifierABI, this.wallets.validator1);
            this.contracts.flux = new ethers.Contract(fluxAddress, fluxABI, this.provider);

            console.log(`📋 Loaded contracts:`);
            console.log(`  PoCWVerifier: ${verifierAddress}`);
            console.log(`  FLUX Token: ${fluxAddress}`);

        } catch (error) {
            throw new Error(`Failed to load contracts: ${error.message}`);
        }
    }

    // Callback function to handle epoch finalized events from the Go subnet
    async handleEpochFinalized(epochNumber, subnetId, epochData) {
        try {
            console.log(`\n🚀 EPOCH ${epochNumber} FINALIZED - IMMEDIATE MAINNET SUBMISSION`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log(`📊 Subnet: ${subnetId}`);
            console.log(`📈 Epoch: ${epochNumber}`);
            console.log(`🔗 Completed Rounds: ${epochData.CompletedRounds.length}`);
            console.log(`⏰ VLC Clock State:`, epochData.VLCClockState);

            // Check if already submitted (prevent duplicates)
            const epochKey = `${subnetId}-${epochNumber}`;
            if (this.epochSubmissions.has(epochKey)) {
                console.log(`⚠️  Epoch ${epochNumber} already submitted, skipping...`);
                return;
            }

            // Extract current epoch data from Dgraph
            const { vlcGraphData, successfulTasks, failedTasks, miners } = await this.extractCurrentEpochData(subnetId, epochNumber);

            // Submit to mainnet
            const result = await this.submitEpochToMainnet(subnetId, vlcGraphData, miners, successfulTasks, failedTasks);

            // Mark as submitted
            this.epochSubmissions.set(epochKey, {
                epochNumber,
                subnetId,
                txHash: result.txHash,
                blockNumber: result.blockNumber,
                fluxMined: result.fluxMined,
                timestamp: Date.now()
            });

            console.log(`✅ Epoch ${epochNumber} submitted successfully!`);
            console.log(`💰 FLUX Mined: ${result.fluxMined} FLUX tokens`);
            console.log(`📤 Transaction: ${result.txHash}`);
            console.log(`📦 Block: ${result.blockNumber}`);

        } catch (error) {
            console.error(`❌ Failed to submit epoch ${epochNumber}:`, error.message);
        }
    }

    // Extract VLC data for the current completed epoch
    async extractCurrentEpochData(subnetId, epochNumber) {
        try {
            console.log(`📊 Extracting VLC data for epoch ${epochNumber}...`);

            // Query Dgraph for events from this specific epoch
            const query = `
            {
                events(func: has(event_id)) @filter(eq(subnet_id, "${subnetId}")) {
                    uid
                    event_id
                    event_name
                    event_type
                    vlc_clock
                    parents {
                        uid
                        event_id
                    }
                    timestamp
                    description
                    request_id
                }
            }`;

            const response = await fetch(`${this.DGRAPH_URL}/query`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ query })
            });

            if (!response.ok) {
                throw new Error('Failed to query Dgraph');
            }

            const data = await response.json();
            const events = data.data.events || [];

            console.log(`✅ Extracted ${events.length} events from Dgraph`);

            // Filter events for current epoch (rough estimation based on timing or event patterns)
            // In a real implementation, you would track epoch boundaries more precisely
            const epochEvents = this.filterEventsForCurrentEpoch(events, epochNumber);

            // Count successful and failed tasks from the epoch
            const { successfulTasks, failedTasks } = this.analyzeEpochTasks(epochEvents);

            // Generate comprehensive VLC graph data for this epoch
            const vlcGraphData = this.generateEpochVLCData(subnetId, epochNumber, epochEvents, successfulTasks, failedTasks);

            console.log(`📈 Epoch ${epochNumber}: ${successfulTasks} successful, ${failedTasks} failed tasks`);

            return {
                vlcGraphData,
                successfulTasks,
                failedTasks,
                miners: [this.accounts.miner.address]
            };

        } catch (error) {
            console.error("❌ Error extracting epoch data:", error.message);
            // Fallback to simulated data for this epoch
            return this.generateSimulatedEpochData(subnetId, epochNumber);
        }
    }

    // Filter events that belong to the current epoch
    filterEventsForCurrentEpoch(events, epochNumber) {
        // For simplicity, assume the most recent events belong to the current epoch
        // In a production system, you would have explicit epoch boundaries
        const eventsPerEpoch = 10; // Approximate events per epoch (3 rounds * ~3 events per round + epoch events)
        const startIndex = Math.max(0, events.length - (eventsPerEpoch * (4 - epochNumber)));
        const endIndex = events.length - (eventsPerEpoch * (3 - epochNumber));

        return events.slice(startIndex, Math.min(endIndex, events.length));
    }

    // Analyze epoch events to count successful/failed tasks
    analyzeEpochTasks(epochEvents) {
        const successfulTasks = epochEvents.filter(e =>
            e.event_name === 'RoundSuccess' ||
            e.description?.includes('OUTPUT DELIVERED TO USER')
        ).length;

        const failedTasks = epochEvents.filter(e =>
            e.event_name === 'RoundFailed' ||
            e.description?.includes('OUTPUT REJECTED')
        ).length;

        return { successfulTasks, failedTasks };
    }

    // Generate VLC graph data for a specific epoch
    generateEpochVLCData(subnetId, epochNumber, epochEvents, successfulTasks, failedTasks) {
        return {
            subnetId,
            epochNumber,
            events: epochEvents.map(event => ({
                id: event.event_id || `epoch_${epochNumber}_${event.uid}`,
                name: event.event_name || 'Unknown',
                vlcClock: event.vlc_clock || {},
                parents: (event.parents || []).map(p => p.event_id || p.uid),
                timestamp: event.timestamp || Date.now(),
                description: event.description || `Epoch ${epochNumber} event`,
                requestId: event.request_id || null
            })),
            miners: [this.accounts.miner.address],
            validators: [
                this.accounts.validator1.address,
                "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
                "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
                "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
            ],
            summary: {
                epochNumber,
                totalTasks: successfulTasks + failedTasks,
                successfulTasks,
                failedTasks,
                validationStatus: "complete",
                consensusReached: true
            },
            statistics: {
                epochProcessingTime: 7500, // 3 rounds * ~2500ms per round
                eventsInEpoch: epochEvents.length,
                avgEventsPerRound: epochEvents.length / 3
            }
        };
    }

    // Generate simulated epoch data as fallback
    generateSimulatedEpochData(subnetId, epochNumber) {
        console.log(`🎭 Generating simulated data for epoch ${epochNumber}...`);

        const currentTime = Date.now();
        const events = [];

        // Simulate 3 rounds (1 per round for this epoch)
        const tasksInEpoch = 1; // Typically 1 task per epoch in our 7-task demo
        for (let round = 1; round <= 3; round++) {
            const baseTime = currentTime - (3000 - round * 1000);
            const globalTaskId = (epochNumber - 1) * 1 + 1; // Map to global task sequence

            if (globalTaskId <= 7) { // Only if within our 7-task demo
                // User input
                events.push({
                    id: `user_input_epoch_${epochNumber}_round_${round}`,
                    name: "UserInput",
                    vlcClock: { 1: globalTaskId - 1, 2: globalTaskId },
                    parents: events.length > 0 ? [events[events.length - 1].id] : [],
                    timestamp: baseTime,
                    description: `Epoch ${epochNumber} Round ${round}: User submits task`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });

                // Miner output
                events.push({
                    id: `miner_output_epoch_${epochNumber}_round_${round}`,
                    name: "MinerOutput",
                    vlcClock: { 1: globalTaskId, 2: globalTaskId },
                    parents: [`user_input_epoch_${epochNumber}_round_${round}`],
                    timestamp: baseTime + 500,
                    description: `Epoch ${epochNumber} Round ${round}: Miner provides solution`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });

                // Round success
                events.push({
                    id: `round_${epochNumber}_${round}_complete`,
                    name: "RoundSuccess",
                    vlcClock: { 1: globalTaskId, 2: globalTaskId + 1 },
                    parents: [`miner_output_epoch_${epochNumber}_round_${round}`],
                    timestamp: baseTime + 1000,
                    description: `Epoch ${epochNumber} Round ${round}: OUTPUT DELIVERED TO USER`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });
            }
        }

        const vlcGraphData = this.generateEpochVLCData(subnetId, epochNumber, events, 1, 0);

        return {
            vlcGraphData,
            successfulTasks: 1,
            failedTasks: 0,
            miners: [this.accounts.miner.address]
        };
    }

    // Submit epoch data to mainnet
    async submitEpochToMainnet(subnetId, vlcGraphData, miners, successfulTasks, failedTasks) {
        console.log(`\n⚡ Submitting epoch ${vlcGraphData.epochNumber} to mainnet...`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");

        // Convert VLC data to bytes
        const vlcDataString = JSON.stringify(vlcGraphData);
        const vlcDataBytes = ethers.toUtf8Bytes(vlcDataString);

        console.log(`📊 Submitting epoch data:`);
        console.log(`  Subnet: ${subnetId}`);
        console.log(`  Epoch: ${vlcGraphData.epochNumber}`);
        console.log(`  Tasks: ${successfulTasks} successful, ${failedTasks} failed`);
        console.log(`  VLC Events: ${vlcGraphData.events.length}`);
        console.log(`  Data Size: ${vlcDataBytes.length} bytes`);

        // Get pre-submission balances
        const minerBalanceBefore = await this.contracts.flux.balanceOf(this.accounts.miner.address);
        const validator1BalanceBefore = await this.contracts.flux.balanceOf(this.accounts.validator1.address);

        try {
            // Submit epoch and mine FLUX tokens
            console.log(`\n🚀 Validator-1 posting epoch ${vlcGraphData.epochNumber} to mainnet...`);
            const tx = await this.contracts.verifier.submitAndDistributeEpoch(
                vlcGraphData.subnetId,
                vlcDataBytes,
                miners,
                successfulTasks,
                failedTasks
            );

            console.log(`📤 Transaction submitted: ${tx.hash}`);
            console.log("⏳ Waiting for confirmation...");

            const receipt = await tx.wait();
            console.log(`✅ Transaction confirmed in block: ${receipt.blockNumber}`);

            // Check post-submission balances
            const minerBalanceAfter = await this.contracts.flux.balanceOf(this.accounts.miner.address);
            const validator1BalanceAfter = await this.contracts.flux.balanceOf(this.accounts.validator1.address);

            const minerEarned = ethers.formatEther(minerBalanceAfter - minerBalanceBefore);
            const validator1Earned = ethers.formatEther(validator1BalanceAfter - validator1BalanceBefore);
            const totalMined = ethers.formatEther((minerBalanceAfter - minerBalanceBefore) + (validator1BalanceAfter - validator1BalanceBefore));

            console.log(`\n🎉 EPOCH ${vlcGraphData.epochNumber} FLUX MINING COMPLETE!`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log(`💰 Miner earned: ${minerEarned} FLUX`);
            console.log(`🏆 Validator-1 earned: ${validator1Earned} FLUX`);
            console.log(`🔑 Total FLUX mined: ${totalMined} FLUX`);

            return {
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                fluxMined: totalMined,
                gasUsed: receipt.gasUsed.toString()
            };

        } catch (error) {
            console.error(`❌ Epoch submission failed: ${error.message}`);
            throw error;
        }
    }

    // Get summary of all submitted epochs
    getSubmissionSummary() {
        console.log(`\n📊 EPOCH SUBMISSION SUMMARY`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        console.log(`📈 Total Epochs Submitted: ${this.epochSubmissions.size}`);

        let totalFluxMined = 0;
        for (const [epochKey, submission] of this.epochSubmissions.entries()) {
            console.log(`  Epoch ${submission.epochNumber}: ${submission.fluxMined} FLUX (tx: ${submission.txHash.substring(0, 10)}...)`);
            totalFluxMined += parseFloat(submission.fluxMined);
        }

        console.log(`💰 Total FLUX Mined: ${totalFluxMined} FLUX across all epochs`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    }

    // Start HTTP server to receive epoch data from Go
    async startHttpServer() {
        const PORT = 3001;

        this.httpServer = http.createServer((req, res) => {
            // Handle CORS
            res.setHeader('Access-Control-Allow-Origin', '*');
            res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
            res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

            if (req.method === 'OPTIONS') {
                res.writeHead(200);
                res.end();
                return;
            }

            const parsedUrl = url.parse(req.url, true);

            if (req.method === 'POST' && parsedUrl.pathname === '/submit-epoch') {
                this.handleEpochSubmission(req, res);
            } else if (req.method === 'GET' && parsedUrl.pathname === '/health') {
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ status: 'healthy', service: 'per-epoch-bridge' }));
            } else {
                res.writeHead(404, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: 'Not Found' }));
            }
        });

        return new Promise((resolve, reject) => {
            this.httpServer.listen(PORT, (err) => {
                if (err) {
                    reject(err);
                } else {
                    console.log(`🌐 HTTP server listening on port ${PORT}`);
                    console.log(`📡 Ready to receive epoch data from Go at http://localhost:${PORT}/submit-epoch`);
                    resolve();
                }
            });
        });
    }

    // Handle epoch submission from Go
    async handleEpochSubmission(req, res) {
        let body = '';

        req.on('data', chunk => {
            body += chunk.toString();
        });

        req.on('end', async () => {
            try {
                const epochData = JSON.parse(body);
                console.log(`\n🚀 RECEIVED EPOCH SUBMISSION FROM GO`);
                console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
                console.log(`📊 Epoch: ${epochData.epochNumber}`);
                console.log(`🌐 Subnet: ${epochData.subnetId}`);
                console.log(`⏰ Timestamp: ${new Date(epochData.timestamp * 1000).toISOString()}`);
                console.log(`🔗 Rounds: ${epochData.completedRounds.length}`);
                console.log(`🔍 Detailed Rounds: ${epochData.detailedRounds ? epochData.detailedRounds.length : 'undefined'}`);
                console.log(`🕘 VLC State: ${JSON.stringify(epochData.vlcClockState)}`);

                // Debug detailed round data
                if (epochData.detailedRounds && epochData.detailedRounds.length > 0) {
                    console.log(`🔍 DEBUG - Detailed rounds received:`);
                    epochData.detailedRounds.forEach((round, index) => {
                        console.log(`   Round ${index + 1}: ${round.userInput ? round.userInput.substring(0, 40) + '...' : 'No input'}`);
                    });
                } else {
                    console.log(`❌ DEBUG - No detailed rounds in payload`);
                }

                // Submit to blockchain
                await this.submitEpochToBlockchain(epochData);

                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    success: true,
                    epochNumber: epochData.epochNumber,
                    message: 'Epoch submitted successfully'
                }));

            } catch (error) {
                console.error('❌ Error handling epoch submission:', error.message);
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    success: false,
                    error: error.message
                }));
            }
        });
    }

    // Submit epoch data to blockchain using the received data
    async submitEpochToBlockchain(epochData) {
        try {
            console.log(`📤 Submitting Epoch ${epochData.epochNumber} to blockchain...`);

            const successfulMiners = [this.accounts.miner.address];

            // Use actual successful/failed counts from detailed round data
            let successfulTasks = (epochData.detailedRounds || []).filter(r => r.success).length;
            let failedTasks = (epochData.detailedRounds || []).filter(r => !r.success).length;

            console.log(`📊 Task breakdown: ${successfulTasks} successful, ${failedTasks} failed (total: ${successfulTasks + failedTasks})`);

            // Verify we have the expected task counts
            const totalRoundsInEpoch = successfulTasks + failedTasks;
            if (totalRoundsInEpoch === 0) {
                console.log(`⚠️  WARNING: No detailed round data available, falling back to completedRounds count`);
                // Fallback to legacy method if detailed rounds are unavailable
                successfulTasks = epochData.completedRounds ? epochData.completedRounds.length : 0;
                failedTasks = 0;
            }

            // Prepare VLC graph data
            let vlcGraphData;
            let ipfsMetadata = null;

            if (this.USE_PINATA) {
                // Upload to IPFS and use URI
                console.log(`📌 Pinata mode: Uploading VLC data to IPFS...`);
                const vlcGraph = this.createVLCGraphObject(epochData);
                const ipfsResult = await this.uploadToPinata(vlcGraph, epochData.epochNumber);

                // Store IPFS URI as bytes (much smaller than full JSON)
                vlcGraphData = ethers.toUtf8Bytes(ipfsResult.ipfsUri);
                ipfsMetadata = ipfsResult;
            } else {
                // Traditional mode: encode full JSON as bytes
                console.log(`📦 Traditional mode: Encoding full VLC data on-chain...`);
                vlcGraphData = this.encodeVLCGraphData(epochData);
            }

            // Submit to contract
            const tx = await this.contracts.verifier.submitAndDistributeEpoch(
                epochData.subnetId,
                vlcGraphData,
                successfulMiners,
                successfulTasks,
                failedTasks
            );

            console.log(`📤 Transaction submitted: ${tx.hash}`);
            const receipt = await tx.wait();
            console.log(`✅ Transaction confirmed in block ${receipt.blockNumber}`);

            // Track submission
            const submissionKey = `${epochData.subnetId}-epoch-${epochData.epochNumber}`;
            this.epochSubmissions.set(submissionKey, {
                epochNumber: epochData.epochNumber,
                subnetId: epochData.subnetId,
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                timestamp: epochData.timestamp,
                fluxMined: "0", // Would calculate from logs
                ipfsUri: ipfsMetadata?.ipfsUri || null,
                ipfsCid: ipfsMetadata?.cid || null,
                gatewayUrl: ipfsMetadata?.gatewayUrl || null
            });

            console.log(`🎉 Epoch ${epochData.epochNumber} submitted successfully!`);
            if (ipfsMetadata) {
                if (this.PINATA_PUBLIC) {
                    console.log(`📌 IPFS: https://ipfs.io/ipfs/${ipfsMetadata.cid}`);
                } else {
                    console.log(`🔒 Private storage - NOT accessible via public IPFS`);
                }
            }
            console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);

            return {
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                gasUsed: receipt.gasUsed.toString()
            };

        } catch (error) {
            console.error(`❌ Blockchain submission failed: ${error.message}`);
            throw error;
        }
    }

    /**
     * Create VLC graph object from epoch data (used for both IPFS and on-chain storage)
     * @param {Object} epochData - Epoch data from Go subnet
     * @returns {Object} Structured VLC graph data
     */
    createVLCGraphObject(epochData) {
        return {
            epochNumber: epochData.epochNumber,
            subnetId: epochData.subnetId,
            vlcClockState: epochData.vlcClockState,
            detailedRounds: epochData.detailedRounds || [],
            epochEventId: epochData.epochEventId || '',
            parentRoundEventId: epochData.parentRoundEventId || '',
            timestamp: epochData.timestamp || Math.floor(Date.now() / 1000)
        };
    }

    // Encode VLC graph data for blockchain submission (traditional mode - full data on-chain)
    encodeVLCGraphData(epochData) {
        // Create a structured representation of the VLC graph for this epoch
        const vlcGraph = this.createVLCGraphObject(epochData);

        // Convert to hex-encoded bytes for smart contract
        const jsonString = JSON.stringify(vlcGraph);
        const hexData = '0x' + Buffer.from(jsonString, 'utf8').toString('hex');

        console.log(`🔗 Encoded VLC graph data: ${jsonString.length} bytes`);
        console.log(`📊 Epoch summary: ${vlcGraph.totalRounds} rounds (${vlcGraph.successfulRounds} success, ${vlcGraph.failedRounds} failed)`);

        // Log detailed round information
        if (epochData.detailedRounds && epochData.detailedRounds.length > 0) {
            console.log(`📋 Round details:`);
            epochData.detailedRounds.forEach(round => {
                const status = round.success ? '✅' : '❌';
                const inputPreview = round.userInput.length > 40 ? round.userInput.substring(0, 40) + '...' : round.userInput;
                console.log(`   Round ${round.roundNumber}: ${status} "${inputPreview}"`);
            });
        }

        return hexData;
    }
}

// Export the class for use in integration scripts
module.exports = PerEpochMainnetBridge;

// If run directly, start in interactive mode
if (require.main === module) {
    const bridge = new PerEpochMainnetBridge();

    async function main() {
        try {
            await bridge.initialize();
            console.log(`\n🔄 Per-Epoch Bridge Ready!`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log("To use this bridge:");
            console.log("1. Set up epoch callback in subnet coordinator");
            console.log("2. Each completed epoch (3 rounds) triggers immediate submission");
            console.log("3. FLUX tokens are mined in real-time per epoch");
            console.log("");
            console.log("Example usage:");
            console.log("const bridge = new PerEpochMainnetBridge();");
            console.log("coordinator.GraphAdapter.SetEpochFinalizedCallback(bridge.handleEpochFinalized.bind(bridge));");

        } catch (error) {
            console.error("❌ Bridge initialization failed:", error.message);
        }
    }

    main().catch(console.error);
}