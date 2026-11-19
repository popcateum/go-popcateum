# Standard beacon deposit contract

These files are the unmodified Solidity source and ABI/creation bytecode from
[Ethereum consensus-specs v1.3.0](https://github.com/ethereum/consensus-specs/tree/01b53691dcc36d37a5ad8994b3a32d8de69fb1aa/solidity_deposit_contract).
`provenance.json` pins the full commit and SHA-256 checksums. The source is CC0-1.0.

No Solidity compiler or npm installation is needed to deploy. The published
artifact was built with Solidity 0.6.11, optimizer 5,000,000 runs and literal
metadata. `runtime.hex` is derived by running its constructor in the local EVM;
Go tests reproduce it under both Berlin and London rules.

`ether` and `gwei` in Solidity are numeric units (10^18 and 10^9 native base
units). On Popcateum, 1 ether in this source is 1 POP. Keep the standard ABI,
DepositEvent and SSZ hashing unchanged for stock beacon clients.

The contract accepts a minimum of 1 POP, in whole gwei increments. The initial
32 POP validator activation policy is a consensus-client/application rule, not
a hardcoded 32 POP restriction here. This contract does not check BLS signatures
or withdrawal credentials. Tests intentionally use synthetic signatures to test
execution behavior only; they are NOT valid validator registration data.

Use one canonical deployment for the network. There is no owner, upgrade proxy,
constructor argument or direct withdrawal function. Plain transfers revert.

[Deployment commands and recovery](../../docs/deposit-deployment.md)
