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

package route

import (
	"errors"
	"strings"

	"math"
	"sync"
)

//Route 路由算法
type Route struct {
	sync.RWMutex
	netTopologyChange bool
	netTopology       *NetworkTopology
	nextHop           map[string]string
	localNode         string
}

//NewRoute initialization
func NewRoute(localNode string) *Route {
	return &Route{
		netTopologyChange: false,
		netTopology:       newNetworkTopology(),
		localNode:         localNode,
	}
}

//UpdateNetworkTopology Update Network Topology
func (r *Route) UpdateNetworkTopology(link *Link) bool {
	r.RLock()
	defer r.RUnlock()
	if r.netTopology.linkIsExist(link) {
		return false
	}

	if len(link.dstNodes) == 0 {
		if !r.netTopology.srcNodeIsExist(link.srcNode) {
			return false
		}
	}

	r.netTopology.deleteLink(link)

	if len(link.dstNodes) != 0 {
		r.netTopology.addLink(link)
	}

	r.netTopology.deleteNilLink()

	r.netTopologyChange = true
	return true
}

//GetNextHop get next hop
func (r *Route) GetNextHop(dstNode string) (string, error) {
	r.RLock()
	defer r.RUnlock()
	if r.netTopologyChange {
		r.UpdateNextHop()
	}
	if r.nextHop[dstNode] == "" {
		return "", errors.New("not find next-hop ")
	}
	return r.nextHop[dstNode], nil
}

//GetNetworkTopology get Network Topology
func (r *Route) GetNetworkTopology() []Link {
	r.RLock()
	defer r.RUnlock()
	return r.netTopology.getLinkList()
}

//UpdateNextHop update next hop
func (r *Route) UpdateNextHop() {
	r.RLock()
	defer r.RUnlock()

	r.dijkstra()
	r.netTopologyChange = false
}

//INFINITE infinitude
const INFINITE = math.MaxInt64

//dijkstra shortest path simplification algorithm, the same weight of adjacent nodes, the default is 1
func (r *Route) dijkstra() {
	if r.netTopology.getLink(r.localNode) == nil {
		return
	}
	r.nextHop = make(map[string]string)
	cost := make(map[string]int)
	cost[r.localNode] = 0

	netTopology := r.netTopology.verifyNetWorkTopology(r.localNode)
	tmpNetTopology := &NetworkTopology{}
	for _, tLink := range netTopology.list {
		link := *tLink
		tmpNetTopology.list = append(tmpNetTopology.list, &link)
	}
	//logger.Info("---", r.netTopology.getLinkList())
	//logger.Info("---tmp", tmpNetTopology.getLinkList())
	for _, tmpLink := range tmpNetTopology.list {
		if !strings.EqualFold(tmpLink.srcNode, r.localNode) {
			cost[tmpLink.srcNode] = INFINITE
		}
	}

	var tempNode string
	for {
		if len(tmpNetTopology.list) == 0 {
			break
		}

		min := INFINITE
		for _, tmpLink := range tmpNetTopology.list {
			if cost[tmpLink.srcNode] < min {
				min = cost[tmpLink.srcNode]
				tempNode = tmpLink.srcNode
			}
		}

		for k, tmpLink := range tmpNetTopology.list {
			if strings.EqualFold(tmpLink.srcNode, tempNode) {
				tmpNetTopology.list = append(tmpNetTopology.list[:k], tmpNetTopology.list[k+1:]...)
			}
		}

		for _, tmpLink := range netTopology.list {
			if strings.EqualFold(tmpLink.srcNode, tempNode) {
				for _, dstNode := range tmpLink.dstNodes {
					if cost[tempNode]+1 < cost[dstNode] {
						cost[dstNode] = cost[tempNode] + 1
						if tempNode == r.localNode {
							r.nextHop[dstNode] = dstNode
						} else {
							r.nextHop[dstNode] = r.getNextPath(tempNode)
						}
					}
				}
			}
		}

	}
	//logger.Infoln(" nextHop: ", r.nextHop)
}

func (r *Route) getNextPath(node string) string {
	if strings.EqualFold(node, r.nextHop[node]) {
		return node
	}
	return r.getNextPath(r.nextHop[node])
}
