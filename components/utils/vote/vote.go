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

package vote

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

//ITicket Interface for ticket of vote
type ITicket interface {
	Serialize() []byte
}

//NewVote Create vote instance
func NewVote() *Vote {
	v := &Vote{}
	v.Clear()
	return v
}

//Vote Defined vote struct
type Vote struct {
	tickets    map[string]ITicket
	hashes     map[string]ITicket
	statistics map[string]int
	sync.RWMutex
}

//Clear Init
func (v *Vote) Clear() {
	v.Lock()
	defer v.Unlock()
	v.tickets = make(map[string]ITicket)
	v.hashes = make(map[string]ITicket)
	v.statistics = make(map[string]int)
}

//Size Len of tickets
func (v *Vote) Size() int {
	v.RLock()
	defer v.RUnlock()
	return len(v.tickets)
}

//Add Add ticket
func (v *Vote) Add(voter string, ticket ITicket) {
	v.Lock()
	defer v.Unlock()
	if _, ok := v.tickets[voter]; ok {
		return
	}
	v.tickets[voter] = ticket
	hash := v.key(ticket)
	v.hashes[hash] = ticket
	v.statistics[hash]++
	return
}

//Voter Get ticket of max num
func (v *Vote) Voter() (int, ITicket) {
	v.RLock()
	defer v.RUnlock()
	if len(v.statistics) == 0 {
		return 0, nil
	}
	max := 0
	var hash string
	for v, n := range v.statistics {
		if max < n {
			max = n
			hash = v
		}
	}
	return max, v.hashes[hash]
}

//VoterByVoter Get ticket of voter
func (v *Vote) VoterByVoter(voter string) (int, ITicket) {
	v.RLock()
	defer v.RUnlock()
	ticket, ok := v.tickets[voter]
	if !ok {
		return 0, nil
	}
	return v.statistics[v.key(ticket)], ticket
}

//VoterByTicket Get num of ticket
func (v *Vote) VoterByTicket(ticket ITicket) int {
	v.RLock()
	defer v.RUnlock()
	return v.statistics[v.key(ticket)]
}

//IterVoter Iter by voter
func (v *Vote) IterVoter(function func(string, ITicket)) {
	v.RLock()
	defer v.RUnlock()
	for voter, ticket := range v.tickets {
		function(voter, ticket)
	}
}

//IterTicket Iter by ticket
func (v *Vote) IterTicket(function func(ITicket, int)) {
	v.RLock()
	defer v.RUnlock()
	for hash, num := range v.statistics {
		function(v.hashes[hash], num)
	}
}

//key Hash of ticket
func (v *Vote) key(ticket ITicket) string {
	hash := sha256.Sum256(ticket.Serialize())
	return hex.EncodeToString(hash[:])

}

func (v *Vote) String() string {
	var str string
	for hash, num := range v.statistics {
		str += fmt.Sprintf("hash: %s , num %d;", hash, num)
	}
	return str
}
