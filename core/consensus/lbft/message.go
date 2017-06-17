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
	"strings"

	"fmt"

	"github.com/golang/protobuf/proto"
)

//Broadcast Define consensus data for broadcast
type Broadcast struct {
	to  string
	msg *Message
}

//To Get target for broadcast
func (broadcast *Broadcast) To() string {
	return broadcast.to
}

//Payload Get consensus data for broadcast
func (broadcast *Broadcast) Payload() []byte {
	return broadcast.msg.Serialize()
}

//Serialize Serialize
func (msg *Message) Serialize() []byte {
	return serialize(msg)
}

//Deserialize Deserialize
func (msg *Message) Deserialize(payload []byte) error {
	return proto.Unmarshal(payload, msg)
}

//Compare Compare, 1 greater, 0 equal , -1 smaller
func (req *Request) Compare(v interface{}) int {
	treq := v.(*Request)
	if req.Nonce > treq.Nonce {
		return 1
	} else if req.Nonce < treq.Nonce {
		return -1
	}
	if req.Time > treq.Time {
		return 1
	} else if req.Time < treq.Time {
		return -1
	}
	reqH := hash(req)
	treqH := hash(treq)
	return strings.Compare(reqH, treqH)
}

//Serialize Serialize
func (req *Request) Serialize() []byte {
	return serialize(req)
}

//Serialize Serialize
func (msg *RequestBatch) Serialize() []byte {
	return serialize(msg)
}

//fromChain from
func (msg *RequestBatch) fromChain() (from string) {
	if len(msg.Requests) == 0 {
		return
	}
	fromChains := map[string]string{}
	for _, req := range msg.Requests {
		from = req.FromChain
		fromChains[req.FromChain] = req.FromChain
		break
	}
	if len(fromChains) != 1 {
		panic("illegal requestBatch")
	}
	return
}

//toChain to
func (msg *RequestBatch) toChain() (to string) {
	if len(msg.Requests) == 0 {
		return
	}
	toChains := map[string]string{}
	for _, req := range msg.Requests {
		to = req.ToChain
		toChains[req.ToChain] = req.ToChain
		break
	}
	if len(toChains) != 1 {
		panic("illegal requestBatch")
	}
	return
}

//key name
func (msg *RequestBatch) key() string {
	keys := make([]string, 3)
	keys[0] = msg.fromChain()
	keys[1] = msg.toChain()
	keys[2] = hash(msg)
	key := strings.Join(keys, "-")
	return key
}

//Serialize Serialize
func (msg *PrePrepare) Serialize() []byte {
	payload := serialize(msg)
	m := &PrePrepare{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Prepare) Serialize() []byte {
	payload := serialize(msg)
	m := &Prepare{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Commit) Serialize() []byte {
	payload := serialize(msg)
	m := &Commit{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *Committed) Serialize() []byte {
	payload := serialize(msg)
	m := &Committed{}
	deserialize(payload, m)
	m.ReplicaID = ""
	return serialize(m)
}

//Serialize Serialize
func (msg *ViewChange) Serialize() []byte {
	payload := serialize(msg)
	m := &ViewChange{}
	deserialize(payload, m)
	m.ReplicaID = ""
	m.Priority = 0
	m.H = 0
	return serialize(m)
}

func (msg *Message) info() string {
	if requestBatch := msg.GetRequestBatch(); requestBatch != nil {
		return "requestBatch"
	} else if preprepare := msg.GetPrePrepare(); preprepare != nil {
		return fmt.Sprintf("preprepare from %s (%s)", preprepare.ReplicaID, preprepare.Name)
	} else if prepare := msg.GetPrepare(); prepare != nil {
		return fmt.Sprintf("prepare from %s (%s)", prepare.ReplicaID, prepare.Name)
	} else if commit := msg.GetCommit(); commit != nil {
		return fmt.Sprintf("commit from %s (%s)", commit.ReplicaID, commit.Name)
	} else if committed := msg.GetCommitted(); committed != nil {
		return fmt.Sprintf("committed from %s (%s)", committed.ReplicaID, committed.Name)
	} else if fecthcommitted := msg.GetFetchCommitted(); fecthcommitted != nil {
		return fmt.Sprintf("fecthcommitted from %s (%d)", fecthcommitted.ReplicaID, fecthcommitted.SeqNo)
	} else if viewchange := msg.GetViewchange(); viewchange != nil {
		return fmt.Sprintf("viewchange from %s", viewchange.ReplicaID)
	} else if nullrequest := msg.GetNullReqest(); nullrequest != nil {
		return fmt.Sprintf("nullrequest from %s", nullrequest.ReplicaID)
	} else {
		return hash(msg)
	}
}
