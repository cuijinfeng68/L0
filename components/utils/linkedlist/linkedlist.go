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
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/bocheninc/L0/components/log"
)

//IElement Interface for element of LinkedList
type IElement interface {
	Serialize() []byte
}

var linkedList *LinkedList

func init() {
	linkedList = NewLinkedList()
}

//NewLinkedList Create LinkedList instance
func NewLinkedList() *LinkedList {
	list := &LinkedList{}
	list.Clear()
	return list
}

//LinkedList Define LinkedList struct
type LinkedList struct {
	list    list.List
	mapping map[string]*list.Element
	sync.RWMutex
}

//Clear Initialize
func Clear() { linkedList.Clear() }

//Clear Initialize
func (lst *LinkedList) Clear() {
	lst.Lock()
	defer lst.Unlock()
	lst.mapping = make(map[string]*list.Element)
	lst.list.Init()
}

//Len Len of elements
func Len() int { return linkedList.Len() }

//Len Len of elements
func (lst *LinkedList) Len() int {
	lst.RLock()
	defer lst.RUnlock()
	return lst.list.Len()
}

//Has Contains
func Has(element IElement) IElement { return linkedList.Has(element) }

//Has Contains
func (lst *LinkedList) Has(element IElement) IElement {
	lst.RLock()
	defer lst.RUnlock()
	key := lst.key(element)
	elem, ok := lst.mapping[key]
	if ok {
		return elem.Value.(IElement)
	}
	return nil
}

//Add Add element
func Add(element IElement) { linkedList.Add(element) }

//Add Add element
func (lst *LinkedList) Add(element IElement) {
	lst.Lock()
	defer lst.Unlock()
	key := lst.key(element)
	if _, ok := lst.mapping[key]; ok {
		return
	}
	lst.mapping[key] = lst.list.PushBack(element)
}

//Remove Remove element
func Remove(element IElement) { linkedList.Remove(element) }

//Remove Remove element
func (lst *LinkedList) Remove(element IElement) {
	lst.Lock()
	defer lst.Unlock()
	key := lst.key(element)
	elem, ok := lst.mapping[key]
	if !ok {
		return
	}
	lst.list.Remove(elem)
	delete(lst.mapping, key)
}

//Removes Remove element
func Removes(elements []IElement) { linkedList.Removes(elements) }

//Removes Remove element
func (lst *LinkedList) Removes(elements []IElement) {
	lst.Lock()
	defer lst.Unlock()
	log.Debugf("txpool removes %d", lst.list.Len())
	for _, element := range elements {
		key := lst.key(element)
		elem, ok := lst.mapping[key]
		if ok {
			lst.list.Remove(elem)
			delete(lst.mapping, key)
		}
	}
	log.Debugf("txpool removes %d %d", lst.list.Len(), len(elements))
}

//RemoveBefore Remove elements before element
func RemoveBefore(element IElement) (elements []IElement) {
	return linkedList.RemoveBefore(element)
}

//RemoveBefore Remove elements before element
func (lst *LinkedList) RemoveBefore(element IElement) (elements []IElement) {
	lst.Lock()
	defer lst.Unlock()
	key := lst.key(element)
	telement, ok := lst.mapping[key]
	if !ok {
		return
	}
	for elem := lst.list.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(IElement) == telement.Value.(IElement) {
			break
		}
		elements = append(elements, elem.Value.(IElement))
	}
	for _, element := range elements {
		key := lst.key(element)
		elem, _ := lst.mapping[key]
		lst.list.Remove(elem)
		delete(lst.mapping, key)
	}
	return
}

//RemoveAll Remove all elements
func RemoveAll() (elements []IElement) { return linkedList.RemoveAll() }

//RemoveAll Remove all elements
func (lst *LinkedList) RemoveAll() (elements []IElement) {
	lst.Lock()
	defer lst.Unlock()
	for elem := lst.list.Front(); elem != nil; elem = elem.Next() {
		elements = append(elements, elem.Value.(IElement))
	}
	for _, element := range elements {
		key := lst.key(element)
		elem, _ := lst.mapping[key]
		lst.list.Remove(elem)
		delete(lst.mapping, key)
	}
	return
}

//Get Get num elements from head
func Get(n int) (elements []IElement) { return linkedList.Get(n) }

//Get Get num elements from head
func (lst *LinkedList) Get(n int) (elements []IElement) {
	lst.RLock()
	defer lst.RUnlock()
	var cnt int
	for elem := lst.list.Front(); elem != nil; elem = elem.Next() {
		cnt++
		elements = append(elements, elem.Value.(IElement))
		if cnt == n {
			break
		}
	}
	return elements
}

//IterElement Iter, thread safe
func IterElement(function func(element IElement) bool) {
	linkedList.IterElement(function)
}

//IterElement Iter, thread safe
func (lst *LinkedList) IterElement(function func(element IElement) bool) bool {
	lst.Lock()
	defer lst.Unlock()
	for elem := lst.list.Front(); elem != nil; elem = elem.Next() {
		if function(elem.Value.(IElement)) {
			return true
		}
	}
	return false
}

//Iter Iter, not thread safe
func Iter() func() IElement { return linkedList.Iter() }

//Iter Iter, not thread safe
func (lst *LinkedList) Iter() func() IElement {
	elem := lst.list.Front()
	return func() IElement {
		if elem != nil {
			element := elem.Value.(IElement)
			elem = elem.Next()
			return element
		}
		return nil
	}
}

func (lst *LinkedList) key(element IElement) string {
	hash := sha256.Sum256(element.Serialize())
	return hex.EncodeToString(hash[:])
}
