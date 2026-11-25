package batch_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/contracts/batch"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/core/vm/runtime"
	"github.com/popcateum/go-popcateum/params"
)

type depositData struct {
	Pubkey                []byte
	WithdrawalCredentials []byte
	Signature             []byte
	DepositDataRoot       [32]byte
}

func sample(n byte) depositData {
	d := depositData{Pubkey: bytes.Repeat([]byte{n}, 48), WithdrawalCredentials: make([]byte, 32), Signature: bytes.Repeat([]byte{n + 1}, 96)}
	p := sha256.Sum256(append(append([]byte{}, d.Pubkey...), make([]byte, 16)...))
	a := sha256.Sum256(d.Signature[:64])
	b := sha256.Sum256(append(append([]byte{}, d.Signature[64:]...), make([]byte, 32)...))
	s := sha256.Sum256(append(a[:], b[:]...))
	amount := make([]byte, 32)
	binary.LittleEndian.PutUint64(amount, 32000000000)
	left := sha256.Sum256(append(p[:], d.WithdrawalCredentials...))
	right := sha256.Sum256(append(amount, s[:]...))
	d.DepositDataRoot = sha256.Sum256(append(left[:], right[:]...))
	return d
}
func TestBatchAtomicityAndRuntime(t *testing.T) {
	for _, london := range []bool{false, true} {
		t.Run(map[bool]string{false: "Berlin", true: "London"}[london], func(t *testing.T) {
			config := *params.MainnetChainConfig
			config.LondonBlock = nil
			if london {
				config.LondonBlock = big.NewInt(0)
			}
			cfg := &runtime.Config{ChainConfig: &config, Origin: common.Address{1}, BlockNumber: big.NewInt(8905001), GasLimit: 15000000, Value: new(big.Int)}
			depABI, creation, err := deposit.Artifact()
			if err != nil {
				t.Fatal(err)
			}
			_, target, _, err := runtime.Create(creation, cfg)
			if err != nil {
				t.Fatal(err)
			}
			code, err := batch.CreationCode(target)
			if err != nil {
				t.Fatal(err)
			}
			deployed, address, _, err := runtime.Create(code, cfg)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := batch.RuntimeCode(target)
			if err != nil || !bytes.Equal(deployed, expected) {
				t.Fatal("immutable runtime mismatch", err)
			}
			extracted, err := batch.TargetFromCreation(code)
			if err != nil || extracted != target {
				t.Fatal("target mismatch", err)
			}
			wrong, _ := batch.CreationCode(common.Address{99})
			if _, _, _, err = runtime.Create(wrong, cfg); err == nil {
				t.Fatal("accepted wrong target")
			}
			cfg.State.SetBalance(cfg.Origin, new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether)))
			a, err := batch.ABI()
			if err != nil {
				t.Fatal(err)
			}
			call := func(ds []depositData, value int64) error {
				input, e := a.Pack("batchDeposit", ds)
				if e != nil {
					t.Fatal(e)
				}
				cfg.Value = new(big.Int).Mul(big.NewInt(value), big.NewInt(params.Ether))
				_, _, e = runtime.Call(address, input, cfg)
				return e
			}
			count := func() uint64 {
				cfg.Value = new(big.Int)
				input, _ := depABI.Pack("get_deposit_count")
				output, _, e := runtime.Call(target, input, cfg)
				if e != nil {
					t.Fatal(e)
				}
				v, e := depABI.Unpack("get_deposit_count", output)
				if e != nil {
					t.Fatal(e)
				}
				return binary.LittleEndian.Uint64(v[0].([]byte))
			}
			bad := sample(2)
			bad.DepositDataRoot = [32]byte{}
			if call([]depositData{sample(1), bad}, 64) == nil || count() != 0 || cfg.State.GetBalance(target).Sign() != 0 || len(cfg.State.Logs()) != 0 {
				t.Fatal("partial deposit was not rolled back")
			}
			if call([]depositData{sample(1), sample(1)}, 64) == nil || count() != 0 {
				t.Fatal("duplicate batch not reverted")
			}
			if call([]depositData{sample(1)}, 31) == nil || call(nil, 0) == nil || call(make([]depositData, 101), 3232) == nil {
				t.Fatal("accepted invalid size/value")
			}
			if err = call([]depositData{sample(1), sample(2)}, 64); err != nil {
				t.Fatal("valid batch after revert", err)
			}
			if count() != 2 || cfg.State.GetBalance(target).Cmp(new(big.Int).Mul(big.NewInt(64), big.NewInt(params.Ether))) != 0 || cfg.State.GetBalance(address).Sign() != 0 || len(cfg.State.Logs()) != 3 {
				t.Fatal("incorrect forwarding/events/balances")
			}
			if call([]depositData{sample(3), sample(1)}, 64) == nil || count() != 2 {
				t.Fatal("previously submitted key not reverted")
			}
			if err = call([]depositData{sample(3)}, 32); err != nil || count() != 3 {
				t.Fatal("reverted mapping entry not restored", err)
			}
		})
	}
}
