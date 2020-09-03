// Copyright 2014 The go-ethereum Authors
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

// Package ups implements the Upschain protocol.
package ups

import (
	"errors"
	"fmt"
	"github.com/iceming123/ups/consensus/tbft"
	config "github.com/iceming123/ups/params"
	"math/big"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"github.com/iceming123/ups/accounts"
	"github.com/iceming123/ups/common/hexutil"
	"github.com/iceming123/ups/consensus"
	elect "github.com/iceming123/ups/consensus/election"
	ethash "github.com/iceming123/ups/consensus/minerva"
	"github.com/iceming123/ups/core"
	"github.com/iceming123/ups/core/bloombits"
	"github.com/iceming123/ups/core/rawdb"
	"github.com/iceming123/ups/core/types"
	"github.com/iceming123/ups/core/vm"
	"github.com/iceming123/ups/crypto"
	"github.com/iceming123/ups/ups/downloader"
	"github.com/iceming123/ups/ups/filters"
	"github.com/iceming123/ups/ups/gasprice"
	"github.com/iceming123/ups/upsdb"
	"github.com/iceming123/ups/event"
	"github.com/iceming123/ups/internal/upsapi"
	"github.com/iceming123/ups/log"
	"github.com/iceming123/ups/node"
	"github.com/iceming123/ups/p2p"
	"github.com/iceming123/ups/params"
	"github.com/iceming123/ups/rlp"
	"github.com/iceming123/ups/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Upschain implements the Upschain full node service.
type Upschain struct {
	config      *Config
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Upschain
	// Handlers
	txPool *core.TxPool
	agent    *PbftAgent
	election *elect.Election
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer
	// DB interfaces
	chainDb upsdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *TrueAPIBackend
	gasPrice  *big.Int
	networkID     uint64
	netRPCService *upsapi.PublicNetAPI

	pbftServer *tbft.Node

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price)
}

func (s *Upschain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Upschain object (including the
// initialisation of the common Upschain object)
func New(ctx *node.ServiceContext, config *Config) (*Upschain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run ups.Upschain in light sync mode, use les.LightTruechain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	//chainDb, err := CreateDB(ctx, config, path)
	if err != nil {
		return nil, err
	}

	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	/*if config.Genesis != nil {
		config.MinerGasFloor = config.Genesis.GasLimit * 9 / 10
		config.MinerGasCeil = config.Genesis.GasLimit * 11 / 10
	}*/

	ups := &Upschain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &ethash.Config{PowMode:ethash.ModeNormal}, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.GasPrice,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}

	log.Info("Initialising Upschain protocol", "versions", ProtocolVersions, "network", config.NetworkId, "syncmode", config.SyncMode)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gups upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Deleted: config.DeletedState, Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)

	ups.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, ups.chainConfig, ups.engine, vmConfig)
	if err != nil {
		return nil, err
	}

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		ups.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	ups.bloomIndexer.Start(ups.blockchain)
	consensus.InitDPos(chainConfig)
	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	ups.txPool = core.NewTxPool(config.TxPool, ups.chainConfig, ups.blockchain)
	ups.election = elect.NewElection(ups.chainConfig, ups.blockchain, ups.config)
	checkpoint := config.Checkpoint
	cacheLimit := cacheConfig.TrieCleanLimit

	ups.engine.SetElection(ups.election)
	ups.election.SetEngine(ups.engine)

	ups.agent = NewPbftAgent(ups, ups.chainConfig, ups.engine, ups.election, config.MinerGasFloor, config.MinerGasCeil)

	if ups.protocolManager, err = NewProtocolManager(
		ups.chainConfig,checkpoint, config.SyncMode, config.NetworkId,
		ups.eventMux, ups.txPool, ups.engine,
		ups.blockchain, chainDb, ups.agent,cacheLimit,config.Whitelist); err != nil {
		return nil, err
	}

	ups.APIBackend = &TrueAPIBackend{ups, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	ups.APIBackend.gpo = gasprice.NewOracle(ups.APIBackend, gpoParams)
	return ups, nil
}
func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gups",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (upsdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*upsdb.LDBDatabase); ok {
		db.Meter("ups/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Upschain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config,db upsdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	// snail chain not need clique
	/*
		if chainConfig.Clique != nil {
			return clique.New(chainConfig.Clique, db)
		}*/
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case ethash.ModeFake:
		log.Info("-----Fake mode")
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case ethash.ModeTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case ethash.ModeShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the ups package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Upschain) APIs() []rpc.API {
	apis := upsapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append ups	APIs and  Eth APIs
	namespaces := []string{"ups", "eth"}
	for _, name := range namespaces {
		apis = append(apis, []rpc.API{
			{
				Namespace: name,
				Version:   "1.0",
				Service:   NewPublicTruechainAPI(s),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
				Public:    true,
			},
		}...)
	}
	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Upschain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Upschain) ResetWithFastGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}
func (s *Upschain) PbftAgent() *PbftAgent             { return s.agent }
func (s *Upschain) AccountManager() *accounts.Manager { return s.accountManager }
func (s *Upschain) BlockChain() *core.BlockChain      { return s.blockchain }
func (s *Upschain) Config() *Config                   { return s.config }
func (s *Upschain) TxPool() *core.TxPool                    { return s.txPool }

func (s *Upschain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Upschain) Engine() consensus.Engine           { return s.engine }
func (s *Upschain) ChainDb() upsdb.Database          { return s.chainDb }
func (s *Upschain) IsListening() bool                  { return true } // Always listening
func (s *Upschain) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Upschain) NetVersion() uint64                 { return s.networkID }
func (s *Upschain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Upschain) Synced() bool                       { return atomic.LoadUint32(&s.protocolManager.acceptTxs) == 1 }
func (s *Upschain) ArchiveMode() bool                  { return s.config.NoPruning }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Upschain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Upschain protocol implementation.
func (s *Upschain) Start(srvr *p2p.Server) error {

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = upsapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers

	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	s.startPbftServer()
	if s.pbftServer == nil {
		log.Error("start pbft server failed.")
		return errors.New("start pbft server failed.")
	}
	s.agent.server = s.pbftServer
	log.Info("", "server", s.agent.server)
	s.agent.Start()

	s.election.Start()
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Upschain protocol.
func (s *Upschain) Stop() error {
	s.stopPbftServer()
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}

func (s *Upschain) startPbftServer() error {
	priv, err := crypto.ToECDSA(s.config.CommitteeKey)
	if err != nil {
		return err
	}

	cfg := config.DefaultConfig()
	cfg.P2P.ListenAddress1 = "tcp://0.0.0.0:" + strconv.Itoa(s.config.Port)
	cfg.P2P.ListenAddress2 = "tcp://0.0.0.0:" + strconv.Itoa(s.config.StandbyPort)

	n1, err := tbft.NewNode(cfg, "1", priv, s.agent)
	if err != nil {
		return err
	}
	s.pbftServer = n1
	return n1.Start()
}

func (s *Upschain) stopPbftServer() error {
	if s.pbftServer != nil {
		s.pbftServer.Stop()
	}
	return nil
}
