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
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
)

//PersistItem
type PersistItem struct {
	dbHandler    *db.BlockchainDB
	columnFamily string
}

//NewPersistItem initialization
func NewPersistItem() *PersistItem {
	return &PersistItem{
		dbHandler:    db.GetDBInstance(),
		columnFamily: "persistCacheTxs",
	}
}

//Exists whether key exist
func (pi *PersistItem) Exists(key string) (bool, []byte) {
	dbKey := []byte(key)
	valueBytes, err := pi.dbHandler.Get(pi.columnFamily, dbKey)
	if err != nil {
		log.Error(err.Error())
		return false, nil
	}

	return valueBytes != nil, valueBytes
}

//Add add new item
func (pi *PersistItem) Add(key string, value []byte) error {
	dbKey := []byte(key)
	err := pi.dbHandler.Put(pi.columnFamily, dbKey, value)
	if err != nil {
		log.Error(err.Error())
	}

	return err
}

//Del delete the item from db
func (pi *PersistItem) Del(key string) error {
	dbKey := []byte(key)
	err := pi.dbHandler.Delete(pi.columnFamily, dbKey)

	if err != nil {
		log.Error(err.Error())
	}

	return err
}
