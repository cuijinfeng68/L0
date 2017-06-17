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
	"reflect"
	"sync"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/blockchain"
	cache "github.com/bocheninc/L0/core/merge/cache"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/msgnet"
)

// PeerTx for where the transaction should be delivered
type PeerTx struct {
	chainID string
	peerID  string
	tx      *types.Transaction
	valid   bool
}

// TxParser for parse tx
type TxParser struct {
	receive     Receiver
	bc          *blockchain.Blockchain
	txCache     *cache.CacheTxs
	txMergeChan chan *PeerTx
	mutux       sync.Mutex
}

// NewTxParser create txParser instance
func NewTxParser(bc *blockchain.Blockchain) *TxParser {
	return &TxParser{
		bc:          bc,
		txCache:     cache.NewCacheTxs(),
		txMergeChan: make(chan *PeerTx, 16),
	}
}

func (tp *TxParser) setReceiver(receiver Receiver) {
	tp.receive = receiver
}

func (tp *TxParser) start() {
	go tp.eventLoop()
}

func (tp *TxParser) callback(peerTx *PeerTx) {
	tp.txMergeChan <- peerTx
}

func (tp *TxParser) sendEvent(event Event) {
	if tp.receive != nil {
		tp.receive.ProcessEvent(event)
	}
}

func (tp *TxParser) recvEvent(chainID, peerID string, event Event) {
	payload := event.(msgnet.Message).Payload
	uploadPayload := new(UploadPayload)
	uploadPayload.Deserialize(payload)
	log.Debugln("parseChainID: ", chainID, " peerID: ", peerID, " uploadPayload: ", *uploadPayload)
	t := SetChainTable(chainID, tp.callback)
	t.AddMergeTxs(chainID, peerID, uploadPayload)
}

func (tp *TxParser) eventLoop() {
	for {
		select {
		case peerTx := <-tp.txMergeChan:
			log.Debug(" ===> peerTx: ", peerTx, " txHash: ", peerTx.tx.Hash().String(), " chainID: ", peerTx.chainID, " peerrID: ", peerTx.peerID)
			go tp.handleMergeTx(peerTx)
		}
	}
}

func (tp *TxParser) handleMergeTx(peerTx *PeerTx) {
	tx := peerTx.tx
	if !peerTx.valid {
		go tp.sendEvent(AckMergedTxEvent{chainID: peerTx.chainID, peerID: peerTx.peerID, msg: msgnet.Message{Cmd: msgnet.ChainAckedMergeTxsMsg, Payload: tx.Serialize()}})
		return
	}
	go tp.sendEvent(AckMergeTxEvent{chainID: peerTx.chainID, peerID: peerTx.peerID, msg: msgnet.Message{Cmd: msgnet.ChainAckMergeTxsMsg, Payload: tx.Serialize()}})

	if tp.isChainCoordinatePoint(tx.FromChain(), tx.ToChain()) {
		tp.mutux.Lock()
		exist, value := tp.txCache.Exists(tx)
		if exist {
			tp.txCache.Add(tx, append(value, []byte(peerTx.chainID)...))
			tp.mutux.Unlock()
			tp.processTransaction(tx)
		} else {
			value := []byte{}
			tp.txCache.Add(tx, append(value, []byte(peerTx.chainID)...))
			tp.mutux.Unlock()
		}

	} else {
		tp.processTransaction(tx)
	}
}

func (tp *TxParser) isChainCoordinatePoint(fromChain, toChain string) bool {
	if reflect.DeepEqual(fromChain, toChain) {
		return true
	}
	return false
}

func (tp *TxParser) processTransaction(tx *types.Transaction) {
	log.Infoln("===> processTransaction", tx.Hash().String())
	tp.bc.ProcessTransaction(tx)
}
