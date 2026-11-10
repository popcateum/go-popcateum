package beacon

import (
	"math/big"
	"strings"
	"testing"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/consensus"
	"github.com/popcateum/go-popcateum/consensus/ethash"
	"github.com/popcateum/go-popcateum/consensus/misc"
	"github.com/popcateum/go-popcateum/core/rawdb"
	"github.com/popcateum/go-popcateum/core/state"
	"github.com/popcateum/go-popcateum/core/types"
	"github.com/popcateum/go-popcateum/params"
)

type transitionChain struct {
	consensus.ChainHeaderReader
	parent *types.Header
	td     *big.Int
}

func (c transitionChain) Config() *params.ChainConfig        { return params.MainnetChainConfig }
func (c transitionChain) GetTd(common.Hash, uint64) *big.Int { return c.td }
func (c transitionChain) GetHeader(h common.Hash, n uint64) *types.Header {
	if c.parent.Hash() == h && c.parent.Number.Uint64() == n {
		return c.parent
	}
	return nil
}

func TestPopcateumTTDTransition(t *testing.T) {
	parent := &types.Header{Number: big.NewInt(8909999), Time: 100, Difficulty: big.NewInt(1000000), GasLimit: 30000000, BaseFee: big.NewInt(params.InitialBaseFee), UncleHash: types.EmptyUncleHash}
	chain := transitionChain{parent: parent, td: new(big.Int).Set(params.MainnetTerminalTotalDifficulty)}
	engine := New(ethash.NewFaker())
	header := &types.Header{ParentHash: parent.Hash(), Number: big.NewInt(8910000), Time: 112, Difficulty: new(big.Int), GasLimit: parent.GasLimit, BaseFee: misc.CalcBaseFee(chain.Config(), parent), UncleHash: types.EmptyUncleHash}
	if err := engine.VerifyHeader(chain, header, true); err != nil {
		t.Fatal(err)
	}
	header.Difficulty.SetInt64(131072)
	if err := engine.VerifyHeader(chain, header, true); err == nil || !strings.Contains(err.Error(), "invalid difficulty") {
		t.Fatalf("accepted PoW child of terminal block: %v", err)
	}
	chain.td.Sub(chain.td, big.NewInt(1))
	header.Difficulty.SetInt64(0)
	if err := engine.VerifyHeader(chain, header, true); err == nil {
		t.Fatal("accepted PoS before TTD")
	}
	chain.td.Add(chain.td, big.NewInt(1))
	db, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		t.Fatal(err)
	}
	header.Coinbase = common.Address{1}
	engine.Finalize(chain, header, db, nil, nil, nil)
	if db.GetBalance(header.Coinbase).Sign() != 0 {
		t.Fatal("issued PoW reward after Merge")
	}
}
