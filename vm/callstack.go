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

// callstack ... todo

package vm

import (
	"container/list"
	"math/big"
)

const (
	stateOpTypeDelete = iota
	stateOpTypePut
)

type stateOpfunc struct {
	optype int
	key    string
	value  []byte
}

type stateQueue struct {
	lst      *list.List
	stateMap map[string][]byte
}

func newStateQueue() *stateQueue {
	lst := list.New()
	state := make(map[string][]byte)
	return &stateQueue{lst, state}
}

func (ss *stateQueue) offer(opfunc *stateOpfunc) {
	ss.lst.PushFront(opfunc)
}

func (ss *stateQueue) poll() *stateOpfunc {
	e := ss.lst.Back()
	if e != nil {
		ss.lst.Remove(e)
		return e.Value.(*stateOpfunc)
	}
	return nil
}

type transferOpfunc struct {
	txType uint32
	from   string
	to     string
	amount *big.Int
}

type transferQueue struct {
	lst         *list.List
	balancesMap map[string]int64
}

func newTransferQueue() *transferQueue {
	lst := list.New()
	balances := make(map[string]int64)
	return &transferQueue{lst, balances}
}

func (tq *transferQueue) offer(opfunc *transferOpfunc) {
	tq.lst.PushFront(opfunc)
}

func (tq *transferQueue) poll() *transferOpfunc {
	e := tq.lst.Back()
	if e != nil {
		tq.lst.Remove(e)
		return e.Value.(*transferOpfunc)
	}
	return nil
}
