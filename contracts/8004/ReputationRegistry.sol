// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IIdentityRegistry {
    function ownerOf(uint256 tokenId) external view returns (address);
    function isApprovedForAll(address owner, address operator) external view returns (bool);
    function getApproved(uint256 tokenId) external view returns (address);
}

/// @title ReputationRegistry - ERC-8004 Reputation Registry v2.0
/// @notice Manages reputation feedback for AI agents
/// @dev Implements ERC-8004 specification v2.0 with int256 values and decimals
contract ReputationRegistry {
    address private immutable identityRegistry;

    event NewFeedback(
        uint256 indexed agentId,
        address indexed clientAddress,
        uint64 feedbackIndex,
        int256 value,
        uint8 valueDecimals,
        string indexed indexedTag1,
        string tag1,
        string tag2,
        string endpoint,
        string feedbackURI,
        bytes32 feedbackHash
    );

    event FeedbackRevoked(
        uint256 indexed agentId,
        address indexed clientAddress,
        uint64 indexed feedbackIndex
    );

    event ResponseAppended(
        uint256 indexed agentId,
        address indexed clientAddress,
        uint64 feedbackIndex,
        address indexed responder,
        string responseURI,
        bytes32 responseHash
    );

    struct Feedback {
        int256 value;
        uint8 valueDecimals;
        string tag1;
        string tag2;
        string endpoint;
        bool isRevoked;
    }

    // agentId => clientAddress => feedbackIndex => Feedback (1-indexed)
    mapping(uint256 => mapping(address => mapping(uint64 => Feedback))) private _feedback;

    // agentId => clientAddress => last feedback index
    mapping(uint256 => mapping(address => uint64)) private _lastIndex;

    // agentId => clientAddress => feedbackIndex => responder => response count
    mapping(uint256 => mapping(address => mapping(uint64 => mapping(address => uint64)))) private _responseCount;

    // Track all unique responders for each feedback
    mapping(uint256 => mapping(address => mapping(uint64 => address[]))) private _responders;
    mapping(uint256 => mapping(address => mapping(uint64 => mapping(address => bool)))) private _responderExists;

    // Track all unique clients that have given feedback for each agent
    mapping(uint256 => address[]) private _clients;
    mapping(uint256 => mapping(address => bool)) private _clientExists;

    constructor(address _identityRegistry) {
        require(_identityRegistry != address(0), "bad identity");
        identityRegistry = _identityRegistry;
    }

    function getIdentityRegistry() external view returns (address) {
        return identityRegistry;
    }

    /// @notice Give feedback for an agent (v2.0 - int256 value with decimals)
    /// @param agentId The agent's identity token ID
    /// @param value The feedback value (can be negative)
    /// @param valueDecimals Decimal places for the value
    /// @param tag1 Primary tag for categorization
    /// @param tag2 Secondary tag for categorization
    /// @param endpoint The service endpoint
    /// @param feedbackURI URI pointing to detailed feedback
    /// @param feedbackHash Hash of the feedback content
    function giveFeedback(
        uint256 agentId,
        int256 value,
        uint8 valueDecimals,
        string calldata tag1,
        string calldata tag2,
        string calldata endpoint,
        string calldata feedbackURI,
        bytes32 feedbackHash
    ) external {
        // Verify agent exists
        require(_agentExists(agentId), "Agent does not exist");

        // Get agent owner
        IIdentityRegistry registry = IIdentityRegistry(identityRegistry);
        address agentOwner = registry.ownerOf(agentId);

        // SECURITY: Prevent self-feedback from owner and operators
        require(
            msg.sender != agentOwner &&
            !registry.isApprovedForAll(agentOwner, msg.sender) &&
            registry.getApproved(agentId) != msg.sender,
            "Self-feedback not allowed"
        );

        // Get current index for this client-agent pair (1-indexed)
        uint64 currentIndex = _lastIndex[agentId][msg.sender] + 1;

        // Store feedback at 1-indexed position
        _feedback[agentId][msg.sender][currentIndex] = Feedback({
            value: value,
            valueDecimals: valueDecimals,
            tag1: tag1,
            tag2: tag2,
            endpoint: endpoint,
            isRevoked: false
        });

        // Update last index
        _lastIndex[agentId][msg.sender] = currentIndex;

        // Track new client
        if (!_clientExists[agentId][msg.sender]) {
            _clients[agentId].push(msg.sender);
            _clientExists[agentId][msg.sender] = true;
        }

        emit NewFeedback(agentId, msg.sender, currentIndex, value, valueDecimals, tag1, tag1, tag2, endpoint, feedbackURI, feedbackHash);
    }

    function revokeFeedback(uint256 agentId, uint64 feedbackIndex) external {
        require(feedbackIndex > 0, "index must be > 0");
        require(feedbackIndex <= _lastIndex[agentId][msg.sender], "index out of bounds");
        require(!_feedback[agentId][msg.sender][feedbackIndex].isRevoked, "Already revoked");

        _feedback[agentId][msg.sender][feedbackIndex].isRevoked = true;
        emit FeedbackRevoked(agentId, msg.sender, feedbackIndex);
    }

    function appendResponse(
        uint256 agentId,
        address clientAddress,
        uint64 feedbackIndex,
        string calldata responseURI,
        bytes32 responseHash
    ) external {
        require(feedbackIndex > 0, "index must be > 0");
        require(feedbackIndex <= _lastIndex[agentId][clientAddress], "index out of bounds");
        require(bytes(responseURI).length > 0, "Empty URI");

        // Track new responder
        if (!_responderExists[agentId][clientAddress][feedbackIndex][msg.sender]) {
            _responders[agentId][clientAddress][feedbackIndex].push(msg.sender);
            _responderExists[agentId][clientAddress][feedbackIndex][msg.sender] = true;
        }

        // Increment response count for this responder
        _responseCount[agentId][clientAddress][feedbackIndex][msg.sender]++;

        emit ResponseAppended(agentId, clientAddress, feedbackIndex, msg.sender, responseURI, responseHash);
    }

    function getLastIndex(uint256 agentId, address clientAddress) external view returns (uint64) {
        return _lastIndex[agentId][clientAddress];
    }

    /// @notice Read feedback details (v2.0)
    function readFeedback(uint256 agentId, address clientAddress, uint64 feedbackIndex)
        external
        view
        returns (int256 value, uint8 valueDecimals, string memory tag1, string memory tag2, bool isRevoked)
    {
        require(feedbackIndex > 0, "index must be > 0");
        require(feedbackIndex <= _lastIndex[agentId][clientAddress], "index out of bounds");
        Feedback storage f = _feedback[agentId][clientAddress][feedbackIndex];
        return (f.value, f.valueDecimals, f.tag1, f.tag2, f.isRevoked);
    }

    /// @notice Get summary of feedback for an agent (v2.0)
    /// @param agentId The agent's identity token ID
    /// @param clientAddresses List of client addresses to include (required, cannot be empty)
    /// @param tag1 Filter by tag1 (empty string for no filter)
    /// @param tag2 Filter by tag2 (empty string for no filter)
    /// @return count Number of feedbacks
    /// @return summaryValue Sum of all values (or average depending on implementation)
    /// @return summaryValueDecimals Decimal places for summaryValue
    function getSummary(
        uint256 agentId,
        address[] calldata clientAddresses,
        string calldata tag1,
        string calldata tag2
    ) external view returns (uint64 count, int256 summaryValue, uint8 summaryValueDecimals) {
        require(clientAddresses.length > 0, "clientAddresses required");

        int256 totalValue = 0;
        count = 0;
        summaryValueDecimals = 0;

        bytes32 tag1Hash = keccak256(bytes(tag1));
        bytes32 tag2Hash = keccak256(bytes(tag2));
        bool filterTag1 = bytes(tag1).length > 0;
        bool filterTag2 = bytes(tag2).length > 0;

        for (uint256 i = 0; i < clientAddresses.length; i++) {
            uint64 lastIdx = _lastIndex[agentId][clientAddresses[i]];
            for (uint64 j = 1; j <= lastIdx; j++) {
                Feedback storage fb = _feedback[agentId][clientAddresses[i]][j];
                if (fb.isRevoked) continue;
                if (filterTag1 && keccak256(bytes(fb.tag1)) != tag1Hash) continue;
                if (filterTag2 && keccak256(bytes(fb.tag2)) != tag2Hash) continue;

                totalValue += fb.value;
                count++;
            }
        }

        // Return average as summaryValue
        summaryValue = count > 0 ? totalValue / int256(uint256(count)) : int256(0);
    }

    /// @notice Read all feedback for an agent (v2.0)
    function readAllFeedback(
        uint256 agentId,
        address[] calldata clientAddresses,
        string calldata tag1,
        string calldata tag2,
        bool includeRevoked
    ) external view returns (
        address[] memory clients,
        uint64[] memory feedbackIndexes,
        int256[] memory values,
        uint8[] memory valueDecimals,
        string[] memory tag1s,
        string[] memory tag2s,
        bool[] memory revokedStatuses
    ) {
        require(clientAddresses.length > 0, "clientAddresses required");

        bytes32 tag1Hash = keccak256(bytes(tag1));
        bytes32 tag2Hash = keccak256(bytes(tag2));
        bool filterTag1 = bytes(tag1).length > 0;
        bool filterTag2 = bytes(tag2).length > 0;

        // First pass: count matching feedback
        uint256 totalCount = 0;
        for (uint256 i = 0; i < clientAddresses.length; i++) {
            uint64 lastIdx = _lastIndex[agentId][clientAddresses[i]];
            for (uint64 j = 1; j <= lastIdx; j++) {
                Feedback storage fb = _feedback[agentId][clientAddresses[i]][j];
                if (!includeRevoked && fb.isRevoked) continue;
                if (filterTag1 && keccak256(bytes(fb.tag1)) != tag1Hash) continue;
                if (filterTag2 && keccak256(bytes(fb.tag2)) != tag2Hash) continue;
                totalCount++;
            }
        }

        // Initialize arrays
        clients = new address[](totalCount);
        feedbackIndexes = new uint64[](totalCount);
        values = new int256[](totalCount);
        valueDecimals = new uint8[](totalCount);
        tag1s = new string[](totalCount);
        tag2s = new string[](totalCount);
        revokedStatuses = new bool[](totalCount);

        // Second pass: populate arrays
        uint256 idx = 0;
        for (uint256 i = 0; i < clientAddresses.length; i++) {
            uint64 lastIdx = _lastIndex[agentId][clientAddresses[i]];
            for (uint64 j = 1; j <= lastIdx; j++) {
                Feedback storage fb = _feedback[agentId][clientAddresses[i]][j];
                if (!includeRevoked && fb.isRevoked) continue;
                if (filterTag1 && keccak256(bytes(fb.tag1)) != tag1Hash) continue;
                if (filterTag2 && keccak256(bytes(fb.tag2)) != tag2Hash) continue;

                clients[idx] = clientAddresses[i];
                feedbackIndexes[idx] = j;
                values[idx] = fb.value;
                valueDecimals[idx] = fb.valueDecimals;
                tag1s[idx] = fb.tag1;
                tag2s[idx] = fb.tag2;
                revokedStatuses[idx] = fb.isRevoked;
                idx++;
            }
        }
    }

    function getResponseCount(
        uint256 agentId,
        address clientAddress,
        uint64 feedbackIndex,
        address[] calldata responders
    ) external view returns (uint64 count) {
        if (clientAddress == address(0)) {
            // Count all responses for all clients
            address[] memory clients = _clients[agentId];
            for (uint256 i = 0; i < clients.length; i++) {
                uint64 lastIdx = _lastIndex[agentId][clients[i]];
                for (uint64 j = 1; j <= lastIdx; j++) {
                    count += _countResponses(agentId, clients[i], j, responders);
                }
            }
        } else if (feedbackIndex == 0) {
            // Count all responses for specific clientAddress
            uint64 lastIdx = _lastIndex[agentId][clientAddress];
            for (uint64 j = 1; j <= lastIdx; j++) {
                count += _countResponses(agentId, clientAddress, j, responders);
            }
        } else {
            // Count responses for specific clientAddress and feedbackIndex
            count = _countResponses(agentId, clientAddress, feedbackIndex, responders);
        }
    }

    function _countResponses(
        uint256 agentId,
        address clientAddress,
        uint64 feedbackIndex,
        address[] calldata responders
    ) internal view returns (uint64 count) {
        if (responders.length == 0) {
            // Count from all responders
            address[] memory allResponders = _responders[agentId][clientAddress][feedbackIndex];
            for (uint256 k = 0; k < allResponders.length; k++) {
                count += _responseCount[agentId][clientAddress][feedbackIndex][allResponders[k]];
            }
        } else {
            // Count from specified responders
            for (uint256 k = 0; k < responders.length; k++) {
                count += _responseCount[agentId][clientAddress][feedbackIndex][responders[k]];
            }
        }
    }

    function getClients(uint256 agentId) external view returns (address[] memory) {
        return _clients[agentId];
    }

    function _agentExists(uint256 agentId) internal view returns (bool) {
        try IIdentityRegistry(identityRegistry).ownerOf(agentId) returns (address owner) {
            return owner != address(0);
        } catch {
            return false;
        }
    }

    /// @notice Get contract version
    /// @return Version string
    function getVersion() external pure returns (string memory) {
        return "2.0.0";
    }
}
