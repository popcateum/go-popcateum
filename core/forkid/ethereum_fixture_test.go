package forkid

import (
	"math/big"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/params"
)

// Official v1.11.6 Ethereum mainnet vectors are independent of Popcateum.
var ethereumMainnetConfig = &params.ChainConfig{
	ChainID:                       big.NewInt(1),
	HomesteadBlock:                big.NewInt(1_150_000),
	DAOForkBlock:                  big.NewInt(1_920_000),
	DAOForkSupport:                true,
	EIP150Block:                   big.NewInt(2_463_000),
	EIP155Block:                   big.NewInt(2_675_000),
	EIP158Block:                   big.NewInt(2_675_000),
	ByzantiumBlock:                big.NewInt(4_370_000),
	ConstantinopleBlock:           big.NewInt(7_280_000),
	PetersburgBlock:               big.NewInt(7_280_000),
	IstanbulBlock:                 big.NewInt(9_069_000),
	MuirGlacierBlock:              big.NewInt(9_200_000),
	BerlinBlock:                   big.NewInt(12_244_000),
	LondonBlock:                   big.NewInt(12_965_000),
	ArrowGlacierBlock:             big.NewInt(13_773_000),
	GrayGlacierBlock:              big.NewInt(15_050_000),
	TerminalTotalDifficulty:       big.NewInt(0), // 58_750_000_000_000_000_000_000
	TerminalTotalDifficultyPassed: true,
	ShanghaiTime:                  func() *uint64 { t := uint64(1681338455); return &t }(),
	Ethash:                        new(params.EthashConfig),
}
var ethereumMainnetGenesisHash = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
