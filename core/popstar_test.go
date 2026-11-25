package core

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/consensus/ethash"
	"github.com/popcateum/go-popcateum/contracts/batch"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/core/rawdb"
	"github.com/popcateum/go-popcateum/core/types"
	"github.com/popcateum/go-popcateum/core/vm"
	"github.com/popcateum/go-popcateum/crypto"
	"github.com/popcateum/go-popcateum/params"
)

// Include a batch deposit in the very first Popstar block, import it on a
// separate database, and verify that the next block preserves the deposit.
func TestPopstarActivation(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	cfg := *params.MainnetChainConfig
	cfg.LondonBlock, cfg.PopstarBlock = big.NewInt(2), big.NewInt(2)
	cfg.PopSongBlock, cfg.PopcornBlock = big.NewInt(0), big.NewInt(0)
	spec := &Genesis{Config: &cfg, GasLimit: 10_000_000, Alloc: GenesisAlloc{
		sender: {Balance: new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))},
	}}
	producer := rawdb.NewMemoryDatabase()
	genesis := spec.MustCommit(producer)
	blocks, receipts := GenerateChain(&cfg, genesis, ethash.NewFaker(), producer, 3, func(i int, b *BlockGen) {
		if i != 1 {
			return
		}
		if b.statedb.GetNonce(popstarDeposit) != 1 || b.statedb.GetNonce(popstarBatch) != 1 {
			t.Fatal("missing predeploy nonce")
		}
		abi, _, err := deposit.Artifact()
		if err != nil {
			t.Fatal(err)
		}
		input, err := abi.Pack("get_deposit_root")
		if err != nil {
			t.Fatal(err)
		}
		evm := vm.NewEVM(NewEVMBlockContext(b.header, nil, &sender), vm.TxContext{}, b.statedb.Copy(), &cfg, vm.Config{})
		root, _, err := evm.StaticCall(vm.AccountRef(sender), popstarDeposit, input, 300000)
		if err != nil || common.BytesToHash(root) != deposit.EmptyRoot() {
			t.Fatalf("empty root %x: %v", root, err)
		}
		// Valid execution-layer deposit data for all-zero public key,
		// credentials and signature, amount 32 billion Gwei.
		data := []struct {
			Pubkey, WithdrawalCredentials, Signature []byte
			DepositDataRoot                          [32]byte
		}{
			{make([]byte, 48), make([]byte, 32), make([]byte, 96), common.HexToHash("05125366a514ddd17fc8158440399c02d631cdb991dffa30623107f27e43673d")},
		}
		batchABI, err := batch.ABI()
		if err != nil {
			t.Fatal(err)
		}
		input, err = batchABI.Pack("batchDeposit", data)
		if err != nil {
			t.Fatal(err)
		}
		tx := types.NewTransaction(0, popstarBatch, new(big.Int).Mul(big.NewInt(32), big.NewInt(params.Ether)), 500000, big.NewInt(2*params.GWei), input)
		signed, err := types.SignTx(tx, types.LatestSigner(&cfg), key)
		if err != nil {
			t.Fatal(err)
		}
		b.AddTx(signed)
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), nil, spec, nil, ethash.NewFaker(), vm.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if _, err = chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	if len(receipts[1]) != 1 || receipts[1][0].Status != 1 || len(receipts[1][0].Logs) != 2 || receipts[1][0].Logs[0].Address != popstarDeposit {
		t.Fatal("batch did not forward the deposit at activation")
	}
	code, err := batch.RuntimeCode(popstarDeposit)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range blocks {
		db, err := chain.StateAt(b.Root())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if db.Exist(popstarDeposit) || db.Exist(popstarBatch) {
				t.Fatal("premature activation")
			}
			continue
		}
		if !bytes.Equal(db.GetCode(popstarDeposit), deposit.RuntimeCode()) || !bytes.Equal(db.GetCode(popstarBatch), code) ||
			db.GetState(popstarDeposit, common.BigToHash(big.NewInt(32))).Big().Uint64() != 1 {
			t.Fatalf("wrong contract state at %d", b.NumberU64())
		}
	}
}
