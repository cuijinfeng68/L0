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

package lbft

import (
	"encoding/json"
	"time"

	"sync"

	"sync/atomic"

	"sort"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils/vote"
	"github.com/bocheninc/L0/core/consensus"
)

//MINQUORUM  Define min quorum
const MINQUORUM = 3

//EMPTYBLOCK empty block id
const EMPTYBLOCK = 1136160000

//NewLbft Create lbft consenter
func NewLbft(options *Options, stack consensus.IStack) *Lbft {
	lbft := &Lbft{
		priority: time.Now().UnixNano(),
		options:  options,
		stack:    stack,
		committedRequestBatch: make(map[uint64]*RequestBatch),
		lbftCoreChan:          make(chan string, options.BufferSize),
		lbftCoreCommittedChan: make(chan *Committed, options.BufferSize),
		lbftCores:             make(map[string]*lbftCore),
		voteViewChange:        vote.NewVote(),
		voteCommitted:         make(map[string]*vote.Vote),

		committedRequestBatchChan: make(chan *committedRequestBatch, options.BufferSize),
		recvConsensusMsgChan:      make(chan *Message, options.BufferSize),
		committedTxsChan:          make(chan *consensus.CommittedTxs, options.BufferSize),
		broadcastChan:             make(chan consensus.IBroadcast, options.BufferSize),
		pool:                      &sync.Pool{New: func() interface{} { return stack.NewTransaction() }},
	}

	if lbft.options.BlockTimeout >= lbft.options.BlockInterval {
		log.Warn("lbft.blockTimeout should is smaller lbft.blockInterval")
		lbft.options.BlockTimeout = 4 * lbft.options.BlockInterval / 5
	}

	if lbft.options.ViewChangePeriod > 0*time.Second && lbft.options.ViewChangePeriod <= lbft.options.BlockInterval {
		log.Warn("lbft.ViewChangePeriod should is greater lbft.blockInterval")
		lbft.options.ViewChangePeriod = 1000 * lbft.options.BlockInterval
	}

	if lbft.options.N < 4 {
		log.Panicf("lbft.N should is greater 3, %d", lbft.options.N)
	}

	if 3*lbft.options.Q+1 < lbft.options.N*2 {
		q := (lbft.options.N*2-1)/3 + 1
		log.Warn("lbft.Q should is greater %d", q)
		lbft.options.Q = q
	}

	lbft.blockTimer = time.NewTimer(lbft.options.BlockInterval)
	lbft.blockTimer.Stop()
	lbft.emptyBlockTimer = time.NewTimer(lbft.options.BlockInterval)
	lbft.emptyBlockTimer.Stop()
	lbft.viewChangeTimer = time.NewTimer(lbft.options.ViewChange)
	lbft.viewChangeTimer.Stop()
	lbft.resendViewChangeTimer = time.NewTimer(lbft.options.ResendViewChange)
	lbft.resendViewChangeTimer.Stop()
	lbft.viewChangePeriodTimer = time.NewTimer(lbft.options.BlockInterval)
	lbft.viewChangePeriodTimer.Stop()
	lbft.nullRequestTimer = time.NewTimer(lbft.options.NullRequest)
	lbft.nullRequestTimer.Stop()
	return lbft
}

//Lbft Define lbft consenter
type Lbft struct {
	priority        int64
	lastPrimaryID   string
	primaryID       string
	execSeqNo       uint64
	lastSeqNo       uint64
	seqNo           uint64
	verifySeqNo     uint64
	votedCnt        int
	concurrentCntTo int

	prePrepareAsync *asyncSeqNo
	commitAsync     *asyncSeqNo

	options                 *Options
	stack                   consensus.IStack
	committedRequestBatch   map[uint64]*RequestBatch
	rwCommittedRequestBatch sync.RWMutex
	lbftCores               map[string]*lbftCore
	rwlbftCores             sync.RWMutex
	lbftCoreChan            chan string
	lbftCoreCommittedChan   chan *Committed
	voteViewChange          *vote.Vote
	voteCommitted           map[string]*vote.Vote

	blockTimer            *time.Timer
	viewChangeTimer       *time.Timer
	resendViewChangeTimer *time.Timer
	viewChangePeriodTimer *time.Timer
	nullRequestTimer      *time.Timer
	emptyBlockTimer       *time.Timer
	emptyBlockTimerStart  bool

	committedBlock            []*committedRequestBatch
	committedRequestBatchChan chan *committedRequestBatch

	recvConsensusMsgChan chan *Message
	committedTxsChan     chan *consensus.CommittedTxs
	broadcastChan        chan consensus.IBroadcast
	exit                 chan struct{}
	waitGroup            sync.WaitGroup
	pool                 *sync.Pool
}

func (lbft *Lbft) String() string {
	bytes, _ := json.Marshal(lbft.options)
	return string(bytes)
}

func (lbft *Lbft) execSeqNum() uint64 {
	return atomic.AddUint64(&lbft.execSeqNo, 0)
}

func (lbft *Lbft) incrExecSeqNum() uint64 {
	return atomic.AddUint64(&lbft.execSeqNo, 1)
}

func (lbft *Lbft) updateExecSeqNo(seqNo uint64) {
	if lbft.execSeqNo == 0 {
		lbft.execSeqNo = seqNo
	}
}

func (lbft *Lbft) updateLastSeqNo(seqNo uint64) {
	if t := seqNo - lbft.lastSeqNum(); int64(t) > 0 {
		atomic.AddUint64(&lbft.lastSeqNo, t)
	}
}

func (lbft *Lbft) lastSeqNum() uint64 {
	return atomic.AddUint64(&lbft.lastSeqNo, 0)
}

func (lbft *Lbft) seqNum() uint64 {
	return atomic.AddUint64(&lbft.seqNo, 0)
}

func (lbft *Lbft) incrSeqNum() uint64 {
	return atomic.AddUint64(&lbft.seqNo, 1)
}

func (lbft *Lbft) updateVerifySeqNo(seqNo uint64) {
	if t := seqNo - lbft.verifySeqNum(); int64(t) > 0 {
		atomic.AddUint64(&lbft.verifySeqNo, t)
	}
}

func (lbft *Lbft) verifySeqNum() uint64 {
	return atomic.AddUint64(&lbft.verifySeqNo, 0)
}

func (lbft *Lbft) incrVerifySeqNum() uint64 {
	return atomic.AddUint64(&lbft.verifySeqNo, 1)
}

//Start Start consenter serverice
func (lbft *Lbft) Start() {
	if lbft.exit != nil {
		log.Warnf("Replica %s consenter alreay started", lbft.options.ID)
		return
	}
	lbft.exit = make(chan struct{})
	go func() {
		lbft.waitGroup.Add(1)
		defer lbft.waitGroup.Done()
		lbft.handleCommittedRequestBatch()
	}()
	go func() {
		lbft.waitGroup.Add(1)
		defer lbft.waitGroup.Done()
		lbft.handleTransaction()
	}()
	go func() {
		lbft.waitGroup.Add(1)
		defer lbft.waitGroup.Done()
		lbft.handleConsensusMsg()
	}()
	log.Debugf("Replica %s consenter started", lbft.options.ID)
	lbft.resetBlockTimer()
}

//Stop Stop consenter serverice
func (lbft *Lbft) Stop() {
	if lbft.exit == nil {
		log.Warnf("Replica %s consenter alreay stopped", lbft.options.ID)
		return
	}
	close(lbft.exit)
	lbft.waitGroup.Wait()
	lbft.exit = nil
	log.Debugf("Replica %s consenter stopped", lbft.options.ID)
}

//RecvConsensus Receive consensus data for consenter
func (lbft *Lbft) RecvConsensus(payload []byte) {
	msg := &Message{}
	if err := msg.Deserialize(payload); err != nil {
		log.Errorf("Replica %s receive consensus message : unkown %v", lbft.options.ID, err)
		return
	}
	//log.Debugf("Replica %s receive broadcast consensus message %s(%s)", lbft.options.ID, msg.info(), hash(msg))
	lbft.recvConsensusMsgChan <- msg
}

//BroadcastTransactionChannel Broadcast consensus data
func (lbft *Lbft) BroadcastTransactionChannel() <-chan consensus.ITransaction {
	return nil
}

//BroadcastConsensusChannel Broadcast consensus data
func (lbft *Lbft) BroadcastConsensusChannel() <-chan consensus.IBroadcast {
	return lbft.broadcastChan
}

//CommittedTxsChannel Commit block data
func (lbft *Lbft) CommittedTxsChannel() <-chan *consensus.CommittedTxs {
	return lbft.committedTxsChan
}

func (lbft *Lbft) intersectionQuorum() int {
	return lbft.options.Q
}

func (lbft *Lbft) handleCommittedRequestBatch() {
	var id int64 = EMPTYBLOCK
	var has bool
	for {
		select {
		case <-lbft.exit:
			return
		case name := <-lbft.lbftCoreChan:
			lbft.removeInstance(name)
		case committed := <-lbft.lbftCoreCommittedChan:
			lbft.addCommittedReqeustBatch(committed.SeqNo, committed.RequestBatch)
		case ctt := <-lbft.committedRequestBatchChan:
			if ctt.requestBatch.Id == EMPTYBLOCK {
				if id != EMPTYBLOCK || has {
					lbft.committedBlock = append(lbft.committedBlock, ctt)
					lbft.writeBlock()
					id = EMPTYBLOCK
					has = false
					lbft.committedBlock = nil
				} else {
					lbft.committedBlock = append(lbft.committedBlock, ctt)
				}
			} else if ctt.requestBatch.fromChain() == lbft.options.Chain { //发送方
				if id != EMPTYBLOCK && id != ctt.requestBatch.Id {
					lbft.writeBlock()
					lbft.committedBlock = nil
					lbft.committedBlock = append(lbft.committedBlock, ctt)
					id = ctt.requestBatch.Id
				} else {
					if has {
						lbft.writeBlock()
						lbft.committedBlock = nil
						has = false
						lbft.committedBlock = append(lbft.committedBlock, ctt)
						id = ctt.requestBatch.Id
					} else {
						lbft.committedBlock = append(lbft.committedBlock, ctt)
						id = ctt.requestBatch.Id
					}
				}
			} else { //接收方
				lbft.committedBlock = append(lbft.committedBlock, ctt)
				has = true
			}
		}
	}
}

func (lbft *Lbft) writeBlock() {
	if len(lbft.committedBlock) == 0 {
		return
	}
	var nano uint32
	var seqNos []uint64
	txs := []consensus.ITransaction{}
	for _, ctt := range lbft.committedBlock {
		seqNos = append(seqNos, ctt.seqNo)
		nano = ctt.requestBatch.Time
		reqBatch := ctt.requestBatch
		for _, req := range reqBatch.Requests {
			tx := lbft.pool.Get().(consensus.ITransaction)
			//tx := lbft.stack.NewTransaction()
			if err := tx.Deserialize(req.Transaction); err == nil {
				txs = append(txs, tx)
			}
		}
	}
	log.Infof("Replica %s write block %v (%d transactions) ", lbft.options.ID, seqNos, len(txs))
	lbft.committedTxsChan <- &consensus.CommittedTxs{Time: nano, Transactions: txs, SeqNos: seqNos}
	lbft.committedBlock = nil
}

func (lbft *Lbft) handleTransaction() {
	for {
		select {
		case <-lbft.exit:
			return
		case <-lbft.viewChangeTimer.C:
			log.Debugf("Replica %s view change timeout", lbft.options.ID)
			lbft.voteViewChange.Clear()
		case <-lbft.resendViewChangeTimer.C:
			lbft.votedCnt++
			if lbft.votedCnt > lbft.options.K*lbft.options.N {
				lbft.voteViewChange.IterVoter(func(voter string, ticket vote.ITicket) {
					tvc := ticket.(*ViewChange)
					log.Infof("Replica %s received view change message from %s for voter %s , lastSeqNo %d", lbft.options.ID, tvc.ReplicaID, tvc.PrimaryID, tvc.H)
				})
				log.Panicf("Replica %s failed to vote new Primary, diff lastSeqNo  %d", lbft.options.ID, lbft.lastSeqNum())
			}
			log.Debugf("Replica %s resend view change %s", lbft.options.ID, time.Now())
			var vc *ViewChange
			lbft.voteViewChange.IterVoter(func(voter string, ticket vote.ITicket) {
				tvc := ticket.(*ViewChange)
				if tvc.PrimaryID != lbft.lastPrimaryID && tvc.H == lbft.lastSeqNum() {
					if vc == nil {
						vc = tvc
					} else if tvc.Priority < vc.Priority {
						vc = tvc
					}
				}
			})
			lbft.voteViewChange.Clear()
			t1 := time.Now()
			t2 := t1.Truncate(time.Second)
			time.Sleep(time.Second - t1.Sub(t2))
			lbft.sendViewChange(vc)
		case <-lbft.viewChangePeriodTimer.C:
			log.Debugf("Replica %s view change period", lbft.options.ID)
			lbft.sendViewChange(nil)
		case <-lbft.nullRequestTimer.C:
			lbft.nullRequestHandler()
		case <-lbft.emptyBlockTimer.C:
			if lbft.isPrimary() {
				requestBath := &RequestBatch{Time: uint32(time.Now().Unix()), Id: EMPTYBLOCK}
				lbft.handleRequestBatch(requestBath)
			}
			lbft.emptyBlockTimerStart = false
			log.Debugf("Replica %s stop empty block", lbft.options.ID)
		case <-lbft.blockTimer.C:
			lbft.maybeSendViewChange()
			lbft.submitRequestBatches()
			lbft.resetBlockTimer()
		}

	}
}

func (lbft *Lbft) submitRequestBatches() {
	if !lbft.isPrimary() || lbft.stack.Len() == 0 {
		return
	}

	id := time.Now().UnixNano()
	requestBatchList := make([]*RequestBatch, 0, lbft.options.MaxConcurrentNumFrom/2)
	reqs := make([]*Request, 0, lbft.options.BlockSize/3)
	var toChain string
	var nano uint32
	var cnt int
	lbft.stack.IterTransaction(func(tx consensus.ITransaction) bool {
		if tx == nil && len(reqs) > 0 {
			requestBath := &RequestBatch{Time: nano, Requests: reqs, Id: id}
			log.Debugf("Replica %s generate requestBatch %s : timestamp %d, transations %d", lbft.options.ID, hash(requestBath), requestBath.Time, len(requestBath.Requests))
			cnt = 0
			nano = 0
			toChain = ""
			reqs = make([]*Request, 0, lbft.options.BlockSize/3)
			requestBatchList = append(requestBatchList, requestBath)
			if len(requestBatchList) == lbft.options.MaxConcurrentNumFrom {
				return true
			}
		}
		req := &Request{
			Time:        tx.CreateTime(),
			Transaction: tx.Serialize(),
			FromChain:   tx.FromChain(),
			ToChain:     tx.ToChain(),
			Nonce:       tx.Nonce(),
		}
		if toChain != "" && req.ToChain != toChain {
			requestBath := &RequestBatch{Time: nano, Requests: reqs, Id: id}
			log.Debugf("Replica %s generate requestBatch %s : timestamp %d, transations %d", lbft.options.ID, hash(requestBath), requestBath.Time, len(requestBath.Requests))
			cnt = 0
			nano = 0
			toChain = ""
			reqs = make([]*Request, 0, lbft.options.BlockSize/3)
			requestBatchList = append(requestBatchList, requestBath)
			if len(requestBatchList) == lbft.options.MaxConcurrentNumFrom {
				return true
			}
		}
		reqs = append(reqs, req)
		toChain = req.ToChain
		if nano < req.Time {
			nano = req.Time
		}
		cnt++
		if cnt == lbft.options.BlockSize {
			requestBath := &RequestBatch{Time: nano, Requests: reqs, Id: id}
			log.Debugf("Replica %s generate requestBatch %s : timestamp %d, transations %d", lbft.options.ID, hash(requestBath), requestBath.Time, len(requestBath.Requests))
			cnt = 0
			nano = 0
			toChain = ""
			reqs = make([]*Request, 0, lbft.options.BlockSize/3)
			requestBatchList = append(requestBatchList, requestBath)
			if len(requestBatchList) == lbft.options.MaxConcurrentNumFrom {
				return true
			}
		}
		return false
	})
	if len(reqs) > 0 {
		requestBath := &RequestBatch{Time: nano, Requests: reqs, Id: id}
		requestBatchList = append(requestBatchList, requestBath)
		log.Debugf("Replica %s generate requestBatch %s : timestamp %d, transations %d", lbft.options.ID, hash(requestBath), requestBath.Time, len(requestBath.Requests))
	}

	for _, requestBatch := range requestBatchList {
		if lbft.isValid(requestBatch, true) {
			lbft.handleRequestBatch(requestBatch)
		} else {
			log.Warnf("Replica %s received requestBatch message for consensus %s : ignore illegal requestBatch (%s == %s)", lbft.options.ID, requestBatch.key(), requestBatch.fromChain(), lbft.options.Chain)
		}
	}
}

func (lbft *Lbft) resetViewChangePeriodTimer() {
	lbft.viewChangePeriodTimer.Stop()
	if lbft.hasPrimary() && lbft.options.ViewChangePeriod > 0*time.Second {
		t1 := time.Now()
		t2 := t1.Truncate(lbft.options.ViewChangePeriod)
		lbft.viewChangePeriodTimer.Reset(lbft.options.ViewChangePeriod - t1.Sub(t2))
	}
}

func (lbft *Lbft) resetBlockTimer() {
	lbft.blockTimer.Stop()
	t1 := time.Now()
	t2 := t1.Truncate(lbft.options.BlockInterval)
	lbft.blockTimer.Reset(lbft.options.BlockInterval - t1.Sub(t2))
}

func (lbft *Lbft) resetEmptyBlockTimer() {
	lbft.emptyBlockTimer.Stop()
	t1 := time.Now()
	t2 := t1.Truncate(lbft.options.BlockInterval)
	lbft.emptyBlockTimer.Reset(2*lbft.options.BlockInterval - t1.Sub(t2))
	lbft.emptyBlockTimerStart = true
	log.Debugf("Replica %s start empty block", lbft.options.ID)
}

func (lbft *Lbft) softResetEmptyBlockTimer() {
	if lbft.emptyBlockTimerStart {
		return
	}
	lbft.emptyBlockTimer.Stop()
	t1 := time.Now()
	t2 := t1.Truncate(lbft.options.BlockInterval)
	lbft.emptyBlockTimer.Reset(2*lbft.options.BlockInterval - t1.Sub(t2))
	lbft.emptyBlockTimerStart = true
	log.Debugf("Replica %s start empty block", lbft.options.ID)
}

func (lbft *Lbft) hasPrimary() bool {
	return lbft.primaryID != ""
}

func (lbft *Lbft) maybeSendViewChange() {
	if lbft.hasPrimary() {
		return
	}
	log.Debugf("Primary %s has no PrimaryID, send view change", lbft.options.ID)
	lbft.sendViewChange(nil)
}

func (lbft *Lbft) isPrimary() bool {
	return lbft.options.ID == lbft.primaryID
}

func (lbft *Lbft) handleConsensusMsg() {
	for {
		select {
		case <-lbft.exit:
			return
		case msg := <-lbft.recvConsensusMsgChan:
			switch tp := msg.Payload.(type) {
			case *Message_RequestBatch:
				if requestBatch := msg.GetRequestBatch(); requestBatch != nil {
					if !lbft.isValid(requestBatch, false) {
						log.Errorf("Replica %s received requestBatch message for consensus %s : ignore illegal requestBatch (%s == %s) ", lbft.options.ID, requestBatch.key(), requestBatch.toChain(), lbft.options.Chain)
					} else if lbft.isPrimary() {
						if lbft.concurrentCntTo > lbft.options.MaxConcurrentNumTo {
							log.Warnf("Replica %s received requestBatch message for consensus %s :  max concurrent %d ", lbft.options.ID, requestBatch.key(), lbft.options.MaxConcurrentNumTo)
						} else {
							lbft.concurrentCntTo++
							lbft.handleRequestBatch(requestBatch)
						}
					}
				}
			case *Message_PrePrepare:
				if preprepare := msg.GetPrePrepare(); preprepare != nil {
					if !lbft.hasPrimary() {
						log.Errorf("Replica %s received prePrepare message from %s for consensus %s : ignore diff primayID (%s==%s)", lbft.options.ID, preprepare.ReplicaID, preprepare.Name, preprepare.PrimaryID, lbft.primaryID)
					} else if preprepare.Chain != lbft.options.Chain || preprepare.ReplicaID != preprepare.PrimaryID {
						log.Errorf("Replica %s received prePrepare message from %s for consensus %s : ignore illegal preprepare (%s==%s) ", lbft.options.ID, preprepare.ReplicaID, preprepare.Name, preprepare.Chain, lbft.options.Chain)
					} else if preprepare.ReplicaID != lbft.primaryID {
						log.Errorf("Replica %s received prePrepare message from %s for consensus %s :  ignore not from primayID (%s==%s)", lbft.options.ID, preprepare.ReplicaID, preprepare.Name, preprepare.ReplicaID, lbft.primaryID)
					} else if preprepare.SeqNo <= lbft.lastSeqNum() {
						log.Debugf("Replica %s received prePrepare message from %s for consensus %s : ignore delay seqNo (%d > %d)", lbft.options.ID, preprepare.ReplicaID, preprepare.Name, preprepare.SeqNo, lbft.lastSeqNum())
					} else {
						lbft.handleLbftCoreMsg(preprepare.Name, msg)
					}
				}
			case *Message_Prepare:
				if prepare := msg.GetPrepare(); prepare != nil {
					if !lbft.hasPrimary() {
						log.Errorf("Replica %s received prepare message from %s for consensus %s : ignore diff primayID (%s==%s)", lbft.options.ID, prepare.ReplicaID, prepare.Name, prepare.PrimaryID, lbft.primaryID)
					} else if prepare.Chain == lbft.options.Chain && prepare.PrimaryID != lbft.primaryID {
						log.Errorf("Replica %s received prepare message from %s for consensus %s : ignore diff primayID (%s==%s)", lbft.options.ID, prepare.ReplicaID, prepare.Name, prepare.PrimaryID, lbft.primaryID)
					} else if prepare.Chain == lbft.options.Chain && prepare.SeqNo <= lbft.lastSeqNum() {
						log.Debugf("Replica %s received prepare message from %s for consensus %s : ingore delay sepNo (%d > %d)", lbft.options.ID, prepare.ReplicaID, prepare.Name, prepare.SeqNo, lbft.lastSeqNum())
					} else {
						lbft.handleLbftCoreMsg(prepare.Name, msg)
					}
				}
			case *Message_Commit:
				if commit := msg.GetCommit(); commit != nil {
					if !lbft.hasPrimary() {
						log.Errorf("Replica %s received commit message from %s for consensus %s : ignore diff primayID (%s==%s)", lbft.options.ID, commit.ReplicaID, commit.Name, commit.PrimaryID, lbft.primaryID)
					} else if commit.Chain == lbft.options.Chain && commit.PrimaryID != lbft.primaryID {
						log.Errorf("Replica %s received commit message from %s for consensus %s : ignore diff primayID (%s==%s)", lbft.options.ID, commit.ReplicaID, commit.Name, commit.PrimaryID, lbft.primaryID)
					} else if commit.Chain == lbft.options.Chain && commit.SeqNo <= lbft.lastSeqNum() {
						log.Debugf("Replica %s received commit message from %s for consensus %s : ignore delay seqNo (%d > %d)", lbft.options.ID, commit.ReplicaID, commit.Name, commit.SeqNo, lbft.lastSeqNum())
					} else {
						lbft.handleLbftCoreMsg(commit.Name, msg)
					}
				}
			case *Message_Committed:
				if committed := msg.GetCommitted(); committed != nil {
					if committed.Chain != lbft.options.Chain {
						log.Errorf("Replica %s received committed message from %s for consensus %s : ignore diff chain (%s==%s) ", lbft.options.ID, committed.ReplicaID, committed.Name, committed.Chain, lbft.options.Chain)
					} else if committed.SeqNo <= lbft.execSeqNum() {
						log.Debugf("Replica %s received committed message from %s for consensus %s : ignore delay seqNo (%d > %d)", lbft.options.ID, committed.ReplicaID, committed.Name, committed.SeqNo, lbft.execSeqNum())
					} else {
						lbft.recvCommitted(committed)
					}
				}
			case *Message_FetchCommitted:
				if committed := msg.GetFetchCommitted(); committed != nil {
					if committed.Chain != lbft.options.Chain {
						log.Errorf("Replica %s received fetch committed message from %s : ignore diff chain  (%s==%s)", lbft.options.ID, committed.ReplicaID, committed.Chain, lbft.options.Chain)
					} else {
						if requestBatch := lbft.getCommittedReqeustBatch(committed.SeqNo); requestBatch != nil {
							key := requestBatch.key()
							log.Infof("Replica %s received fetch committed message from %s : broadcast committed for %s ", lbft.options.ID, committed.ReplicaID, key)
							ctt := &Committed{
								Name:         key,
								Chain:        lbft.options.Chain,
								ReplicaID:    lbft.options.ID,
								SeqNo:        committed.SeqNo,
								RequestBatch: requestBatch,
							}
							lbft.broadcast(lbft.options.Chain, &Message{Payload: &Message_Committed{Committed: ctt}})
						} else {
							log.Warnf("Replica %s received fetch committed message from %s : ignore missing seqno ", lbft.options.ID, committed.ReplicaID)
						}
					}
				}
			case *Message_Viewchange:
				if vc := msg.GetViewchange(); vc != nil {
					if vc.Chain != lbft.options.Chain {
						log.Errorf("Replica %s received view change from %s : ignore diff chain (%s==%s) ", lbft.options.ID, vc.ReplicaID, vc.Chain, lbft.options.Chain)
						return
					}
					lbft.recvViewChange(vc)
				}
			case *Message_NullReqest:
				if np := msg.GetNullReqest(); np != nil {
					if np.Chain != lbft.options.Chain {
						log.Errorf("Replica %s received null request from %s : ignore diff chain (%s==%s) ", lbft.options.ID, np.ReplicaID, np.Chain, lbft.options.Chain)
						return
					}
					log.Debugf("Replica %s received null request from %s", lbft.options.ID, np.ReplicaID)
					if lbft.primaryID != np.PrimaryID && np.PrimaryID == np.ReplicaID {
						log.Infof("Replica %s view change : vote new PrimaryID %s (%s), null request", lbft.options.ID, np.PrimaryID, lbft.primaryID)
						lbft.primaryID = np.PrimaryID
						lbft.lastSeqNo = np.H
						lbft.seqNo = np.H
						lbft.updateExecSeqNo(np.H)
						lbft.verifySeqNo = np.H
						lbft.prePrepareAsync = newAsyncSeqNo(np.H)
						lbft.commitAsync = newAsyncSeqNo(np.H)
					}
					lbft.nullRequestTimerStart()
				}
			default:
				log.Warnf("unsupport consensus message type %v ", tp)
			}
		}
	}
}

func (lbft *Lbft) nullRequestHandler() {
	if !lbft.hasPrimary() {
		return
	}
	if lbft.isPrimary() {
		log.Debugf("Primary %s null request timer expired, sending null request", lbft.options.ID)
		nullRequest := &NullRequest{
			ReplicaID: lbft.options.ID,
			Chain:     lbft.options.Chain,
			PrimaryID: lbft.primaryID,
			H:         lbft.lastSeqNum(),
		}
		lbft.broadcast(lbft.options.Chain, &Message{Payload: &Message_NullReqest{NullReqest: nullRequest}})
		lbft.nullRequestTimerStart()
	} else {
		log.Debugf("Replica %s null request timer expired, sending view change", lbft.options.ID)
		lbft.sendViewChange(nil)
	}
}

func (lbft *Lbft) nullRequestTimerStart() {
	lbft.nullRequestTimer.Stop()
	if !(lbft.options.NullRequest > 0*time.Second) {
		return
	}
	if !lbft.hasPrimary() {
		return
	}
	if lbft.isPrimary() {
		lbft.nullRequestTimer.Reset(lbft.options.NullRequest)
	} else {
		lbft.nullRequestTimer.Reset(lbft.options.BlockInterval + lbft.options.NullRequest)
	}
}

func (lbft *Lbft) sendViewChange(vc *ViewChange) {
	if quorum, _ := lbft.voteViewChange.VoterByVoter(lbft.options.ID); quorum > 0 {
		return
	}
	if vc != nil {
		vc.Chain = lbft.options.Chain
		vc.ReplicaID = lbft.options.ID
	} else {
		lbft.updateLastSeqNo(lbft.stack.GetLastSeqNo())
		vc = &ViewChange{
			Chain:     lbft.options.Chain,
			ReplicaID: lbft.options.ID,
			Priority:  lbft.priority,
			PrimaryID: lbft.options.ID,
			H:         lbft.lastSeqNum(),
		}
	}
	lbft.recvViewChange(vc)
	lbft.broadcast(lbft.options.Chain, &Message{Payload: &Message_Viewchange{Viewchange: vc}})
}

func (lbft *Lbft) recvViewChange(vc *ViewChange) {
	if vc.Chain != lbft.options.Chain {
		log.Warningf("Replica %s received view change message form other chain (%s-%s)", lbft.options.ID, lbft.options.Chain, vc.Chain)
		return
	}

	if lbft.options.ID != vc.ReplicaID && vc.ReplicaID == lbft.primaryID && vc.PrimaryID == lbft.primaryID {
		lbft.sendViewChange(nil)
	}

	var added bool
	lbft.voteViewChange.IterVoter(func(voter string, ticket vote.ITicket) {
		tvc := ticket.(*ViewChange)
		if tvc.ReplicaID == vc.ReplicaID {
			if vc.H > tvc.H {
				tvc.H = vc.H
			}
			added = true
		}
	})
	if !added {
		lbft.voteViewChange.Add(vc.ReplicaID, vc)
	}
	cnt := lbft.voteViewChange.Size()
	log.Infof("Replica %s received view change message from %s for voter %s , vote size %d", lbft.options.ID, vc.ReplicaID, vc.PrimaryID, cnt)
	if cnt == 1 {
		lbft.viewChangeTimer.Reset(lbft.options.ViewChange)
	} else if cnt == lbft.intersectionQuorum() {
		lbft.lastPrimaryID = lbft.primaryID
		lbft.primaryID = ""
		log.Infof("Replica %s start to vote new PrimaryID, view change", lbft.options.ID)
		lbft.viewChangeTimer.Stop()
		lbft.resetViewChangePeriodTimer()
		lbft.nullRequestTimerStart()
		t1 := time.Now()
		t2 := t1.Truncate(lbft.options.ResendViewChange)
		lbft.resendViewChangeTimer.Reset(lbft.options.ResendViewChange - t1.Sub(t2))
	}
	if quorum, ticker := lbft.voteViewChange.Voter(); quorum >= lbft.intersectionQuorum() {
		lbft.resendViewChangeTimer.Stop()
		lbft.voteViewChange.Clear()
		vc := ticker.(*ViewChange)
		lbft.newView(vc)
	}
}

func (lbft *Lbft) newView(vc *ViewChange) {
	lbft.primaryID = vc.PrimaryID
	if lbft.isPrimary() {
		lbft.priority = time.Now().UnixNano()
		lbft.blockTimer.Stop()
	}
	lbft.lastSeqNo = vc.H
	lbft.seqNo = vc.H
	lbft.votedCnt = 0
	log.Infof("Replica %s view change : vote new PrimaryID %s", lbft.options.ID, vc.PrimaryID)
	lbft.verifySeqNo = vc.H
	lbft.updateExecSeqNo(vc.H)
	lbft.iterInstance(func(key string, instance *lbftCore) {
		if !instance.isPassCommit {
			delete(lbft.lbftCores, key)
			instance.stop()
		} else {
			log.Debugf("Replica %s alreay commmit for consensus %s, view change", lbft.options.ID, instance.name)
		}
	})
	for len(lbft.lbftCoreChan) > 0 {
		name := <-lbft.lbftCoreChan
		lbft.removeInstance(name)
	}
	lbft.prePrepareAsync = newAsyncSeqNo(vc.H)
	lbft.commitAsync = newAsyncSeqNo(vc.H)
	lbft.resetViewChangePeriodTimer()
	lbft.nullRequestTimerStart()
	lbft.concurrentCntTo = 0
	if lbft.isPrimary() {
		for len(lbft.lbftCoreCommittedChan) > 0 {
			committed := <-lbft.lbftCoreCommittedChan
			lbft.addCommittedReqeustBatch(committed.SeqNo, committed.RequestBatch)
		}
		lbft.resetBlockTimer()
		lbft.resetEmptyBlockTimer()
	}
}

func (lbft *Lbft) recvCommitted(ct *Committed) {
	if ct.SeqNo <= lbft.execSeqNum() || lbft.hasCommittedReqeustBatch(ct.SeqNo) {
		log.Debugf("Replica %s received committed message from %s for consensus %s, delay", lbft.options.ID, ct.ReplicaID, ct.Name)
		delete(lbft.voteCommitted, ct.Name)
		for k, v := range lbft.voteCommitted {
			_, ticket := v.Voter()
			ct := ticket.(*Committed)
			if ct.SeqNo <= lbft.execSeqNum() {
				delete(lbft.voteCommitted, k)
			}
		}
		return
	}

	v, ok := lbft.voteCommitted[ct.Name]
	if !ok {
		v = vote.NewVote()
		lbft.voteCommitted[ct.Name] = v
	}
	v.Add(ct.ReplicaID, ct)
	log.Infof("Replica %s received committed message from %s for consensus %s, vote %d", lbft.options.ID, ct.ReplicaID, ct.Name, v.Size())
	if quorum := v.VoterByTicket(ct); quorum >= lbft.intersectionQuorum() {
		lbft.addCommittedReqeustBatch(ct.SeqNo, ct.RequestBatch)
		delete(lbft.voteCommitted, ct.Name)
	}
}

func (lbft *Lbft) addCommittedReqeustBatch(seqNo uint64, requestBatch *RequestBatch) {
	lbft.rwCommittedRequestBatch.Lock()
	defer lbft.rwCommittedRequestBatch.Unlock()
	if _, ok := lbft.committedRequestBatch[seqNo]; ok {
		return
	}
	log.Infof("Replica %s add committed requestBatch %d (%s)", lbft.options.ID, seqNo, hash(requestBatch))
	lbft.committedRequestBatch[seqNo] = requestBatch
	lbft.updateLastSeqNo(seqNo)
	lbft.updateVerifySeqNo(seqNo)
	go lbft.checkpoint()
	lbft.removeInstance(requestBatch.key())
	// && lbft.options.Chain == requestBatch.fromChain()
	if len(requestBatch.Requests) > 0 {
		lbft.stack.Removes(lbft.toTxs(requestBatch))
		// go func(requestBatch *RequestBatch) {
		// 	tts := lbft.toTxs(requestBatch)
		// 	for _, tt := range tts {
		// 		tx := tt.(*types.Transaction)
		// 		sender := tx.Sender()
		// 		log.Info("yyy", " ", sender, " ", tx.Nonce(), " ", tx.Hash(), requestBatch.key())
		// 	}
		// }(requestBatch)
	}
}

//Uint64Slice sortable
type Uint64Slice []uint64

func (us Uint64Slice) Len() int {
	return len(us)
}
func (us Uint64Slice) Less(i, j int) bool {
	return us[i] < us[j]
}
func (us Uint64Slice) Swap(i, j int) {
	us[i], us[j] = us[j], us[i]
}

func (lbft *Lbft) checkpoint() {
	lbft.rwCommittedRequestBatch.Lock()
	defer lbft.rwCommittedRequestBatch.Unlock()
	if len(lbft.committedRequestBatch) == 0 {
		return
	}
	keys := Uint64Slice{}
	for seqNo := range lbft.committedRequestBatch {
		keys = append(keys, seqNo)
	}
	sort.Sort(keys)
	checkpoint := lbft.execSeqNum() + 1
	for _, seqNo := range keys {
		reqBatch := lbft.committedRequestBatch[seqNo]
		if seqNo < checkpoint {
			if n := seqNo - uint64(lbft.options.K); n >= keys[0] {
				delete(lbft.committedRequestBatch, n)
			}
		} else if seqNo == checkpoint {
			height := lbft.incrExecSeqNum()
			log.Debugf("Replica %s write requestBatch %d (%s, %d transactions) ", lbft.options.ID, seqNo, hash(reqBatch), len(reqBatch.Requests))
			lbft.committedRequestBatchChan <- &committedRequestBatch{requestBatch: reqBatch, seqNo: height}
			delete(lbft.committedRequestBatch, seqNo-uint64(lbft.options.K))
			checkpoint = lbft.execSeqNum() + 1
		} else if seqNo-checkpoint > uint64(lbft.options.K) {
			log.Warnf("Replica %s fetch committed %d ", lbft.options.ID, checkpoint)
			lbft.broadcast(lbft.options.Chain, &Message{Payload: &Message_FetchCommitted{FetchCommitted: &FetchCommitted{
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
				SeqNo:     checkpoint,
			}}})
			break
		}
	}

	if cnt := len(keys); cnt > 0 && keys[cnt-1] > checkpoint+uint64(lbft.options.K) {
		log.Infof("Replica %s need seqNo %d ", lbft.options.ID, lbft.execSeqNum()+1)
		for seqNo, reqBatch := range lbft.committedRequestBatch {
			log.Infof("Replica %s seqNo %d : %s", lbft.options.ID, seqNo, reqBatch.key())
		}
		log.Panicf("Replica %s fallen behind over %d", lbft.options.ID, lbft.options.K)
	}
}

func (lbft *Lbft) removeCommittedReqeustBatch(seqNo uint64) {
	lbft.rwCommittedRequestBatch.Lock()
	defer lbft.rwCommittedRequestBatch.Unlock()
	delete(lbft.committedRequestBatch, seqNo)
}

func (lbft *Lbft) iterCommittedReqeustBatch(function func(uint64, *RequestBatch)) {
	lbft.rwCommittedRequestBatch.Lock()
	defer lbft.rwCommittedRequestBatch.Unlock()
	for seqNo, requestBatch := range lbft.committedRequestBatch {
		function(seqNo, requestBatch)
	}
}

func (lbft *Lbft) hasCommittedReqeustBatch(seqNo uint64) bool {
	lbft.rwCommittedRequestBatch.RLock()
	defer lbft.rwCommittedRequestBatch.RUnlock()
	_, ok := lbft.committedRequestBatch[seqNo]
	return ok
}

func (lbft *Lbft) getCommittedReqeustBatch(seqNo uint64) *RequestBatch {
	lbft.rwCommittedRequestBatch.RLock()
	defer lbft.rwCommittedRequestBatch.RUnlock()
	req, ok := lbft.committedRequestBatch[seqNo]
	if ok {
		return req
	}
	return nil
}

func (lbft *Lbft) handleRequestBatch(requestBatch *RequestBatch) {
	lbft.rwlbftCores.Lock()
	defer lbft.rwlbftCores.Unlock()
	key := requestBatch.key()
	instance, ok := lbft.lbftCores[key]
	if !ok {
		instance = newLbftCore(key, lbft)
		lbft.lbftCores[key] = instance
	}
	instance.handleRequestBatch(lbft.incrSeqNum(), requestBatch)
}

func (lbft *Lbft) handleLbftCoreMsg(key string, msg *Message) {
	lbft.rwlbftCores.Lock()
	defer lbft.rwlbftCores.Unlock()
	instance, ok := lbft.lbftCores[key]
	if !ok {
		instance = newLbftCore(key, lbft)
		lbft.lbftCores[key] = instance
	}
	instance.recvMessage(msg)
}

func (lbft *Lbft) getInstance(key string) *lbftCore {
	lbft.rwlbftCores.Lock()
	defer lbft.rwlbftCores.Unlock()
	if instance, ok := lbft.lbftCores[key]; ok {
		return instance
	}
	instance := newLbftCore(key, lbft)
	lbft.lbftCores[key] = instance
	return instance
}

func (lbft *Lbft) removeInstance(key string) {
	lbft.rwlbftCores.Lock()
	defer lbft.rwlbftCores.Unlock()
	if instance, ok := lbft.lbftCores[key]; ok {
		delete(lbft.lbftCores, key)
		if lbft.concurrentCntTo > 0 && instance.isCross() {
			lbft.concurrentCntTo--
		}
		instance.stop()
	}
}

func (lbft *Lbft) iterInstance(function func(string, *lbftCore)) {
	lbft.rwlbftCores.Lock()
	defer lbft.rwlbftCores.Unlock()
	for key, instance := range lbft.lbftCores {
		function(key, instance)
	}
}

func (lbft *Lbft) hasInstance(key string) bool {
	lbft.rwlbftCores.RLock()
	defer lbft.rwlbftCores.RUnlock()
	_, ok := lbft.lbftCores[key]
	return ok
}

func (lbft *Lbft) broadcast(to string, msg *Message) {
	//log.Debugf("Replica %s send broadcast consensus message %s(%s) from %s to %s", lbft.options.ID, msg.info(), hash(msg), lbft.options.Chain, to)
	lbft.broadcastChan <- &Broadcast{
		to:  to,
		msg: msg,
	}
}

func (lbft *Lbft) isValid(requestBatch *RequestBatch, from bool) bool {
	if from {
		if requestBatch.Id == EMPTYBLOCK && len(requestBatch.Requests) == 0 {
			return true
		}
		fromChain := requestBatch.fromChain()
		return fromChain == lbft.options.Chain
	}

	if requestBatch.Id == EMPTYBLOCK {
		return false
	}
	toChain := requestBatch.toChain()
	return toChain == lbft.options.Chain
}

func (lbft *Lbft) toTxs(requestBatch *RequestBatch) []consensus.ITransaction {
	txs := make([]consensus.ITransaction, 0, len(requestBatch.Requests))
	for _, req := range requestBatch.Requests {
		tx := lbft.pool.Get().(consensus.ITransaction)
		//tx := lbft.stack.NewTransaction()
		if err := tx.Deserialize(req.Transaction); err == nil {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (lbft *Lbft) toRequestBatch(txs []consensus.ITransaction) *RequestBatch {
	reqs := make([]*Request, 0, len(txs))
	var nano uint32
	for _, tx := range txs {
		req := &Request{
			Time:        tx.CreateTime(),
			Transaction: tx.Serialize(),
			FromChain:   tx.FromChain(),
			ToChain:     tx.ToChain(),
			Nonce:       tx.Nonce(),
		}
		if nano < req.Time {
			nano = req.Time
		}
		reqs = append(reqs, req)
	}
	return &RequestBatch{Requests: reqs, Time: nano}
}

type committedRequestBatch struct {
	seqNo        uint64
	requestBatch *RequestBatch
}
