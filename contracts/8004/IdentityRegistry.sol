// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/cryptography/EIP712.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/interfaces/IERC1271.sol";

/// @title IdentityRegistry - ERC-8004 Agent Identity Registry
/// @notice Manages decentralized AI agent identities as ERC-721 tokens
/// @dev Implements ERC-8004 specification with EIP-712 signature verification
contract IdentityRegistry is ERC721URIStorage, Ownable, EIP712 {
    uint256 private _lastId;

    // agentId => metadataKey => metadataValue
    mapping(uint256 => mapping(string => bytes)) private _metadata;

    // agentId => verified agent wallet address
    mapping(uint256 => address) private _agentWallet;

    struct MetadataEntry {
        string metadataKey;
        bytes metadataValue;
    }

    // EIP-712 typehash for setAgentWallet
    bytes32 private constant AGENT_WALLET_SET_TYPEHASH =
        keccak256("AgentWalletSet(uint256 agentId,address newWallet,address owner,uint256 deadline)");

    // ERC-1271 magic value for contract signature validation
    bytes4 private constant ERC1271_MAGICVALUE = 0x1626ba7e;

    // Maximum deadline delay (5 minutes)
    uint256 private constant MAX_DEADLINE_DELAY = 5 minutes;

    // Reserved metadata key hash (cannot be set via setMetadata)
    bytes32 private constant RESERVED_AGENT_WALLET_KEY_HASH = keccak256("agentWallet");

    event Registered(uint256 indexed agentId, string agentURI, address indexed owner);
    event MetadataSet(uint256 indexed agentId, string indexed indexedMetadataKey, string metadataKey, bytes metadataValue);
    event URIUpdated(uint256 indexed agentId, string newURI, address indexed updatedBy);
    event AgentWalletSet(uint256 indexed agentId, address indexed wallet, address indexed setBy);

    constructor()
        ERC721("AgentIdentity", "AGENT")
        Ownable(msg.sender)
        EIP712("ERC8004IdentityRegistry", "1")
    {}

    /// @notice Register a new agent identity
    /// @return agentId The ID of the newly registered agent
    function register() external returns (uint256 agentId) {
        agentId = _lastId++;
        _safeMint(msg.sender, agentId);
        _agentWallet[agentId] = msg.sender;
        emit Registered(agentId, "", msg.sender);
    }

    /// @notice Register a new agent identity with URI
    /// @param agentURI The URI pointing to agent metadata (AgentCard)
    /// @return agentId The ID of the newly registered agent
    function register(string memory agentURI) external returns (uint256 agentId) {
        agentId = _lastId++;
        _safeMint(msg.sender, agentId);
        _agentWallet[agentId] = msg.sender;
        _setTokenURI(agentId, agentURI);
        emit Registered(agentId, agentURI, msg.sender);
    }

    /// @notice Register a new agent identity with URI and metadata
    /// @param agentURI The URI pointing to agent metadata
    /// @param metadata Array of metadata key-value pairs
    /// @return agentId The ID of the newly registered agent
    function register(string memory agentURI, MetadataEntry[] memory metadata) external returns (uint256 agentId) {
        agentId = _lastId++;
        _safeMint(msg.sender, agentId);
        _agentWallet[agentId] = msg.sender;
        _setTokenURI(agentId, agentURI);
        emit Registered(agentId, agentURI, msg.sender);

        for (uint256 i = 0; i < metadata.length; i++) {
            require(keccak256(bytes(metadata[i].metadataKey)) != RESERVED_AGENT_WALLET_KEY_HASH, "reserved key");
            _metadata[agentId][metadata[i].metadataKey] = metadata[i].metadataValue;
            emit MetadataSet(agentId, metadata[i].metadataKey, metadata[i].metadataKey, metadata[i].metadataValue);
        }
    }

    /// @notice Get metadata value for an agent
    /// @param agentId The agent ID
    /// @param metadataKey The metadata key to retrieve
    /// @return The metadata value as bytes
    function getMetadata(uint256 agentId, string memory metadataKey) external view returns (bytes memory) {
        return _metadata[agentId][metadataKey];
    }

    /// @notice Set metadata for an agent
    /// @param agentId The agent ID
    /// @param metadataKey The metadata key (cannot be "agentWallet")
    /// @param metadataValue The metadata value
    function setMetadata(uint256 agentId, string memory metadataKey, bytes memory metadataValue) external {
        require(
            msg.sender == _ownerOf(agentId) ||
            isApprovedForAll(_ownerOf(agentId), msg.sender) ||
            msg.sender == getApproved(agentId),
            "Not authorized"
        );
        require(keccak256(bytes(metadataKey)) != RESERVED_AGENT_WALLET_KEY_HASH, "reserved key");
        _metadata[agentId][metadataKey] = metadataValue;
        emit MetadataSet(agentId, metadataKey, metadataKey, metadataValue);
    }

    /// @notice Update the URI for an agent
    /// @param agentId The agent ID
    /// @param newURI The new URI
    function setAgentURI(uint256 agentId, string calldata newURI) external {
        address owner = ownerOf(agentId);
        require(
            msg.sender == owner ||
            isApprovedForAll(owner, msg.sender) ||
            msg.sender == getApproved(agentId),
            "Not authorized"
        );
        _setTokenURI(agentId, newURI);
        emit URIUpdated(agentId, newURI, msg.sender);
    }

    /// @notice Get the verified wallet address for an agent
    /// @param agentId The agent ID
    /// @return The verified wallet address
    function getAgentWallet(uint256 agentId) external view returns (address) {
        ownerOf(agentId); // Ensure token exists
        return _agentWallet[agentId];
    }

    /// @notice Set a verified wallet for an agent using EIP-712 signature
    /// @param agentId The agent ID
    /// @param newWallet The wallet address to set (must sign the message)
    /// @param deadline Signature expiration timestamp
    /// @param signature The EIP-712 signature from newWallet
    function setAgentWallet(
        uint256 agentId,
        address newWallet,
        uint256 deadline,
        bytes calldata signature
    ) external {
        address owner = ownerOf(agentId);
        require(
            msg.sender == owner ||
            isApprovedForAll(owner, msg.sender) ||
            msg.sender == getApproved(agentId),
            "Not authorized"
        );
        require(newWallet != address(0), "bad wallet");
        require(block.timestamp <= deadline, "expired");
        require(deadline <= block.timestamp + MAX_DEADLINE_DELAY, "deadline too far");

        // Build EIP-712 digest
        bytes32 structHash = keccak256(abi.encode(
            AGENT_WALLET_SET_TYPEHASH,
            agentId,
            newWallet,
            owner,
            deadline
        ));
        bytes32 digest = _hashTypedDataV4(structHash);

        // Verify signature - support both EOA and contract wallets (ERC-1271)
        if (newWallet.code.length == 0) {
            // EOA signature
            address recovered = ECDSA.recover(digest, signature);
            require(recovered == newWallet, "invalid wallet sig");
        } else {
            // Contract wallet (ERC-1271)
            bytes4 result = IERC1271(newWallet).isValidSignature(digest, signature);
            require(result == ERC1271_MAGICVALUE, "invalid wallet sig");
        }

        _agentWallet[agentId] = newWallet;

        // Store as metadata for discoverability
        _metadata[agentId]["agentWallet"] = abi.encodePacked(newWallet);
        emit MetadataSet(agentId, "agentWallet", "agentWallet", abi.encodePacked(newWallet));
        emit AgentWalletSet(agentId, newWallet, msg.sender);
    }

    /// @notice Clear agent wallet (only owner/approved can call)
    /// @param agentId The agent ID
    function clearAgentWallet(uint256 agentId) external {
        address owner = ownerOf(agentId);
        require(
            msg.sender == owner ||
            isApprovedForAll(owner, msg.sender) ||
            msg.sender == getApproved(agentId),
            "Not authorized"
        );
        _agentWallet[agentId] = address(0);
        _metadata[agentId]["agentWallet"] = "";
        emit MetadataSet(agentId, "agentWallet", "agentWallet", "");
        emit AgentWalletSet(agentId, address(0), msg.sender);
    }

    /// @dev Override _update to clear agentWallet on transfer
    /// @notice Ensures verified wallet doesn't persist to new owners
    function _update(address to, uint256 tokenId, address auth) internal override returns (address) {
        address from = _ownerOf(tokenId);

        // Call parent implementation
        address result = super._update(to, tokenId, auth);

        // If this is a transfer (not mint/burn), clear agentWallet
        if (from != address(0) && to != address(0)) {
            _agentWallet[tokenId] = address(0);
            _metadata[tokenId]["agentWallet"] = "";
            emit MetadataSet(tokenId, "agentWallet", "agentWallet", "");
        }

        return result;
    }

    /// @notice Get contract version
    /// @return Version string
    function getVersion() external pure returns (string memory) {
        return "1.0.0";
    }

    /// @notice Get the next agent ID that will be assigned
    /// @return The next agent ID
    function getNextAgentId() external view returns (uint256) {
        return _lastId;
    }
}