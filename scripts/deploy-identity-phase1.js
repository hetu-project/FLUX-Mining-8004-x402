#!/usr/bin/env node

/**
 * Phase 1: Deploy and setup basic ERC-8004 identity for existing miners
 * Run this after deploying the MinerIdentity contract
 */

const { ethers } = require('ethers');
const fs = require('fs').promises;

async function deployPhase1() {
    console.log("🆔 ERC-8004 Phase 1: Basic Identity Setup");
    console.log("========================================");

    // Setup provider and signers
    const provider = new ethers.JsonRpcProvider("http://localhost:8545");

    // Use your existing test accounts
    const accounts = {
        deployer: new ethers.Wallet(
            "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
            provider
        ),
        miner: new ethers.Wallet(
            "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba",
            provider
        ),
        validator1: new ethers.Wallet(
            "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
            provider
        )
    };

    console.log("📜 Step 1: Deploy MinerIdentity contract...");

    // Deploy MinerIdentity
    const MinerIdentity = await ethers.getContractFactory(
        JSON.parse(await fs.readFile('./out/MinerIdentity.sol/MinerIdentity.json', 'utf8')).abi,
        JSON.parse(await fs.readFile('./out/MinerIdentity.sol/MinerIdentity.json', 'utf8')).bytecode.object,
        accounts.deployer
    );

    const minerIdentity = await MinerIdentity.deploy();
    await minerIdentity.waitForDeployment();
    const identityAddress = await minerIdentity.getAddress();

    console.log("✅ MinerIdentity deployed at:", identityAddress);

    // Save the address
    const addresses = JSON.parse(await fs.readFile('./contract_addresses.json', 'utf8').catch(() => '{}'));
    addresses.minerIdentity = identityAddress;
    await fs.writeFile('./contract_addresses.json', JSON.stringify(addresses, null, 2));

    console.log("\n🎫 Step 2: Register existing miners with identity NFTs...");

    // Register the demo miner
    console.log("Registering miner:", accounts.miner.address);
    const tx1 = await minerIdentity.registerMiner(
        accounts.miner.address,
        "AI_AGENT",
        "LLM,OPTIMIZATION,ANALYSIS,CAUSAL_REASONING"
    );
    await tx1.wait();

    const minerTokenId = await minerIdentity.getTokenIdByAddress(accounts.miner.address);
    console.log("✅ Miner registered with token ID:", minerTokenId.toString());

    // Register validator as a miner too (they can dual-role)
    console.log("\nRegistering validator1 as dual-role:", accounts.validator1.address);
    const tx2 = await minerIdentity.registerMiner(
        accounts.validator1.address,
        "HYBRID",
        "VALIDATION,QUALITY_ASSESSMENT,CONSENSUS"
    );
    await tx2.wait();

    const validatorTokenId = await minerIdentity.getTokenIdByAddress(accounts.validator1.address);
    console.log("✅ Validator registered with token ID:", validatorTokenId.toString());

    console.log("\n📊 Step 3: Verify registrations...");

    // Check metadata
    const minerMeta = await minerIdentity.minerMetadata(minerTokenId);
    console.log("\nMiner metadata:");
    console.log("  Type:", minerMeta.minerType);
    console.log("  Capabilities:", minerMeta.capabilities);
    console.log("  Active:", minerMeta.isActive);

    console.log("\n🎉 Phase 1 Complete!");
    console.log("========================");
    console.log("Next steps:");
    console.log("1. Run 'npm run test:identity' to test the identity system");
    console.log("2. PoCWVerifier already includes identity verification");
    console.log("3. Proceed to Phase 2 for reputation integration");

    return identityAddress;
}

// Run if called directly
if (require.main === module) {
    deployPhase1()
        .then(() => process.exit(0))
        .catch(error => {
            console.error("❌ Error:", error);
            process.exit(1);
        });
}

module.exports = { deployPhase1 };