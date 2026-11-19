// depositctl prepares, signs and verifies a standard deposit contract deployment.
// Only the explicit send command broadcasts; plan and verify are read-only RPC.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterh/liner"
	ethereum "github.com/popcateum/go-popcateum"
	"github.com/popcateum/go-popcateum/accounts/keystore"
	"github.com/popcateum/go-popcateum/common"
	"github.com/popcateum/go-popcateum/common/hexutil"
	"github.com/popcateum/go-popcateum/contracts/batch"
	"github.com/popcateum/go-popcateum/contracts/deposit"
	"github.com/popcateum/go-popcateum/core/types"
	"github.com/popcateum/go-popcateum/crypto"
	"github.com/popcateum/go-popcateum/ethclient"
	"github.com/popcateum/go-popcateum/params"
	"github.com/popcateum/go-popcateum/rpc"
)

// Fixed in the shipped binary. Tests can substitute an isolated chain genesis.
var networkGenesis = params.MainnetGenesisHash

type plan struct {
	TargetDeposit common.Address `json:"target_deposit_address,omitempty"`
	Schema        string         `json:"schema"`
	ChainID       string         `json:"chain_id"`
	Genesis       common.Hash    `json:"genesis_hash"`
	From          common.Address `json:"from"`
	Nonce         uint64         `json:"nonce"`
	Address       common.Address `json:"expected_contract_address"`
	CreationHash  common.Hash    `json:"creation_code_keccak256"`
	ObservedBlock uint64         `json:"observed_block"`
	ObservedHash  common.Hash    `json:"observed_block_hash"`
	ObservedTD    string         `json:"observed_total_difficulty"`
	TTD           string         `json:"terminal_total_difficulty"`
	London        bool           `json:"london_active"`
	GasEstimate   uint64         `json:"gas_estimate"`
	GasLimit      uint64         `json:"gas_limit"`
	GasPrice      string         `json:"gas_price_wei"`
	MaxFee        string         `json:"maximum_fee_wei"`
	MaxFeePOP     string         `json:"maximum_fee_pop"`
	Value         string         `json:"value_wei"`
}

type record struct {
	ContractKind  string         `json:"contract_kind"`
	BatchAddress  common.Address `json:"batch_contract_address,omitempty"`
	Schema        string         `json:"schema"`
	Phase         string         `json:"phase"`
	Transaction   common.Hash    `json:"transaction_hash"`
	Plan          *plan          `json:"plan,omitempty"`
	ChainID       string         `json:"chain_id"`
	Genesis       common.Hash    `json:"genesis_hash"`
	Address       common.Address `json:"deposit_contract_address"`
	From          common.Address `json:"deployer"`
	Block         uint64         `json:"deployment_block,omitempty"`
	BlockHash     common.Hash    `json:"deployment_block_hash,omitempty"`
	RuntimeHash   common.Hash    `json:"runtime_code_keccak256,omitempty"`
	Confirmations uint64         `json:"confirmations,omitempty"`
	ObservedBlock uint64         `json:"observed_block,omitempty"`
	ObservedHash  common.Hash    `json:"observed_block_hash,omitempty"`
	DepositRoot   common.Hash    `json:"deposit_root,omitempty"`
	DepositCount  uint64         `json:"deposit_count"`
	GasUsed       uint64         `json:"gas_used,omitempty"`
}

type node struct {
	raw *rpc.Client
	eth *ethclient.Client
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "depositctl:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: depositctl plan|send|verify [flags]; use <command> --help")
	}
	command := args[0]
	if command != "plan" && command != "send" && command != "verify" {
		return fmt.Errorf("unknown command %q", command)
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	endpoint := f.String("rpc", "http://127.0.0.1:9506", "EL HTTP URL or local IPC path (never Engine port)")
	out := f.String("out", "", "new JSON output file; existing files are never overwritten")
	timeout := f.Duration("timeout", 5*time.Minute, "RPC and receipt wait timeout")
	var from, price, planPath, keyPath, passwordPath, txHash, batchTarget string
	var confirmations uint64
	switch command {
	case "plan":
		f.StringVar(&batchTarget, "batch-target", "", "deploy BatchDeposit forwarding to this verified standard deposit address")
		f.StringVar(&from, "from", "", "funded deployer address")
		f.StringVar(&price, "gas-price-gwei", "2", "legacy gas price in whole gwei (1 gwei = 10^-9 POP)")
	case "send":
		f.StringVar(&planPath, "plan", "", "reviewed deployment plan JSON")
		f.StringVar(&keyPath, "keystore", "", "encrypted execution-wallet keystore JSON; not validator keystore")
		f.StringVar(&passwordPath, "password-file", "", "optional password file; otherwise hidden terminal prompt")
		f.Uint64Var(&confirmations, "confirmations", 3, "required PoW confirmations, including deployment block")
	case "verify":
		f.StringVar(&txHash, "tx", "", "deployment transaction hash")
		f.Uint64Var(&confirmations, "confirmations", 3, "required confirmations; read-only, can be run after Merge too")
	}
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 || *out == "" {
		return errors.New("--out is required and positional arguments are not accepted")
	}
	if *timeout <= 0 || (command != "plan" && confirmations == 0) {
		return errors.New("timeout and confirmations must be positive")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		return errors.New("output already exists or cannot be accessed; use a new --out path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	raw, err := rpc.DialContext(ctx, *endpoint)
	if err != nil {
		return err
	}
	defer raw.Close()
	n := node{raw, ethclient.NewClient(raw)}
	if err = n.checkNetwork(ctx); err != nil {
		return err
	}
	switch command {
	case "plan":
		if !common.IsHexAddress(from) {
			return errors.New("--from must be a 20-byte address")
		}
		gasPrice, ok := new(big.Int).SetString(price, 10)
		if !ok || gasPrice.Sign() <= 0 {
			return errors.New("--gas-price-gwei must be a positive integer")
		}
		gasPrice.Mul(gasPrice, big.NewInt(params.GWei))
		target := common.Address{}
		if batchTarget != "" {
			if !common.IsHexAddress(batchTarget) || common.HexToAddress(batchTarget) == (common.Address{}) {
				return errors.New("invalid --batch-target")
			}
			target = common.HexToAddress(batchTarget)
		}
		p, err := n.prepare(ctx, common.HexToAddress(from), gasPrice, target)
		if err != nil {
			return err
		}
		if err = writeNew(*out, p); err != nil {
			return err
		}
		return printJSON(p)
	case "send":
		if planPath == "" || keyPath == "" {
			return errors.New("--plan and --keystore are required")
		}
		var p plan
		data, err := os.ReadFile(planPath)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(data, &p); err != nil {
			return err
		}
		// Validate all public plan fields before asking for a password.
		if err = validatePlan(p); err != nil {
			return err
		}
		encrypted, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}
		password, err := readPassword(passwordPath)
		if err != nil {
			return err
		}
		key, err := keystore.DecryptKey(encrypted, password)
		if err != nil {
			return fmt.Errorf("decrypt execution keystore: %w", err)
		}
		defer func() {
			for i := range key.PrivateKey.D.Bits() {
				key.PrivateKey.D.Bits()[i] = 0
			}
		}()
		if crypto.PubkeyToAddress(key.PrivateKey.PublicKey) != p.From {
			return errors.New("keystore does not match plan deployer")
		}
		gasPrice, _ := new(big.Int).SetString(p.GasPrice, 10)
		fresh, err := n.prepare(ctx, p.From, gasPrice, p.TargetDeposit)
		if err != nil {
			return err
		}
		if fresh.Nonce != p.Nonce || fresh.Address != p.Address {
			return errors.New("nonce changed; create and review a new plan")
		}
		original, err := n.eth.HeaderByNumber(ctx, new(big.Int).SetUint64(p.ObservedBlock))
		if err != nil {
			return err
		}
		if original.Hash() != p.ObservedHash {
			return errors.New("plan observation was reorganized; create a new plan")
		}
		head, err := n.eth.HeaderByNumber(ctx, nil)
		if err != nil {
			return err
		}
		if p.GasLimit < fresh.GasEstimate || p.GasLimit > head.GasLimit {
			return errors.New("plan gas limit no longer fits; create a new plan")
		}
		maxFee := new(big.Int).Mul(new(big.Int).SetUint64(p.GasLimit), gasPrice)
		balance, err := n.eth.BalanceAt(ctx, p.From, nil)
		if err != nil {
			return err
		}
		if balance.Cmp(maxFee) < 0 {
			return errors.New("insufficient balance for plan maximum fee")
		}
		code, err := creationCode(p.TargetDeposit)
		if err != nil {
			return err
		}
		tx, err := types.SignTx(types.NewContractCreation(p.Nonce, new(big.Int), p.GasLimit, gasPrice, code), types.NewEIP155Signer(big.NewInt(1213)), key.PrivateKey)
		if err != nil {
			return err
		}
		r := record{Schema: "popcateum-deposit-deployment-v1", Phase: "prepared", Transaction: tx.Hash(), Plan: &p, ChainID: p.ChainID, Genesis: p.Genesis, Address: p.Address, From: p.From}
		r.ContractKind = "deposit"
		if p.TargetDeposit != (common.Address{}) {
			r.ContractKind = "batch"
			r.BatchAddress = p.Address
			r.Address = p.TargetDeposit
		}
		// Reserve a journal and persist the transaction hash BEFORE broadcasting.
		if err = writeNew(*out, r); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Deployment transaction:", tx.Hash(), "expected address:", p.Address)
		if err = n.eth.SendTransaction(ctx, tx); err != nil {
			return fmt.Errorf("broadcast result uncertain: %w; keep %s and use verify --tx %s before any retry", err, *out, tx.Hash())
		}
		r.Phase = "submitted"
		if err = writeOwned(*out, r); err != nil {
			return fmt.Errorf("transaction submitted, journal update failed: %w; verify --tx %s", err, tx.Hash())
		}
		for {
			receipt, err := n.eth.TransactionReceipt(ctx, tx.Hash())
			if err == nil {
				if receipt.Status != types.ReceiptStatusSuccessful {
					return errors.New("deployment reverted; transaction hash remains in journal")
				}
				latest, err := n.eth.BlockNumber(ctx)
				if err != nil {
					return err
				}
				if latest >= receipt.BlockNumber.Uint64() && latest-receipt.BlockNumber.Uint64()+1 >= confirmations {
					break
				}
			} else if !errors.Is(err, ethereum.NotFound) {
				return fmt.Errorf("receipt lookup: %w; recover with verify --tx %s", err, tx.Hash())
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("receipt/confirmation wait ended; do not redeploy automatically: verify --tx %s (%w)", tx.Hash(), ctx.Err())
			case <-time.After(time.Second):
			}
		}
		verified, err := n.verify(ctx, tx.Hash(), confirmations)
		if err != nil {
			return err
		}
		verified.Plan = &p
		if err = writeOwned(*out, verified); err != nil {
			return err
		}
		return printJSON(verified)
	case "verify":
		hashBytes, err := hexutil.Decode(txHash)
		if err != nil || len(hashBytes) != 32 {
			return errors.New("--tx must be a 32-byte transaction hash")
		}
		r, err := n.verify(ctx, common.BytesToHash(hashBytes), confirmations)
		if err != nil {
			return err
		}
		if err = writeNew(*out, r); err != nil {
			return err
		}
		return printJSON(r)
	}
	return nil
}

func (n node) checkNetwork(ctx context.Context) error {
	id, err := n.eth.ChainID(ctx)
	if err != nil {
		return err
	}
	network, err := n.eth.NetworkID(ctx)
	if err != nil {
		return err
	}
	genesis, err := n.eth.HeaderByNumber(ctx, big.NewInt(0))
	if err != nil {
		return err
	}
	if id.Cmp(big.NewInt(1213)) != 0 || network.Cmp(big.NewInt(1213)) != 0 || genesis.Hash() != networkGenesis {
		return errors.New("wrong network: require Popcateum chain/network 1213 and the original genesis hash")
	}
	return nil
}

func (n node) prepare(ctx context.Context, from common.Address, price *big.Int, targets ...common.Address) (*plan, error) {
	if price == nil || price.Sign() <= 0 || price.BitLen() > 256 {
		return nil, errors.New("invalid gas price")
	}
	sync, err := n.eth.SyncProgress(ctx)
	if err != nil {
		return nil, err
	}
	if sync != nil {
		return nil, errors.New("node is syncing; wait for completion")
	}
	head, err := n.eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	var block struct {
		TD *hexutil.Big `json:"totalDifficulty"`
	}
	if err = n.raw.CallContext(ctx, &block, "eth_getBlockByHash", head.Hash(), false); err != nil {
		return nil, err
	}
	if block.TD == nil || head.Difficulty.Sign() == 0 || (*big.Int)(block.TD).Cmp(params.MainnetTerminalTotalDifficulty) >= 0 {
		return nil, errors.New("PoW preparation requires a head below the configured TTD; verify remains available after Merge")
	}
	if head.BaseFee != nil && price.Cmp(head.BaseFee) < 0 {
		return nil, errors.New("gas price below base fee; increase --gas-price-gwei and re-plan")
	}
	nonce, err := n.eth.NonceAt(ctx, from, nil)
	if err != nil {
		return nil, err
	}
	pending, err := n.eth.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	if nonce != pending {
		return nil, errors.New("deployer has pending transactions; wait before planning")
	}
	existing, err := n.eth.CodeAt(ctx, from, nil)
	if err != nil {
		return nil, err
	}
	if len(existing) != 0 || from == (common.Address{}) {
		return nil, errors.New("deployer must be a nonzero externally owned account")
	}
	target := common.Address{}
	if len(targets) > 0 {
		target = targets[0]
	}
	if target != (common.Address{}) {
		if err := n.checkDepositTarget(ctx, target, head.Number); err != nil {
			return nil, err
		}
	}
	code, err := creationCode(target)
	if err != nil {
		return nil, err
	}
	gas, err := n.eth.EstimateGas(ctx, ethereum.CallMsg{From: from, GasPrice: price, Data: code})
	if err != nil {
		return nil, fmt.Errorf("estimate deployment (fund deployer first): %w", err)
	}
	limit := gas + gas/5 + 10000
	if limit < gas || limit > head.GasLimit {
		return nil, errors.New("deployment with gas margin exceeds current block gas limit")
	}
	cost := new(big.Int).Mul(new(big.Int).SetUint64(limit), price)
	balance, err := n.eth.BalanceAt(ctx, from, nil)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(cost) < 0 {
		return nil, fmt.Errorf("deployer requires at least %s wei for maximum deployment fee", cost)
	}
	return &plan{Schema: "popcateum-deposit-plan-v1", ChainID: "1213", Genesis: networkGenesis, From: from, Nonce: nonce, Address: crypto.CreateAddress(from, nonce), TargetDeposit: target, CreationHash: crypto.Keccak256Hash(code), ObservedBlock: head.Number.Uint64(), ObservedHash: head.Hash(), ObservedTD: (*big.Int)(block.TD).String(), TTD: params.MainnetTerminalTotalDifficulty.String(), London: head.BaseFee != nil, GasEstimate: gas, GasLimit: limit, GasPrice: price.String(), MaxFee: cost.String(), MaxFeePOP: new(big.Rat).SetFrac(cost, big.NewInt(params.Ether)).FloatString(18), Value: "0"}, nil
}

func validatePlan(p plan) error {
	price, ok := new(big.Int).SetString(p.GasPrice, 10)
	if !ok || price.Sign() <= 0 || price.BitLen() > 256 {
		return errors.New("invalid plan gas price")
	}
	code, err := creationCode(p.TargetDeposit)
	if err != nil {
		return err
	}
	cost := new(big.Int).Mul(price, new(big.Int).SetUint64(p.GasLimit))
	if p.Schema != "popcateum-deposit-plan-v1" || p.ChainID != "1213" || p.Genesis != networkGenesis || p.CreationHash != crypto.Keccak256Hash(code) || p.TTD != params.MainnetTerminalTotalDifficulty.String() || p.Value != "0" || p.From == (common.Address{}) || p.Address != crypto.CreateAddress(p.From, p.Nonce) || p.GasLimit == 0 || p.MaxFee != cost.String() || p.MaxFeePOP != new(big.Rat).SetFrac(cost, big.NewInt(params.Ether)).FloatString(18) {
		return errors.New("invalid or incompatible plan; generate it using this binary")
	}
	return nil
}

func (n node) verify(ctx context.Context, hash common.Hash, confirmations uint64) (*record, error) {
	tx, pending, err := n.eth.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, errors.New("transaction is still pending; mine and verify again")
	}
	_, creation, err := deposit.Artifact()
	if err != nil {
		return nil, err
	}
	target := common.Address{}
	if !bytes.Equal(tx.Data(), creation) {
		target, err = batch.TargetFromCreation(tx.Data())
		if err != nil {
			return nil, err
		}
	}
	if tx.To() != nil || tx.Value().Sign() != 0 || tx.ChainId().Cmp(big.NewInt(1213)) != 0 {
		return nil, errors.New("transaction is not this standard deposit contract deployment")
	}
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(1213)), tx)
	if err != nil {
		return nil, err
	}
	receipt, err := n.eth.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	address := crypto.CreateAddress(from, tx.Nonce())
	if receipt.Status != types.ReceiptStatusSuccessful || receipt.ContractAddress != address {
		return nil, errors.New("deployment reverted or receipt address mismatch")
	}
	block, err := n.eth.HeaderByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return nil, err
	}
	if block.Hash() != receipt.BlockHash {
		return nil, errors.New("deployment receipt is not canonical; wait and verify again")
	}
	head, err := n.eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	if head.Number.Cmp(receipt.BlockNumber) < 0 || head.Number.Uint64()-receipt.BlockNumber.Uint64()+1 < confirmations {
		return nil, errors.New("not enough confirmations; verify again after mining")
	}
	code, err := n.eth.CodeAt(ctx, address, head.Number)
	if err != nil {
		return nil, err
	}
	expected := deposit.RuntimeCode()
	inspectAddress := address
	kind := "deposit"
	batchAddress := common.Address{}
	if target != (common.Address{}) {
		kind = "batch"
		batchAddress = address
		inspectAddress = target
		expected, err = batch.RuntimeCode(target)
		if err != nil {
			return nil, err
		}
		if err = n.checkDepositTarget(ctx, target, head.Number); err != nil {
			return nil, err
		}
	}
	if !bytes.Equal(code, expected) {
		return nil, errors.New("deployed runtime differs from pinned contract")
	}
	contractABI, _, err := deposit.Artifact()
	if err != nil {
		return nil, err
	}
	call := func(method string) ([]interface{}, error) {
		input, err := contractABI.Pack(method)
		if err != nil {
			return nil, err
		}
		output, err := n.eth.CallContract(ctx, ethereum.CallMsg{To: &inspectAddress, Data: input}, head.Number)
		if err != nil {
			return nil, err
		}
		return contractABI.Unpack(method, output)
	}
	rootValues, err := call("get_deposit_root")
	if err != nil {
		return nil, err
	}
	countValues, err := call("get_deposit_count")
	if err != nil {
		return nil, err
	}
	root := common.Hash(rootValues[0].([32]byte))
	countBytes := countValues[0].([]byte)
	if len(countBytes) != 8 {
		return nil, errors.New("invalid deposit count encoding")
	}
	count := binary.LittleEndian.Uint64(countBytes)
	if count == 0 && root != deposit.EmptyRoot() {
		return nil, errors.New("empty deposit tree root mismatch")
	}
	finalHead, err := n.eth.HeaderByNumber(ctx, head.Number)
	if err != nil {
		return nil, err
	}
	if finalHead.Hash() != head.Hash() {
		return nil, errors.New("observation reorganized; verify again")
	}
	return &record{Schema: "popcateum-deposit-deployment-v1", Phase: "verified", Transaction: hash, ChainID: "1213", Genesis: networkGenesis, ContractKind: kind, BatchAddress: batchAddress, Address: inspectAddress, From: from, Block: receipt.BlockNumber.Uint64(), BlockHash: receipt.BlockHash, RuntimeHash: crypto.Keccak256Hash(code), Confirmations: head.Number.Uint64() - receipt.BlockNumber.Uint64() + 1, ObservedBlock: head.Number.Uint64(), ObservedHash: head.Hash(), DepositRoot: root, DepositCount: count, GasUsed: receipt.GasUsed}, nil
}

func readPassword(path string) (string, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r"), nil
	}
	if !liner.TerminalSupported() {
		return "", errors.New("no secure terminal; use --password-file")
	}
	term := liner.NewLiner()
	defer term.Close()
	return term.PasswordPrompt("Execution keystore password: ")
}
func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
func writeNew(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
func writeOwned(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".deposit-journal-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func creationCode(target common.Address) ([]byte, error) {
	if target != (common.Address{}) {
		return batch.CreationCode(target)
	}
	_, code, err := deposit.Artifact()
	return code, err
}
func (n node) checkDepositTarget(ctx context.Context, target common.Address, height *big.Int) error {
	code, err := n.eth.CodeAt(ctx, target, height)
	if err != nil {
		return err
	}
	if !bytes.Equal(code, deposit.RuntimeCode()) {
		return errors.New("batch target is not the pinned standard deposit contract")
	}
	return nil
}
