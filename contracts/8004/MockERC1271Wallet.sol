// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";
import "@openzeppelin/contracts/interfaces/IERC1271.sol";

/**
 * @title MockERC1271Wallet
 * @dev Mock implementation of ERC-1271 for testing signature validation
 * Used to test how the ReputationRegistry handles contract-based signatures
 */
contract MockERC1271Wallet is IERC1271 {
    using ECDSA for bytes32;
    using MessageHashUtils for bytes32;

    address public owner;

    constructor(address _owner) {
        owner = _owner;
    }

    function isValidSignature(
        bytes32 hash,
        bytes memory signature
    ) public view override returns (bytes4) {
        address recoveredSigner = hash.recover(signature);
        if (recoveredSigner == owner) {
            return IERC1271.isValidSignature.selector;
        } else {
            return bytes4(0xffffffff);
        }
    }
}