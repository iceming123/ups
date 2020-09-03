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

package minerva

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/consensus"
	"github.com/iceming123/ups/core/state"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/core/vm"
	"github.com/iceming123/ups/log"
	"github.com/iceming123/ups/params"
)

// Minerva protocol constants.
var (
	allowedFutureBlockTime = 15 * time.Second // Max time from current time allowed for blocks, before they're considered future blocks
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
	errInvalidFast       = errors.New("invalid fast number")
	//ErrRewardedBlock is returned if a block to import is already rewarded.
	ErrRewardedBlock = errors.New("block already rewarded")
	ErrRewardEnd     = errors.New("Reward end")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (m *Minerva) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}
// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Upschain m engine.
func (m *Minerva) VerifyHeader(chain consensus.ChainReader, header *types.Header) error {
	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}

	if chain.GetHeaderByNumber(number) != nil {
		return consensus.ErrForkFastBlock
	}

	return m.verifyHeader(chain, header, parent)
}
// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (m *Minerva) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header,
	seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if m.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = m.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (m *Minerva) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header,
	seals []bool, index int) error {
	var parent *types.Header

	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}

	return m.verifyHeader(chain, headers[index], parent)
	//return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Upschain minerva engine.
func (m *Minerva) verifyHeader(chain consensus.ChainReader, header, parent *types.Header) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if header.Time.Cmp(big.NewInt(time.Now().Add(allowedFutureBlockTime).Unix())) > 0 {
		fmt.Println(consensus.ErrFutureBlock.Error(), "header", header.Time, "now", time.Now().Unix(),
			"cmp:", big.NewInt(time.Now().Add(allowedFutureBlockTime).Unix()))
		return consensus.ErrFutureBlock
	}

	if header.Time.Cmp(parent.Time) < 0 {
		return errZeroBlockTime
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}

	return nil
}

// VerifySigns check the sings included in fast block or fruit
func (m *Minerva) VerifySigns(fastnumber *big.Int, fastHash common.Hash, signs []*types.PbftSign) error {
	// validate the signatures of this fruit
	ms := make(map[common.Address]uint)
	members := m.election.GetCommittee(fastnumber)
	if members == nil {
		log.Warn("VerifySigns get committee failed.", "number", fastnumber)
		return consensus.ErrInvalidSign
	}
	for _, member := range members {
		addr := member.CommitteeBase
		ms[addr] = 0
	}

	count := 0
	for _, sign := range signs {
		if sign.FastHash != fastHash || sign.FastHeight.Cmp(fastnumber) != 0 {
			log.Warn("VerifySigns signs hash error", "number", fastnumber, "hash", fastHash, "signHash", sign.FastHash, "signNumber", sign.FastHeight)
			return consensus.ErrInvalidSign
		}
		if sign.Result == types.VoteAgree {
			count++
		}
	}
	if count <= len(members)*2/3 {
		log.Warn("VerifySigns number error", "signs", len(signs), "agree", count, "members", len(members))
		return consensus.ErrInvalidSign
	}

	signMembers, errs := m.election.VerifySigns(signs)
	for i, err := range errs {
		if err != nil {
			log.Warn("VerifySigns error", "err", err)
			return err
		}
		addr := signMembers[i].CommitteeBase
		if _, ok := ms[addr]; !ok {
			// is not a committee member
			log.Warn("VerifySigns member error", "signs", len(signs), "member", hex.EncodeToString(members[i].Publickey))
			return consensus.ErrInvalidSign
		}
		if ms[addr] == 1 {
			// the committee member's sign is already exist
			log.Warn("VerifySigns member already exist", "signs", len(signs), "member", hex.EncodeToString(members[i].Publickey))
			return consensus.ErrInvalidSign
		}
		ms[addr] = 1
	}

	return nil
}

//VerifySwitchInfo verify the switch info of Committee
func (m *Minerva) VerifySwitchInfo(fastnumber *big.Int, info []*types.CommitteeMember) error {

	return m.election.VerifySwitchInfo(fastnumber, info)

}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big8          = big.NewInt(8)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	big32         = big.NewInt(32)

	big90 = big.NewInt(90)

	bigMinus1  = big.NewInt(-1)
	bigMinus99 = big.NewInt(-99)
	big2999999 = big.NewInt(2999999)
)

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the minerva protocol. The changes are done inline.
func (m *Minerva) Prepare(chain consensus.ChainReader, header *types.Header) error {
	if parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1); parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return nil
}

// Finalize implements consensus.Engine, accumulating the block fruit and uncle rewards,
// setting the final state and assembling the block.
func (m *Minerva) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB,
	txs []*types.Transaction, receipts []*types.Receipt, feeAmount *big.Int) (*types.Block, *types.ChainReward,error) {

	infos,err := accumulateRewardsFast(state, header.Number.Uint64(),header.Time.Uint64())
	if err != nil {
		log.Error("Finalize Error", "accumulateRewardsFast", err.Error())
		return nil,nil, err
	}
	if err := m.finalizeFastGas(state, header.Number, header.Hash(), feeAmount); err != nil {
		return nil,nil, err
	}

	if err := m.finalizeValidators(chain, state, header.Number); err != nil {
		return nil,nil, err
	}
	header.Root = state.IntermediateRoot(true)
	return types.NewBlock(header, txs, receipts, nil, nil),infos, nil
}

// FinalizeCommittee upddate current committee state
func (m *Minerva) FinalizeCommittee(block *types.Block) error {
	return m.election.FinalizeCommittee(block)
}

// gas allocation
func (m *Minerva) finalizeFastGas(state *state.StateDB, fastNumber *big.Int, fastHash common.Hash, feeAmount *big.Int) error {
	if feeAmount == nil || feeAmount.Uint64() == 0 {
		return nil
	}
	committee := m.election.GetCommittee(fastNumber)
	committeeGas := big.NewInt(0)
	if len(committee) == 0 {
		return errors.New("not have committee")
	}
	committeeGas = new(big.Int).Div(feeAmount, big.NewInt(int64(len(committee))))
	for _, v := range committee {
		state.AddBalance(v.Coinbase, committeeGas)
		LogPrint("committee's gas award", v.Coinbase, committeeGas)
	}
	return nil
}

// gas allocation
func (m *Minerva) finalizeValidators(chain consensus.ChainReader, state *state.StateDB, fastNumber *big.Int) error {

	next := new(big.Int).Add(fastNumber, big1)
	// init the first epoch in the fork
	first := types.GetFirstEpoch()
	// fmt.Println("first.BeginHeight", first.BeginHeight, "next", next)
	if first.BeginHeight == next.Uint64() {
		i := vm.NewImpawnImpl()
		error := i.Load(state, types.StakingAddress)
		if es, err := i.DoElections(first.EpochID, next.Uint64()); err != nil {
			return err
		} else {
			log.Info("init in first forked, Do pre election", "height", next, "epoch:", first.EpochID, "len:", len(es), "err", error)
		}
		if err := i.Shift(first.EpochID); err != nil {
			return err
		}
		i.Save(state, types.StakingAddress)
		log.Info("init in first forked,", "height", next, "epoch:", first.EpochID)
	}
	epoch := types.GetEpochFromHeight(fastNumber.Uint64())
	if fastNumber.Uint64() == epoch.EndHeight-params.ElectionPoint {
		i := vm.NewImpawnImpl()
		error := i.Load(state, types.StakingAddress)
		if es, err := i.DoElections(epoch.EpochID+1, fastNumber.Uint64()); err != nil {
			return err
		} else {
			log.Info("Do validators election", "height", fastNumber, "epoch:", epoch.EpochID+1, "len:", len(es), "err", error)
		}
		i.Save(state, types.StakingAddress)
	}

	if fastNumber.Uint64() == epoch.EndHeight {
		i := vm.NewImpawnImpl()
		err := i.Load(state, types.StakingAddress)
		log.Info("Force new epoch", "height", fastNumber, "err", err)
		if err := i.Shift(epoch.EpochID + 1); err != nil {
			return err
		}
		i.Save(state, types.StakingAddress)
	}
		
	return nil
}

//LogPrint log debug
func LogPrint(info string, addr common.Address, amount *big.Int) {
	log.Debug("[Consensus AddBalance]", "info", info, "CoinBase:", addr.String(), "amount", amount)
}

func accumulateRewardsFast(stateDB *state.StateDB, fast,btime uint64) (*types.ChainReward,error) {
	committeeCoin := GetBlockReward(fast)
	
	impawn := vm.NewImpawnImpl()
	impawn.Load(stateDB, types.StakingAddress)
	defer impawn.Save(stateDB, types.StakingAddress)
	//committee reward
	infos, err := impawn.Reward(fast, committeeCoin)
	if err != nil {
		return nil,err
	}
	for _, v := range infos {
		for _, vv := range v.Items {
			stateDB.AddBalance(vv.Address, vv.Amount)
			LogPrint("committee:", vv.Address, vv.Amount)
		}
	}
	rewardsInfos := types.NewChainReward(fast,btime,infos)
	return rewardsInfos,nil
}
//GetBlockReward Reward for block allocation
func GetBlockReward(num uint64) *big.Int {
	return new(big.Int).Set(Big1e16)
}
