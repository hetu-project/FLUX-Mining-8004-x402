// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title USDC Mock Token
 * @dev Standard ERC20 stablecoin for x402 payment system
 * This is a simplified USDC implementation for testing/demo purposes
 * In production, you would use the official USDC contract
 */
contract USDC is ERC20, Ownable {
    uint8 private _decimals;

    constructor() ERC20("USD Coin", "USDC") Ownable(msg.sender) {
        _decimals = 6; // USDC uses 6 decimals (same as real USDC)
    }

    /**
     * @dev Returns the number of decimals used for token amounts
     */
    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    /**
     * @dev Mint new USDC tokens (only owner)
     * @param to Address to receive tokens
     * @param amount Amount to mint (in USDC units with 6 decimals)
     */
    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }

    /**
     * @dev Burn USDC tokens
     * @param amount Amount to burn
     */
    function burn(uint256 amount) external {
        _burn(msg.sender, amount);
    }

    /**
     * @dev Burn USDC tokens from specified account (with allowance)
     * @param from Address to burn from
     * @param amount Amount to burn
     */
    function burnFrom(address from, uint256 amount) external {
        uint256 currentAllowance = allowance(from, msg.sender);
        require(currentAllowance >= amount, "ERC20: burn amount exceeds allowance");
        _approve(from, msg.sender, currentAllowance - amount);
        _burn(from, amount);
    }
}
