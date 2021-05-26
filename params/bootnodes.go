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
	"enode://ccec1a0bb4f1c2a005652f7f39cd25f359a0de01246a2d8990b85d70f085be56b55956898341dbf1a3071ff339545f16b2f7a24acdcfc65deafe28c0b7cbd192@34.122.106.50:60606",

	// gcp-eu-1
	"enode://6c77ade73b5a69250f4d96cbc5247720ddeaddaa66d1162019909e278c6b40c4a0450f3c43c41b054280010feb4484a4b57105d05f854fa281df1fb60f21f987@34.118.106.111:60606",

	// gcp-au-1
	"enode://03efe8ebd19e1f7cbddf828be9306a081e7f85d264418aad719a8ba54208e14d8d6b590f7ebdc778e7cb4af0c175e5d76b49667da71fd717071173d6faafe96b@34.151.90.235:60606",

	// gcp-ko-1
	"enode://ef8984a56f4e9ca2b67697b5515df597607cd4544c6565997b55febbeec06f923840e9d3345480c79c8e3e3e7226d8c7931ac42c2f9a516f01657dd512254f04@34.64.237.162:60606",
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
	"enr:-J24QBmTAElTorQ2G_cPeueiBIMfz7RCLsBHVBouTv2QASlxDkmNqqNy2y7ZDub85IQHtXhGGy88RobqIAJO-Cnq5HsDg2V0aMfGhPyPDe6AgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQLM7BoLtPHCoAVlL385zSXzWaDeASRqLYmQuF1w8IW-VoRzbmFwwIN0Y3CC7L6DdWRwguy-",

	// gcp-eu-1
	"enr:-J24QBdX0A3XI1_n82tqVWj4PnrAg1gaRO7_bsyvoR5apWH5RPRvDdsA3qU15MRW7uO8wByZK83u4yWAdT8BeYfRzI0Bg2V0aMfGhPyPDe6AgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQNsd63nO1ppJQ9NlsvFJHcg3erdqmbRFiAZkJ4njGtAxIRzbmFwwIN0Y3CC7L6DdWRwguy-",

	// gcp-au-1
	"enr:-J24QFEFSF6pDflOes9vr8RY_KOAsrt-4fBMfYhI-9cAw96GTrT042rby1t9V7hsaTAHDR_yBO2c_T6Z0uwNfsdo5t8Bg2V0aMfGhPyPDe6AgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQMD7-jr0Z4ffL3fgovpMGoIHn-F0mRBiq1xmoulQgjhTYRzbmFwwIN0Y3CC7L6DdWRwguy-",

	// gcp-ko-1
	"enr:-J24QErx4dLES3M_Kb1OP2hE1Z2UE1x2Cvc7ORX2MxTguMAqWfYN6Y8q8ic4tpIrGjJos4r6mcEf6TLdhLbEgil4RuoBg2V0aMfGhPyPDe6AgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQLviYSlb06corZ2l7VRXfWXYHzUVExlZZl7Vf677sBvkoRzbmFwwIN0Y3CC7L6DdWRwguy-",

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
