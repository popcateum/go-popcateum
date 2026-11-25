package ethash

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/consensus"
	"github.com/popcateum/go-popcateum/core/rawdb"
	"github.com/popcateum/go-popcateum/core/state"
	"github.com/popcateum/go-popcateum/core/types"
	"github.com/popcateum/go-popcateum/params"
)

type popHeaderChain struct{ consensus.ChainHeaderReader }

func (popHeaderChain) Config() *params.ChainConfig { return params.MainnetChainConfig }

func popHeaders(t *testing.T) map[uint64]*types.Header {
	t.Helper()
	data, err := os.ReadFile("testdata/popcateum-headers.json")
	if err != nil {
		t.Fatal(err)
	}
	var headers []*types.Header
	if err := json.Unmarshal(data, &headers); err != nil {
		t.Fatal(err)
	}
	result := make(map[uint64]*types.Header)
	for _, h := range headers {
		result[h.Number.Uint64()] = h
	}
	return result
}

// Public historical headers cover the actual PopSong/Popcorn boundaries and
// the migration head. Verify real Ethash seals, not the fake mining engine.
func TestPopcateumHistoricalHeaders(t *testing.T) {
	headers := popHeaders(t)
	engine := New(Config{PowMode: ModeNormal, CachesInMem: 2}, nil, false)
	defer engine.Close()
	for _, n := range []uint64{1, 2, 84999, 85000, 85001, 3999999, 4000000, 4000001, 8904999} {
		h, parent := headers[n], headers[n-1]
		if h.ParentHash != parent.Hash() {
			t.Fatalf("%d: fixture parent hash mismatch", n)
		}
		if err := engine.verifyHeader(popHeaderChain{}, h, parent, false, true, time.Now().Unix()); err != nil {
			t.Fatalf("historical header %d: %v", n, err)
		}
	}
	if headers[0].Hash() != params.MainnetGenesisHash {
		t.Fatal("wrong genesis fixture")
	}
	if headers[8904999].Hash() != common.HexToHash("0x185e96f351cdaf980c707b80b37097803d092bf36f9841ad384ea5b4dba4ab00") {
		t.Fatal("wrong migration head")
	}
}

func TestPopcateumLondonBoundary(t *testing.T) {
	parent := popHeaders(t)[8904999]
	header := types.CopyHeader(parent)
	header.ParentHash = parent.Hash()
	header.Number = big.NewInt(8905000)
	header.Time = parent.Time + 100000 // Restart after a long mining pause.
	header.Difficulty = CalcDifficulty(params.MainnetChainConfig, header.Time, parent)
	header.BaseFee = big.NewInt(params.InitialBaseFee)
	header.GasLimit = parent.GasLimit * params.MainnetChainConfig.ElasticityMultiplier()
	header.GasUsed = 0
	if header.Difficulty.Cmp(parent.Difficulty) != 0 {
		t.Fatalf("restart difficulty: %v", header.Difficulty)
	}
	engine := NewFaker()
	if err := engine.verifyHeader(popHeaderChain{}, header, parent, false, false, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	header.BaseFee = nil
	if err := engine.verifyHeader(popHeaderChain{}, header, parent, false, false, time.Now().Unix()); err == nil {
		t.Fatal("accepted London without base fee")
	}
}

func TestPopcateumRewardsWithoutSeaPop(t *testing.T) {
	for _, tc := range []struct{ block, reward int64 }{{1, 50}, {84999, 50}, {85000, 50}, {3999999, 50}, {4000000, 30}, {8905000, 30}, {9000000, 30}, {9000001, 30}} {
		db, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		if err != nil {
			t.Fatal(err)
		}
		miner, uncleMiner := common.Address{1}, common.Address{2}
		header := &types.Header{Number: big.NewInt(tc.block), Coinbase: miner}
		uncle := &types.Header{Number: big.NewInt(tc.block - 1), Coinbase: uncleMiner}
		accumulateRewards(params.MainnetChainConfig, db, header, []*types.Header{uncle})
		reward := new(big.Int).Mul(big.NewInt(tc.reward), big.NewInt(params.Ether))
		wantMiner := new(big.Int).Add(reward, new(big.Int).Div(new(big.Int).Set(reward), big.NewInt(32)))
		wantUncle := new(big.Int).Div(new(big.Int).Mul(reward, big.NewInt(7)), big.NewInt(8))
		if db.GetBalance(miner).Cmp(wantMiner) != 0 || db.GetBalance(uncleMiner).Cmp(wantUncle) != 0 {
			t.Fatalf("wrong rewards at %d", tc.block)
		}
	}
}

func TestPopstarMainnetTerminalDifficulty(t *testing.T) {
	parent := popHeaders(t)[8_904_999]
	config := params.MainnetChainConfig
	td := big.NewInt(7_288_670_200_714_051)
	for n := int64(8_905_000); n <= 8_999_999; n++ {
		before := new(big.Int).Set(td)
		diff := CalcDifficulty(config, parent.Time+12, parent)
		td.Add(td, diff)
		if config.IsTerminalPoWBlock(before, td) != (n == 8_999_999) {
			t.Fatalf("unexpected terminal classification at %d, TD %v", n, td)
		}
		parent = &types.Header{Number: big.NewInt(n), Difficulty: diff, Time: parent.Time + 12}
	}
	if td.Cmp(config.TerminalTotalDifficulty) != 0 || td.String() != "7288759620794051" {
		t.Fatalf("terminal TD %s", td)
	}
}
