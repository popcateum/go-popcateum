// Package deposit contains the unmodified Ethereum Phase 0 deposit contract.
// See provenance.json for the pinned upstream source and artifact checksums.
package deposit

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/popcateum/go-popcateum/accounts/abi"
	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/crypto"
)

//go:embed deposit_contract.json
var artifactJSON []byte

//go:embed runtime.hex
var runtimeHex string

// RuntimeCode is derived by executing the pinned creation code; tests compare
// it with deployments under both Berlin and London rules.
func RuntimeCode() []byte { return common.FromHex(strings.TrimSpace(runtimeHex)) }

const ArtifactSHA256 = "fbb573648e4fe96a6b731768cbf5165f5037d7bd29f43359c5316eeb9edc78e6"

// Artifact verifies the vendored artifact before exposing its deployment code.
func Artifact() (abi.ABI, []byte, error) {
	if fmt.Sprintf("%x", sha256.Sum256(artifactJSON)) != ArtifactSHA256 {
		return abi.ABI{}, nil, fmt.Errorf("deposit artifact checksum mismatch")
	}
	var data struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode string          `json:"bytecode"`
	}
	if err := json.Unmarshal(artifactJSON, &data); err != nil {
		return abi.ABI{}, nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(string(data.ABI)))
	return contractABI, common.FromHex(data.Bytecode), err
}

// EmptyRoot is the SSZ deposit list root before the first deposit.
func EmptyRoot() common.Hash {
	var zero [32]byte
	for i := 0; i < 32; i++ {
		zero = sha256.Sum256(append(zero[:], zero[:]...))
	}
	return sha256.Sum256(append(zero[:], make([]byte, 32)...))
}

// CreationHash identifies the exact deployment transaction data.
func CreationHash() common.Hash {
	_, code, err := Artifact()
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(code)
}
