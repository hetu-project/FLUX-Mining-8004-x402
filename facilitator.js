#!/usr/bin/env node

/**
 * x402 V2 Facilitator Service
 *
 * Handles payment verification and settlement for x402 V2 protocol.
 * Supports two payment modes:
 * - direct: per-task payments
 * - session: per-epoch payments (batched tasks)
 */

const express = require('express');
const { ethers } = require('ethers');
const cors = require('cors');

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

        this.network = config.network || 'local';
        this.rpcUrl = config.rpcUrl || 'http://localhost:8545';
        this.provider = new ethers.JsonRpcProvider(this.rpcUrl);

        this.privateKey = config.privateKey;
        this.wallet = new ethers.Wallet(this.privateKey, this.provider);

        this.usdcAddress = config.usdcAddress;
        this.paymentMode = config.paymentMode || 'direct';

        // Pending transactions awaiting validation
        this.pendingTransactions = new Map();

        console.log('🌐 x402 V2 Facilitator:');
        console.log(`   Network: ${this.network}`);
        console.log(`   Facilitator: ${this.wallet.address}`);
        console.log(`   Payment Mode: ${this.paymentMode}`);
        console.log(`   USDC: ${this.usdcAddress}`);

        this.setupEndpoints();
    }

    setupEndpoints() {
        // Health check
        this.app.get('/health', (req, res) => {
            res.json({
                status: 'healthy',
                facilitator: this.wallet.address,
                network: this.network,
                paymentMode: this.paymentMode
            });
        });

        // Payment requirements
        this.app.get('/payment-requirements', (req, res) => {
            res.json({
                x402Version: 2,
                accepts: [{
                    scheme: 'direct',
                    network: this.network,
                    asset: this.usdcAddress,
                    recipient: this.wallet.address,
                    minAmount: '1000000',
                    maxAmount: '1000000000'
                }]
            });
        });

        // Settle payment (store pending transaction)
        this.app.post('/settle', async (req, res) => {
            try {
                const { payment, scheme, taskId } = req.body;

                if (scheme !== 'direct' && scheme !== 'exact') {
                    return res.status(400).json({ error: `Unsupported scheme: ${scheme}` });
                }

                if (!payment.signedTx) {
                    return res.status(400).json({ error: 'No signed transaction provided' });
                }

                this.pendingTransactions.set(payment.taskId, {
                    taskId: payment.taskId,
                    client: payment.client,
                    recipient: payment.recipient,
                    amount: payment.amount,
                    signedTx: payment.signedTx,
                    timestamp: Date.now(),
                    status: 'pending_validation'
                });

                res.json({
                    taskId: payment.taskId,
                    status: 'pending_validation',
                    scheme: 'direct',
                    amount: payment.amount,
                    recipient: payment.recipient
                });
            } catch (error) {
                console.error('❌ Settlement error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Finalize payment (broadcast or discard)
        this.app.post('/direct/finalize', async (req, res) => {
            try {
                const { taskId, approved, partial, approvedTasks, totalTasks } = req.body;

                if (!this.pendingTransactions.has(taskId)) {
                    return res.status(404).json({ error: `No pending transaction: ${taskId}` });
                }

                const pendingTx = this.pendingTransactions.get(taskId);

                if (!approved) {
                    this.pendingTransactions.delete(taskId);
                    return res.json({ status: 'discarded', taskId });
                }

                // Partial payment (session mode)
                if (partial && approvedTasks !== undefined && totalTasks !== undefined) {
                    const fullAmountWei = ethers.parseUnits(pendingTx.amount, 6);
                    const partialAmountWei = (fullAmountWei * BigInt(approvedTasks)) / BigInt(totalTasks);

                    console.log(`📊 Partial payment: ${approvedTasks}/${totalTasks} = ${ethers.formatUnits(partialAmountWei, 6)} USDC`);

                    const usdcContract = new ethers.Contract(this.usdcAddress, USDC_ABI, this.wallet);
                    const tx = await usdcContract.transferFrom(pendingTx.client, pendingTx.recipient, partialAmountWei);
                    const receipt = await tx.wait();

                    this.pendingTransactions.delete(taskId);

                    return res.json({
                        transactionHash: tx.hash,
                        blockNumber: receipt.blockNumber,
                        status: 'completed',
                        scheme: 'direct',
                        taskId,
                        approvedTasks,
                        totalTasks,
                        amountPaid: ethers.formatUnits(partialAmountWei, 6)
                    });
                }

                // Full payment
                const txResponse = await this.provider.broadcastTransaction(pendingTx.signedTx);
                const receipt = await txResponse.wait();

                this.pendingTransactions.delete(taskId);

                res.json({
                    transactionHash: txResponse.hash,
                    blockNumber: receipt.blockNumber,
                    status: 'completed',
                    scheme: 'direct',
                    taskId
                });
            } catch (error) {
                console.error('❌ Finalization error:', error);
                res.status(500).json({ error: error.message });
            }
        });

        // Check pending transaction status
        this.app.get('/direct/status/:taskId', (req, res) => {
            const { taskId } = req.params;

            if (!this.pendingTransactions.has(taskId)) {
                return res.status(404).json({ error: `Not found: ${taskId}` });
            }

            const tx = this.pendingTransactions.get(taskId);
            res.json({
                taskId: tx.taskId,
                status: tx.status,
                recipient: tx.recipient,
                amount: tx.amount
            });
        });
    }

    start(port) {
        this.app.listen(port, () => {
            console.log(`✅ Facilitator running on port ${port}`);
        });
    }
}

// Start facilitator
const config = {
    network: process.env.NETWORK || 'local',
    rpcUrl: process.env.RPC_URL || 'http://localhost:8545',
    privateKey: process.env.FACILITATOR_KEY,
    usdcAddress: process.env.USDC_ADDRESS || process.env.PAYMENT_TOKEN_ADDRESS,
    paymentMode: process.env.PAYMENT_MODE || 'direct',
    port: process.env.FACILITATOR_PORT || 3002
};

if (!config.privateKey) {
    console.error('❌ FACILITATOR_KEY not set');
    process.exit(1);
}

const facilitator = new X402Facilitator(config);
facilitator.start(config.port);
