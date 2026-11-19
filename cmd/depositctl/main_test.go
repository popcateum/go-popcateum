package main

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/crypto"
	"github.com/popcateum/go-popcateum/params"
)

func TestPlanTampering(t *testing.T) {
	from := common.Address{1}
	original := plan{Schema: "popcateum-deposit-plan-v1", ChainID: "1213", Genesis: params.MainnetGenesisHash, From: from, Nonce: 1, Address: crypto.CreateAddress(from, 1), CreationHash: deposit.CreationHash(), TTD: params.MainnetTerminalTotalDifficulty.String(), GasLimit: 3000000, GasPrice: "2000000000", MaxFee: new(big.Int).Mul(big.NewInt(3000000), big.NewInt(2000000000)).String(), Value: "0", MaxFeePOP: "0.006000000000000000"}
	if err := validatePlan(original); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*plan){
		func(p *plan) { p.ChainID = "1" }, func(p *plan) { p.Genesis = common.Hash{} },
		func(p *plan) { p.CreationHash = common.Hash{} }, func(p *plan) { p.TTD = "1" },
		func(p *plan) { p.Address = common.Address{} }, func(p *plan) { p.Nonce++ },
		func(p *plan) { p.Value = "32000000000000000000" }, func(p *plan) { p.GasPrice = "-1" },
		func(p *plan) { p.GasLimit++ }, func(p *plan) { p.MaxFee = "0" },
	} {
		p := original
		change(&p)
		if err := validatePlan(p); err == nil {
			t.Fatal("accepted tampered plan")
		}
	}
}

func TestJournalNeverOverwritesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	if err := writeNew(path, map[string]string{"phase": "prepared"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeNew(path, map[string]string{"phase": "another deployment"}); err == nil {
		t.Fatal("overwrote deployment journal")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("changed existing journal")
	}
	if err = writeOwned(path, map[string]string{"phase": "verified"}); err != nil {
		t.Fatal(err)
	}
}
