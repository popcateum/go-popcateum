// Copyright 2016 The go-popcateum Authors
// This file is part of the go-popcateum library.
//
// The go-popcateum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-popcateum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-popcateum library. If not, see <http://www.gnu.org/licenses/>.

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gpop

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/popcateum/go-popcateum/core"
	"github.com/popcateum/go-popcateum/eth/downloader"
	"github.com/popcateum/go-popcateum/eth/ethconfig"
	"github.com/popcateum/go-popcateum/ethclient"
	"github.com/popcateum/go-popcateum/ethstats"
	"github.com/popcateum/go-popcateum/internal/debug"
	"github.com/popcateum/go-popcateum/les"
	"github.com/popcateum/go-popcateum/node"
	"github.com/popcateum/go-popcateum/p2p"
	"github.com/popcateum/go-popcateum/p2p/nat"
	"github.com/popcateum/go-popcateum/params"
)

// NodeConfig represents the collection of configuration values to fine tune the Gpop
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-popcateum to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// PopcateumEnabled specifies whether the node should run the Popcateum protocol.
	PopcateumEnabled bool

	// PopcateumNetworkID is the network identifier used by the Popcateum protocol to
	// decide if remote peers should be accepted or not.
	PopcateumNetworkID int64 // uint64 in truth, but Java can't handle that...

	// PopcateumGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	PopcateumGenesis string

	// PopcateumDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	PopcateumDatabaseCache int

	// PopcateumNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	PopcateumNetStats string

	// Listening address of pprof server.
	PprofAddress string
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	PopcateumEnabled:       true,
	PopcateumNetworkID:     1213,
	PopcateumDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// AddBootstrapNode adds an additional bootstrap node to the node config.
func (conf *NodeConfig) AddBootstrapNode(node *Enode) {
	conf.BootstrapNodes.Append(node)
}

// EncodeJSON encodes a NodeConfig into a JSON data dump.
func (conf *NodeConfig) EncodeJSON() (string, error) {
	data, err := json.Marshal(conf)
	return string(data), err
}

// String returns a printable representation of the node config.
func (conf *NodeConfig) String() string {
	return encodeOrError(conf)
}

// Node represents a Gpop Popcateum node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gpop node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}

	if config.PprofAddress != "" {
		debug.StartPProf(config.PprofAddress, true)
	}

	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.VersionWithMeta,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}

	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	debug.Memsize.Add("node", rawStack)

	var genesis *core.Genesis
	if config.PopcateumGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.PopcateumGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the Longcat testnet, hard code the chain configs too
		if config.PopcateumGenesis == LongcatGenesis() {
			genesis.Config = params.LongcatChainConfig
			if config.PopcateumNetworkID == 1213 {
				config.PopcateumNetworkID = 777
			}
		}
		// If we have the Rinkeby testnet, hard code the chain configs too
		if config.PopcateumGenesis == RinkebyGenesis() {
			genesis.Config = params.RinkebyChainConfig
			if config.PopcateumNetworkID == 1213 {
				config.PopcateumNetworkID = 4
			}
		}
		// If we have the Goerli testnet, hard code the chain configs too
		if config.PopcateumGenesis == GoerliGenesis() {
			genesis.Config = params.GoerliChainConfig
			if config.PopcateumNetworkID == 1213 {
				config.PopcateumNetworkID = 5
			}
		}
	}
	// Register the Popcateum protocol if requested
	if config.PopcateumEnabled {
		ethConf := ethconfig.Defaults
		ethConf.Genesis = genesis
		ethConf.SyncMode = downloader.LightSync
		ethConf.NetworkId = uint64(config.PopcateumNetworkID)
		ethConf.DatabaseCache = config.PopcateumDatabaseCache
		lesBackend, err := les.New(rawStack, &ethConf)
		if err != nil {
			return nil, fmt.Errorf("popcateum init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.PopcateumNetStats != "" {
			if err := ethstats.New(rawStack, lesBackend.ApiBackend, lesBackend.Engine(), config.PopcateumNetStats); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	return &Node{rawStack}, nil
}

// Close terminates a running node along with all it's services, tearing internal state
// down. It is not possible to restart a closed node.
func (n *Node) Close() error {
	return n.node.Close()
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	// TODO: recreate the node so it can be started multiple times
	return n.node.Start()
}

// Stop terminates a running node along with all its services. If the node was not started,
// an error is returned. It is not possible to restart a stopped node.
//
// Deprecated: use Close()
func (n *Node) Stop() error {
	return n.node.Close()
}

// GetPopcateumClient retrieves a client to access the Popcateum subsystem.
func (n *Node) GetPopcateumClient() (client *PopcateumClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &PopcateumClient{ethclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
