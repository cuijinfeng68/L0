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
	"sync"
	"testing"
	"time"
)

func TestAsyncSeqNo(t *testing.T) {
	as := newAsyncSeqNo(0)
	ws := &sync.WaitGroup{}

	for i := 10; i > 0; i-- {
		go func(i int) {
			ws.Add(1)
			defer ws.Done()
			if i%3 == 0 {
				time.AfterFunc(time.Second, func() {
					as.wait(uint64(i), func() {
						fmt.Println(i)
					})
				})
			} else {
				as.wait(uint64(i), func() {
					fmt.Println(i)
				})
			}
		}(i)
	}
	ws.Wait()
}
