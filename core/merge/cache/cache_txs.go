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

package cache

import (
	"github.com/bocheninc/L0/core/types"
	"sync"
	"time"
)

type CacheTxs struct {
	persistItem *PersistItem
	cacheTable  *CacheTable
}

var (
	cachTxs         *CacheTxs
	once            sync.Once
	defaultLiftSpan = 1800 * time.Second
)

// NewCacheTxs returns a basic db instance
func NewCacheTxs() *CacheTxs {
	once.Do(func() {
		cachTxs = &CacheTxs{
			persistItem: NewPersistItem(),
			cacheTable:  NewCacheTable("cacheTxsTable"),
		}
		// Now we put tx to db and cache together, we only delete item from cache
		//cachTxs.cacheTable.SetAboutToDeleteItemCallback(func(item *CacheItem) {
		//	cachTxs.persistItem.Add(item.Key(), item.Data())
		//})
	})
	return cachTxs
}

//NotFoundAdd add tx if not exists, return if exists
func (ntc *CacheTxs) NotFoundAdd(tx *types.Transaction, chainIDs []byte) (bool, error) {
	if ok, _ := ntc.Exists(tx); ok {
		return ok, nil
	}

	err := ntc.Add(tx, chainIDs)

	return false, err
}

//Exists whether tx exists
func (ntc *CacheTxs) Exists(tx *types.Transaction) (bool, []byte) {
	txKey := tx.SignHash().String()
	if ok, value := ntc.cacheTable.Exists(txKey); ok {
		return true, value
	}

	if ok, value := ntc.persistItem.Exists(txKey); ok {
		return true, value
	}

	return false, nil
}

//Add add new tx, at the same time we put tx to db
func (ntc *CacheTxs) Add(tx *types.Transaction, chainIDs []byte) error {
	err := ntc.cacheTable.Add(tx.SignHash().String(), defaultLiftSpan, chainIDs)
	err = ntc.persistItem.Add(tx.SignHash().String(), chainIDs)
	return err
}

//Del del this tx from cache
func (ntc *CacheTxs) Del(tx *types.Transaction) error {
	var err = error(nil)
	txKey := string(tx.SignHash().Bytes())
	if ok, _ := ntc.cacheTable.Exists(txKey); ok {
		ntc.cacheTable.Del(txKey)
	}
	//ok := ntc.cacheTable.Exists(txKey)
	//if ok {
	//	err = ntc.cacheTable.Del(txKey)
	//}
	//else {
	//	err = ntc.persistItem.Del(txKey)
	//}

	return err
}
