// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

interface IStandardDeposit {
    function deposit(bytes calldata pubkey, bytes calldata withdrawal_credentials,
        bytes calldata signature, bytes32 deposit_data_root) external payable;
}

/// @notice Atomically forwards exactly 32 native POP per validator to the
/// pinned standard deposit contract. No owner, proxy, refunds or withdrawals.
/// BLS validity and existing deposits made directly to the standard contract
/// must be checked by the caller; the standard contract does not check BLS.
contract BatchDeposit {
    uint256 public constant STAKE = 32 ether;
    uint256 public constant MAX_BATCH_SIZE = 100;
    bytes32 public constant DEPOSIT_CODEHASH =
        0x6c029a231254fadb724d63be769f75eedd66362df034a3e663252b49d062a666;
    IStandardDeposit public immutable depositContract;
    mapping(bytes32 => bool) public deposited;

    struct DepositData {
        bytes pubkey;
        bytes withdrawal_credentials;
        bytes signature;
        bytes32 deposit_data_root;
    }
    event BatchDeposited(address indexed sender, uint256 count);

    constructor(address target) {
        require(target.codehash == DEPOSIT_CODEHASH, "Wrong deposit contract");
        depositContract = IStandardDeposit(target);
    }

    function batchDeposit(DepositData[] calldata deposits) external payable {
        uint256 count = deposits.length;
        require(count > 0 && count <= MAX_BATCH_SIZE, "Batch size must be 1..100");
        require(msg.value == count * STAKE, "Expected 32 POP per validator");
        require(address(depositContract).codehash == DEPOSIT_CODEHASH, "Deposit code changed");
        for (uint256 i = 0; i < count; i++) {
            bytes32 key = keccak256(deposits[i].pubkey);
            require(!deposited[key], "Pubkey already submitted through batch");
            deposited[key] = true;
            depositContract.deposit{value: STAKE}(
                deposits[i].pubkey, deposits[i].withdrawal_credentials,
                deposits[i].signature, deposits[i].deposit_data_root
            );
        }
        emit BatchDeposited(msg.sender, count);
    }
}
