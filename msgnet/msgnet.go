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

package msgnet

import (
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	msgnet "github.com/bocheninc/msg-net/peer"
)

const (
	ChainTxMsg = iota + 100
	ChainConsensusMsg
	ChainMergeTxsMsg
	ChainAckMergeTxsMsg
	ChainAckedMergeTxsMsg
)

// Message represents the message transfer in msg-net
type Message struct {
	Cmd     uint8
	Payload []byte
}

// Serialize message to bytes
func (m *Message) Serialize() []byte {
	return utils.Serialize(*m)
}

// Deserialize bytes to message
func (m *Message) Deserialize(data []byte) {
	utils.Deserialize(data, m)
}

type (
	// Stack defines send interface
	Stack interface {
		Send(dst string, payload, signature []byte) bool
	}

	// MsgHandler handles the message of the msg-net
	MsgHandler func(src string, dst string, payload, sig []byte) error
)

// NewMsgnet start client msg-net service and returns a msg-net peer
func NewMsgnet(id string, routeAddress []string, fn MsgHandler, logOutPath string) *msgnet.Peer {
	// msg-net services
	if len(routeAddress) > 0 {
		msgnet.SetLogOut(logOutPath)
		msgnetPeer := msgnet.NewPeer(id, routeAddress, fn)
		msgnetPeer.Start()
		log.Debug("Msg-net Service Start ...")
		return msgnetPeer
	}
	return nil
}
