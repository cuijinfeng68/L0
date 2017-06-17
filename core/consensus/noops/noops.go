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

package noops

import (
	"time"

	"encoding/json"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/consensus"
)

// NewNoops Create Noops
func NewNoops(options *Options, stack consensus.IStack) *Noops {
	noops := &Noops{
		options: options,
		stack:   stack,
	}
	if noops.options.CommitTxChanSize < options.BlockSize {
		noops.options.CommitTxChanSize = options.BlockSize
	}
	noops.committedTxsChan = make(chan *consensus.CommittedTxs, noops.options.CommittedTxsChanSize)
	noops.broadcastChan = make(chan consensus.IBroadcast, noops.options.BroadcastChanSize)
	noops.blockTimer = time.NewTimer(noops.options.BlockInterval)
	noops.blockTimer.Stop()
	noops.seqNo = noops.stack.GetLastSeqNo()
	return noops
}

// Noops Define Noops
type Noops struct {
	options          *Options
	stack            consensus.IStack
	committedTxsChan chan *consensus.CommittedTxs
	broadcastChan    chan consensus.IBroadcast
	blockTimer       *time.Timer
	seqNo            uint64
	exit             chan struct{}
}

func (noops *Noops) String() string {
	bytes, _ := json.Marshal(noops.options)
	return string(bytes)
}

// IsRunning Noops consenter serverice already started
func (noops *Noops) IsRunning() bool {
	return noops.exit != nil
}

// Start Start consenter serverice of Noops
func (noops *Noops) Start() {
	if noops.IsRunning() {
		return
	}
	noops.exit = make(chan struct{})
	noops.blockTimer = time.NewTimer(noops.options.BlockInterval)
	for {
		select {
		case <-noops.exit:
			noops.exit = nil
			return
		case <-noops.blockTimer.C:
			noops.processBlock()
		}
	}
}

func (noops *Noops) processBlock() {
	noops.blockTimer.Stop()
	if noops.stack.Len() > 0 {
		txs := []consensus.ITransaction{}
		noops.stack.IterTransaction(func(tx consensus.ITransaction) bool {
			txs = append(txs, tx)
			if len(txs) == noops.options.BlockSize {
				return true
			}
			return false
		})
		txs = noops.stack.VerifyTxsInConsensus(txs, true)
		noops.seqNo++
		log.Infof("Noops write block (%d transactions)  %d", len(txs), noops.seqNo)
		seqNos := []uint64{noops.seqNo}
		noops.committedTxsChan <- &consensus.CommittedTxs{Time: uint32(time.Now().Unix()), Transactions: txs, SeqNos: seqNos}
		noops.stack.Removes(txs)
	}
	noops.blockTimer = time.NewTimer(noops.options.BlockInterval)
}

// Stop Stop consenter serverice of Noops
func (noops *Noops) Stop() {
	if noops.IsRunning() {
		close(noops.exit)
	}
}

// RecvConsensus Receive consensus data
func (noops *Noops) RecvConsensus(payload []byte) {
	//noops.broadcastChan<-
}

// BroadcastTransactionChannel Broadcast consensus data
func (noops *Noops) BroadcastTransactionChannel() <-chan consensus.ITransaction {
	return nil
}

// BroadcastConsensusChannel Broadcast consensus data
func (noops *Noops) BroadcastConsensusChannel() <-chan consensus.IBroadcast {
	return noops.broadcastChan
}

// CommittedTxsChannel Commit block data
func (noops *Noops) CommittedTxsChannel() <-chan *consensus.CommittedTxs {
	return noops.committedTxsChan
}
