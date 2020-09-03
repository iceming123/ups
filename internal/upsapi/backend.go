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

// Package ethapi implements the general True API functions.
package upsapi

import (
	"context"
	"math/big"

	"github.com/iceming123/ups/accounts"
	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/core"
	"github.com/iceming123/ups/core/state"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/core/vm"
	"github.com/iceming123/ups/ups/downloader"
	"github.com/iceming123/ups/upsdb"
	"github.com/iceming123/ups/event"
	"github.com/iceming123/ups/params"
	"github.com/iceming123/ups/rpc"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// General True API
	Downloader() *downloader.Downloader
	ProtocolVersion() int
	SuggestPrice(ctx context.Context) (*big.Int, error)
	ChainDb() upsdb.Database
	EventMux() *event.TypeMux
	AccountManager() *accounts.Manager

	// BlockChain API
	SetHead(number uint64)
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error)
	StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error)
	GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error)
	SubscribeChainEvent(ch chan<- types.FastChainEvent) event.Subscription
	SubscribeChainHeadEvent(ch chan<- types.FastChainHeadEvent) event.Subscription
	SubscribeChainSideEvent(ch chan<- types.FastChainSideEvent) event.Subscription
	GetCommittee(id rpc.BlockNumber) (map[string]interface{}, error)
	GetCurrentCommitteeNumber() *big.Int

	GetStateChangeByFastNumber(fastNumber rpc.BlockNumber) *types.BlockBalance
	GetChainRewardContent(blockNr rpc.BlockNumber) *types.ChainReward

	// TxPool API
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash common.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)
	Stats() (pending int, queued int)
	TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	SubscribeNewTxsEvent(chan<- types.NewTxsEvent) event.Subscription

	ChainConfig() *params.ChainConfig
	CurrentBlock() *types.Block
}

func GetAPIs(apiBackend Backend) []rpc.API {
	nonceLock := new(AddrLocker)
	var apis []rpc.API
	namespaces := []string{"ups", "eth"}
	for _, name := range namespaces {
		apis = append(apis, []rpc.API{
			{
				Namespace: name,
				Version:   "1.0",
				Service:   NewPublicTrueAPI(apiBackend),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   NewPublicBlockChainAPI(apiBackend),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   NewPublicTransactionPoolAPI(apiBackend, nonceLock),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   NewPublicAccountAPI(apiBackend.AccountManager()),
				Public:    true,
			},
		}...)
	}
	apis = append(apis, []rpc.API{
		{
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewPublicTxPoolAPI(apiBackend),
			Public:    true,
		},{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(apiBackend, nonceLock),
			Public:    false,
		}, {
			Namespace: "impawn",
			Version:   "1.0",
			Service:   NewPublicImpawnAPI(apiBackend),
			Public:    true,
		},
	}...)
	return apis
}
