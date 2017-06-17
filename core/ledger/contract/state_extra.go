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

package contract

import (
	"github.com/bocheninc/L0/components/db"
)

type StateExtra struct {
	ContractStateDeltas map[string]*ContractStateDelta
	success             bool
}

func NewStateExtra() *StateExtra {
	return &StateExtra{make(map[string]*ContractStateDelta), false}
}

func (stateExtra *StateExtra) get(scAddr string, key string) []byte {
	contractStateDelta, ok := stateExtra.ContractStateDeltas[scAddr]
	if ok {
		return contractStateDelta.get(EnSmartContractKey(scAddr, key))
	}
	return nil
}

func (stateExtra *StateExtra) set(scAddr string, key string, value []byte) {
	contractStateDelta := stateExtra.getOrCreateContractStateDelta(scAddr)

	contractStateDelta.set(EnSmartContractKey(scAddr, key), value)
	return
}

func (stateExtra *StateExtra) delete(scAddr string, key string) {
	contractStateDelta := stateExtra.getOrCreateContractStateDelta(scAddr)
	contractStateDelta.remove(EnSmartContractKey(scAddr, key))
	return
}

func (stateExtra *StateExtra) getOrCreateContractStateDelta(scAddr string) *ContractStateDelta {
	contractStateDelta, ok := stateExtra.ContractStateDeltas[scAddr]
	if !ok {
		contractStateDelta = newContractStateDelta(scAddr)
		stateExtra.ContractStateDeltas[scAddr] = contractStateDelta
	}
	return contractStateDelta
}

func (stateExtra *StateExtra) getUpdatedContractStateDelta() map[string]*ContractStateDelta {
	return stateExtra.ContractStateDeltas
}

type ContractStateDelta struct {
	contract string
	cacheKVs map[string]*CacheKVs
}

func newContractStateDelta(scAddr string) *ContractStateDelta {
	return &ContractStateDelta{scAddr, make(map[string]*CacheKVs)}
}

func (csd *ContractStateDelta) get(key string) []byte {
	value, ok := csd.cacheKVs[key]
	if ok {
		if value.optype != db.OperationDelete {
			return value.value
		}
	}

	return nil
}

func (csd *ContractStateDelta) set(key string, value []byte) {
	csd.cacheKVs[key] = newCacheKVs(db.OperationPut, key, value)
}

func (csd *ContractStateDelta) remove(key string) {
	csd.cacheKVs[key] = newCacheKVs(db.OperationDelete, key, []byte(""))
}

func (csd *ContractStateDelta) getUpdatedKVs() map[string]*CacheKVs {
	return csd.cacheKVs
}

type CacheKVs struct {
	optype uint
	key    string
	value  []byte
}

func newCacheKVs(typ uint, key string, value []byte) *CacheKVs {
	return &CacheKVs{optype: typ, key: key, value: value}
}
