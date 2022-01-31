// Copyright 2015 The go-popcateum Authors
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

package params

import "github.com/popcateum/go-popcateum/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Popcateum network.
var MainnetBootnodes = []string{
	// gcp-usa-1
	"enode://74bd8ce92043ad08f9c9f4318a2f13e2cc17139a4c3f776608f02ca37fd3799fd38d4447a88a3695c14c14bfc546e342b2793ebca3421f7194b495c9586e8396@34.125.181.37:60606",

	// gcp-eu-1
	"enode://3af84c81b030b1e852555b150e6b60229862dc8bab719d5eb623f7a31041f168b34a94fe9b9db11a780929e2271d0bc2d15571b972c24e82ef356dc0eeac1414@34.118.9.192:60606",

	// gcp-au-1
	"enode://59ff84730e94c0b1dc51a1323ec34416fb1b5fc72a96406d0f050e6d42f54d59e3d250c71d48d96644ee075eada8bcb3e7d68f2ae996ddfa3df313af11e0793d@34.129.113.83:60606",

	// gcp-jp-1
	"enode://1ee12060b7c46f3bc5c12345258d236debd9ce32714e1a3f84af9d9bde85fa086cebc3e6760da96d5df254adf9bb208bc97101e9d2b6fa0ea1cc8feecf5b1eaa@35.200.73.88:60606",

	// Hun-1
	"enode://e577e880ca38b8b935748ee4b8cf77b9be1017b00abaf287fd3148a2fd378865e3c36b188442c16dd293fddb311033b4cc82c109c6143505c412919245d432c3@124.54.168.230:60606",
}

// LongcatBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Longcat test network.
var LongcatBootnodes = []string{

}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	
}

// GoerliBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Görli test network.
var GoerliBootnodes = []string{
	
}

// YoloV3Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// YOLOv3 ephemeral test network.
// TODO: Set Yolov3 bootnodes
var YoloV3Bootnodes = []string{
	
}

var V5Bootnodes = []string{
	// gcp-usa-1
	"enr:-KK4QIaiRiK2Ssn102qStidWrgpBno1IbqTCycnTgWgizM15eejCr96uradLrzf2LtE1k_AH20KjuC5ganEJtEGYSx2CGpKDZXRoysmEo1hwp4M9CQCCaWSCdjSCaXCEfwAAAYlzZWNwMjU2azGhAnS9jOkgQ60I-cn0MYovE-LMFxOaTD93ZgjwLKN_03mfhHNuYXDAg3RjcILsvoN1ZHCC7L4",

	// gcp-eu-1
	"enr:-KK4QBOSzjDp571hNKt0kE2FBfSbsVSbq-ieIGP2XNtw12dqHCeU-OYg-YKNcn-RpSX1GqNM8AGzck3OUKjVsN6BKneCEMmDZXRoysmEo1hwp4M9CQCCaWSCdjSCaXCEInYJwIlzZWNwMjU2azGhAjr4TIGwMLHoUlVbFQ5rYCKYYtyLq3GdXrYj96MQQfFohHNuYXDAg3RjcILsvoN1ZHCC7L4",

	// gcp-au-1
	"enr:-KK4QJp1mMDyca8nUh0wjQSCZwizXjMknXYnscBON99wV24wPEH__FAsZOD1ygrTBTuPCp187eqNkTHzbrpFZmV-5P6CGZyDZXRoysmEo1hwp4M9CQCCaWSCdjSCaXCEIoFxU4lzZWNwMjU2azGhA1n_hHMOlMCx3FGhMj7DRBb7G1_HKpZAbQ8FDm1C9U1ZhHNuYXDAg3RjcILsvoN1ZHCC7L4",

	// gcp-jp-1
	"enr:-KK4QGnr0XsOBUvkz1UIxK8PE-QJkVD2Rg-Fa53pRFNtaanTA5kfSOrbm5abmhV3yrIxFIQ9KHxFLyYBJte91_uXflKCGMGDZXRoysmEo1hwp4M9CQCCaWSCdjSCaXCEI8hJWIlzZWNwMjU2azGhAh7hIGC3xG87xcEjRSWNI23r2c4ycU4aP4SvnZvehfoIhHNuYXDAg3RjcILsvoN1ZHCC7L4",

}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/popcateum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case LongcatGenesisHash:
		net = "longcat"
	case RinkebyGenesisHash:
		net = "rinkeby"
	case GoerliGenesisHash:
		net = "goerli"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".popcateum.org"
}
