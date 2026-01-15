// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IIdentityRegistry {
    function ownerOf(uint256 tokenId) external view returns (address);
    function isApprovedForAll(address owner, address operator) external view returns (bool);
    function getApproved(uint256 tokenId) external view returns (address);
}

/// @title ValidationRegistry - ERC-8004 Validation Registry
/// @notice Manages validation requests and responses for AI agents
/// @dev Implements ERC-8004 specification
contract ValidationRegistry {
    address private immutable identityRegistry;

    event ValidationRequest(
        address indexed validatorAddress,
        uint256 indexed agentId,
        string requestURI,
        bytes32 indexed requestHash
    );

    event ValidationResponse(
        address indexed validatorAddress,
        uint256 indexed agentId,
        bytes32 indexed requestHash,
        uint8 response,
        string responseURI,
        bytes32 responseHash,
        string tag
    );

    struct ValidationStatus {
        address validatorAddress;
        uint256 agentId;
        uint8 response;
        bytes32 responseHash;
        string tag;
        uint256 lastUpdate;
        bool hasResponse;
    }

    // requestHash => validation status
    mapping(bytes32 => ValidationStatus) private _validations;

    // agentId => list of requestHashes
    mapping(uint256 => bytes32[]) private _agentValidations;

    // validatorAddress => list of requestHashes
    mapping(address => bytes32[]) private _validatorRequests;

    constructor(address _identityRegistry) {
        require(_identityRegistry != address(0), "bad identity");
        identityRegistry = _identityRegistry;
    }

    /// @notice Get the identity registry address
    function getIdentityRegistry() external view returns (address) {
        return identityRegistry;
    }

    /// @notice Request validation for an agent
    /// @param validatorAddress The address of the validator to perform validation
    /// @param agentId The agent ID to validate
    /// @param requestURI URI containing validation request details
    /// @param requestHash Hash of the request data
    function validationRequest(
        address validatorAddress,
        uint256 agentId,
        string calldata requestURI,
        bytes32 requestHash
    ) external {
        require(validatorAddress != address(0), "bad validator");
        require(_validations[requestHash].validatorAddress == address(0), "exists");

        // Check permission: caller must be owner or approved operator
        IIdentityRegistry registry = IIdentityRegistry(identityRegistry);
        address owner = registry.ownerOf(agentId);
        require(
            msg.sender == owner ||
            registry.isApprovedForAll(owner, msg.sender) ||
            registry.getApproved(agentId) == msg.sender,
            "Not authorized"
        );

        _validations[requestHash] = ValidationStatus({
            validatorAddress: validatorAddress,
            agentId: agentId,
            response: 0,
            responseHash: bytes32(0),
            tag: "",
            lastUpdate: block.timestamp,
            hasResponse: false
        });

        // Track for lookups
        _agentValidations[agentId].push(requestHash);
        _validatorRequests[validatorAddress].push(requestHash);

        emit ValidationRequest(validatorAddress, agentId, requestURI, requestHash);
    }

    /// @notice Submit validation response
    /// @param requestHash The request hash to respond to
    /// @param response Validation score (0-100)
    /// @param responseURI URI containing response details
    /// @param responseHash Hash of the response data
    /// @param tag Category tag for the validation
    function validationResponse(
        bytes32 requestHash,
        uint8 response,
        string calldata responseURI,
        bytes32 responseHash,
        string calldata tag
    ) external {
        ValidationStatus storage s = _validations[requestHash];
        require(s.validatorAddress != address(0), "unknown");
        require(msg.sender == s.validatorAddress, "not validator");
        require(response <= 100, "resp>100");

        s.response = response;
        s.responseHash = responseHash;
        s.tag = tag;
        s.lastUpdate = block.timestamp;
        s.hasResponse = true;

        emit ValidationResponse(s.validatorAddress, s.agentId, requestHash, response, responseURI, responseHash, tag);
    }

    /// @notice Get validation status for a request
    /// @param requestHash The request hash to query
    function getValidationStatus(bytes32 requestHash)
        external
        view
        returns (
            address validatorAddress,
            uint256 agentId,
            uint8 response,
            bytes32 responseHash,
            string memory tag,
            uint256 lastUpdate,
            bool hasResponse
        )
    {
        ValidationStatus storage s = _validations[requestHash];
        require(s.validatorAddress != address(0), "unknown");
        return (s.validatorAddress, s.agentId, s.response, s.responseHash, s.tag, s.lastUpdate, s.hasResponse);
    }

    /// @notice Get aggregated validation summary for an agent
    /// @param agentId The agent ID to query
    /// @param validatorAddresses Filter by validators (empty = all)
    /// @param tag Filter by tag (empty = all)
    function getSummary(
        uint256 agentId,
        address[] calldata validatorAddresses,
        string calldata tag
    ) external view returns (uint64 count, uint8 avgResponse) {
        uint256 totalResponse = 0;
        count = 0;

        bytes32[] storage requestHashes = _agentValidations[agentId];
        bytes32 tagHash = keccak256(bytes(tag));
        bool filterTag = bytes(tag).length > 0;

        for (uint256 i = 0; i < requestHashes.length; i++) {
            ValidationStatus storage s = _validations[requestHashes[i]];

            // Only count responses that have been submitted
            if (!s.hasResponse) continue;

            // Filter by validator if specified
            bool matchValidator = (validatorAddresses.length == 0);
            if (!matchValidator) {
                for (uint256 j = 0; j < validatorAddresses.length; j++) {
                    if (s.validatorAddress == validatorAddresses[j]) {
                        matchValidator = true;
                        break;
                    }
                }
            }

            // Filter by tag
            bool matchTag = !filterTag || (keccak256(bytes(s.tag)) == tagHash);

            if (matchValidator && matchTag) {
                totalResponse += s.response;
                count++;
            }
        }

        avgResponse = count > 0 ? uint8(totalResponse / count) : 0;
    }

    /// @notice Get all validation request hashes for an agent
    /// @param agentId The agent ID to query
    function getAgentValidations(uint256 agentId) external view returns (bytes32[] memory) {
        return _agentValidations[agentId];
    }

    /// @notice Get all validation request hashes for a validator
    /// @param validatorAddress The validator address to query
    function getValidatorRequests(address validatorAddress) external view returns (bytes32[] memory) {
        return _validatorRequests[validatorAddress];
    }

    /// @notice Get contract version
    /// @return Version string
    function getVersion() external pure returns (string memory) {
        return "1.0.0";
    }
}
