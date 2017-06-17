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
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	cache "github.com/bocheninc/L0/core/merge/cache"
	"github.com/bocheninc/L0/core/types"
)

//DefaultDurationCnt default timeout
const (
	DefaultDurationCnt      = 5 * time.Second
	DefaultStoreCntDuration = 20 * time.Second
)

var (
	allChainCacheTable = make(map[string]*ChainCacheTable)
	mutex              sync.RWMutex
)

//CountDot store the count and flag of transaction
type CountDot struct {
	cnt        uint32
	flag       bool
	createTime uint32
}

//NewCountDot create instance
func NewCountDot() *CountDot {
	return &CountDot{cnt: 0, flag: false, createTime: utils.CurrentTimestamp()}
}

func (cntDot *CountDot) incr() {
	cntDot.cnt++
}

func (cntDot *CountDot) getCnt() uint32 {
	return cntDot.cnt
}

func (cntDot *CountDot) getFlag() bool {
	return cntDot.flag
}

func (cntDot *CountDot) setFlag(flag bool) {
	cntDot.flag = flag
}

//ChainCacheTable manage all tables for sub-chain
type ChainCacheTable struct {
	mutex         sync.RWMutex
	cacheTxsTable *cache.CacheTable
	countDot      map[string]*CountDot
	txCache       *cache.CacheTxs

	sendTxBack func(peerTx *PeerTx)
}

// NewChainCacheTable create instance
func NewChainCacheTable(table string) *ChainCacheTable {
	return &ChainCacheTable{
		cacheTxsTable: cache.NewCacheTable(table),
		countDot:      make(map[string]*CountDot),
		txCache:       cache.NewCacheTxs(),
	}
}

// setTxBack set callback, will call when have a agreement transaction
func (cct *ChainCacheTable) setTxBack(callback func(peerTx *PeerTx)) {
	cct.mutex.Lock()
	defer cct.mutex.Unlock()
	cct.sendTxBack = callback
}

// SetChainTable table == chainID
func SetChainTable(table string, callback func(peerTx *PeerTx)) *ChainCacheTable {
	mutex.RLock()
	t, ok := allChainCacheTable[table]
	mutex.RUnlock()

	if !ok {
		mutex.Lock()
		allChainCacheTable[table] = NewChainCacheTable(table)
		t = allChainCacheTable[table]
		t.setTxBack(callback)
		mutex.Unlock()
	}

	go t.countDotLoop()

	return t
}

func (cct *ChainCacheTable) countDotLoop() {
	ticker := time.NewTicker(DefaultStoreCntDuration)
	for {
		select {
		case <-ticker.C:
			cct.removeTimeOutCountDot()
		}
	}
}

func (cct *ChainCacheTable) removeTimeOutCountDot() {
	cct.mutex.Lock()
	defer cct.mutex.Unlock()
	// to remove timeout count Dot
}

func (cct *ChainCacheTable) hasHandledTx(chainID string, tx *types.Transaction) bool {
	if exist, data := cct.txCache.Exists(tx); exist {
		strData := string(data)
		return strings.Contains(strData, chainID)

	}

	return false
}

func (cct *ChainCacheTable) handleTx(chainID, peerID string, peerNum uint32, duration uint32, tx *types.Transaction) {
	txHash := tx.SignHash().String()
	if _, ok := cct.countDot[txHash]; !ok {
		cct.countDot[txHash] = NewCountDot()
	}

	exist, _ := cct.cacheTxsTable.NotFoundAdd(peerID+txHash, time.Duration(duration)*DefaultDurationCnt, []byte(""))
	if !exist {
		cct.countDot[txHash].incr()
	} else {
		return
	}

	log.Debugln("===> handleTx TxHash: ", tx.Hash().String(), " Cnt: ", cct.countDot[txHash].getCnt(), " chainID: ", chainID, "  peerId: ", peerID, " peerNum: ", peerNum)
	if cct.countDot[txHash].getCnt() > peerNum/2 && !cct.countDot[txHash].getFlag() {
		cct.countDot[txHash].setFlag(true)
		if cct.sendTxBack != nil {
			cct.sendTxBack(&PeerTx{chainID: chainID, peerID: peerID, tx: tx, valid: true})
			delete(cct.countDot, txHash)
		}
	}
}

// AddMergeTxs send peerTx when get the agreement transaction from all sub-peers
// delete txHash-count from cct.countDot when get the agreement transaction
// delete the cache item(peerID+txHash) when timeout = duration * DEFAULT_DURATION_CNT
func (cct *ChainCacheTable) AddMergeTxs(chainID, peerID string, upload *UploadPayload) {
	cct.mutex.Lock()
	defer cct.mutex.Unlock()

	for _, tx := range upload.RepeatMergeTxs {
		if !cct.hasHandledTx(chainID, tx) {
			cct.handleTx(chainID, peerID, upload.PeerNum, upload.MergeDuration, tx)
		} else {
			cct.sendTxBack(&PeerTx{chainID: chainID, peerID: peerID, tx: tx, valid: false})
		}

	}

	for _, tx := range upload.MergeTxs {
		cct.handleTx(chainID, peerID, upload.PeerNum, upload.MergeDuration, tx)
	}
}
