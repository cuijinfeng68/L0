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
	"sync"
	"time"
)

type CacheItem struct {
	sync.Mutex

	//key: tx.hash, value: tx
	key  string
	data []byte

	//the time of keeping tx in memory
	lifeSpan  time.Duration
	createdOn time.Time

	//callback when item removed from memory, and put item to db
	aboutToExpire func(key interface{})
}

// NewCacheItem returns a newly created CacheItem.
func NewCacheItem(key string, lifeSpan time.Duration, data []byte) *CacheItem {
	t := time.Now()
	return &CacheItem{
		key:           key,
		lifeSpan:      lifeSpan,
		createdOn:     t,
		aboutToExpire: nil,
		data:          data,
	}
}

// CreatedOn returns when this item was added to the cache.
func (item *CacheItem) CreatedOn() time.Time {
	return item.createdOn
}

// LifeSpan returns when this item was added to the cache.
func (item *CacheItem) LifeSpan() time.Duration {
	return item.lifeSpan
}

// Key returns the key of this cached item.
func (item *CacheItem) Key() string {
	return item.key
}

// Data returns the value of this cached item.
func (item *CacheItem) Data() []byte {
	return item.data
}

// SetAboutToExpireCallback
// before the item is about to be removed from the cache.
func (item *CacheItem) SetAboutToExpireCallback(f func(interface{})) {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = f
}
