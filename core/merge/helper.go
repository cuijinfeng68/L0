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

package merge

import (
	"fmt"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/blockchain"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/msgnet"
)

// Event merge event
type Event interface{}

// Receiver for processing the event
type Receiver interface {
	ProcessEvent(e Event)
}

type pmHandler interface {
	SendMsgnetMessage(src, dst string, msg msgnet.Message) bool
	Relay(inv types.IInventory)
}

// Helper manages merge service
type Helper struct {
	txMerge  *TxMerge
	txParser *TxParser
	pmSender pmHandler
}

// NewHelper create instance
func NewHelper(ledger *ledger.Ledger, bc *blockchain.Blockchain, pmSender pmHandler, mergeConfig *Config) *Helper {
	config = mergeConfig
	h := &Helper{
		txMerge:  NewTxMerge(ledger),
		txParser: NewTxParser(bc),
		pmSender: pmSender,
	}

	h.txMerge.setReceiver(h)
	h.txParser.setReceiver(h)

	return h
}

// Start starts service
func (h *Helper) Start() {

	h.txMerge.start()
	h.txParser.start()
	log.Infoln("merge start...:")
}

// HandleNetMsg handle msg from msg_net
func (h *Helper) HandleNetMsg(msgType uint8, chainID string, peerID string, event Event) {
	switch msgType {
	case msgnet.ChainMergeTxsMsg:
		h.txParser.recvEvent(chainID, peerID, event)
	case msgnet.ChainAckMergeTxsMsg:
		h.txMerge.recvEvent(event)
	case msgnet.ChainAckedMergeTxsMsg:
		h.txMerge.recvEvent(event)
	}
}

// HandleLocalMsg handle msg from local peers
func (h *Helper) HandleLocalMsg(event Event) {
	h.txMerge.recvEvent(event)
}

// ProcessEvent broadcast msg to peers or msg_net
func (h *Helper) ProcessEvent(event Event) {
	switch et := event.(type) {
	case TxEvent:
		log.Debugln("mergeSendMsgnet: ", " peetID: ", et.peerID, " dstChainID :", et.dstChainID)
		h.pmSender.SendMsgnetMessage(et.peerID, et.dstChainID, et.msg)
	case AckMergeTxEvent:
		tx := new(types.Transaction)
		tx.Deserialize(et.msg.Payload)
		log.Debugln("AckMergeTxEvent: ", " peerID: ", et.peerID, " TxHash: ", tx.Hash().String())
		h.pmSender.SendMsgnetMessage(config.PeerID, h.peerAddress(et.chainID, et.peerID), et.msg)
	case AckMergedTxEvent:
		h.pmSender.SendMsgnetMessage(config.PeerID, h.peerAddress(et.chainID, et.peerID), et.msg)
	case BroadcastAckMergeTxEvent:
		h.pmSender.Relay(et.tx)
	}
}

func (h *Helper) peerAddress(chainID, peerID string) string {
	return fmt.Sprintf("%s:%s", chainID, peerID)
}

// TxEvent for merge txs msg
type TxEvent struct {
	msg        msgnet.Message
	dstChainID string
	peerID     string
}

// AckMergeTxEvent for ack merge msg
type AckMergeTxEvent struct {
	msg     msgnet.Message
	chainID string
	peerID  string
}

// AckMergedTxEvent for ack merged msg
type AckMergedTxEvent struct {
	msg     msgnet.Message
	chainID string
	peerID  string
}

// BroadcastAckMergeTxEvent for broadcast ack merge msg to local peers
type BroadcastAckMergeTxEvent struct {
	tx *types.Transaction
}
