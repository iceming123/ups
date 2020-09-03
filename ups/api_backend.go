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

package ups

import (
	"context"
	"math/big"

	"github.com/iceming123/ups/accounts"
	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/common/math"
	"github.com/iceming123/ups/core"
	"github.com/iceming123/ups/core/bloombits"
	"github.com/iceming123/ups/core/rawdb"
	"github.com/iceming123/ups/core/state"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/core/vm"
	"github.com/iceming123/ups/ups/downloader"
	"github.com/iceming123/ups/ups/gasprice"
	"github.com/iceming123/ups/upsdb"
	"github.com/iceming123/ups/event"
	"github.com/iceming123/ups/params"
	"github.com/iceming123/ups/rpc"
)

// TRUEAPIBackend implements ethapi.Backend for full nodes
type TrueAPIBackend struct {
	ups *Upschain
	gpo   *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *TrueAPIBackend) ChainConfig() *params.ChainConfig {
	return b.ups.chainConfig
}

// CurrentBlock return the fast chain current Block
func (b *TrueAPIBackend) CurrentBlock() *types.Block {
	return b.ups.blockchain.CurrentBlock()
}
// SetHead Set the newest position of Fast Chain, that will reset the fast blockchain comment
func (b *TrueAPIBackend) SetHead(number uint64) {
	b.ups.protocolManager.downloader.Cancel()
	b.ups.blockchain.SetHead(number)
}

// HeaderByNumber returns Header of fast chain by the number
// rpc.PendingBlockNumber == "pending"; rpc.LatestBlockNumber == "latest" ; rpc.LatestBlockNumber == "earliest"
func (b *TrueAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ups.blockchain.CurrentBlock().Header(), nil
	}
	return b.ups.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

// HeaderByHash returns header of fast chain by the hash
func (b *TrueAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.ups.blockchain.GetHeaderByHash(hash), nil
}

// BlockByNumber returns block of fast chain by the number
func (b *TrueAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	if blockNr == rpc.PendingBlockNumber {
		block := b.ups.blockchain.CurrentBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ups.blockchain.CurrentBlock(), nil
	}
	return b.ups.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

// StateAndHeaderByNumber returns the state of block by the number
func (b *TrueAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		state, _ := b.ups.blockchain.State()
		block := b.ups.blockchain.CurrentBlock()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.ups.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

// GetBlock returns the block by the block's hash
func (b *TrueAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.ups.blockchain.GetBlockByHash(hash), nil
}

// GetReceipts returns the Receipt details by txhash
func (b *TrueAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.ups.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.ups.chainDb, hash, *number), nil
	}
	return nil, nil
}

// GetLogs returns the logs by txhash
func (b *TrueAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.ups.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.ups.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

// GetEVM returns the EVM
func (b *TrueAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.ups.BlockChain(), nil, nil)
	return vm.NewEVM(context, state, b.ups.chainConfig, vmCfg), vmError, nil
}

// SubscribeRemovedLogsEvent registers a subscription of RemovedLogsEvent in fast blockchain
func (b *TrueAPIBackend) SubscribeRemovedLogsEvent(ch chan<- types.RemovedLogsEvent) event.Subscription {
	return b.ups.BlockChain().SubscribeRemovedLogsEvent(ch)
}

// SubscribeChainEvent registers a subscription of chainEvnet in fast blockchain
func (b *TrueAPIBackend) SubscribeChainEvent(ch chan<- types.FastChainEvent) event.Subscription {
	return b.ups.BlockChain().SubscribeChainEvent(ch)
}

// SubscribeChainHeadEvent registers a subscription of chainHeadEvnet in fast blockchain
func (b *TrueAPIBackend) SubscribeChainHeadEvent(ch chan<- types.FastChainHeadEvent) event.Subscription {
	return b.ups.BlockChain().SubscribeChainHeadEvent(ch)
}

// SubscribeChainSideEvent registers a subscription of chainSideEvnet in fast blockchain,deprecated
func (b *TrueAPIBackend) SubscribeChainSideEvent(ch chan<- types.FastChainSideEvent) event.Subscription {
	return b.ups.BlockChain().SubscribeChainSideEvent(ch)
}

// SubscribeLogsEvent registers a subscription of log in fast blockchain
func (b *TrueAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.ups.BlockChain().SubscribeLogsEvent(ch)
}

func (b *TrueAPIBackend) GetChainRewardContent(blockNr rpc.BlockNumber) *types.ChainReward {
	sheight := uint64(blockNr)
	return b.ups.blockchain.GetRewardInfos(sheight)
}

// GetStateChangeByFastNumber returns the Committee info by committee number
func (b *TrueAPIBackend) GetStateChangeByFastNumber(fastNumber rpc.BlockNumber) *types.BlockBalance {
	return b.ups.blockchain.GetBalanceInfos(uint64(fastNumber))
}

func (b *TrueAPIBackend) GetCommittee(number rpc.BlockNumber) (map[string]interface{}, error) {
	if number == rpc.LatestBlockNumber {
		return b.ups.election.GetCommitteeById(new(big.Int).SetUint64(b.ups.agent.CommitteeNumber())), nil
	}
	return b.ups.election.GetCommitteeById(big.NewInt(number.Int64())), nil
}

func (b *TrueAPIBackend) GetCurrentCommitteeNumber() *big.Int {
	return b.ups.election.GetCurrentCommitteeNumber()
}

// SendTx returns nil by success to add local txpool
func (b *TrueAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.ups.txPool.AddLocal(signedTx)
}

// GetPoolTransactions returns Transactions by pending state in txpool
func (b *TrueAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.ups.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

// GetPoolTransaction returns Transaction by txHash in txpool
func (b *TrueAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.ups.txPool.Get(hash)
}

// GetPoolNonce returns user nonce by user address in txpool
func (b *TrueAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.ups.txPool.State().GetNonce(addr), nil
}

// Stats returns the count tx in txpool
func (b *TrueAPIBackend) Stats() (pending int, queued int) {
	return b.ups.txPool.Stats()
}

func (b *TrueAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.ups.TxPool().Content()
}

// SubscribeNewTxsEvent returns the subscript event of new tx
func (b *TrueAPIBackend) SubscribeNewTxsEvent(ch chan<- types.NewTxsEvent) event.Subscription {
	return b.ups.TxPool().SubscribeNewTxsEvent(ch)
}

// Downloader returns the fast downloader
func (b *TrueAPIBackend) Downloader() *downloader.Downloader {
	return b.ups.Downloader()
}

// ProtocolVersion returns the version of protocol
func (b *TrueAPIBackend) ProtocolVersion() int {
	return b.ups.EthVersion()
}

// SuggestPrice returns tht suggest gas price
func (b *TrueAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

// ChainDb returns tht database of fastchain
func (b *TrueAPIBackend) ChainDb() upsdb.Database {
	return b.ups.ChainDb()
}

// EventMux returns Event locker
func (b *TrueAPIBackend) EventMux() *event.TypeMux {
	return b.ups.EventMux()
}

// AccountManager returns Account Manager
func (b *TrueAPIBackend) AccountManager() *accounts.Manager {
	return b.ups.AccountManager()
}

// BloomStatus returns Bloom Status
func (b *TrueAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.ups.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

// ServiceFilter make the Filter for the truechian
func (b *TrueAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.ups.bloomRequests)
	}
}
