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
	"crypto/sha256"
	"encoding/hex"

	proto "github.com/golang/protobuf/proto"
)

//Serializer Supply Serialize() interface
type Serializer interface {
	Serialize() []byte
}

func hash(val Serializer) string {
	h := sha256.Sum256(val.Serialize())
	return hex.EncodeToString(h[:])
}

func serialize(msg proto.Message) []byte {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bytes
}

func deserialize(payload []byte, msg proto.Message) {
	err := proto.Unmarshal(payload, msg)
	if err != nil {
		panic(err)
	}
}
