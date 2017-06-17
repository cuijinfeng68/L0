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
	"errors"
	"sort"

	"sync"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/types"
)

const (
	timeKey string = "timeKey"
)

// Storage represents Merged transactions
type Storage struct {
	sync.Mutex
	dbHandler    *db.BlockchainDB
	columnFamily string
	timeArray    utils.Times
	m            map[uint32]types.Transactions
}

// NewStorage initialization
func NewStorage(db *db.BlockchainDB) *Storage {
	return &Storage{
		dbHandler:    db,
		columnFamily: "storage",
		timeArray:    utils.Times{},
		m:            make(map[uint32]types.Transactions),
	}
}

// ClassifiedTransaction classifies transaction and save in db
func (storage *Storage) ClassifiedTransaction(txs types.Transactions) error {
	storage.Lock()
	defer storage.Unlock()
	array, err := storage.getTxTime()
	if err != nil {
		return err
	}

	storage.timeArray = array

	for _, tx := range txs {
		if err := storage.persistenceTransaction(tx); err != nil {
			return err
		}
	}

	for time, txs := range storage.m {
		if err := storage.dbHandler.Put(storage.columnFamily, utils.Uint32ToBytes(time), utils.Serialize(txs)); err != nil {
			return err
		}
	}

	if err := storage.putTxTime(storage.timeArray); err != nil {
		return err
	}
	return nil
}

// GetMergedTransaction returns to be merged transactions
func (storage *Storage) GetMergedTransaction(delay uint32) (types.Transactions, error) {
	storage.Lock()
	defer storage.Unlock()

	array, err := storage.getTxTime()

	if err != nil {
		return nil, err
	}

	sort.Sort(utils.Times(array))

	var txs types.Transactions

	if !storage.checkTxsIsEnough(delay, array) {
		return nil, nil
	}
	for k, v := range array {
		if v < array[0]+delay {
			tmpTxs, err := storage.getTxByTime(v)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tmpTxs...)

			if err := storage.deleteTxByTime(v); err != nil {
				return nil, err
			}
		} else {
			if err := storage.putTxTime(array[k:]); err != nil {
				return nil, err
			}
			break
		}

		if k == len(array)-1 {
			if err := storage.putTxTime([]uint32{}); err != nil {
				return nil, err
			}
		}
	}

	return txs, nil
}

func (storage *Storage) checkTxsIsEnough(delay uint32, array utils.Times) bool {
	//todo check make better
	if len(array) == 0 {
		return false
	}
	if array[len(array)-1] > array[0]+2*delay {
		return true
	}
	return false
}

// PutTxsHashByMergeTxHash put transactions hashs by merge transaction hash
func (storage *Storage) PutTxsHashByMergeTxHash(mergeTXHash crypto.Hash, txsHashs []crypto.Hash) error {
	if err := storage.dbHandler.Put(storage.columnFamily, mergeTXHash.Bytes(), utils.Serialize(txsHashs)); err != nil {
		return err
	}
	return nil
}

// GetTxsByMergeTxHash get transactions hashs by merge transaction hash
func (storage *Storage) GetTxsByMergeTxHash(mergeTXHash crypto.Hash) ([]crypto.Hash, error) {
	txsHashsBytes, err := storage.dbHandler.Get(storage.columnFamily, mergeTXHash.Bytes())
	if err != nil {
		return nil, err
	}
	if len(txsHashsBytes) == 0 {
		return nil, errors.New("not found txsHashs")
	}
	txsHashs := make([]crypto.Hash, 0)
	utils.Deserialize(txsHashsBytes, &txsHashs)
	return txsHashs, nil
}

func (storage *Storage) persistenceTransaction(tx *types.Transaction) error {

	if !utils.Contain(tx.CreateTime(), storage.timeArray) {
		storage.timeArray = append(storage.timeArray, tx.CreateTime())
	}

	storage.m[tx.CreateTime()] = append(storage.m[tx.CreateTime()], tx)

	return nil
}

func (storage *Storage) getTxTime() (utils.Times, error) {
	timeBytes, err := storage.dbHandler.Get(storage.columnFamily, []byte(timeKey))
	if err != nil {
		return nil, err
	}
	if len(timeBytes) == 0 {
		return utils.Times{}, nil
	}
	array := utils.BytesToUint32Arrary(timeBytes)
	return array, nil
}

func (storage *Storage) putTxTime(array []uint32) error {

	if err := storage.dbHandler.Put(storage.columnFamily, []byte(timeKey), utils.Uint32ArrayToBytes(array)); err != nil {
		return err
	}
	return nil
}

func (storage *Storage) getTxByTime(time uint32) (types.Transactions, error) {
	txsBytes, err := storage.dbHandler.Get(storage.columnFamily, utils.Uint32ToBytes(time))
	if err != nil {
		return nil, err
	}
	txs := make(types.Transactions, 0)
	utils.Deserialize(txsBytes, &txs)
	return txs, nil
}

func (storage *Storage) deleteTxByTime(time uint32) error {
	if err := storage.dbHandler.Delete(storage.columnFamily, utils.Uint32ToBytes(time)); err != nil {
		return err
	}
	return nil
}
