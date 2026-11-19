// Package batch contains the fixed-target, 32 POP atomic deposit forwarder.
package batch

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/popcateum/go-popcateum/accounts/abi"
	"github.com/popcateum/go-popcateum/common"
)

//go:embed artifact.json
var artifactJSON []byte

type artifact struct {
	ABI                 json.RawMessage `json:"abi"`
	Bytecode            string          `json:"bytecode"`
	DeployedBytecode    string          `json:"deployedBytecode"`
	ImmutableReferences map[string][]struct {
		Start  int `json:"start"`
		Length int `json:"length"`
	} `json:"immutableReferences"`
}

func load() (artifact, error) { var a artifact; err := json.Unmarshal(artifactJSON, &a); return a, err }
func CreationCode(target common.Address) ([]byte, error) {
	a, err := load()
	if err != nil {
		return nil, err
	}
	parsed, err := abi.JSON(strings.NewReader(string(a.ABI)))
	if err != nil {
		return nil, err
	}
	args, err := parsed.Pack("", target)
	if err != nil {
		return nil, err
	}
	return append(common.FromHex(a.Bytecode), args...), nil
}
func RuntimeCode(target common.Address) ([]byte, error) {
	a, err := load()
	if err != nil {
		return nil, err
	}
	code, err := hex.DecodeString(strings.TrimPrefix(a.DeployedBytecode, "0x"))
	if err != nil {
		return nil, err
	}
	if len(a.ImmutableReferences) != 1 {
		return nil, fmt.Errorf("expected only the depositContract immutable")
	}
	for _, refs := range a.ImmutableReferences {
		for _, ref := range refs {
			if ref.Length != 32 || ref.Start < 0 || ref.Start+32 > len(code) {
				return nil, fmt.Errorf("invalid immutable reference")
			}
			copy(code[ref.Start:ref.Start+32], common.LeftPadBytes(target.Bytes(), 32))
		}
	}
	return code, nil
}
func ABI() (abi.ABI, error) {
	a, err := load()
	if err != nil {
		return abi.ABI{}, err
	}
	return abi.JSON(strings.NewReader(string(a.ABI)))
}

func TargetFromCreation(input []byte) (common.Address, error) {
	a, err := load()
	if err != nil {
		return common.Address{}, err
	}
	prefix := common.FromHex(a.Bytecode)
	if len(input) != len(prefix)+32 || !bytes.Equal(input[:len(prefix)], prefix) || !bytes.Equal(input[len(prefix):len(prefix)+12], make([]byte, 12)) {
		return common.Address{}, fmt.Errorf("not a pinned BatchDeposit deployment")
	}
	target := common.BytesToAddress(input[len(input)-20:])
	if target == (common.Address{}) {
		return target, fmt.Errorf("zero deposit target")
	}
	return target, nil
}
