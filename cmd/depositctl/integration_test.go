package main

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/popcateum/go-popcateum/accounts/keystore"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/core"
	"github.com/popcateum/go-popcateum/crypto"
	"github.com/popcateum/go-popcateum/ethclient"
	"github.com/popcateum/go-popcateum/params"
	"github.com/popcateum/go-popcateum/rpc"
)

// The shipped CLI accepts only the real genesis. This test first checks that
// rejection, then substitutes a test-only genesis for a funded isolated chain.
// The gpop subprocess has no peers, no HTTP listener, and uses a small Ethash test DAG.
func TestPoWDeploymentIntegration(t *testing.T) {
	binary := os.Getenv("GPOP_BINARY")
	if binary == "" {
		t.Skip("set GPOP_BINARY to the built gpop for isolated RPC integration")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("/tmp", "pop-deposit-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	key, err := crypto.HexToECDSA(strings.Repeat("01", 32))
	if err != nil {
		t.Fatal(err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	ks := keystore.NewKeyStore(filepath.Join(dir, "keys"), keystore.LightScryptN, keystore.LightScryptP)
	account, err := ks.ImportECDSA(key, "public-test-password")
	if err != nil {
		t.Fatal(err)
	}
	password := filepath.Join(dir, "password")
	if err = os.WriteFile(password, []byte("public-test-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config := *params.MainnetChainConfig
	config.PopSongBlock, config.PopcornBlock, config.LondonBlock = big.NewInt(0), big.NewInt(0), big.NewInt(0)
	genesis := &core.Genesis{Config: &config, GasLimit: 15000000, Difficulty: big.NewInt(131072), BaseFee: big.NewInt(params.InitialBaseFee), Alloc: core.GenesisAlloc{from: {Balance: new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))}}}
	genesisPath := filepath.Join(dir, "genesis.json")
	data, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(genesisPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	datadir := filepath.Join(dir, "chain")
	if output, err := exec.Command(binary, "--datadir", datadir, "init", genesisPath).CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, output)
	}
	configPath := filepath.Join(dir, "node.toml")
	if err = os.WriteFile(configPath, []byte("[Eth.Ethash]\nPowMode = 2\nCachesOnDisk = 0\nDatasetsOnDisk = 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ipc := filepath.Join(dir, "node.ipc")
	logFile, err := os.Create(filepath.Join(dir, "node.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	process := exec.Command(binary, "--datadir", datadir, "--config", configPath, "--ipcpath", ipc, "--networkid", "1213", "--port", "0", "--authrpc.port", "0", "--nodiscover", "--maxpeers", "0", "--nat", "none", "--syncmode", "full", "--gcmode", "archive", "--cache", "64", "--verbosity", "2", "--datadir.minfreedisk", "0", "--miner.etherbase", from.Hex())
	process.Stdout, process.Stderr = logFile, logFile
	if err = process.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		process.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- process.Wait() }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			process.Process.Kill()
			<-done
		}
		if t.Failed() {
			data, _ := os.ReadFile(logFile.Name())
			if len(data) > 12000 {
				data = data[len(data)-12000:]
			}
			t.Log(string(data))
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	var raw *rpc.Client
	for i := 0; i < 100; i++ {
		raw, err = rpc.DialIPC(ctx, ipc)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	eth := ethclient.NewClient(raw)
	out := func(name string) string { return filepath.Join(dir, name+".json") }
	invoke := func(command string, args ...string) error {
		return run(append([]string{command, "--rpc", ipc}, args...))
	}
	if err = invoke("plan", "--from", from.Hex(), "--out", out("wrong-network")); err == nil || !strings.Contains(err.Error(), "wrong network") {
		t.Fatalf("accepted synthetic genesis: %v", err)
	}
	originalGenesis := networkGenesis
	networkGenesis = genesis.ToBlock().Hash()
	defer func() { networkGenesis = originalGenesis }()
	if err = invoke("plan", "--from", from.Hex(), "--out", out("plan")); err != nil {
		t.Fatal(err)
	}
	nonce, err := eth.PendingNonceAt(ctx, from)
	if err != nil || nonce != 0 {
		t.Fatal("plan sent a transaction", err)
	}
	oldTTD := params.MainnetTerminalTotalDifficulty
	params.MainnetTerminalTotalDifficulty = big.NewInt(0)
	blocked := invoke("plan", "--from", from.Hex(), "--out", out("past-ttd"))
	params.MainnetTerminalTotalDifficulty = oldTTD
	if blocked == nil || !strings.Contains(blocked.Error(), "below the configured TTD") {
		t.Fatalf("TTD guard: %v", blocked)
	}
	if err = raw.CallContext(ctx, nil, "miner_start", 1); err != nil {
		t.Fatal(err)
	}
	if err = invoke("send", "--plan", out("plan"), "--keystore", account.URL.Path, "--password-file", password, "--confirmations", "3", "--timeout", "45s", "--out", out("deployment")); err != nil {
		t.Fatal(err)
	}
	var first record
	data, err = os.ReadFile(out("deployment"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &first); err != nil {
		t.Fatal(err)
	}
	if first.Phase != "verified" || first.DepositCount != 0 || first.DepositRoot != deposit.EmptyRoot() || first.Confirmations < 3 {
		t.Fatalf("bad verified record: %+v", first)
	}
	if err = invoke("verify", "--tx", first.Transaction.Hex(), "--out", out("verified-again")); err != nil {
		t.Fatal(err)
	}
	if err = invoke("send", "--plan", out("plan"), "--keystore", account.URL.Path, "--password-file", password, "--out", out("stale-plan")); err == nil || !strings.Contains(err.Error(), "nonce changed") {
		t.Fatalf("stale nonce not rejected: %v", err)
	}

	if err = invoke("plan", "--from", from.Hex(), "--batch-target", first.Address.Hex(), "--out", out("batch-plan")); err != nil {
		t.Fatal(err)
	}
	if err = invoke("send", "--plan", out("batch-plan"), "--keystore", account.URL.Path, "--password-file", password, "--confirmations", "3", "--timeout", "45s", "--out", out("batch-deployment")); err != nil {
		t.Fatal(err)
	}
	var wrapper record
	data, err = os.ReadFile(out("batch-deployment"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &wrapper); err != nil {
		t.Fatal(err)
	}
	if wrapper.Phase != "verified" || wrapper.ContractKind != "batch" || wrapper.Address != first.Address || wrapper.BatchAddress == first.Address {
		t.Fatalf("bad batch manifest: %+v", wrapper)
	}
	if err = invoke("verify", "--tx", wrapper.Transaction.Hex(), "--out", out("batch-verified-again")); err != nil {
		t.Fatal(err)
	}
	if err = raw.CallContext(ctx, nil, "miner_stop"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	if err = invoke("plan", "--from", from.Hex(), "--out", out("plan2")); err != nil {
		t.Fatal(err)
	}
	if err = invoke("send", "--plan", out("plan2"), "--keystore", account.URL.Path, "--password-file", password, "--timeout", "3s", "--out", out("pending")); err == nil {
		t.Fatal("unmined deployment unexpectedly completed")
	}
	var pending record
	data, err = os.ReadFile(out("pending"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &pending); err != nil {
		t.Fatal(err)
	}
	if pending.Phase != "submitted" {
		t.Fatalf("lost broadcast journal: %+v", pending)
	}
	if err = raw.CallContext(ctx, nil, "miner_start", 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		err = invoke("verify", "--tx", pending.Transaction.Hex(), "--confirmations", "1", "--out", out("recovered"))
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("could not recover timed-out deployment: %v", err)
	}
	t.Logf("verified deployment %s at block %d; gas used %d; timeout recovered by transaction hash", first.Address, first.Block, first.GasUsed)
}
