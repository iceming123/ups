// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Upschain network.
var MainnetBootnodes = []string{
	"enode://cd99daa76de43e2b7a5806c3455d33012cd127bca9b2e271be3af5d78e402c153a77e1d408f708770fb390e597621407f963f1c444090c21f91e03e03caa2110@39.98.216.197:30313", // CN
	"enode://e95937d68263a59c95ac1199eecc450b3590624accaf1542c7e51d8dc3ca3bfa6d3f60785b021c408b4a9a67b2869da33237c75448ae29b70506164a2bfe6931@13.52.156.74:30313",  // US WEST
	"enode://9032cc37954363b4d2dd37a898959aadf213718ff1bdb146848fb8c9a5adfd31d543ca870a08a223b27da2309051d0ce41775fa6de9337ed519b64cfa85b5b0c@47.241.22.155:30313", // SG
	"enode://4c64220af42271b6a6ea5463e97a125fef86d0bbb077db7d669af9d020d8ccf8ef4b617e3b36bbb9c10096404ecc1a7e06bcec3210a2cdf49b2bce5a0e1c7eb5@8.209.88.41:30313",   // DE

	"enode://fb331ff6aded86b393d9de2f9c449d313b356af0c4c0b9500e0f6c51bcb4ed31ca45dc2ab64c6182d1876eb9e3fd073d488277a40a6d357bc6e63350a2e00ffc@101.132.183.35:30313", // CN
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://dd593b5714c4e619ee4bb72083d3895e8637a81db302a6974e5513e8b823fd0004120ccb5efcb1428573ddbe85a61fde75de2878357ad4bcac3c9a9f7a58a56c@127.0.0.1:30303",
	"enode://eddf857f0b17098b1e8190d17003630f4f65e5a439dca4c8d2421badf3e28728ded8f638cb72aba6b5fd9b28cb1a53188c77f3652fd497fef0d9bb7e88f18974@127.0.0.1:30304",
	"enode://7e423f1603a43a5e185a055e58b0f1a2631a3442673ae08b325906e744d61e89463ae05594f82655afaaa399af924af538c6457f2765592c938f979f1ee608d0@127.0.0.1:30305",
	"enode://d30477149f09146eec3e77994ec133c2f3a711e084960d1066dcb20619b170733b03d5fc68a0b3d2b77fe1563ef7b7f3413bd46289b248af9cef0914eccac0c3@127.0.0.1:30306",
}

// DevnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the dev Upschain network.
var DevnetBootnodes = []string{
	"enode://dd593b5714c4e619ee4bb72083d3895e8637a81db302a6974e5513e8b823fd0004120ccb5efcb1428573ddbe85a61fde75de2878357ad4bcac3c9a9f7a58a56c@127.0.0.1:30303",
	"enode://eddf857f0b17098b1e8190d17003630f4f65e5a439dca4c8d2421badf3e28728ded8f638cb72aba6b5fd9b28cb1a53188c77f3652fd497fef0d9bb7e88f18974@127.0.0.1:30304",
	"enode://7e423f1603a43a5e185a055e58b0f1a2631a3442673ae08b325906e744d61e89463ae05594f82655afaaa399af924af538c6457f2765592c938f979f1ee608d0@127.0.0.1:30305",
	"enode://d30477149f09146eec3e77994ec133c2f3a711e084960d1066dcb20619b170733b03d5fc68a0b3d2b77fe1563ef7b7f3413bd46289b248af9cef0914eccac0c3@127.0.0.1:30306",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://ebb007b1efeea668d888157df36cf8fe49aa3f6fd63a0a67c45e4745dc081feea031f49de87fa8524ca29343a21a249d5f656e6daeda55cbe5800d973b75e061@39.98.171.41:30315",
	"enode://b5062c25dc78f8d2a8a216cebd23658f170a8f6595df16a63adfabbbc76b81b849569145a2629a65fe50bfd034e38821880f93697648991ba786021cb65fb2ec@39.98.43.179:30312",
}
