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

import "github.com/golang/protobuf/proto"
import "crypto/sha256"
import "bytes"

// Broadcast Define consensus data for broadcast
type Broadcast struct {
	to      string
	payload *NbftMessage
}

// To Get target for broadcast
func (broadcast *Broadcast) To() string { return broadcast.to }

// Payload Get consensus data for broadcast
func (broadcast *Broadcast) Payload() []byte { return broadcast.payload.Serialize() }

// Compare Compare request, 1 greater, 0 equal , -1 smaller
func (req *Request) Compare(v interface{}) int {
	treq := v.(*Request)
	if req.Time > treq.Time {
		return 1
	} else if req.Time < treq.Time {
		return -1
	}
	hash := sha256.Sum256(req.Serialize())
	thash := sha256.Sum256(treq.Serialize())
	return bytes.Compare(hash[:], thash[:])
}

// Deserialize Request deserialize
func (req *Request) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, req); err != nil {
		panic(err)
	}
}

// Serialize Request serialize
func (req *Request) Serialize() []byte {
	bytes, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}
	return bytes
}

// Deserialize PrePrepare deserialize
func (preprep *PrePrepare) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, preprep); err != nil {
		panic(err)
	}
}

// Serialize PrePrepare serialize
func (preprep *PrePrepare) Serialize() []byte {
	bytes, err := proto.Marshal(preprep)
	if err != nil {
		panic(err)
	}
	tpreprep := &PrePrepare{}
	tpreprep.Deserialize(bytes)
	tpreprep.ReplicaID = ""
	tbytes, err := proto.Marshal(tpreprep)
	return tbytes
}

// Deserialize Prepare deserialize
func (prep *Prepare) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, prep); err != nil {
		panic(err)
	}
}

// Serialize Prepare serialize
func (prep *Prepare) Serialize() []byte {
	bytes, err := proto.Marshal(prep)
	if err != nil {
		panic(err)
	}
	tpre := &Prepare{}
	tpre.Deserialize(bytes)
	tpre.ReplicaID = ""
	tbytes, err := proto.Marshal(tpre)
	return tbytes
}

// Deserialize Commit deserialize
func (commit *Commit) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, commit); err != nil {
		panic(err)
	}
}

// Serialize Commit serialize
func (commit *Commit) Serialize() []byte {
	bytes, err := proto.Marshal(commit)
	if err != nil {
		panic(err)
	}
	tcommit := &Commit{}
	tcommit.Deserialize(bytes)
	tcommit.ReplicaID = ""
	tbytes, err := proto.Marshal(tcommit)
	return tbytes
}

// Deserialize Committed deserialize
func (committed *Committed) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, committed); err != nil {
		panic(err)
	}
}

// Serialize Commit serialize
func (committed *Committed) Serialize() []byte {
	bytes, err := proto.Marshal(committed)
	if err != nil {
		panic(err)
	}
	return bytes
}

// Deserialize NbftMessage deserialize
func (msg *NbftMessage) Deserialize(payload []byte) {
	if err := proto.Unmarshal(payload, msg); err != nil {
		panic(err)
	}
}

// Serialize NbftMessage serialize
func (msg *NbftMessage) Serialize() []byte {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bytes
}
