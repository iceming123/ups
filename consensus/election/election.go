// Copyright 2018 The UpsChain Authors
// This file is part of the ups library.
//
// The ups library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ups library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ups library. If not, see <http://www.gnu.org/licenses/>.

package election

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/consensus"
	"github.com/iceming123/ups/core/state"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/core/vm"
	"github.com/iceming123/ups/crypto"
	"github.com/iceming123/ups/event"
	"github.com/iceming123/ups/log"
	"github.com/iceming123/ups/params"
)

const (
	snailchainHeadSize  = 64
	committeeCacheLimit = 256
)

type ElectMode uint

const (
	// ElectModeEtrue for ups
	ElectModeEtrue = iota
	// ElectModeFake for Test purpose
	ElectModeFake
)

var (
	// maxUint256 is a big integer representing 2^256-1
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

var (
	ErrCommittee     = errors.New("get committee failed")
	ErrInvalidMember = errors.New("invalid committee member")
	ErrInvalidSwitch = errors.New("invalid switch block info")
)

type committee struct {
	id                  *big.Int
	beginFastNumber     *big.Int // the first fast block proposed by this committee
	endFastNumber       *big.Int // the lastproposed by this committee
	members             types.CommitteeMembers
	backupMembers       types.CommitteeMembers
	switches            []*big.Int // blocknumbers whose block include switchinfos
}

// Members returns dump of the committee members
func (c *committee) Members() []*types.CommitteeMember {
	members := make([]*types.CommitteeMember, len(c.members))
	copy(members, c.members)
	return members
}

// Members returns dump of the backup committee members
func (c *committee) BackupMembers() []*types.CommitteeMember {
	members := make([]*types.CommitteeMember, len(c.backupMembers))
	copy(members, c.backupMembers)
	return members
}

func (c *committee) setMemberState(pubkey []byte, flag uint32) {
	for i, m := range c.members {
		if bytes.Equal(m.Publickey, pubkey) {
			c.members[i] = &types.CommitteeMember{
				Coinbase:  m.Coinbase,
				Publickey: m.Publickey,
				Flag:      flag,
			}
			break
		}
	}
	for i, m := range c.backupMembers {
		if bytes.Equal(m.Publickey, pubkey) {
			c.backupMembers[i] = &types.CommitteeMember{
				Coinbase:  m.Coinbase,
				Publickey: m.Publickey,
				Flag:      flag,
			}
			break
		}
	}
}

type Election struct {
	chainConfig *params.ChainConfig

	genesisCommittee []*types.CommitteeMember
	defaultMembers   []*types.CommitteeMember

	commiteeCache *lru.Cache
	epochCache    *lru.Cache

	electionMode    ElectMode
	committee       *committee
	mu              sync.RWMutex
	testPrivateKeys []*ecdsa.PrivateKey

	startSwitchover bool //Flag bit for handling event switching
	singleNode      bool

	electionFeed event.Feed
	scope        event.SubscriptionScope

	fastchain  BlockChain
	engine consensus.Engine
}

type BlockChain interface {
	CurrentBlock() *types.Block

	CurrentHeader() *types.Header

	GetBlockByNumber(number uint64) *types.Block

	StateAt(root common.Hash) (*state.StateDB, error)
}


type Config interface {
	GetNodeType() bool
}

// NewElection create election processor and load genesis committee
func NewElection(chainConfig *params.ChainConfig, fastBlockChain BlockChain, config Config) *Election {
	// init
	election := &Election{
		chainConfig:       chainConfig,
		fastchain:         fastBlockChain,
		singleNode:        config.GetNodeType(),
		electionMode:      ElectModeEtrue,
	}

	// get genesis committee
	election.genesisCommittee = election.getGenesisCommittee()
	if len(election.genesisCommittee) == 0 {
		log.Error("Election creation get no genesis committee members")
	}

	election.commiteeCache, _ = lru.New(committeeCacheLimit)
	election.epochCache, _ = lru.New(committeeCacheLimit)

	if election.singleNode {
		committeeMember := election.getGenesisCommittee()
		if committeeMember == nil {
			log.Error("genesis block committee member is nil.")
		}
		election.genesisCommittee = election.getGenesisCommittee()[:1]
	}
	if !election.singleNode && len(election.genesisCommittee) < 4 {
		log.Error("Election creation get insufficient genesis committee members")
	}
	for _, m := range election.genesisCommittee {
		var member = *m
		member.Flag = types.StateUnusedFlag
		election.defaultMembers = append(election.defaultMembers, &member)
	}

	return election
}

func NewLightElection(fastBlockChain BlockChain) *Election {
	// init
	election := &Election{
		fastchain:    fastBlockChain,
		electionMode: ElectModeEtrue,
	}
	return election
}

// NewFakeElection create fake mode election only for testing
func NewFakeElection() *Election {
	var priKeys []*ecdsa.PrivateKey
	var members []*types.CommitteeMember

	for i := 0; i < params.MinimumCommitteeNumber; i++ {
		priKey, err := crypto.GenerateKey()
		priKeys = append(priKeys, priKey)
		if err != nil {
			log.Error("initMembers", "error", err)
		}
		coinbase := crypto.PubkeyToAddress(priKey.PublicKey)
		m := &types.CommitteeMember{Coinbase: coinbase, CommitteeBase: coinbase, Publickey: crypto.FromECDSAPub(&priKey.PublicKey), Flag: types.StateUsedFlag, MType: types.TypeFixed}
		members = append(members, m)
	}

	// Backup members are empty in FakeMode Election
	elected := &committee{
		id:                  new(big.Int).Set(common.Big0),
		beginFastNumber:     new(big.Int).Set(common.Big1),
		endFastNumber:       new(big.Int).Set(common.Big0),
		members:             members,
	}

	election := &Election{
		fastchain:         nil,
		singleNode:        false,
		committee:         elected,
		electionMode:      ElectModeFake,
		testPrivateKeys:   priKeys,
	}
	return election
}

func (e *Election) GenerateFakeSigns(fb *types.Block) ([]*types.PbftSign, error) {
	var signs []*types.PbftSign
	for _, privateKey := range e.testPrivateKeys {
		voteSign := &types.PbftSign{
			Result:     types.VoteAgree,
			FastHeight: fb.Header().Number,
			FastHash:   fb.Hash(),
		}
		var err error
		signHash := voteSign.HashWithNoSign().Bytes()
		voteSign.Sign, err = crypto.Sign(signHash, privateKey)
		if err != nil {
			log.Error("fb GenerateSign error ", "err", err)
		}
		signs = append(signs, voteSign)
	}
	return signs, nil
}

func (e *Election) GetGenesisCommittee() []*types.CommitteeMember {
	return e.genesisCommittee
}

func (e *Election) GetCurrentCommittee() *committee {
	return e.committee
}

func (e *Election) GetCurrentCommitteeNumber() *big.Int {
	return e.committee.id
}

// GetMemberByPubkey returns committeeMember specified by public key bytes
func (e *Election) GetMemberByPubkey(members []*types.CommitteeMember, publickey []byte) *types.CommitteeMember {
	if len(members) == 0 {
		log.Error("GetMemberByPubkey method len(members)= 0")
		return nil
	}
	for _, member := range members {
		if bytes.Equal(publickey, member.Publickey) {
			return member
		}
	}
	return nil
}

// IsCommitteeMember reports whether the provided public key is in committee
func (e *Election) GetMemberFlag(members []*types.CommitteeMember, publickey []byte) uint32 {
	if len(members) == 0 {
		log.Error("IsCommitteeMember method len(members)= 0")
		return 0
	}
	for _, member := range members {
		if bytes.Equal(publickey, member.Publickey) {
			return member.Flag
		}
	}
	return 0
}

func (e *Election) IsCommitteeMember(members []*types.CommitteeMember, publickey []byte) bool {
	flag := e.GetMemberFlag(members, publickey)
	return flag == types.StateUsedFlag
}

// VerifyPublicKey get the committee member by public key
func (e *Election) VerifyPublicKey(fastHeight *big.Int, pubKeyByte []byte) (*types.CommitteeMember, error) {
	members := e.GetCommittee(fastHeight)
	if members == nil {
		log.Info("GetCommittee members is nil", "fastHeight", fastHeight)
		return nil, ErrCommittee
	}
	member := e.GetMemberByPubkey(members, pubKeyByte)
	/*if member == nil {
		return nil, ErrInvalidMember
	}*/
	return member, nil
}

// VerifySign lookup the pbft sign and return the committee member who signs it
func (e *Election) VerifySign(sign *types.PbftSign) (*types.CommitteeMember, error) {
	pubkey, err := crypto.SigToPub(sign.HashWithNoSign().Bytes(), sign.Sign)
	if err != nil {
		return nil, err
	}
	pubkeyByte := crypto.FromECDSAPub(pubkey)
	member, err := e.VerifyPublicKey(sign.FastHeight, pubkeyByte)
	return member, err
}

// VerifySigns verify signatures of bft committee in batches
func (e *Election) VerifySigns(signs []*types.PbftSign) ([]*types.CommitteeMember, []error) {
	members := make([]*types.CommitteeMember, len(signs))
	errs := make([]error, len(signs))

	if len(signs) == 0 {
		log.Warn("Veriry signs get nil pbftsigns")
		return nil, nil
	}
	// All signs should have the same fastblock height
	committeeMembers := e.GetCommittee(signs[0].FastHeight)
	if len(committeeMembers) == 0 {
		log.Error("Election get none committee for verify pbft signs")
		for i := range errs {
			errs[i] = ErrCommittee
		}
		return members, errs
	}

	for i, sign := range signs {
		// member, err := e.VerifySign(sign)
		pubkey, _ := crypto.SigToPub(sign.HashWithNoSign().Bytes(), sign.Sign)
		member := e.GetMemberByPubkey(committeeMembers, crypto.FromECDSAPub(pubkey))
		if member == nil {
			errs[i] = ErrInvalidMember
		} else {
			members[i] = member
		}
	}
	return members, errs
}

// VerifySwitchInfo verify committee members and it's state
func (e *Election) VerifySwitchInfo(fastNumber *big.Int, info []*types.CommitteeMember) error {
	if e.singleNode == true {
		return nil
	}
	begin, members := e.getMembers(fastNumber)
	if begin == nil || members == nil {
		log.Error("Failed to fetch elected committee", "fast", fastNumber)
		return ErrCommittee
	}
	if begin.Cmp(fastNumber) == 0 && len(members) == len(info) {
		for i := range info {
			if !info[i].Compared(members[i]) {
				log.Error("SwitchInfo members invalid", "num", fastNumber)
				return ErrInvalidSwitch
			}
		}
	}

	return nil
}

func (e *Election) getGenesisCommittee() []*types.CommitteeMember {
	block := e.fastchain.GetBlockByNumber(0)
	if block != nil {
		return block.SwitchInfos()
	}
	return nil
}

func (e *Election) getValidators(fastNumber *big.Int) []*types.CommitteeMember {
	epoch := types.GetEpochFromHeight(fastNumber.Uint64())
	current := e.fastchain.CurrentBlock().Number()

	if cache, ok := e.epochCache.Get(epoch.EpochID); ok {
		members := cache.(*[]*types.CommitteeMember)
		return *members
	}

	if current.Cmp(fastNumber) > 0 {
		// Read committee from block body
		block := e.fastchain.GetBlockByNumber(epoch.BeginHeight)
		if block != nil {
			var (
				members []*types.CommitteeMember
				backups []*types.CommitteeMember
			)
			for _, m := range e.fastchain.GetBlockByNumber(epoch.BeginHeight).SwitchInfos() {
				if m.Flag == types.StateUsedFlag {
					members = append(members, m)
				}
				if m.Flag == types.StateUnusedFlag {
					backups = append(backups, m)
				}
			}
			committee := &types.ElectionCommittee{Members: members, Backups: backups}
			// cache validators by epoch
			e.epochCache.Add(epoch.EpochID, &committee.Members)
			return committee.Members
		}
	}
	log.Info("getValidators in state", "number", fastNumber, "current", current)
	block := e.fastchain.CurrentBlock()
	stateDb, err := e.fastchain.StateAt(block.Root())
	if err != nil {
		log.Warn("Fetch committee from state failed", "number", fastNumber, "err", err)
		return nil
	}
	validators := vm.GetValidatorsByEpoch(stateDb, epoch.EpochID, fastNumber.Uint64())
	if len(validators) > 0 {
		e.epochCache.Add(epoch.EpochID, &validators)
	}
	return validators
}

// GetCommittee gets committee members propose this fast block
func (e *Election) GetCommittee(fastNumber *big.Int) []*types.CommitteeMember {
	return e.getValidators(fastNumber)
}

// GetCommitteeById return committee info sepecified by Committee ID
func (e *Election) GetCommitteeById(id *big.Int) map[string]interface{} {
	info := make(map[string]interface{})
	epoch := types.GetEpochFromID(id.Uint64())
	members := e.getValidators(big.NewInt(int64(epoch.BeginHeight)))
	if id.Cmp(common.Big0) <= 0 && members == nil {
		members = e.genesisCommittee
	} else if members == nil {
		log.Error("GetCommitteeById failed", "epoch", epoch)
		return nil
	}
	info["id"] = id.Uint64()
	info["memberCount"] = len(members)
	info["members"] = membersDisplay(members)
	info["beginNumber"] = epoch.BeginHeight
	info["endNumber"] = epoch.EndHeight
	return info
}
func (e *Election) getMembers(fastNumber *big.Int) (*big.Int, []*types.CommitteeMember) {
	epoch := types.GetEpochFromHeight(fastNumber.Uint64())
	return new(big.Int).SetUint64(epoch.BeginHeight), e.getValidators(fastNumber)
}
func membersDisplay(members []*types.CommitteeMember) []map[string]interface{} {
	var attrs []map[string]interface{}
	for _, member := range members {
		attrs = append(attrs, map[string]interface{}{
			"coinbase": member.Coinbase,
			"PKey":     hex.EncodeToString(member.Publickey),
			"flag":     member.Flag,
			"type":     member.MType,
		})
	}
	return attrs
}

// FinalizeCommittee upddate current committee state
func (e *Election) FinalizeCommittee(block *types.Block) error {
	return nil
}

// Start load current committ and starts election processing
func (e *Election) Start() error {
	log.Info("Election enable stake at launch")
	return nil
}
// SubscribeElectionEvent adds a channel to feed on committee change event
func (e *Election) SubscribeElectionEvent(ch chan<- types.ElectionEvent) event.Subscription {
	return e.scope.Track(e.electionFeed.Subscribe(ch))
}

// SetEngine set election backend consesus
func (e *Election) SetEngine(engine consensus.Engine) {
	e.engine = engine
}

func printCommittee(c *committee) {
	log.Info("Committee Info", "ID", c.id, "count", len(c.members), "start", c.beginFastNumber)
	for _, member := range c.members {
		log.Info("Committee member: ", "PKey", hex.EncodeToString(member.Publickey), "coinbase", member.Coinbase)
	}
	for _, member := range c.backupMembers {
		log.Info("Committee backup: ", "PKey", hex.EncodeToString(member.Publickey), "coinbase", member.Coinbase)
	}
}
