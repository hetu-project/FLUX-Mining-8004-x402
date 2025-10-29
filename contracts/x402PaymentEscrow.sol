// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./AIUSD.sol";

/**
 * @title x402PaymentEscrow
 * @notice Escrow contract for x402 payment protocol in FLUX-Mining
 * @dev Manages payment locks, releases, and refunds for AI task payments
 */
contract x402PaymentEscrow is Ownable, ReentrancyGuard {
    AIUSD public immutable aiusd;

    enum PaymentStatus {
        NONE,
        DEPOSITED,
        COMPLETED,
        REFUNDED,
        EXPIRED
    }

    struct TaskPayment {
        bytes32 taskId;
        address client;
        address agent;
        uint256 amount;
        uint256 depositTime;
        uint256 deadline;
        PaymentStatus status;
    }

    // Mapping from taskId to payment details
    mapping(bytes32 => TaskPayment) public payments;

    // Authorized coordinators (V1 validator)
    mapping(address => bool) public authorizedCoordinators;

    // Nonce management for replay protection
    mapping(address => uint256) public nonces;

    // Events
    event PaymentDeposited(
        bytes32 indexed taskId,
        address indexed client,
        address indexed agent,
        uint256 amount,
        uint256 deadline
    );
    event PaymentReleased(bytes32 indexed taskId, address indexed agent, uint256 amount);
    event PaymentRefunded(bytes32 indexed taskId, address indexed client, uint256 amount);
    event PaymentExpired(bytes32 indexed taskId, address indexed client, uint256 amount);
    event CoordinatorAuthorized(address indexed coordinator);
    event CoordinatorRevoked(address indexed coordinator);

    modifier onlyCoordinator() {
        require(authorizedCoordinators[msg.sender], "Not authorized coordinator");
        _;
    }

    constructor(address _aiusd) Ownable(msg.sender) {
        require(_aiusd != address(0), "Invalid AIUSD address");
        aiusd = AIUSD(_aiusd);
    }

    /**
     * @notice Authorize a coordinator (V1 validator)
     * @param coordinator Address to authorize
     */
    function authorizeCoordinator(address coordinator) external onlyOwner {
        require(coordinator != address(0), "Invalid coordinator address");
        authorizedCoordinators[coordinator] = true;
        emit CoordinatorAuthorized(coordinator);
    }

    /**
     * @notice Revoke coordinator authorization
     * @param coordinator Address to revoke
     */
    function revokeCoordinator(address coordinator) external onlyOwner {
        authorizedCoordinators[coordinator] = false;
        emit CoordinatorRevoked(coordinator);
    }

    /**
     * @notice Deposit payment using standard transferFrom (requires prior approval)
     * @param taskId Unique task identifier
     * @param client Client address (payer)
     * @param agent Agent address (payee)
     * @param amount Payment amount
     * @param deadline Payment deadline timestamp
     */
    function depositPayment(
        bytes32 taskId,
        address client,
        address agent,
        uint256 amount,
        uint256 deadline
    ) external onlyCoordinator nonReentrant {
        require(payments[taskId].status == PaymentStatus.NONE, "Payment already exists");
        require(client != address(0) && agent != address(0), "Invalid addresses");
        require(amount > 0, "Amount must be positive");
        require(deadline > block.timestamp, "Deadline already passed");

        // Transfer AIUSD from client to escrow (requires prior approval)
        require(aiusd.transferFrom(client, address(this), amount), "Transfer failed");

        // Store payment details
        payments[taskId] = TaskPayment({
            taskId: taskId,
            client: client,
            agent: agent,
            amount: amount,
            depositTime: block.timestamp,
            deadline: deadline,
            status: PaymentStatus.DEPOSITED
        });

        emit PaymentDeposited(taskId, client, agent, amount, deadline);
    }

    /**
     * @notice Deposit payment using EIP-3009 transferWithAuthorization
     * @param taskId Unique task identifier
     * @param client Client address (payer)
     * @param agent Agent address (payee)
     * @param amount Payment amount
     * @param validAfter Timestamp after which authorization is valid
     * @param validBefore Timestamp before which authorization is valid
     * @param nonce Unique nonce for replay protection
     * @param v ECDSA signature parameter
     * @param r ECDSA signature parameter
     * @param s ECDSA signature parameter
     */
    function depositWithAuthorization(
        bytes32 taskId,
        address client,
        address agent,
        uint256 amount,
        uint256 validAfter,
        uint256 validBefore,
        bytes32 nonce,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) external onlyCoordinator nonReentrant {
        require(payments[taskId].status == PaymentStatus.NONE, "Payment already exists");
        require(client != address(0) && agent != address(0), "Invalid addresses");
        require(amount > 0, "Amount must be positive");
        require(validBefore > block.timestamp, "Deadline already passed");

        // Use EIP-3009 to pull funds from client (gasless for client)
        aiusd.transferWithAuthorization(
            client,
            address(this),
            amount,
            validAfter,
            validBefore,
            nonce,
            v,
            r,
            s
        );

        // Store payment details
        payments[taskId] = TaskPayment({
            taskId: taskId,
            client: client,
            agent: agent,
            amount: amount,
            depositTime: block.timestamp,
            deadline: validBefore,
            status: PaymentStatus.DEPOSITED
        });

        emit PaymentDeposited(taskId, client, agent, amount, validBefore);
    }

    /**
     * @notice Release payment to agent (on successful task completion)
     * @param taskId Task identifier
     */
    function releasePayment(bytes32 taskId) external onlyCoordinator nonReentrant {
        TaskPayment storage payment = payments[taskId];
        require(payment.status == PaymentStatus.DEPOSITED, "Invalid payment status");
        require(block.timestamp <= payment.deadline, "Payment expired");

        payment.status = PaymentStatus.COMPLETED;

        // Transfer 100% to agent
        require(aiusd.transfer(payment.agent, payment.amount), "Transfer failed");

        emit PaymentReleased(taskId, payment.agent, payment.amount);
    }

    /**
     * @notice Refund payment to client (on task failure or rejection)
     * @param taskId Task identifier
     */
    function refundPayment(bytes32 taskId) external onlyCoordinator nonReentrant {
        TaskPayment storage payment = payments[taskId];
        require(
            payment.status == PaymentStatus.DEPOSITED || payment.status == PaymentStatus.EXPIRED,
            "Invalid payment status"
        );

        payment.status = PaymentStatus.REFUNDED;

        // Return 100% to client
        require(aiusd.transfer(payment.client, payment.amount), "Refund failed");

        emit PaymentRefunded(taskId, payment.client, payment.amount);
    }

    /**
     * @notice Mark payment as expired (callable by anyone after deadline)
     * @param taskId Task identifier
     */
    function markExpired(bytes32 taskId) external nonReentrant {
        TaskPayment storage payment = payments[taskId];
        require(payment.status == PaymentStatus.DEPOSITED, "Invalid payment status");
        require(block.timestamp > payment.deadline, "Not expired yet");

        payment.status = PaymentStatus.EXPIRED;

        emit PaymentExpired(taskId, payment.client, payment.amount);
    }

    /**
     * @notice Auto-refund expired payment (callable by coordinator after expiration)
     * @param taskId Task identifier
     */
    function autoRefundExpired(bytes32 taskId) external onlyCoordinator nonReentrant {
        TaskPayment storage payment = payments[taskId];
        require(payment.status == PaymentStatus.DEPOSITED, "Invalid payment status");
        require(block.timestamp > payment.deadline, "Not expired yet");

        payment.status = PaymentStatus.REFUNDED;

        // Auto-refund to client
        require(aiusd.transfer(payment.client, payment.amount), "Auto-refund failed");

        emit PaymentExpired(taskId, payment.client, payment.amount);
        emit PaymentRefunded(taskId, payment.client, payment.amount);
    }

    /**
     * @notice Get payment details
     * @param taskId Task identifier
     * @return Payment details
     */
    function getPayment(bytes32 taskId) external view returns (TaskPayment memory) {
        return payments[taskId];
    }

    /**
     * @notice Check if payment is active and not expired
     * @param taskId Task identifier
     * @return True if payment is active
     */
    function isPaymentActive(bytes32 taskId) external view returns (bool) {
        TaskPayment memory payment = payments[taskId];
        return payment.status == PaymentStatus.DEPOSITED && block.timestamp <= payment.deadline;
    }

    /**
     * @notice Get next nonce for a client
     * @param client Client address
     * @return Next nonce value
     */
    function getNextNonce(address client) external view returns (uint256) {
        return nonces[client];
    }

    /**
     * @notice Increment and return nonce for a client (coordinator only)
     * @param client Client address
     * @return New nonce value
     */
    function incrementNonce(address client) external onlyCoordinator returns (uint256) {
        nonces[client]++;
        return nonces[client];
    }
}
