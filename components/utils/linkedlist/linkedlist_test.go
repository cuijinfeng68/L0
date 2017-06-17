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

package linkedlist

import (
	"fmt"
	"testing"
)

type INT int

func (e INT) Compare(v interface{}) int {
	if e < v.(INT) {
		return -1
	} else if e > v.(INT) {
		return 1
	}
	return 0
}

func (e INT) Serialize() []byte {
	str := fmt.Sprintf("%d", e)
	return []byte(str)
}

func Test(t *testing.T) {
	list := NewLinkedList()
	list.Clear()

	list.Add(INT(10))
	list.Add(INT(8))
	list.Add(INT(14))
	list.Add(INT(5))
	list.Add(INT(7))

	fmt.Println("len:", list.Len())

	next := list.Iter()
	for elem := next(); elem != nil; elem = next() {
		fmt.Println(elem.(INT))
	}

	list.Remove(INT(14))
	fmt.Println("len:", list.Len())

	next = list.Iter()
	for elem := next(); elem != nil; elem = next() {
		fmt.Println(elem.(INT))
	}
}
