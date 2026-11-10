#!/usr/bin/env python3
"""Read-only PopSong TTD estimate. Never changes chain config or sends transactions.

The default offline fixture reproduces the release estimate. --rpc samples a
running node instead. The model holds observed hash power constant, NOT difficulty.
Quantiles describe model runs, not uncertainty about future miners.
"""

import argparse
import json
import math
from pathlib import Path
import random
import statistics
import urllib.request

EMPTY_UNCLES = "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"
GENESIS = "0x07fd751edea3b3a2d790002e4712b14e2d0e43670f25952d4734ee31b526fa1b"


def rpc_block(url, number):
    payload = json.dumps({"jsonrpc": "2.0", "id": 1,
                          "method": "eth_getBlockByNumber", "params": [number, False]}).encode()
    request = urllib.request.Request(url, payload, {"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=30) as response:
        data = json.load(response)
    if data.get("error") or not data.get("result"):
        raise ValueError(f"Cannot read block {number}: {data.get('error')}")
    return data["result"]


def quantity(block, field):
    return int(block[field], 16)


def estimate(blocks, target, runs, seed, scale, long_pause):
    if len(blocks) < 2 or runs < 2 or not math.isfinite(scale) or scale <= 0:
        raise ValueError("Need >= 2 headers/runs and a positive finite hash-rate scale")
    for parent, child in zip(blocks, blocks[1:]):
        if (quantity(child, "number") != quantity(parent, "number") + 1
                or child["parentHash"] != parent["hash"]
                or quantity(child, "timestamp") <= quantity(parent, "timestamp")
                or quantity(child, "totalDifficulty") - quantity(parent, "totalDifficulty") != quantity(child, "difficulty")):
            raise ValueError("Headers must form a contiguous PoW chain with consistent TD")
    head, count = blocks[-1], len(blocks)-1
    height, head_td = quantity(head, "number"), quantity(head, "totalDifficulty")
    pow_blocks = target - height - 1
    if height < 85000 or pow_blocks < 1 or target >= 12000000:
        raise ValueError("This estimator supports future targets in the pre-bomb PopSong era only")
    difficulties = [quantity(b, "difficulty") for b in blocks[1:]]
    mean_difficulty = sum(difficulties) // count
    interval = (quantity(head, "timestamp") - quantity(blocks[0], "timestamp")) / count
    hashrate = mean_difficulty / interval * scale
    uncle_probability = sum(b["sha3Uncles"] != EMPTY_UNCLES for b in blocks[:-1]) / count
    additions, durations = [], []
    for run in range(runs):
        rng = random.Random(seed + run)
        difficulty = quantity(head, "difficulty")
        uncle = head["sha3Uncles"] != EMPTY_UNCLES
        total, elapsed = 0, 0
        for index in range(pow_blocks):
            restart = index == 0 and long_pause
            # Integer timestamps; parent difficulty approximates the next waiting time.
            seconds = 100000 if restart else max(1, int(rng.expovariate(hashrate / difficulty)))
            adjustment = max((2 if uncle else 1) - seconds // 6, -99)
            difficulty = max(131072, difficulty + (difficulty // 2048) * adjustment)
            total += difficulty
            if not restart:
                elapsed += seconds
            uncle = rng.random() < uncle_probability
        additions.append(total)
        durations.append(elapsed)
    ordered = sorted(additions)
    median = int(statistics.median(additions))
    return {
        "basis_head": height, "basis_hash": head["hash"], "basis_td": str(head_td),
        "target_first_pos_height": target, "estimated_additional_pow_blocks": pow_blocks,
        "sample_count": count, "sample_mean_difficulty": mean_difficulty,
        "head_difficulty": quantity(head, "difficulty"), "average_block_seconds": interval,
        "historical_hashrate_proxy": hashrate / scale, "hashrate_scale": scale,
        "historical_uncle_parent_fraction": uncle_probability,
        "runs": runs, "seed": seed, "long_pause": long_pause,
        "additional_td_median": median,
        "additional_td_p10": ordered[int(0.1 * (runs-1))],
        "additional_td_p90": ordered[int(0.9 * (runs-1))],
        "projected_seconds_median": statistics.median(durations),
        "ttd": str(head_td + median),
        "limitations": "Conditional constant-hash-power model; no height/time guarantee. Not a beacon readiness check."
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group()
    source.add_argument("--rpc", help="Read-only EL JSON-RPC endpoint")
    source.add_argument("--headers", type=Path, default=Path(__file__).resolve().parents[1] / "docs/evidence/ttd-headers.json")
    parser.add_argument("--samples", type=int, default=512, help="Historical intervals, with --rpc")
    parser.add_argument("--target-first-pos", type=int, default=8910000)
    parser.add_argument("--runs", type=int, default=128)
    parser.add_argument("--seed", type=int, default=20261110)
    parser.add_argument("--hashrate-scale", type=float, default=1.0)
    parser.add_argument("--continuous", action="store_true", help="No long-pause adjustment on next block")
    args = parser.parse_args()
    if args.rpc:
        if rpc_block(args.rpc, "0x0")["hash"] != GENESIS:
            parser.error("RPC is not Popcateum mainnet")
        head = rpc_block(args.rpc, "latest")
        height = quantity(head, "number")
        if args.samples < 1 or args.samples > height:
            parser.error("Invalid sample count")
        blocks = [rpc_block(args.rpc, hex(n)) for n in range(height-args.samples, height)] + [head]
    else:
        blocks = json.loads(args.headers.read_text())
    try:
        result = estimate(blocks, args.target_first_pos, args.runs, args.seed,
                          args.hashrate_scale, not args.continuous)
    except ValueError as error:
        parser.error(str(error))
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
