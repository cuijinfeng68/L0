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

package lbft

import (
	"fmt"
	"sort"
	"sync"
)

func newAsyncSeqNo(start uint64) *asyncSeqNo {
	return &asyncSeqNo{
		start: start,
		seqNo: make(map[uint64]chan struct{}),
	}
}

type asyncSeqNo struct {
	start uint64
	seqNo map[uint64]chan struct{}
	rw    sync.RWMutex
}

func (as *asyncSeqNo) wait(seqNo uint64, function func()) {
	if seqNo <= as.start {
		panic(fmt.Sprintf("wrong seqNo, %d > %d", seqNo, as.start))
	}

	as.rw.Lock()
	if _, ok := as.seqNo[seqNo]; !ok {
		as.seqNo[seqNo] = make(chan struct{})
	}

	c, ok := as.seqNo[seqNo-1]
	if !ok {
		c = make(chan struct{})
		as.seqNo[seqNo-1] = c
	}
	as.rw.Unlock()

	if seqNo == as.start+1 {
	} else {
		if c != nil {
			<-c
		}
	}

	function()

	as.rw.Lock()
	delete(as.seqNo, seqNo-1)
	if c := as.seqNo[seqNo]; c != nil {
		close(c)
		as.seqNo[seqNo] = nil
	}
	as.rw.Unlock()
}

func (as *asyncSeqNo) notify(seqNo uint64) {
	as.rw.Lock()
	defer as.rw.Unlock()
	if c, ok := as.seqNo[seqNo-1]; ok && c != nil {
		close(c)
		as.seqNo[seqNo-1] = nil
	}
}

func (as *asyncSeqNo) notifyAll(seqNo uint64) {
	as.rw.Lock()
	defer as.rw.Unlock()

	keys := Uint64Slice{}
	for n := range as.seqNo {
		keys = append(keys, n)
	}
	sort.Sort(keys)

	for _, n := range keys {
		if n > seqNo {
			break
		}
		if c, ok := as.seqNo[n]; ok && c != nil {
			close(c)
			as.seqNo[n] = nil
		}
	}
}
