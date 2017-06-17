// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package lcnd

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"

	"runtime"

	"syscall"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/config"
	"github.com/bocheninc/L0/core/accounts/keystore"
	"github.com/bocheninc/L0/core/blockchain"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/consensus/consenter"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/merge"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/node"
)

// Lcnd represents the blockchain l0
type Lcnd struct {
	*config.Config
	mu              sync.Mutex
	bc              *blockchain.Blockchain
	protocolManager *node.ProtocolManager
	consenter       consensus.Consenter
	wg              sync.WaitGroup
}

// NewLcnd returns l0 daemon instance
func NewLcnd(cfgFile string) *Lcnd {
	var (
		lcnd    Lcnd
		chainDb *db.BlockchainDB

		newLedger *ledger.Ledger
		bc        *blockchain.Blockchain
		ks        *keystore.KeyStore

		netConfig   *p2p.Config
		mergeConfig *merge.Config
		cfg         *config.Config
		err         error
	)

	if cfg, err = config.New(cfgFile); err != nil {
		log.Errorf("loadConfig error %v", err)
		return nil
	}
	lcnd.Config = cfg

	lcnd.initLog()

	netConfig = cfg.NetConfig

	mergeConfig = cfg.MergeConfig

	chainDb = db.NewDB(cfg.DbConfig)

	newLedger = ledger.NewLedger(chainDb)
	bc = blockchain.NewBlockchain(newLedger)
	consenter := consenter.NewConsenter(config.ConsenterOptions(), bc)
	ks = keystore.NewPlaintextKeyStore(chainDb, cfg.KeyStoreDir)
	lcnd.protocolManager = node.NewProtocolManager(chainDb, netConfig, bc, consenter, newLedger, ks, mergeConfig, cfg.LogDir)

	bc.SetBlockchainConsenter(consenter)
	bc.SetNetworkStack(lcnd.protocolManager)

	lcnd.bc = bc

	lcnd.consenter = consenter
	lcnd.wg = sync.WaitGroup{}

	return &lcnd
}

// Start starts the blockchain service
func (l *Lcnd) Start() {
	if l.Config.CPUFile != "" {
		startPProf(l.Config.CPUFile, l.Config.CPUFile+".mem")
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	l.bc.Start()
	l.protocolManager.Start()

	// TODO: every service start here, and make waitgroup usefull
	l.wg.Add(1)
	l.wg.Wait()
}

func startPProf(cpuFile, memFile string) {
	cpuProfile, _ := os.Create(cpuFile)

	pprof.StartCPUProfile(cpuProfile)

	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-abort
		pprof.StopCPUProfile()

		memProfile, _ := os.Create(memFile)
		pprof.WriteHeapProfile(memProfile)
		memProfile.Close()
		cpuProfile.Close()
		os.Exit(0)
	}()

}

func (l *Lcnd) initLog() {
	log.New(l.Config.LogFile)
	log.SetLevel(l.Config.LogLevel)
}
