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

type CacheTable struct {
	sync.RWMutex

	name  string
	items map[string]*CacheItem

	cleanupTimer    *time.Timer
	cleanupInterval time.Duration

	// Callback when adding a new item to the cache.
	addedItem func(item *CacheItem)
	// Callback before deleting an item from the cache.
	aboutToDeleteItem func(item *CacheItem)
}

func NewCacheTable(table string) *CacheTable {
	return &CacheTable{
		name:  table,
		items: make(map[string]*CacheItem),
	}
}

// SetAddedItemCallback set Callback for a new item is added to the cache.
func (table *CacheTable) SetAddedItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.addedItem = f
}

// SetAboutToDeleteItemCallback set Callback for an item is about to be removed from the cache.
func (table *CacheTable) SetAboutToDeleteItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = f
}

func (table *CacheTable) NotFoundAdd(key string, liftSpan time.Duration, data []byte) (bool, error) {

	if ok, _ := table.Exists(key); ok {
		return true, nil
	}

	err := table.Add(key, liftSpan, data)
	return false, err
}

func (table *CacheTable) Exists(key string) (bool, []byte) {
	table.RLock()
	defer table.RUnlock()

	item, ok := table.items[key]

	if ok {
		return ok, item.Data()
	}

	return ok, nil
}

func (table *CacheTable) Add(key string, liftSpan time.Duration, data []byte) error {
	item := NewCacheItem(key, liftSpan, data)
	table.Lock()
	table.addInternal(item)

	return nil
}

func (table *CacheTable) Del(key string) error {
	table.Lock()
	_, ok := table.items[key]
	if !ok {
		table.Unlock()
		return ErrKeyNotFound
	}

	defer table.Unlock()
	delete(table.items, key)

	return nil
}

func (table *CacheTable) addInternal(item *CacheItem) {
	table.items[item.key] = item
	addedItem := table.addedItem
	expDur := table.cleanupInterval
	table.cleanupInterval = 10
	table.Unlock()

	if addedItem != nil {
		addedItem(item)
	}

	if expDur == 0 {
		table.expirationCheck()
	}
}

func (table *CacheTable) expirationCheck() {
	table.Lock()

	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}

	aboutToDeleteItem := table.aboutToDeleteItem
	items := table.items
	table.Unlock()

	now := time.Now()
	smallestDuration := 0 * time.Second

	for _, item := range items {
		lifeSpan := item.LifeSpan()
		createdOn := item.CreatedOn()

		if now.Sub(createdOn) > lifeSpan {
			table.Lock()
			table.Del(item.Key())
			table.Unlock()
			if aboutToDeleteItem != nil {
				aboutToDeleteItem(item)
			}
		} else {
			if smallestDuration == 0 || lifeSpan-now.Sub(createdOn) < smallestDuration {
				smallestDuration = lifeSpan - now.Sub(createdOn)
			}
		}
	}

	table.Lock()
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go table.expirationCheck()
		})
	}
	table.Unlock()
}
