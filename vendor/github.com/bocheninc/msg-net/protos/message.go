// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of msg-net 
// 
// The msg-net is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// The msg-net is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package protos

import (
	"github.com/golang/protobuf/proto"
)

//Serialize serializes message
func (m *Message) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes message
func (m *Message) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil

}

//Serialize serializes router message
func (m *Router) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes router message
func (m *Router) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil
}

//Serialize serializes routers message
func (m *Routers) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes routers message
func (m *Routers) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil
}

//Serialize serializes peer message
func (m *Peer) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes peer message
func (m *Peer) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil
}

//Serialize serializes peers message
func (m *Peers) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes peers message
func (m *Peers) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil
}

//Serialize serializes chainMessage message
func (m *ChainMessage) Serialize() ([]byte, error) {
	msgData, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return msgData, nil
}

//Deserialize deserializes chainMessage message
func (m *ChainMessage) Deserialize(data []byte) error {
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	return nil
}
