// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./HETUToken.sol";

// Interface for ERC-8004 IdentityRegistry
interface IIdentityRegistry {
    function ownerOf(uint256 tokenId) external view returns (address);
}

/**
 * @title Subnet Registry
 * @dev Subnet registration with ERC-8004 identity verification
 * Miners must provide their agent ID and ownership is verified
 */
contract SubnetRegistry {
    HETUToken public hetuToken;
    IIdentityRegistry public identityRegistry;

    struct Subnet {
        string subnetId;
        address miner;
        uint256 minerAgentId;  // Added: Store the miner's agent ID
        address[4] validators;
        uint256 minerDeposit;
        uint256 validatorDeposit;
        bool isActive;
        uint256 registeredAt;
    }

    mapping(string => Subnet) public subnets;
    mapping(address => string) public participantToSubnet;
    mapping(uint256 => string) public agentIdToSubnet;  // Added: Track which subnet an agent ID is in

    uint256 public constant MINER_DEPOSIT = 500 * 10**18; // 500 HETU
    uint256 public constant VALIDATOR_DEPOSIT = 100 * 10**18; // 100 HETU per validator

    event SubnetRegistered(string subnetId, address miner, uint256 agentId, address[4] validators);
    event SubnetDeactivated(string subnetId);
    event IdentityVerified(address miner, uint256 agentId);

    address public owner;
    bool public initialized;

    constructor() {
        owner = msg.sender;
    }

    function initialize(address _hetuToken, address _identityRegistry) external {
        require(msg.sender == owner, "Only owner can initialize");
        require(!initialized, "Already initialized");
        hetuToken = HETUToken(_hetuToken);
        identityRegistry = IIdentityRegistry(_identityRegistry);
        initialized = true;
    }

    /**
     * @dev Register a new subnet with identity verification
     * @param subnetId Unique subnet identifier
     * @param minerAgentId The ERC-8004 agent ID that the miner owns
     * @param miner Address of the miner (must match agent ID owner)
     * @param validators Array of 4 validator addresses
     */
    function registerSubnet(
        string memory subnetId,
        uint256 minerAgentId,
        address miner,
        address[4] memory validators
    ) external {
        require(initialized, "Contract not initialized");
        require(bytes(subnets[subnetId].subnetId).length == 0, "Subnet already exists");
        require(miner != address(0), "Invalid miner address");

        // IDENTITY VERIFICATION: Check that the miner owns the provided agent ID
        address agentOwner = identityRegistry.ownerOf(minerAgentId);
        require(agentOwner == miner, "Miner does not own the provided agent ID");
        emit IdentityVerified(miner, minerAgentId);

        // Check that this agent ID is not already registered in another subnet
        require(bytes(agentIdToSubnet[minerAgentId]).length == 0, "Agent ID already registered in a subnet");

        // Check all validators are unique and valid
        for (uint i = 0; i < 4; i++) {
            require(validators[i] != address(0), "Invalid validator address");
            require(validators[i] != miner, "Miner cannot be validator");
            for (uint j = i + 1; j < 4; j++) {
                require(validators[i] != validators[j], "Duplicate validator");
            }
        }

        // Collect miner deposit
        require(
            hetuToken.transferFrom(miner, address(this), MINER_DEPOSIT),
            "Miner deposit failed"
        );

        // Collect validator deposits
        for (uint i = 0; i < 4; i++) {
            require(
                hetuToken.transferFrom(validators[i], address(this), VALIDATOR_DEPOSIT),
                "Validator deposit failed"
            );
        }

        // Create subnet with agent ID
        subnets[subnetId] = Subnet({
            subnetId: subnetId,
            miner: miner,
            minerAgentId: minerAgentId,  // Store the agent ID
            validators: validators,
            minerDeposit: MINER_DEPOSIT,
            validatorDeposit: VALIDATOR_DEPOSIT,
            isActive: true,
            registeredAt: block.timestamp
        });

        // Track participants
        participantToSubnet[miner] = subnetId;
        agentIdToSubnet[minerAgentId] = subnetId;  // Track agent ID to subnet

        for (uint i = 0; i < 4; i++) {
            participantToSubnet[validators[i]] = subnetId;
        }

        emit SubnetRegistered(subnetId, miner, minerAgentId, validators);
    }

    /**
     * @dev Get subnet details including agent ID
     */
    function getSubnet(string memory subnetId) external view returns (
        address miner,
        uint256 minerAgentId,
        address[4] memory validators,
        bool isActive
    ) {
        Subnet memory subnet = subnets[subnetId];
        return (subnet.miner, subnet.minerAgentId, subnet.validators, subnet.isActive);
    }

    /**
     * @dev Check if a subnet is active
     */
    function isSubnetActive(string memory subnetId) external view returns (bool) {
        return subnets[subnetId].isActive;
    }

    /**
     * @dev Deactivate a subnet (owner only)
     */
    function deactivateSubnet(string memory subnetId) external {
        require(msg.sender == owner, "Only owner");
        require(subnets[subnetId].isActive, "Subnet not active");

        subnets[subnetId].isActive = false;
        emit SubnetDeactivated(subnetId);
    }

    /**
     * @dev Get which subnet an agent ID is registered in
     */
    function getSubnetByAgentId(uint256 agentId) external view returns (string memory) {
        return agentIdToSubnet[agentId];
    }
}