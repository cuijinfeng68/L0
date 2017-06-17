// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of msg-net 
// 
// The msg-net is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// The msg-net is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package router

import (
	"encoding/json"
	"strings"
	"sync"

	pb "github.com/bocheninc/msg-net/protos"
)

//NewPeers make new peers struct
func NewPeers() *Peers {
	peers := &Peers{}
	peers.m = make(map[string][]*pb.Peer)
	return peers
}

//Peers peers struct
type Peers struct {
	m map[string][]*pb.Peer
	sync.RWMutex
}

//Update update peers
func (p *Peers) Update(key string, peers []*pb.Peer) {
	p.Lock()
	defer p.Unlock()
	p.m[key] = peers
}

//String returns summary
func (p *Peers) String() string {
	p.RLock()
	defer p.RUnlock()
	bytes, _ := json.Marshal(p.m)
	return string(bytes)
}

//GetKeys gets keys by id
func (p *Peers) GetKeys(id string) (res []string) {
	p.RLock()
	defer p.RUnlock()
	m := make(map[string]string)
	if strings.HasSuffix(id, ":") {
		for k, v := range p.m {
			for _, p := range v {
				if strings.HasPrefix(p.Id, id) {
					m[k] = k
				}
			}
		}
	} else {
		for k, v := range p.m {
			for _, p := range v {
				if p.Id == id {
					m[k] = k
					break
				}
			}
		}
	}

	for k := range m {
		res = append(res, k)
	}
	return res
}
