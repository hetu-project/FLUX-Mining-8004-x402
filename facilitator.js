#!/usr/bin/env node

/**
 * x402 Facilitator Service
 *
 * This service implements the x402 protocol facilitator that handles payment
 * verification and settlement for both standard (direct) and escrow payment modes.
 *
 * Previously, Validator-1 was handling these responsibilities directly.
 * Now, this dedicated facilitator service provides proper x402 compliance.
 */

const express = require('express');
const { ethers } = require('ethers');
const cors = require('cors');

// ABI for escrow contract interactions
const ESCROW_ABI = [
    "function depositPayment(bytes32 taskId, address client, address agent, uint256 amount, uint256 deadline)",
    "function releasePayment(bytes32 taskId)",
    "function refundPayment(bytes32 taskId)",
    "function getPayment(bytes32 taskId) view returns (tuple(bytes32 taskId, address client, address agent, uint256 amount, uint256 depositTime, uint256 deadline, uint8 status))"
];

// ABI for USDC token
const USDC_ABI = [
    "function transfer(address to, uint256 amount) returns (bool)",
    "function transferFrom(address from, address to, uint256 amount) returns (bool)",
    "function balanceOf(address account) view returns (uint256)",
    "function approve(address spender, uint256 amount) returns (bool)"
];

class X402Facilitator {
    constructor(config) {
        this.app = express();
        this.app.use(express.json());
        this.app.use(cors());

        // Network configuration
        this.network = config.network || 'local';
        this.rpcUrl = config.rpcUrl || 'http://localhost:8545';
        this.provider = new ethers.JsonRpcProvider(this.rpcUrl);

        // Facilitator wallet
        this.privateKey = config.privateKey;
        this.wallet = new ethers.Wallet(this.privateKey, this.provider);

        // Contract addresses
        this.escrowAddress = config.escrowAddress;
        this.usdcAddress = config.usdcAddress;
        this.validators = config.validators || [];

        // Payment mode configuration
        this.paymentMode = config.paymentMode || 'hybrid';  // direct, escrow, or hybrid

        // Store for pending signed transactions (for direct payments)
        this.pendingTransactions = new Map(); // taskId -> signed transaction

        // Nonce management for transaction ordering
        this.nonceTracker = null;
        this.initializeNonce();

        console.log('🌐 X402 Facilitator Configuration:');
        console.log(`   Network: ${this.network}`);
        console.log(`   RPC URL: ${this.rpcUrl}`);
        console.log(`   Facilitator: ${this.wallet.address}`);
        console.log(`   Payment Mode: ${this.paymentMode}`);
        console.log(`   Escrow Contract: ${this.escrowAddress}`);
        console.log(`   USDC Token: ${this.usdcAddress}`);

        this.setupEndpoints();
    }

    async initializeNonce() {
        this.nonceTracker = await this.provider.getTransactionCount(this.wallet.address, 'latest');
    }

    async getNextNonce() {
        // Get the current nonce from the network
        const networkNonce = await this.provider.getTransactionCount(this.wallet.address, 'pending');

        // Use the maximum of our tracker and the network nonce
        if (this.nonceTracker === null || networkNonce > this.nonceTracker) {
            this.nonceTracker = networkNonce;
        }

        const nonce = this.nonceTracker;
        this.nonceTracker++; // Increment for next transaction

        return nonce;
    }

    setupEndpoints() {
        // Health check endpoint
        this.app.get('/health', (req, res) => {
            res.json({
                status: 'healthy',
                facilitator: this.wallet.address,
                network: this.network,
                paymentMode: this.paymentMode,
                capabilities: this.getCapabilities()
            });
        });

        // x402 standard: Payment verification
        this.app.post('/verify', async (req, res) => {
            try {
                const { payment, scheme } = req.body;

                // Check if scheme is allowed based on payment mode
                if (!this.isSchemeAllowed(scheme)) {
                    return res.status(400).json({
                        valid: false,
                        error: `Scheme '${scheme}' not allowed in ${this.paymentMode} mode`
                    });
                }

                // Verify payment signature and format
                const valid = await this.verifyPayment(payment, scheme);

                res.json({
                    valid,
                    facilitator: this.wallet.address,
                    scheme,
                    capabilities: this.getCapabilities()
                });
            } catch (error) {
                console.error('❌ Verification error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // x402 standard: Payment settlement
        this.app.post('/settle', async (req, res) => {
            try {
                const { payment, scheme, taskId } = req.body;

                // Check if scheme is allowed
                if (!this.isSchemeAllowed(scheme)) {
                    return res.status(400).json({
                        error: `Scheme '${scheme}' not allowed in ${this.paymentMode} mode`
                    });
                }

                let result;
                if (scheme === 'exact' || scheme === 'direct') {
                    // Direct payment (standard x402)
                    result = await this.settleDirectPayment(payment);
                    // Ensure scheme is included in response
                    result.scheme = 'direct';
                } else if (scheme === 'escrow') {
                    // Escrow payment (our enhancement)
                    result = await this.initiateEscrowPayment(payment, taskId);
                    // Ensure scheme is included in response
                    result.scheme = 'escrow';
                } else {
                    return res.status(400).json({ error: 'Unknown payment scheme' });
                }

                res.json(result);
            } catch (error) {
                console.error('❌ Settlement error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Escrow extension: Check escrow status
        this.app.get('/escrow/status/:taskId', async (req, res) => {
            try {
                if (!this.isSchemeAllowed('escrow')) {
                    return res.status(400).json({
                        error: 'Escrow not enabled in current payment mode'
                    });
                }

                const status = await this.getEscrowStatus(req.params.taskId);
                res.json(status);
            } catch (error) {
                console.error('❌ Status check error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Escrow extension: Release payment from escrow
        this.app.post('/escrow/release', async (req, res) => {
            try {
                if (!this.isSchemeAllowed('escrow')) {
                    return res.status(400).json({
                        error: 'Escrow not enabled in current payment mode'
                    });
                }

                const { taskId, validatorApprovals } = req.body;
                const result = await this.releaseFromEscrow(taskId, validatorApprovals);
                res.json(result);
            } catch (error) {
                console.error('❌ Release error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Escrow extension: Refund payment from escrow
        this.app.post('/escrow/refund', async (req, res) => {
            try {
                if (!this.isSchemeAllowed('escrow')) {
                    return res.status(400).json({
                        error: 'Escrow not enabled in current payment mode'
                    });
                }

                const { taskId, reason } = req.body;
                const result = await this.refundFromEscrow(taskId, reason);
                res.json(result);
            } catch (error) {
                console.error('❌ Refund error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Direct payment finalization: Broadcast or discard pending transaction
        this.app.post('/direct/finalize', async (req, res) => {
            try {
                const { taskId, approved, validatorApprovals } = req.body;

                if (!this.pendingTransactions.has(taskId)) {
                    return res.status(404).json({
                        error: `No pending transaction found for task ${taskId}`
                    });
                }

                const pendingTx = this.pendingTransactions.get(taskId);

                if (approved) {
                    // Broadcast the transaction to blockchain
                    const txResponse = await this.provider.broadcastTransaction(pendingTx.signedTx);
                    const receipt = await txResponse.wait();

                    // Remove from pending
                    this.pendingTransactions.delete(taskId);

                    return res.json({
                        transactionHash: txResponse.hash,
                        blockNumber: receipt.blockNumber,
                        status: 'completed',
                        taskId: taskId
                    });
                } else {
                    // Discard the transaction
                    this.pendingTransactions.delete(taskId);

                    return res.json({
                        status: 'discarded',
                        taskId: taskId,
                        reason: 'Payment rejected based on validation results'
                    });
                }
            } catch (error) {
                console.error('❌ Finalization error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Get pending direct payment status
        this.app.get('/direct/status/:taskId', (req, res) => {
            const { taskId } = req.params;

            if (!this.pendingTransactions.has(taskId)) {
                return res.status(404).json({
                    error: `No pending transaction found for task ${taskId}`
                });
            }

            const pendingTx = this.pendingTransactions.get(taskId);

            res.json({
                taskId: pendingTx.taskId,
                status: pendingTx.status,
                recipient: pendingTx.recipient,
                amount: pendingTx.amount,
                timestamp: pendingTx.timestamp
            });
        });

        // x402 extension: Get payment requirements
        this.app.get('/payment-requirements', (req, res) => {
            const requirements = {
                x402Version: 1,
                accepts: []
            };

            // Add available payment schemes based on mode
            if (this.paymentMode === 'direct' || this.paymentMode === 'hybrid') {
                requirements.accepts.push({
                    scheme: 'exact',
                    network: this.network,
                    asset: this.usdcAddress,
                    recipient: this.wallet.address,
                    minAmount: '1000000',  // 1 USDC
                    maxAmount: '1000000000'  // 1000 USDC
                });
            }

            if (this.paymentMode === 'escrow' || this.paymentMode === 'hybrid') {
                requirements.accepts.push({
                    scheme: 'escrow',
                    network: this.network,
                    asset: this.usdcAddress,
                    escrowContract: this.escrowAddress,
                    minAmount: '1000000',  // 1 USDC
                    maxAmount: '1000000000',  // 1000 USDC
                    extras: {
                        validatorRequired: true,
                        validators: this.validators,
                        taskBased: true
                    }
                });
            }

            res.json(requirements);
        });
    }

    // Check if a payment scheme is allowed based on current mode
    isSchemeAllowed(scheme) {
        if (this.paymentMode === 'hybrid') return true;
        if (this.paymentMode === 'direct' && (scheme === 'exact' || scheme === 'direct')) return true;
        if (this.paymentMode === 'escrow' && scheme === 'escrow') return true;
        return false;
    }

    // Get available capabilities based on payment mode
    getCapabilities() {
        const capabilities = [];
        if (this.paymentMode === 'direct' || this.paymentMode === 'hybrid') {
            capabilities.push('exact', 'direct');
        }
        if (this.paymentMode === 'escrow' || this.paymentMode === 'hybrid') {
            capabilities.push('escrow');
        }
        return capabilities;
    }

    // Verify payment signature and format
    async verifyPayment(payment, scheme) {
        // Basic validation
        if (!payment.amount || !payment.recipient) {
            return false;
        }

        // Check amount is positive
        const amount = ethers.parseUnits(payment.amount, 6);  // USDC has 6 decimals
        if (amount <= 0) {
            return false;
        }

        // For escrow, verify taskId exists
        if (scheme === 'escrow' && !payment.taskId) {
            return false;
        }

        // TODO: Add signature verification
        // This would verify the payment authorization signature

        return true;
    }

    // Store direct payment transaction (standard x402)
    // In direct mode, facilitator receives and holds the signed transaction until validation
    async settleDirectPayment(payment) {
        // Validate that we received a signed transaction from the client
        if (!payment.signedTx) {
            throw new Error('No signed transaction provided by client');
        }

        // Store the signed transaction (already signed by client)
        this.pendingTransactions.set(payment.taskId, {
            taskId: payment.taskId,
            client: payment.client,
            recipient: payment.recipient,
            amount: payment.amount,
            signedTx: payment.signedTx,  // Store the client's signed transaction
            timestamp: Date.now(),
            status: 'pending_validation'
        });

        return {
            taskId: payment.taskId,
            status: 'pending_validation',
            scheme: 'direct',
            amount: payment.amount,
            recipient: payment.recipient,
            message: 'Transaction stored, awaiting validation results'
        };
    }

    // Initiate escrow payment (our enhancement)
    async initiateEscrowPayment(payment, taskId) {
        console.log(`🔒 Processing escrow payment for task ${taskId}`);
        console.log(`   Client: ${payment.client}`);
        console.log(`   Agent: ${payment.agent}`);
        console.log(`   Amount: ${payment.amount} USDC`);

        // Get contracts
        const escrow = new ethers.Contract(this.escrowAddress, ESCROW_ABI, this.wallet);
        const usdc = new ethers.Contract(this.usdcAddress, USDC_ABI, this.wallet);

        const amount = ethers.parseUnits(payment.amount, 6);
        const deadline = Math.floor(Date.now() / 1000) + 3600;  // 1 hour deadline

        // Note: Facilitator has already approved escrow during setup (unlimited allowance)
        // No need to approve again here

        // Convert taskId to bytes32
        const taskIdBytes32 = ethers.id(taskId); // Hash the string to get bytes32

        // Get next nonce from tracker to avoid conflicts
        const nonce = await this.getNextNonce();

        // Deposit to escrow using depositPayment
        // Signature: depositPayment(bytes32 taskId, address client, address agent, uint256 amount, uint256 deadline)
        const tx = await escrow.depositPayment(
            taskIdBytes32,
            payment.client,
            payment.agent,
            amount,
            deadline,
            { nonce }
        );
        const receipt = await tx.wait();

        console.log(`✅ Payment escrowed for task ${taskId}: ${tx.hash}`);

        return {
            transactionHash: tx.hash,
            blockNumber: receipt.blockNumber,
            status: 'escrowed',
            scheme: 'escrow',
            amount: payment.amount,
            taskId: taskId,
            escrowContract: this.escrowAddress,
            deadline: deadline,
            validators: this.validators
        };
    }

    // Get escrow payment status
    async getEscrowStatus(taskId) {
        const escrow = new ethers.Contract(this.escrowAddress, ESCROW_ABI, this.provider);

        // Convert taskId to bytes32
        const taskIdBytes32 = ethers.id(taskId);
        const payment = await escrow.getPayment(taskIdBytes32);

        return {
            taskId: taskId,
            client: payment.client,
            agent: payment.agent,
            amount: ethers.formatUnits(payment.amount, 6),
            depositTime: payment.depositTime.toString(),
            deadline: payment.deadline.toString(),
            status: payment.status, // This is uint8 enum value
            statusName: ['NONE', 'DEPOSITED', 'COMPLETED', 'REFUNDED', 'EXPIRED'][payment.status]
        };
    }

    // Release payment from escrow
    async releaseFromEscrow(taskId, validatorApprovals) {
        console.log(`🔓 Releasing payment for task ${taskId}`);

        // In production, would verify validator signatures
        // For now, just check count
        if (!validatorApprovals || validatorApprovals.length === 0) {
            throw new Error('No validator approvals provided');
        }

        const escrow = new ethers.Contract(this.escrowAddress, ESCROW_ABI, this.wallet);

        // Convert taskId to bytes32
        const taskIdBytes32 = ethers.id(taskId);

        // Get next nonce from tracker to avoid conflicts
        const nonce = await this.getNextNonce();

        const tx = await escrow.releasePayment(taskIdBytes32, { nonce });
        const receipt = await tx.wait();

        console.log(`✅ Payment released for task ${taskId}: ${tx.hash}`);

        return {
            transactionHash: tx.hash,
            blockNumber: receipt.blockNumber,
            status: 'released',
            taskId: taskId
        };
    }

    // Refund payment from escrow
    async refundFromEscrow(taskId, reason) {
        console.log(`↩️  Refunding payment for task ${taskId}`);
        console.log(`   Reason: ${reason || 'Not specified'}`);

        const escrow = new ethers.Contract(this.escrowAddress, ESCROW_ABI, this.wallet);

        // Convert taskId to bytes32
        const taskIdBytes32 = ethers.id(taskId);

        // Get next nonce from tracker to avoid conflicts
        const nonce = await this.getNextNonce();

        const tx = await escrow.refundPayment(taskIdBytes32, { nonce });
        const receipt = await tx.wait();

        console.log(`✅ Payment refunded for task ${taskId}: ${tx.hash}`);

        return {
            transactionHash: tx.hash,
            blockNumber: receipt.blockNumber,
            status: 'refunded',
            taskId: taskId,
            reason: reason
        };
    }

    // Start the server
    listen(port) {
        this.app.listen(port, () => {
            console.log(`✅ X402 Facilitator running on port ${port}`);
            console.log(`   Health check: http://localhost:${port}/health`);
            console.log(`   Payment requirements: http://localhost:${port}/payment-requirements`);
        });
    }
}

// Parse command line arguments
const args = process.argv.slice(2);
const config = {
    network: process.env.NETWORK || 'local',
    rpcUrl: process.env.RPC_URL || 'http://localhost:8545',
    privateKey: process.env.FACILITATOR_KEY,
    escrowAddress: process.env.ESCROW_ADDRESS,
    usdcAddress: process.env.USDC_ADDRESS || process.env.PAYMENT_TOKEN_ADDRESS,
    validators: process.env.VALIDATORS ? process.env.VALIDATORS.split(',') : [],
    paymentMode: process.env.PAYMENT_MODE || 'hybrid',
    port: process.env.FACILITATOR_PORT || 3002
};

// Parse command line args
for (let i = 0; i < args.length; i++) {
    if (args[i] === '--network') config.network = args[++i];
    if (args[i] === '--port') config.port = parseInt(args[++i]);
    if (args[i] === '--mode') config.paymentMode = args[++i];
    if (args[i] === '--key') config.privateKey = args[++i];
    if (args[i] === '--escrow') config.escrowAddress = args[++i];
    if (args[i] === '--usdc') config.usdcAddress = args[++i];
}

// Validate required configuration
if (!config.privateKey) {
    console.error('❌ Facilitator private key not provided');
    process.exit(1);
}

// Create and start facilitator
const facilitator = new X402Facilitator(config);
facilitator.listen(config.port);