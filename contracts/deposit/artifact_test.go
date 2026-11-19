package deposit

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/core/vm/runtime"
	"github.com/popcateum/go-popcateum/crypto"
	"github.com/popcateum/go-popcateum/params"
)

// Test keys/signatures here deliberately test the execution contract only.
// The standard contract does NOT check BLS signatures; these are not validators.
func TestDepositBerlinAndLondon(t *testing.T) {
	for _, london := range []bool{false, true} {
		name := "Berlin"
		if london {
			name = "London"
		}
		t.Run(name, func(t *testing.T) {
			config := *params.MainnetChainConfig
			config.LondonBlock = nil
			if london {
				config.LondonBlock = big.NewInt(0)
			}
			cfg := &runtime.Config{ChainConfig: &config, Origin: common.Address{1}, BlockNumber: big.NewInt(8905001), GasLimit: 10000000, Value: new(big.Int)}
			contractABI, creation, err := Artifact()
			if err != nil {
				t.Fatal(err)
			}
			code, address, _, err := runtime.Create(creation, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(code, RuntimeCode()) || crypto.Keccak256Hash(code) != common.HexToHash("0x6c029a231254fadb724d63be769f75eedd66362df034a3e663252b49d062a666") {
				t.Fatal("runtime mismatch")
			}
			cfg.State.SetBalance(cfg.Origin, new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether)))
			query := func(method string) interface{} {
				t.Helper()
				cfg.Value = new(big.Int)
				input, err := contractABI.Pack(method)
				if err != nil {
					t.Fatal(err)
				}
				output, _, err := runtime.Call(address, input, cfg)
				if err != nil {
					t.Fatal(err)
				}
				values, err := contractABI.Unpack(method, output)
				if err != nil {
					t.Fatal(err)
				}
				return values[0]
			}
			if !bytes.Equal(query("get_deposit_count").([]byte), make([]byte, 8)) || common.Hash(query("get_deposit_root").([32]byte)) != EmptyRoot() {
				t.Fatal("wrong initial tree")
			}
			var leaves [][32]byte
			for index := 0; index < 2; index++ {
				pubkey := bytes.Repeat([]byte{byte(index + 1)}, 48)
				credentials := make([]byte, 32)
				credentials[0] = 1
				credentials[31] = 9
				signature := bytes.Repeat([]byte{byte(index + 3)}, 96)
				amount := uint64(32000000000)
				leaf := depositDataRoot(pubkey, credentials, signature, amount)
				input, err := contractABI.Pack("deposit", pubkey, credentials, signature, leaf)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Value = new(big.Int).Mul(big.NewInt(32), big.NewInt(params.Ether))
				if _, _, err = runtime.Call(address, input, cfg); err != nil {
					t.Fatal(err)
				}
				leaves = append(leaves, leaf)
				count := binary.LittleEndian.Uint64(query("get_deposit_count").([]byte))
				if count != uint64(index+1) || common.Hash(query("get_deposit_root").([32]byte)) != listRoot(leaves) {
					t.Fatal("incorrect deposit tree root/count")
				}
				logs := cfg.State.Logs()
				if len(logs) != index+1 {
					t.Fatal("missing deposit event")
				}
				event := logs[index]
				if event.Address != address || len(event.Topics) != 1 || event.Topics[0] != contractABI.Events["DepositEvent"].ID {
					t.Fatal("wrong event signature")
				}
				values, err := contractABI.Events["DepositEvent"].Inputs.Unpack(event.Data)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(values[0].([]byte), pubkey) || !bytes.Equal(values[1].([]byte), credentials) || binary.LittleEndian.Uint64(values[2].([]byte)) != amount || !bytes.Equal(values[3].([]byte), signature) || binary.LittleEndian.Uint64(values[4].([]byte)) != uint64(index) {
					t.Fatal("wrong event fields")
				}
			}
			wantedBalance := new(big.Int).Mul(big.NewInt(64), big.NewInt(params.Ether))
			pubkey, credentials, signature := make([]byte, 48), make([]byte, 32), make([]byte, 96)
			validRoot := depositDataRoot(pubkey, credentials, signature, 32000000000)
			for _, tc := range []struct {
				name   string
				value  *big.Int
				pubkey []byte
				root   [32]byte
			}{
				{"bad root", new(big.Int).Mul(big.NewInt(32), big.NewInt(params.Ether)), pubkey, [32]byte{}},
				{"less than 1 POP", big.NewInt(params.Ether / 2), pubkey, validRoot},
				{"fractional gwei", new(big.Int).Add(new(big.Int).Mul(big.NewInt(32), big.NewInt(params.Ether)), big.NewInt(1)), pubkey, validRoot},
				{"bad pubkey length", new(big.Int).Mul(big.NewInt(32), big.NewInt(params.Ether)), pubkey[:47], validRoot},
			} {
				input, err := contractABI.Pack("deposit", tc.pubkey, credentials, signature, tc.root)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Value = tc.value
				if _, _, err = runtime.Call(address, input, cfg); err == nil {
					t.Fatalf("accepted %s", tc.name)
				}
				if cfg.State.GetBalance(address).Cmp(wantedBalance) != 0 || len(cfg.State.Logs()) != 2 || common.Hash(query("get_deposit_root").([32]byte)) != listRoot(leaves) {
					t.Fatalf("revert did not preserve state: %s", tc.name)
				}
			}
			cfg.Value = big.NewInt(params.Ether)
			if _, _, err = runtime.Call(address, nil, cfg); err == nil {
				t.Fatal("accepted plain value transfer")
			}
		})
	}
}

func depositDataRoot(pubkey, credentials, signature []byte, amount uint64) [32]byte {
	pubRoot := sha256.Sum256(append(append([]byte{}, pubkey...), make([]byte, 16)...))
	a := sha256.Sum256(signature[:64])
	b := sha256.Sum256(append(append([]byte{}, signature[64:]...), make([]byte, 32)...))
	sigRoot := sha256.Sum256(append(a[:], b[:]...))
	amountChunk := make([]byte, 32)
	binary.LittleEndian.PutUint64(amountChunk, amount)
	left := sha256.Sum256(append(pubRoot[:], credentials...))
	right := sha256.Sum256(append(amountChunk, sigRoot[:]...))
	return sha256.Sum256(append(left[:], right[:]...))
}

// Construct a full level of the sparse tree from leaves, independent of the
// contract's progressive branch-update implementation.
func listRoot(leaves [][32]byte) common.Hash {
	nodes := append([][32]byte{}, leaves...)
	var zero [32]byte
	for level := 0; level < 32; level++ {
		if len(nodes)%2 != 0 {
			nodes = append(nodes, zero)
		}
		next := make([][32]byte, len(nodes)/2)
		for i := range next {
			next[i] = sha256.Sum256(append(nodes[i*2][:], nodes[i*2+1][:]...))
		}
		nodes = next
		zero = sha256.Sum256(append(zero[:], zero[:]...))
	}
	chunk := make([]byte, 32)
	binary.LittleEndian.PutUint64(chunk, uint64(len(leaves)))
	return sha256.Sum256(append(nodes[0][:], chunk...))
}
