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

package nbft

import (
	"encoding/json"
	"strings"
	"time"

	"sync"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils/sortedlinkedlist"
	"github.com/bocheninc/L0/components/utils/vote"
	"github.com/bocheninc/L0/core/consensus"
)

// MINQUORUM  Define min quorum
const MINQUORUM = 3

// FORMAT time layout
const FORMAT = "2006-01-02 15:04:05.999999999 -0700 MST"

// NewNbft Create nbft consenter
func NewNbft(options *Options, stack consensus.IStack) *Nbft {
	nbft := &Nbft{
		options: options,
		stack:   stack,
	}
	nbft.nbftCores = make(map[string]*nbftCore)
	nbft.list = sortedlinkedlist.NewSortedLinkedList()
	nbft.commitReqChan = make(chan *Request, nbft.options.CommitTxChanSize)
	nbft.committedReqsChan = make(chan *Committed, nbft.options.CommitTxChanSize)
	nbft.committedTxsChan = make(chan *consensus.CommittedTxs, nbft.options.CommittedTxsChanSize)
	nbft.broadcastChan = make(chan consensus.IBroadcast, nbft.options.BroadcastChanSize)
	nbft.blockTimeChan = make(chan time.Time, 100)
	nbft.blockTimer = time.NewTimer(nbft.options.BlockInterval)
	nbft.blockTimer.Stop()

	nbft.unexecuteCommittedReqs = make(map[string]*Committed)
	nbft.executedCommittedReqs = make(map[string]*Committed)
	nbft.returnCommittedReqsList = make(map[string][]*ReturnCommitted)
	nbft.hLastExec = make(map[string]time.Time)
	if nbft.options.BlockTimeout > nbft.options.BlockInterval {
		log.Warn("nbft.blockTimeout should is smaller nbft.blockInterval")
		nbft.options.BlockTimeout = 2 * nbft.options.BlockInterval / 3
	}
	if nbft.options.BlockDelay < nbft.options.BlockInterval {
		log.Warn("nbft.BlockDelay should is greater nbft.BlockInterval")
		nbft.options.BlockDelay = nbft.options.BlockInterval
	}
	if nbft.options.Q < MINQUORUM {
		log.Warn("nbft.Q should is not smaller %d", MINQUORUM)
		nbft.options.Q = MINQUORUM
	}
	return nbft
}

// Nbft Define nbft consenter
type Nbft struct {
	options                 *Options
	stack                   consensus.IStack
	commitReqChan           chan *Request
	committedReqsChan       chan *Committed
	committedTxsChan        chan *consensus.CommittedTxs
	broadcastChan           chan consensus.IBroadcast
	blockTimeChan           chan time.Time
	blockTimer              *time.Timer
	exit                    chan struct{}
	list                    *sortedlinkedlist.SortedLinkedList
	nbftCores               map[string]*nbftCore
	executedCommittedReqs   map[string]*Committed
	unexecuteCommittedReqs  map[string]*Committed
	returnCommittedReqsList map[string][]*ReturnCommitted
	lastNbftCore            time.Time
	lastExecCommittedReqs   time.Time
	hLastExec               map[string]time.Time
	lastRequestTime         int64
	sync.RWMutex
}

func (nbft *Nbft) String() string {
	bytes, _ := json.Marshal(nbft.options)
	return string(bytes)
}

// IsRunning nbft consenter serverice already started
func (nbft *Nbft) IsRunning() bool {
	return nbft.exit != nil
}

// Start Start consenter serverice of nbft
func (nbft *Nbft) Start() {
	if nbft.IsRunning() {
		return
	}
	nbft.exit = make(chan struct{})
	for {
		select {
		case <-nbft.exit:
			nbft.exit = nil
			return
		case req := <-nbft.commitReqChan:
			log.Debugf("Replica %s received transaction", nbft.options.ID)
			if req.Time > nbft.lastRequestTime {
				nbft.list.Add(req)
				if nbft.list.Len() == 1 {
					nbft.resetBlockTimer()
				}
			}
		case <-nbft.blockTimer.C:
			nbft.processBlock()
		case committed := <-nbft.committedReqsChan:
			nbft.addUnexecuteCommittedReqs(committed)
		}
	}

}

func (nbft *Nbft) resetBlockTimer() {
	t := time.Now()
	t2 := t.Truncate(nbft.options.BlockInterval)
	log.Debugf("Replica %s will be start nbft service after %s", nbft.options.ID, nbft.options.BlockInterval-t.Sub(t2))
	nbft.blockTimer.Reset(nbft.options.BlockInterval - t.Sub(t2))
	nbft.blockTimeChan <- t2.Add(-nbft.options.BlockDelay)
}

func (nbft *Nbft) processBlock() {
	nbft.blockTimer.Stop()

	t := <-nbft.blockTimeChan
	nanos := t.UnixNano()
	var cnt int
	var toChain string
	var element sortedlinkedlist.IElement
	next := nbft.list.Iter()
	for elem := next(); elem != nil; elem = next() {
		req := elem.(*Request)
		if req.Time > nanos {
			break
		}
		if toChain == "" {
			toChain = req.ToChain
		} else if req.ToChain != toChain {
			break
		}
		cnt++
		element = elem
		if cnt == nbft.options.BlockSize {
			break
		}
		nbft.lastRequestTime = req.Time
	}

	if element != nil {
		elems := nbft.list.RemoveBefore(element)
		if len(elems) > 0 {
			log.Debugf("Replica %s start nbft service,  %d transactions", nbft.options.ID, len(elems))
			id := nbft.options.Chain + ":" + t.Format(FORMAT)
			reqs := []*Request{}
			for index, elem := range elems {
				reqs = append(reqs, elem.(*Request))
				log.Debugf("Replica %s consensus %s : transaction %d, %v", nbft.options.ID, id, index, elem.(*Request))
			}
			instance := nbft.getInstance(id)
			instance.sendPrePrepare(reqs)
		}
	} else {
		log.Debugf("Replica %s start nbft service,  no transactions", nbft.options.ID)
	}

	if nbft.list.Len() != 0 {
		nbft.resetBlockTimer()
	}
}

// Stop Stop consenter serverice of Noops
func (nbft *Nbft) Stop() {
	if nbft.IsRunning() {
		close(nbft.exit)
	}
}

// RecvTransaction Receive transaction data
func (nbft *Nbft) RecvTransaction(tx consensus.ITransaction) {
	req := &Request{
		Time:        time.Now().UnixNano(),
		Transaction: tx.Serialize(),
		FromChain:   tx.FromChain(),
		ToChain:     tx.ToChain(),
	}
	if req.FromChain == nbft.options.Chain {
		nbft.commitReqChan <- req
		nbft.broadcastChan <- &Broadcast{
			to:      nbft.options.Chain,
			payload: &NbftMessage{Payload: &NbftMessage_Request{Request: req}},
		}
	} else {
		log.Errorf("Replica %s failed to receive transaction, fromchain %s is diff localchain %s", nbft.options.ID, req.FromChain, nbft.options.Chain)
	}
}

// RecvConsensus Receive consensus data
func (nbft *Nbft) RecvConsensus(payload []byte) {
	nbftMessage := &NbftMessage{}
	nbftMessage.Deserialize(payload)
	switch tp := nbftMessage.Payload.(type) {
	case *NbftMessage_Request:
		req := nbftMessage.GetRequest()
		log.Debugf("Replica %s received consensus request", nbft.options.ID)
		if req.FromChain == nbft.options.Chain {
			nbft.commitReqChan <- req
		} else {
			log.Errorf("Replica %s failed to receive transaction, fromchain %s is not localchain %s", nbft.options.ID, req.FromChain, nbft.options.Chain)
		}
	case *NbftMessage_Preprepare:
		preprep := nbftMessage.GetPreprepare()
		instance := nbft.getInstance(preprep.Name)
		if instance != nil {
			log.Debugf("Replica %s received consensus preprepare for consensus %s", nbft.options.ID, preprep.Name)
			instance.recvPrePrepare(preprep)
		} else {
			log.Warnf("Replica %s received preprepare timeout for consensus %s", nbft.options.ID, preprep.Name)
		}
	case *NbftMessage_Prepare:
		prep := nbftMessage.GetPrepare()
		instance := nbft.getInstance(prep.Name)
		if instance != nil {
			log.Debugf("Replica %s received consensus prepare for consensus %s", nbft.options.ID, prep.Name)
			instance.recvPrepare(prep)
		} else {
			log.Warnf("Replica %s received prepare timeout for consensus %s", nbft.options.ID, prep.Name)
		}
	case *NbftMessage_Commit:
		commit := nbftMessage.GetCommit()
		instance := nbft.getInstance(commit.Name)
		if instance != nil {
			log.Debugf("Replica %s received consensus commit for consensus %s", nbft.options.ID, commit.Name)
			instance.recvCommit(commit)
		} else {
			log.Warnf("Replica %s received commit timeout for consensus %s", nbft.options.ID, commit.Name)
		}
	case *NbftMessage_FetchCommitted:
		fetchCommitted := nbftMessage.GetFetchCommitted()
		nbft.recvFetchCommitted(fetchCommitted)
	case *NbftMessage_ReturnCommitted:
		returnCommitted := nbftMessage.GetReturnCommitted()
		nbft.recvReturnCommitted(returnCommitted)
	default:
		log.Warnf("unsupport nbft message type %v ", tp)
	}
}

// BroadcastTransactionChannel Broadcast consensus data
func (nbft *Nbft) BroadcastTransactionChannel() <-chan consensus.ITransaction {
	return nil
}

// BroadcastConsensusChannel Broadcast consensus data
func (nbft *Nbft) BroadcastConsensusChannel() <-chan consensus.IBroadcast {
	return nbft.broadcastChan
}

// CommittedTxsChannel Commit block data
func (nbft *Nbft) CommittedTxsChannel() <-chan *consensus.CommittedTxs {
	return nbft.committedTxsChan
}

func (nbft *Nbft) getInstance(key string) *nbftCore {
	nbft.Lock()
	defer nbft.Unlock()
	instance, ok := nbft.nbftCores[key]
	if ok {
		return instance
	}

	//out date

	instance = newNbftCore(key, nbft.options, nbft.committedReqsChan, nbft.broadcastChan)
	nbft.nbftCores[key] = instance
	time.AfterFunc(nbft.options.BlockTimeout, func() {
		nbft.removeInstance(key)
	})
	return instance
}

func (nbft *Nbft) removeInstance(key string) {
	nbft.Lock()
	defer nbft.Unlock()
	if instance, ok := nbft.nbftCores[key]; ok {
		if !instance.isCommit {
			instance.committedReqsChan <- &Committed{Key: instance.name, Requests: nil}
			log.Warnf("Replica %s is failed for consensus %s, timeout (%d transaction)", nbft.options.ID, instance.name, len(instance.reqs))
		}
		delete(nbft.nbftCores, key)
	}
}

func (nbft *Nbft) iterInstance(function func(string, *nbftCore)) {
	nbft.RLock()
	defer nbft.RUnlock()
	for key, instance := range nbft.nbftCores {
		function(key, instance)
	}
}

func (nbft *Nbft) toCommittedTxs(committed *Committed) *consensus.CommittedTxs {
	txs := []consensus.ITransaction{}
	for _, req := range committed.Requests {
		tx := nbft.stack.NewTransaction()
		if err := tx.Deserialize(req.Transaction); err == nil {
			txs = append(txs, tx)
		}
	}
	return &consensus.CommittedTxs{Time: uint32(time.Unix(0, nbft.time(committed.Key)).Unix()), Transactions: txs}
}

func (nbft *Nbft) time(key string) int64 {
	str := strings.Replace(key, nbft.options.Chain+":", "", 1)
	t, err := time.Parse(FORMAT, str)
	if err != nil {
		log.Panic(err)
	}
	return t.UnixNano()
}

func (nbft *Nbft) recvFetchCommitted(fetchCommitted *FetchCommitted) {
	log.Debugf("Replica %s received fetchCommitted from %s for consensus %s", nbft.options.ID, fetchCommitted.ReplicaID, fetchCommitted.Key)

	if committedReqs, ok := nbft.executedCommittedReqs[fetchCommitted.Key]; ok {
		nbft.broadcastChan <- &Broadcast{
			to:      fetchCommitted.ReplicaID,
			payload: &NbftMessage{Payload: &NbftMessage_ReturnCommitted{ReturnCommitted: &ReturnCommitted{ReplicaID: nbft.options.ID, Committed: committedReqs}}},
		}
		return
	}

	if committedReqs, ok := nbft.unexecuteCommittedReqs[fetchCommitted.Key]; ok {
		nbft.broadcastChan <- &Broadcast{
			to:      fetchCommitted.ReplicaID,
			payload: &NbftMessage{Payload: &NbftMessage_ReturnCommitted{ReturnCommitted: &ReturnCommitted{ReplicaID: nbft.options.ID, Committed: committedReqs}}},
		}
		return
	}
}

func (nbft *Nbft) recvReturnCommitted(returnCommitted *ReturnCommitted) {
	log.Debugf("Replica %s received returnCommitted from %s for consensus %s", nbft.options.ID, returnCommitted.ReplicaID, returnCommitted.Committed.Key)
	t := time.Unix(0, nbft.time(returnCommitted.Committed.Key))
	if !t.After(nbft.lastExecCommittedReqs) {
		return
	}
	if returnCommittedList, ok := nbft.returnCommittedReqsList[returnCommitted.Committed.Key]; ok {
		vote := vote.NewVote()
		for _, treturnCommitted := range returnCommittedList {
			if treturnCommitted.ReplicaID == returnCommitted.ReplicaID {
				return
			}
			vote.Add(treturnCommitted.ReplicaID, treturnCommitted.Committed)
		}
		vote.Add(returnCommitted.ReplicaID, returnCommitted.Committed)
		returnCommittedList = append(returnCommittedList, returnCommitted)
		n, ticket := vote.Voter()
		log.Debugf("Replica %s received returnCommitted from %s for consensus %s, vote %d >= %d", nbft.options.ID, returnCommitted.ReplicaID, returnCommitted.Committed.Key, n, nbft.options.Q)
		if n < nbft.options.Q {
			return
		}
		committed := ticket.(*Committed)
		if committed.Requests == nil {
			committed.Requests = make([]*Request, 0)
		}
		nbft.addUnexecuteCommittedReqs(committed)
		nbft.returnCommittedReqsList[returnCommitted.Committed.Key] = returnCommittedList
	} else {
		returnCommittedList = append(returnCommittedList, returnCommitted)
		nbft.returnCommittedReqsList[returnCommitted.Committed.Key] = returnCommittedList
	}
}

func (nbft *Nbft) addUnexecuteCommittedReqs(committed *Committed) {
	nbft.unexecuteCommittedReqs[committed.Key] = committed
	nbft.executeCommittedReqs()
}

func (nbft *Nbft) addExecutedCommittdReqs(committed *Committed) {
	log.Debugf("Replica %s write block for consensus %s", nbft.options.ID, committed.Key)
	nbft.committedTxsChan <- nbft.toCommittedTxs(committed)
	nbft.executedCommittedReqs[committed.Key] = committed
	delete(nbft.unexecuteCommittedReqs, committed.Key)
}

func (nbft *Nbft) executeCommittedReqs() {
	for key, committed := range nbft.unexecuteCommittedReqs {
		t := time.Unix(0, nbft.time(key))
		if nbft.lastExecCommittedReqs.Equal(time.Time{}) {
			nbft.lastExecCommittedReqs = t.Add(-nbft.options.BlockInterval)
		}
		d := t.Sub(nbft.lastExecCommittedReqs)
		if d == nbft.options.BlockInterval {
			if committed.Requests != nil {
				nbft.addExecutedCommittdReqs(committed)
				nbft.lastExecCommittedReqs = t
			} else {
				nbft.broadcastChan <- &Broadcast{
					to:      nbft.options.Chain,
					payload: &NbftMessage{Payload: &NbftMessage_FetchCommitted{FetchCommitted: &FetchCommitted{ReplicaID: nbft.options.ID, Key: committed.Key}}},
				}
			}
			break
		}
	}

	for key, committed := range nbft.executedCommittedReqs {
		_ = committed
		t := time.Unix(0, nbft.time(key))
		if nbft.lastExecCommittedReqs.Sub(t) > 10*nbft.options.BlockInterval {
			delete(nbft.executedCommittedReqs, key)
		}
	}
}
