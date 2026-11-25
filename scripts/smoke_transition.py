#!/usr/bin/env python3
"""Exercise the built gpop on an isolated disposable chain (small test Ethash DAG).

Uses a PUBLIC TEST KEY, a temporary datadir, no peers and no public RPC listener.
Never point this at an existing datadir. This does not rehearse real beacon consensus.
"""
import json
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
BIN = ROOT / "build/bin/gpop"


def run(args, timeout=90):
    result = subprocess.run([str(BIN)] + args, capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    return result.stdout


with tempfile.TemporaryDirectory(prefix="gpop-transition-smoke-") as directory:
    temp = Path(directory)
    isolated = ["--datadir", str(temp / "mainnet"), "--maxpeers", "0", "--nodiscover",
                "--nat", "none", "--port", "0", "--authrpc.port", "0", "--ipcdisable",
                "--syncmode", "full", "--cache", "64", "--datadir.minfreedisk", "0"]
    output = run(isolated + ["--exec", '''
      if (eth.getBlock(0).hash != "0x07fd751edea3b3a2d790002e4712b14e2d0e43670f25952d4734ee31b526fa1b") throw "genesis";
      if (net.version != "1213") throw "network";
      if (admin.nodeInfo.protocols.eth.config.terminalTotalDifficulty.toString() != "7288759620794051") throw "TTD";
      if (admin.nodeInfo.protocols.eth.config.popstarBlock != 8905000) throw "Popstar";
      console.log("MAINNET_DEFAULTS_OK");
    ''', "console"])
    assert "MAINNET_DEFAULTS_OK" in output, output

    # This fixed key is intentionally public and only funds the disposable chain.
    (temp / "key").write_text("01" * 32)
    (temp / "password").write_text("isolated-public-test-key\n")
    args = ["--datadir", str(temp / "synthetic")]
    output = run(args + ["account", "import", "--password", str(temp / "password"), str(temp / "key")])
    address = "0x" + re.search(r"[0-9a-fA-F]{40}", output).group(0)
    config = {"chainId": 1213, "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0,
              "eip158Block": 0, "byzantiumBlock": 0, "constantinopleBlock": 0,
              "petersburgBlock": 0, "istanbulBlock": 0, "muirGlacierBlock": 0,
              "berlinBlock": 0, "popsongBlock": 0, "popcornBlock": 0, "londonBlock": 2, "popstarBlock": 2,
              "terminalTotalDifficulty": 9000000, "ethash": {}}
    genesis = {"config": config, "difficulty": "0x20000", "gasLimit": "0x989680",
               "alloc": {address: {"balance": "1000000000000000000000000"}}}
    (temp / "genesis.json").write_text(json.dumps(genesis))
    (temp / "node.toml").write_text("[Eth.Ethash]\nPowMode = 2\nCachesOnDisk = 0\nDatasetsOnDisk = 0\n")
    run(args + ["init", str(temp / "genesis.json")])
    output = run(args + isolated[2:] + ["--config", str(temp / "node.toml"),
        "--password", str(temp / "password"), "--unlock", address,
        "--miner.popcatbase", address, "--exec", '''
      function check(ok, message) { if (!ok) throw message; }
      function waitFor(predicate) { for (var n=0; n<45 && !predicate(); n++) admin.sleep(1); check(predicate(), "timeout"); }
      check(miner.setPopcatbase(eth.accounts[0]), "legacy miner alias");
      var tx = eth.sendTransaction({from:eth.accounts[0], data:"0x6001600c60003960016000f300", gas:100000, gasPrice:2000000000});
      miner.start(1);
      waitFor(function() { return eth.getTransactionReceipt(tx) != null && eth.blockNumber >= 2; });
      var receipt = eth.getTransactionReceipt(tx);
      check(receipt.status == "0x1" && eth.getCode(receipt.contractAddress) == "0x00", "deployment");
      check(eth.getBlock(1).baseFeePerGas == null, "pre London");
      var deposit = "0x909ca70000000000000000000000000000000001";
      var batch = "0x909ca70000000000000000000000000000000002";
      check(eth.getCode(deposit, 1) == "0x" && eth.getCode(batch, 1) == "0x", "early Popstar");
      check(eth.getCode(deposit, 2).length > 2 && eth.getCode(batch, 2).length > 2, "missing Popstar contracts");
      check(eth.getBlock(2).difficulty.toString() == "941264", "Popstar difficulty");
      check(eth.getBlock(2).baseFeePerGas.toString() == "1000000000", "London base fee");
      var tx2 = eth.sendTransaction({from:eth.accounts[0], to:"0x000000000000000000000000000000000000dead", value:1, gas:21000, maxFeePerGas:2000000000, maxPriorityFeePerGas:1000000000});
      waitFor(function() { return eth.getTransactionReceipt(tx2) != null; });
      check(eth.getTransactionReceipt(tx2).status == "0x1" && eth.getTransaction(tx2).type == "0x2", "type 2 transaction");
      waitFor(function() { return eth.getBlock("latest").totalDifficulty.gte(9000000); });
      var terminal = eth.getBlock("latest");
      check(eth.getBlock(terminal.number-1).totalDifficulty.lt(9000000), "terminal parent TD");
      admin.sleep(3);
      check(eth.blockNumber == terminal.number, "mined past TTD");
      miner.stop();
      console.log("TRANSITION_SMOKE_OK " + JSON.stringify({terminalHeight:terminal.number, terminalTD:terminal.totalDifficulty.toString(), deploymentBlock:receipt.blockNumber, type2Block:eth.getTransactionReceipt(tx2).blockNumber}));
    ''', "console"])
    assert "TRANSITION_SMOKE_OK" in output, output
    print("MAINNET_DEFAULTS_OK")
    print(next(line for line in output.splitlines() if "TRANSITION_SMOKE_OK" in line))
    output = run(args + isolated[2:] + ["--config", str(temp / "node.toml"),
        "--miner.etherbase", address, "--exec", '''
      var head = eth.getBlock("latest");
      if (!head.totalDifficulty.gte(9000000)) throw "lost terminal head";
      miner.start(1);
      admin.sleep(3);
      if (eth.getBlock("latest").hash != head.hash) throw "mined past TTD after restart";
      miner.stop();
      console.log("RESTART_AT_TTD_OK");
      function engine(method, args) {
        var reply = web3.currentProvider.send({jsonrpc:"2.0", id:1, method:"engine_"+method, params:args});
        if (reply.error) throw JSON.stringify(reply.error);
        return reply.result;
      }
      var zero = "0x" + Array(65).join("0");
      var choice = {headBlockHash:head.hash, safeBlockHash:zero, finalizedBlockHash:zero};
      var attrs = {timestamp:web3.toHex(head.timestamp+12), prevRandao:zero, suggestedFeeRecipient:eth.accounts[0]};
      var proposal = engine("forkchoiceUpdatedV1", [choice, attrs]);
      if (proposal.payloadStatus.status != "VALID" || !proposal.payloadId) throw "terminal forkchoice";
      var payload = engine("getPayloadV1", [proposal.payloadId]);
      if (payload.parentHash != head.hash || parseInt(payload.blockNumber,16) != head.number+1) throw "PoS parent/height";
      if (engine("newPayloadV1", [payload]).status != "VALID") throw "invalid first PoS payload";
      choice.headBlockHash = payload.blockHash;
      if (engine("forkchoiceUpdatedV1", [choice,null]).payloadStatus.status != "VALID") throw "PoS forkchoice";
      var pos = eth.getBlock("latest");
      if (pos.hash != payload.blockHash || !pos.difficulty.eq(0)) throw "first PoS canonical head";
      console.log("FIRST_POS_ENGINE_OK " + pos.number);
    ''', "console"])
    assert "RESTART_AT_TTD_OK" in output, output
    print("RESTART_AT_TTD_OK")
    assert "FIRST_POS_ENGINE_OK" in output, output
    print(next(line for line in output.splitlines() if "FIRST_POS_ENGINE_OK" in line))
