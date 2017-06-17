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
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils/vote"
)

func newLbftCore(name string, lbft *Lbft) *lbftCore {
	lbftCore := &lbftCore{
		name:        name,
		lbft:        lbft,
		msgChan:     make(chan *Message, lbft.options.BufferSize),
		prepareVote: vote.NewVote(),
		commitVote:  vote.NewVote(),
	}
	lbftCore.clsTimeoutTimer = time.NewTimer(2 * lbft.options.BlockTimeout)
	go func() {
		select {
		case <-lbftCore.clsTimeoutTimer.C:
			close(lbftCore.msgChan)
			lbft.removeInstance(name)
		case <-lbftCore.exit:
		}
	}()
	return lbftCore
}

type lbftCore struct {
	name            string
	lbft            *Lbft
	msgChan         chan *Message
	prepareVote     *vote.Vote
	commitVote      *vote.Vote
	clsTimeoutTimer *time.Timer

	isPassPrePrepare bool
	isPassPrepare    bool
	isPassCommit     bool
	requestBatch     *RequestBatch
	fromChain        string
	toChain          string
	seqNo            uint64
	digest           string

	exit      chan struct{}
	isRunnig  bool
	startTime time.Time
	deltaTime [5]time.Duration
}

func (instance *lbftCore) recvMessage(msg *Message) {
	if msg != nil {
		if msg.GetPrePrepare() != nil {
			instance.start()
		}
		instance.msgChan <- msg
	}
}

func (instance *lbftCore) start() {
	if instance.isRunnig {
		log.Warnf("Replica %s core consenter %s alreay started", instance.lbft.options.ID, instance.name)
		return
	}
	instance.isRunnig = true
	instance.exit = make(chan struct{})
	instance.clsTimeoutTimer.Stop()
	timeoutTimer := time.NewTimer(instance.lbft.options.BlockTimeout)
	go func() {
		for {
			select {
			case <-timeoutTimer.C:
				if !instance.isPassCommit {
					if instance.seqNo > instance.lbft.lastSeqNum() {
						log.Debugf("Replica %s send view change for consensus %s : timeout (%d > %d)", instance.lbft.options.ID, instance.name, instance.seqNo, instance.lbft.lastSeqNum())
						if instance.lbft.options.AutoVote {
							instance.lbft.sendViewChange(nil)
						}
					}
				}
			case <-instance.exit:
				close(instance.msgChan)
				instance.exit = nil
				if !instance.isPassCommit {
					if instance.seqNo > instance.lbft.verifySeqNum() {
						log.Errorf("Replica %s failed to verify for consensus %s :  wrong verifySeqNo (%d <= %d),  previous verify failed ", instance.lbft.options.ID, instance.name, instance.seqNo, instance.lbft.verifySeqNum())
					}
					if instance.seqNo > instance.lbft.lastSeqNum() {
						log.Warnf("Replica %s is failed for consensus %s (%d<=%d)", instance.lbft.options.ID, instance.name, instance.seqNo, instance.lbft.lastSeqNum())
					}
				}
				//instance.deltaTime[4] = time.Since(instance.startTime)
				//log.Infof("lbft_core_cost_time(%s)  deltatime(%s,%s,%s,%s) txs(%d)", instance.name, instance.deltaTime[1], instance.deltaTime[2], instance.deltaTime[3], instance.deltaTime[4], len(instance.requestBatch.Requests))
				log.Debugf("Replica %s core consenter %s stopped", instance.lbft.options.ID, instance.name)
				return
			case msg := <-instance.msgChan:
				switch tp := msg.Payload.(type) {
				case *Message_PrePrepare:
					instance.handlePrePrepare(msg.GetPrePrepare())
				case *Message_Prepare:
					instance.handlePrepare(msg.GetPrepare())
				case *Message_Commit:
					instance.handleCommit(msg.GetCommit())
				default:
					log.Warnf("unsupport core consensus message type %v ", tp)
				}
			}
		}
	}()
	instance.startTime = time.Now()
	log.Debugf("Replica %s core consenter %s started", instance.lbft.options.ID, instance.name)
}

func (instance *lbftCore) stop() {
	if !instance.isRunnig {
		log.Warnf("Replica %s core consenter %s alreay stopped", instance.lbft.options.ID, instance.name)
		return
	}
	instance.isRunnig = false
	instance.lbft.prePrepareAsync.notify(instance.seqNo)
	instance.lbft.commitAsync.notify(instance.seqNo)
	close(instance.exit)
}

func (instance *lbftCore) isCross() bool {
	return instance.fromChain != instance.toChain
}

func (instance *lbftCore) handleRequestBatch(seqNo uint64, requestBatch *RequestBatch) {
	instance.seqNo = seqNo
	instance.start()
	if requestBatch.Id == EMPTYBLOCK {
		instance.fromChain = instance.lbft.options.Chain
		instance.toChain = instance.lbft.options.Chain
	} else {
		instance.fromChain = requestBatch.fromChain()
		instance.toChain = requestBatch.toChain()
	}
	var verfiy bool
	instance.lbft.prePrepareAsync.wait(instance.seqNo, func() {
		log.Debugf("Replica %s handle requestBatch for consensus %s : seqNo %d (async preprepare)", instance.lbft.options.ID, instance.name, instance.seqNo)
		instance.waitForVerify()
		if requestBatch.Id != EMPTYBLOCK && instance.fromChain == instance.lbft.options.Chain {
			//instance.lbft.stack.Removes(instance.lbft.toTxs(requestBatch))
			id := requestBatch.Id
			t := time.Now()
			txs := instance.lbft.stack.VerifyTxsInConsensus(instance.lbft.toTxs(requestBatch), true)
			log.Debugf("Replica %s VerifyTxsInConsensus elapsed %s for consensus %s(%d)", instance.lbft.options.ID, time.Now().Sub(t), instance.name, instance.seqNo)
			requestBatch = instance.lbft.toRequestBatch(txs)
			requestBatch.Id = id

			// go func(requestBatch *RequestBatch) {
			// 	tts := instance.lbft.toTxs(requestBatch)
			// 	for _, tt := range tts {
			// 		tx := tt.(*types.Transaction)
			// 		sender := tx.Sender()
			// 		log.Info("xxx verify primay", " ", sender, " ", tx.Nonce(), " ", tx.Hash(), instance.name)
			// 	}
			// }(requestBatch)

			if instance.fromChain != instance.toChain {
				log.Infof("Replica %s broadcast requestBatch message to %s  for consensus %s (%d transactions)", instance.lbft.options.ID, instance.toChain, instance.name, len(requestBatch.Requests))
				instance.lbft.broadcast(instance.toChain, &Message{Payload: &Message_RequestBatch{RequestBatch: requestBatch}})
			}
		}

		instance.lbft.incrVerifySeqNum()
		verfiy = true
	})

	if !verfiy {
		log.Errorf("Replica %s for consensus %s failed to verify %d", instance.lbft.options.ID, instance.name, instance.seqNo)
	}

	log.Infof("Replica %s received requestBatch message for consensus %s (%d transactions) (seqNo %d)", instance.lbft.options.ID, instance.name, len(requestBatch.Requests), instance.seqNo)

	prePrepare := &PrePrepare{
		Name:      instance.name,
		PrimaryID: instance.lbft.options.ID,
		Chain:     instance.lbft.options.Chain,
		ReplicaID: instance.lbft.options.ID,
		SeqNo:     instance.seqNo,
		// Digest:    hash(requestBatch),
		// Quorum:    uint64(instance.lbft.intersectionQuorum()),
		Requests: requestBatch,
	}
	log.Infof("Replica %s send prePrepare message for consensus %s (%d transactions)", instance.lbft.options.ID, instance.name, len(requestBatch.Requests))
	instance.handlePrePrepare(prePrepare)
	instance.lbft.broadcast(instance.lbft.options.Chain, &Message{Payload: &Message_PrePrepare{PrePrepare: prePrepare}})

}

func (instance *lbftCore) handlePrePrepare(preprep *PrePrepare) {
	if preprep == nil {
		return
	}
	if instance.isPassPrePrepare {
		log.Errorf("Replica %s received prePrepare message from %s for consensus %s : alreay exist ", instance.lbft.options.ID, preprep.ReplicaID, instance.name)
		return
	}

	requestBatch := preprep.Requests
	var fromChain, toChain string
	if requestBatch.Id == EMPTYBLOCK {
		fromChain = instance.lbft.options.Chain
		toChain = instance.lbft.options.Chain
	} else {
		fromChain = requestBatch.fromChain()
		toChain = requestBatch.toChain()
	}

	instance.fromChain = fromChain
	instance.toChain = toChain
	instance.seqNo = preprep.SeqNo

	if !instance.lbft.isPrimary() {
		if !instance.lbft.isValid(requestBatch, fromChain == instance.lbft.options.Chain) {
			log.Errorf("Replica %s received requestBatch message  for consensus %s (%d transactions): illegal requestBatch", instance.lbft.options.ID, instance.name, len(requestBatch.Requests))
			return
		}
		var verify bool
		instance.lbft.prePrepareAsync.wait(instance.seqNo, func() {
			log.Debugf("Replica %s handle preprepare for consensus %s : seqNo %d (async preprepare)", instance.lbft.options.ID, instance.name, instance.seqNo)
			instance.waitForVerify()
			if requestBatch.Id != EMPTYBLOCK && instance.lbft.options.Chain == fromChain && instance.seqNo > instance.lbft.seqNum() {
				//instance.lbft.stack.Removes(instance.lbft.toTxs(requestBatch))
				t := time.Now()
				txs := instance.lbft.stack.VerifyTxsInConsensus(instance.lbft.toTxs(requestBatch), false)
				log.Debugf("Replica %s VerifyTxsInConsensus elapsed %s for consensus %s(%d)", instance.lbft.options.ID, time.Now().Sub(t), instance.name, instance.seqNo)
				trequestBatch := instance.lbft.toRequestBatch(txs)
				trequestBatch.Id = requestBatch.Id
				trequestBatch.Time = requestBatch.Time
				// go func(requestBatch *RequestBatch) {
				// 	tts := instance.lbft.toTxs(requestBatch)
				// 	for _, tt := range tts {
				// 		tx := tt.(*types.Transaction)
				// 		sender := tx.Sender()
				// 		log.Info("xxx verify", " ", sender, " ", tx.Nonce(), " ", tx.Hash(), instance.name)
				// 	}
				// }(requestBatch)

				if hash(requestBatch) != hash(trequestBatch) {
					log.Errorf("Replica %s received prePrepare message from %s for consensus %s : different digest (%d==%d)", instance.lbft.options.ID, preprep.ReplicaID, instance.name, len(requestBatch.Requests), len(trequestBatch.Requests))
					return
				}
			}
			instance.lbft.incrVerifySeqNum()
			verify = true
		})
		if !verify {
			log.Errorf("Replica %s for consensus %s failed to verify %d", instance.lbft.options.ID, instance.name, instance.seqNo)
			return
		}
	} else if requestBatch.Id != EMPTYBLOCK {
		if instance.toChain == instance.fromChain {
			instance.lbft.resetEmptyBlockTimer()
		} else {
			log.Debugf("Replica %s start cross chain empty block", instance.lbft.options.ID)
			instance.lbft.softResetEmptyBlockTimer()
		}
	}

	log.Infof("Replica %s received prePrepare message from %s for consensus %s (%d transactions)", instance.lbft.options.ID, preprep.ReplicaID, instance.name, len(requestBatch.Requests))

	instance.requestBatch = requestBatch
	// instance.fromChain = fromChain
	// instance.toChain = toChain
	//instance.seqNo = preprep.SeqNo
	instance.digest = hash(instance.requestBatch)
	instance.isPassPrePrepare = true
	instance.deltaTime[1] = time.Since(instance.startTime)
	prepare := &Prepare{
		Name:      instance.name,
		PrimaryID: instance.lbft.primaryID,
		Chain:     instance.lbft.options.Chain,
		ReplicaID: instance.lbft.options.ID,
		SeqNo:     instance.seqNo,
		Digest:    instance.digest,
		Quorum:    uint64(instance.lbft.intersectionQuorum()),
	}
	log.Infof("Replica %s send prepare message for consensus %s (%d transactions)", instance.lbft.options.ID, instance.name, len(instance.requestBatch.Requests))
	instance.handlePrepare(prepare)
	instance.broadcast(&Message{Payload: &Message_Prepare{Prepare: prepare}})
}

func (instance *lbftCore) handlePrepare(prepare *Prepare) {
	if prepare == nil {
		return
	}
	if instance.isPassPrePrepare {
		if prepare.Chain != instance.fromChain && prepare.Chain != instance.toChain {
			log.Errorf("Replica %s received prepare message from %s for consensus %s: illegal prepare", instance.lbft.options.ID, prepare.ReplicaID, instance.name)
			return
		}

		if prepare.Chain == instance.lbft.options.Chain && prepare.SeqNo != instance.seqNo {
			log.Errorf("Replica %s received prepare message from %s for consensus %s : different seqNo (%d == %d) ", instance.lbft.options.ID, prepare.ReplicaID, instance.name, instance.seqNo, prepare.SeqNo)
			return
		}

		if prepare.Digest != instance.digest {
			log.Errorf("Replica %s received prepare message from %s for consensus %s : different digest ", instance.lbft.options.ID, prepare.ReplicaID, instance.name)
			return
		}
	}

	instance.prepareVote.Add(prepare.ReplicaID, prepare)
	log.Infof("Replica %s received prepare message from %s for consensus %s, voted %d", instance.lbft.options.ID, prepare.ReplicaID, prepare.Name, instance.prepareVote.Size())
	if instance.isPassPrepare == false && instance.maybePreparePass() {
		instance.deltaTime[2] = time.Since(instance.startTime)

		commit := &Commit{
			Name:      instance.name,
			PrimaryID: instance.lbft.primaryID,
			Chain:     instance.lbft.options.Chain,
			ReplicaID: instance.lbft.options.ID,
			SeqNo:     instance.seqNo,
			Digest:    instance.digest,
			Quorum:    uint64(instance.lbft.intersectionQuorum()),
		}
		log.Infof("Replica %s send commit message for consensus %s (%d transactions)", instance.lbft.options.ID, instance.name, len(instance.requestBatch.Requests))
		instance.handleCommit(commit)
		instance.broadcast(&Message{Payload: &Message_Commit{Commit: commit}})
	}
}

func (instance *lbftCore) handleCommit(commit *Commit) {
	if commit == nil {
		return
	}
	if instance.isPassPrePrepare {
		if commit.Chain != instance.fromChain && commit.Chain != instance.toChain {
			log.Errorf("Replica %s received commit message from %s for consensus %s: illegal commit", instance.lbft.options.ID, commit.ReplicaID, instance.name)
			return
		}

		if commit.Chain == instance.lbft.options.Chain && commit.SeqNo != instance.seqNo {
			log.Errorf("Replica %s received prepare message from %s for consensus %s : different seqNo (%d == %d) ", instance.lbft.options.ID, commit.ReplicaID, instance.name, instance.seqNo, commit.SeqNo)
			return
		}

		if commit.Digest != instance.digest {
			log.Errorf("Replica %s received prepare message from %s for consensus %s : different digest ", instance.lbft.options.ID, commit.ReplicaID, instance.name)
			return
		}
	}

	instance.commitVote.Add(commit.ReplicaID, commit)
	log.Infof("Replica %s received commit message from %s for consensus %s, voted %d", instance.lbft.options.ID, commit.ReplicaID, commit.Name, instance.commitVote.Size())

	if instance.isPassCommit == false && instance.maybeCommitPass() {
		instance.deltaTime[3] = time.Since(instance.startTime)
		instance.lbft.commitAsync.wait(instance.seqNo, func() {
			log.Infof("Replica %s succeed to commit for consensus %s (%d transactions)", instance.lbft.options.ID, instance.name, len(instance.requestBatch.Requests))
			ctt := &Committed{
				Name:         instance.name,
				Chain:        instance.lbft.options.Chain,
				ReplicaID:    instance.lbft.options.ID,
				SeqNo:        instance.seqNo,
				RequestBatch: instance.requestBatch,
			}
			instance.lbft.lbftCoreCommittedChan <- ctt
			instance.lbft.broadcast(instance.lbft.options.Chain, &Message{Payload: &Message_Committed{Committed: ctt}})
		})
	}
}

func (instance *lbftCore) maybePreparePass() bool {
	if !instance.isPassPrePrepare {
		return false
	}

	//from chain
	max := 0
	var quorum uint64
	if instance.fromChain == instance.lbft.options.Chain {
		instance.prepareVote.IterTicket(func(ticket vote.ITicket, num int) {
			prepare := ticket.(*Prepare)
			if prepare.Chain == instance.fromChain && prepare.Digest == instance.digest && prepare.SeqNo == instance.seqNo && num > max {
				max = num
				quorum = prepare.Quorum
			}
		})
	} else {
		instance.prepareVote.IterTicket(func(ticket vote.ITicket, num int) {
			prepare := ticket.(*Prepare)
			if prepare.Chain == instance.fromChain && num > max {
				max = num
				quorum = prepare.Quorum
			}
		})
	}

	if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
		return false
	}

	if instance.fromChain != instance.toChain {
		max := 0
		var quorum uint64
		if instance.toChain == instance.lbft.options.Chain {
			instance.prepareVote.IterTicket(func(ticket vote.ITicket, num int) {
				prepare := ticket.(*Prepare)
				if prepare.Chain == instance.toChain && prepare.Digest == instance.digest && prepare.SeqNo == instance.seqNo && num > max {
					max = num
					quorum = prepare.Quorum
				}
			})
		} else {
			instance.prepareVote.IterTicket(func(ticket vote.ITicket, num int) {
				prepare := ticket.(*Prepare)
				if prepare.Chain == instance.toChain && num > max {
					max = num
					quorum = prepare.Quorum
				}
			})
		}
		if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
			return false
		}
	}

	instance.isPassPrepare = true
	return true
}

func (instance *lbftCore) maybeCommitPass() bool {
	if !instance.isPassPrepare {
		return false
	}
	//from chain
	max := 0
	var quorum uint64
	if instance.fromChain == instance.lbft.options.Chain {
		instance.commitVote.IterTicket(func(ticket vote.ITicket, num int) {
			commit := ticket.(*Commit)
			if commit.Chain == instance.fromChain && commit.Digest == instance.digest && commit.SeqNo == instance.seqNo && num > max {
				max = num
				quorum = commit.Quorum
			}
		})
	} else {
		instance.commitVote.IterTicket(func(ticket vote.ITicket, num int) {
			commit := ticket.(*Commit)
			if commit.Chain == instance.fromChain && num > max {
				max = num
				quorum = commit.Quorum
			}
		})
	}
	if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
		return false
	}

	if instance.fromChain != instance.toChain {
		max := 0
		var quorum uint64
		if instance.toChain == instance.lbft.options.Chain {
			instance.commitVote.IterTicket(func(ticket vote.ITicket, num int) {
				commit := ticket.(*Commit)
				if commit.Chain == instance.toChain && commit.Digest == instance.digest && commit.SeqNo == instance.seqNo && num > max {
					max = num
					quorum = commit.Quorum
				}
			})
		} else {
			instance.commitVote.IterTicket(func(ticket vote.ITicket, num int) {
				commit := ticket.(*Commit)
				if commit.Chain == instance.toChain && num > max {
					max = num
					quorum = commit.Quorum
				}
			})
		}

		if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
			return false
		}
	}
	instance.isPassCommit = true
	return true
}

func (instance *lbftCore) broadcast(msg *Message) {
	instance.lbft.broadcast(instance.fromChain, msg)
	if instance.fromChain != instance.toChain {
		instance.lbft.broadcast(instance.toChain, msg)
	}
}

func (instance *lbftCore) waitForVerify() {
	if instance.seqNo <= instance.lbft.verifySeqNum()+1 {
		return
	}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-instance.exit:
			return
		case <-ticker.C:
			if instance.seqNo <= instance.lbft.verifySeqNum()+1 {
				return
			}
		}
	}
}
