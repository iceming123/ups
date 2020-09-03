// Copyright 2017 The go-ethereum Authors
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

// Package consensus implements different Ethereum consensus engines.
package consensus

import (
	"math/big"

	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/core/state"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/log"
	"github.com/iceming123/ups/params"
	"github.com/iceming123/ups/rpc"
	"github.com/iceming123/ups/core/vm"
)

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header and/or uncle verification.
type ChainReader interface {
	// Config retrieves the blockchain's chain configuration.
	Config() *params.ChainConfig

	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header

	// GetHeader retrieves a block header from the database by hash and number.
	GetHeader(hash common.Hash, number uint64) *types.Header

	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByNumber(number uint64) *types.Header

	// GetHeaderByHash retrieves a block header from the database by its hash.
	GetHeaderByHash(hash common.Hash) *types.Header

	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block
}

type RewardInfosAccess interface {
	GetRewardInfos(number uint64) *types.ChainReward
	SetRewardInfos(number uint64,infos *types.ChainReward) error
}


// Engine is an algorithm agnostic consensus engine.
type Engine interface {
	SetElection(e CommitteeElection)

	GetElection() CommitteeElection

	// Author retrieves the Ethereum address of the account that minted the given
	// block, which may be different from the header's coinbase if a consensus
	// engine is based on signatures.
	Author(header *types.Header) (common.Address, error)

	// VerifyHeader checks whether a header conforms to the consensus rules of a
	// given engine. Verifying the seal may be done optionally here, or explicitly
	// via the VerifySeal method.
	VerifyHeader(chain ChainReader, header *types.Header) error

	// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
	// concurrently. The method returns a quit channel to abort the operations and
	// a results channel to retrieve the async verifications (the order is that of
	// the input slice).
	VerifyHeaders(chain ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error)

	VerifySigns(fastnumber *big.Int, fastHash common.Hash, signs []*types.PbftSign) error

	VerifySwitchInfo(fastnumber *big.Int, info []*types.CommitteeMember) error

	// Prepare initializes the consensus fields of a block header according to the
	// rules of a particular engine. The changes are executed inline.
	Prepare(chain ChainReader, header *types.Header) error
	// Finalize runs any post-transaction state modifications (e.g. block rewards)
	// and assembles the final block.
	// Note: The block header and state database might be updated to reflect any
	// consensus rules that happen at finalization (e.g. block rewards).
	Finalize(chain ChainReader, header *types.Header, state *state.StateDB,txs []*types.Transaction,
		 receipts []*types.Receipt, feeAmount *big.Int) (*types.Block,*types.ChainReward, error)

	FinalizeCommittee(block *types.Block) error

	// APIs returns the RPC APIs this consensus engine provides.
	APIs(chain ChainReader) []rpc.API
}

//CommitteeElection module implementation committee interface
type CommitteeElection interface {
	// VerifySigns verify the fast chain committee signatures in batches
	VerifySigns(pvs []*types.PbftSign) ([]*types.CommitteeMember, []error)

	// VerifySwitchInfo verify committee members and it's state
	VerifySwitchInfo(fastnumber *big.Int, info []*types.CommitteeMember) error

	FinalizeCommittee(block *types.Block) error

	//Get a list of committee members
	//GetCommittee(FastNumber *big.Int, FastHash common.Hash) (*big.Int, []*types.CommitteeMember)
	GetCommittee(fastNumber *big.Int) []*types.CommitteeMember

	GenerateFakeSigns(fb *types.Block) ([]*types.PbftSign, error)
}

// PoW is a consensus engine based on proof-of-work.
type PoW interface {
	Engine
}

func InitDPos(config *params.ChainConfig) {
	params.FirstNewEpochID = common.Big1.Uint64()
	params.DposForkPoint = 0
}
func makeImpawInitState(config *params.ChainConfig,state *state.StateDB) bool {
	stateAddress := types.StakingAddress
	key := common.BytesToHash(stateAddress[:])
	obj := state.GetPOSState(stateAddress, key)
	if len(obj) == 0 {
		i := vm.NewImpawnImpl()
		i.Save(state,stateAddress)
		state.SetNonce(stateAddress,1)
		state.SetCode(stateAddress,stateAddress[:])
		log.Info("makeImpawInitState success")
		return true
	}			
	return true
}
func OnceInitImpawnState(config *params.ChainConfig,state *state.StateDB) bool {
	return makeImpawInitState(config,state)
}
