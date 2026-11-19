# BatchDeposit

Native POP atomic forwarder, compiled with Solidity 0.8.19, optimizer 200, EVM London. Each entry forwards exactly 32 POP to the immutable canonical deposit contract. The canonical runtime hash is checked at deployment and at every call. There is no administrator, proxy, arbitrary call, withdrawal or upgrade path.

`deposited[keccak256(pubkey)]` rejects repeated submissions through this wrapper. It cannot see previous deposits submitted directly to the standard contract. BLS signatures are verified by the deposit page; the contracts only validate standard deposit encoding/roots. CL validation remains authoritative.

Sources and artifact are embedded in `depositctl`. From the sibling `deposit-page` directory, `npm ci && npm run contracts:build` reproduces `artifact.json` and the identical web artifact. Runtime bytecode contains an immutable target, so compare against `RuntimeCode(target)`, not the unpatched compiler output.

Deploy with `depositctl plan --batch-target <standard-address>`, followed by the existing `send` / `verify` flow. The CL must still track the **standard** deposit address. Operational commands are in [the page README](../../../deposit-page/README.md).
