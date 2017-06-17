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
	"bytes"

	"crypto/sha256"

	"strings"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils/vote"
	"github.com/bocheninc/L0/core/consensus"
)

func newNbftCore(name string, options *Options, committedReqsChan chan<- *Committed, broadcastChan chan<- consensus.IBroadcast) *nbftCore {
	instance := &nbftCore{
		name:              name,
		options:           options,
		committedReqsChan: committedReqsChan,
		broadcastChan:     broadcastChan,
	}

	instance.prePrepareVote = vote.NewVote()
	instance.prepareVote = vote.NewVote()
	instance.commitVote = vote.NewVote()
	return instance
}

type nbftCore struct {
	name              string
	options           *Options
	reqs              []*Request
	chains            []string
	committedReqsChan chan<- *Committed
	broadcastChan     chan<- consensus.IBroadcast

	prePrepareVote *vote.Vote
	prepareVote    *vote.Vote
	commitVote     *vote.Vote

	digest           string
	primaryChain     string
	isPassPrePrepare bool
	isPassPrepare    bool
	isCommit         bool
}

func (instance *nbftCore) sendPrePrepare(reqs []*Request) {
	log.Debugf("Replica %s send prePrepare for consensus %s (%d transactions)", instance.options.ID, instance.name, len(reqs))

	instance.reqs = reqs
	chains := make(map[string]string)
	instance.chains = append(instance.chains, instance.options.Chain)
	chains[instance.options.Chain] = instance.options.Chain
	for _, req := range instance.reqs {
		if _, ok := chains[req.ToChain]; !ok {
			chains[req.ToChain] = req.ToChain
			instance.chains = append(instance.chains, req.ToChain)
		}
	}

	buf := bytes.NewBuffer([]byte{})
	for _, tx := range instance.reqs {
		buf.Write(tx.Serialize())
	}
	for _, chain := range instance.chains {
		buf.Write([]byte(chain))
	}
	hash := sha256.Sum256(buf.Bytes())
	instance.digest = string(hash[:])

	preprep := &PrePrepare{
		ReplicaID: instance.options.ID,
		Chain:     instance.options.Chain,
		Quorum:    instance.intersectionQuorum(),
		Name:      instance.name,
		Digest:    instance.digest,
		Chains:    instance.chains,
		Requests:  instance.reqs,
	}
	instance.broadcast(&NbftMessage_Preprepare{Preprepare: preprep})
	instance.recvPrePrepare(preprep)
}

func (instance *nbftCore) recvPrePrepare(preprep *PrePrepare) {
	if instance.isPrimaryChain() && preprep.ReplicaID != instance.options.ID {
		return
	}
	log.Debugf("Replica %s received prePrepare from %s for consensus %s, voted %d", instance.options.ID, preprep.ReplicaID, preprep.Name, instance.prePrepareVote.Size())
	instance.prePrepareVote.Add(preprep.ReplicaID, preprep)
	if instance.isPassPrePrepare == false && instance.maybePrePreparePass() {
		prep := &Prepare{
			ReplicaID: instance.options.ID,
			Chain:     instance.options.Chain,
			Quorum:    instance.intersectionQuorum(),
			Name:      instance.name,
			Digest:    instance.digest,
		}
		log.Debugf("Replica %s send prepare for consensus %s", instance.options.ID, instance.name)
		instance.broadcast(&NbftMessage_Prepare{Prepare: prep})
		instance.recvPrepare(prep)
	}
}

func (instance *nbftCore) recvPrepare(prep *Prepare) {
	log.Debugf("Replica %s received prepare from %s for consensus %s, voted %d", instance.options.ID, prep.ReplicaID, prep.Name, instance.prepareVote.Size())
	instance.prepareVote.Add(prep.ReplicaID, prep)
	if instance.isPassPrepare == false && instance.maybePreparePass() {
		commit := &Commit{
			ReplicaID: instance.options.ID,
			Chain:     instance.options.Chain,
			Quorum:    instance.intersectionQuorum(),
			Name:      instance.name,
			Digest:    instance.digest,
		}
		log.Debugf("Replica %s send commit for consensus %s", instance.options.ID, instance.name)
		instance.broadcast(&NbftMessage_Commit{Commit: commit})
		instance.recvCommit(commit)
	}
}

func (instance *nbftCore) recvCommit(commit *Commit) {
	log.Debugf("Replica %s received commit from %s for consensus %s, voted %d", instance.options.ID, commit.ReplicaID, commit.Name, instance.commitVote.Size())
	instance.commitVote.Add(commit.ReplicaID, commit)
	if instance.isCommit == false && instance.maybeCommitPass() {
		log.Infof("Replica %s succeed to commit for consensus %s (%d transactions)", instance.options.ID, instance.name, len(instance.reqs))
		instance.committedReqsChan <- &Committed{Key: instance.name, Requests: instance.reqs}
	}
}

func (instance *nbftCore) intersectionQuorum() uint64 {
	return uint64(instance.options.Q)
}

func (instance *nbftCore) maybePrePreparePass() bool {
	if !instance.isPrimaryChain() {
		max, ticket := instance.prePrepareVote.Voter()
		preprep := ticket.(*PrePrepare)
		quorum := preprep.Quorum
		log.Debugf("Replica %s preprepare quorum of chain %s for consensus %s: %d >= %d", instance.options.ID, preprep.Chain, instance.name, max, quorum)
		if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
			return false
		}
		instance.reqs = preprep.Requests
		instance.digest = preprep.Digest
	}
	instance.isPassPrePrepare = true
	return true
}

func (instance *nbftCore) maybePreparePass() bool {
	if !instance.isPassPrePrepare {
		return false
	}
	for _, chain := range instance.chains {
		max := 0
		var quorum uint64
		if instance.options.Chain == chain {
			num, ticket := instance.prepareVote.VoterByVoter(instance.options.ID)
			if num != 0 {
				max = num
				quorum = ticket.(*Prepare).Quorum
			}
		} else {
			instance.prepareVote.IterTicket(func(ticket vote.ITicket, num int) {
				prep := ticket.(*Prepare)
				if prep.Chain == chain && num > max {
					max = num
					quorum = prep.Quorum
				}
			})
		}
		log.Debugf("Replica %s prepare quorum of chain %s for consensus %s : voter %s", instance.options.ID, chain, instance.name, instance.prepareVote.String())
		log.Debugf("Replica %s prepare quorum of chain %s for consensus %s : %d >= %d", instance.options.ID, chain, instance.name, max, quorum)
		if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
			return false
		}
	}
	//pass
	instance.isPassPrepare = true
	return true
}

func (instance *nbftCore) maybeCommitPass() bool {
	if !instance.isPassPrepare {
		return false
	}
	for _, chain := range instance.chains {
		max := 0
		var quorum uint64
		if instance.options.Chain == chain {
			num, ticket := instance.commitVote.VoterByVoter(instance.options.ID)
			if num != 0 {
				max = num
				quorum = ticket.(*Commit).Quorum
			}
		} else {
			instance.commitVote.IterTicket(func(ticket vote.ITicket, num int) {
				commit := ticket.(*Commit)
				if commit.Chain == chain && num > max {
					max = num
					quorum = commit.Quorum
				}
			})
		}
		log.Debugf("Replica %s commit quorum of chain %s for consensus %s : vote %s", instance.options.ID, chain, instance.name, instance.commitVote.String())
		log.Debugf("Replica %s commit quorum of chain %s for consensus %s : %d >= %d", instance.options.ID, chain, instance.name, max, quorum)
		if max == 0 || quorum < MINQUORUM || uint64(max) < quorum {
			return false
		}
	}
	instance.isCommit = true
	return true
}

func (instance *nbftCore) broadcast(payload isNbftMessage_Payload) {
	for _, chain := range instance.chains {
		instance.broadcastChan <- &Broadcast{
			to:      chain,
			payload: &NbftMessage{Payload: payload},
		}
	}
}

func (instance *nbftCore) isPrimaryChain() bool {
	return strings.HasPrefix(instance.name, instance.options.Chain+":")
}
