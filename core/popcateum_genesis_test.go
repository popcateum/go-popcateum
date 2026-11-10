package core

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/popcateum/go-popcateum/core/rawdb"
	"github.com/popcateum/go-popcateum/core/types"
	"github.com/popcateum/go-popcateum/params"
	"github.com/popcateum/go-popcateum/trie"
)

func TestPopcateumConfigMigration(t *testing.T) {
	for _, height := range []int64{8904999, 9000000} {
		db := rawdb.NewMemoryDatabase()
		genesis := DefaultGenesisBlock().MustCommit(db)
		legacy := *params.MainnetChainConfig
		legacy.LondonBlock, legacy.TerminalTotalDifficulty = nil, nil
		legacy.LegacySeaPopBlock = big.NewInt(9000000)
		rawdb.WriteChainConfig(db, genesis.Hash(), &legacy)
		head := &types.Header{Number: big.NewInt(height), Difficulty: big.NewInt(941264)}
		rawdb.WriteHeader(db, head)
		rawdb.WriteHeadHeaderHash(db, head.Hash())
		config, hash, err := SetupGenesisBlock(db, trie.NewDatabase(db), nil)
		if height == 9000000 {
			if err == nil || !strings.Contains(err.Error(), "already-activated SeaPop") {
				t.Fatalf("unsafe migration: %v", err)
			}
			if rawdb.ReadChainConfig(db, genesis.Hash()).LegacySeaPopBlock == nil {
				t.Fatal("changed rejected database")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if hash != params.MainnetGenesisHash || config.LegacySeaPopBlock != nil {
			t.Fatal("changed genesis or retained SeaPop")
		}
		stored := rawdb.ReadChainConfig(db, hash)
		if stored.LondonBlock.Cmp(big.NewInt(8905000)) != 0 || stored.TerminalTotalDifficulty.Cmp(params.MainnetTerminalTotalDifficulty) != 0 {
			t.Fatal("upgrade configuration not persisted")
		}
		data, _ := json.Marshal(stored)
		if strings.Contains(string(data), "seapopBlock") {
			t.Fatal("SeaPop scheduled in new configuration")
		}
		if rawdb.ReadHeadHeaderHash(db) != head.Hash() {
			t.Fatal("migration rewound head")
		}
	}
}
