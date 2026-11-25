package forkid

import (
	"math/big"
	"testing"

	"github.com/popcateum/go-popcateum/params"
)

func TestPopcateumForkID(t *testing.T) {
	for _, tc := range []struct {
		head uint64
		hash uint32
		next uint64
	}{
		{0, 0xfc8f0dee, 85000},
		{84999, 0xfc8f0dee, 85000},
		{85000, 0xa35870a7, 4000000},
		{4000000, 0x03e94381, 8905000},
		{8904999, 0x03e94381, 8905000},
		{8905000, 0x6f116201, 0},
		{9000000, 0x6f116201, 0},
	} {
		want := ID{Hash: checksumToBytes(tc.hash), Next: tc.next}
		if have := NewID(params.MainnetChainConfig, params.MainnetGenesisHash, tc.head, 0); have != want {
			t.Fatalf("at %d: have %v want %v", tc.head, have, want)
		}
	}
	legacy := *params.MainnetChainConfig
	legacy.LondonBlock, legacy.PopstarBlock = nil, nil
	legacy.LegacySeaPopBlock = big.NewInt(9000000)
	remote := NewID(&legacy, params.MainnetGenesisHash, 8904999, 0)
	before := newFilter(params.MainnetChainConfig, params.MainnetGenesisHash, func() (uint64, uint64) { return 8904999, 0 })
	if err := before(remote); err != nil {
		t.Fatalf("cannot connect to pre-London legacy peer: %v", err)
	}
	after := newFilter(params.MainnetChainConfig, params.MainnetGenesisHash, func() (uint64, uint64) { return 8905000, 0 })
	if err := after(remote); err == nil {
		t.Fatal("accepted incompatible legacy fork after London")
	}
}
