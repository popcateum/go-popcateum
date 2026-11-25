// Copyright 2026 The Popcateum Authors
// SPDX-License-Identifier: LGPL-3.0-or-later

package core

import (
	"crypto/sha256"
	"math/big"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/contracts/batch"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/core/state"
	"github.com/popcateum/go-popcateum/params"
)

var (
	popstarDeposit = common.HexToAddress("0x909ca70000000000000000000000000000000001")
	popstarBatch   = common.HexToAddress("0x909ca70000000000000000000000000000000002")
)

// ApplyPopstar installs the constructor-equivalent contract states before the
// first transaction of the activation block, including when the block is empty.
func ApplyPopstar(config *params.ChainConfig, number *big.Int, db *state.StateDB) {
	if config.PopstarBlock == nil || config.PopstarBlock.Cmp(number) != 0 {
		return
	}
	batchCode, err := batch.RuntimeCode(popstarDeposit)
	if err != nil {
		panic(err) // Invalid embedded consensus artifact.
	}
	db.SetNonce(popstarDeposit, 1)
	db.SetCode(popstarDeposit, deposit.RuntimeCode())
	// Solidity storage: branch[32], deposit_count, then zero_hashes[32].
	// The constructor initializes zero_hashes[1..31] in slots 34..64.
	var pair [64]byte
	for index := int64(1); index < 32; index++ {
		hash := sha256.Sum256(pair[:])
		db.SetState(popstarDeposit, common.BigToHash(big.NewInt(33+index)), hash)
		copy(pair[:32], hash[:])
		copy(pair[32:], hash[:])
	}
	db.SetNonce(popstarBatch, 1)
	db.SetCode(popstarBatch, batchCode)
	db.Finalise(true)
}
