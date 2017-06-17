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
	"testing"
)

func makeRoute() *Route {
	return &Route{
		netTopology: &NetworkTopology{
			list: []*Link{
				&Link{srcNode: "1", dstNodes: []string{"2", "3", "5"}},
				&Link{srcNode: "2", dstNodes: []string{"1", "3"}},
				&Link{srcNode: "3", dstNodes: []string{"1", "2", "5", "6"}},
				&Link{srcNode: "4", dstNodes: []string{"6"}},
				&Link{srcNode: "5", dstNodes: []string{"1", "3"}},
				&Link{srcNode: "6", dstNodes: []string{"3", "4"}},
			},
		},
		localNode:         "1",
		netTopologyChange: false,
	}
}

func printNetworkTopologyList(list []*Link, t *testing.T) {
	for _, v := range list {
		t.Log(*v)
	}
}

func TestRoute(t *testing.T) {
	route := makeRoute()

	t.Log("test add Link1")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{"4"}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
		t.Log(route.GetNextHop("7"))

	} else {
		t.Log("not update network topology")
	}

	t.Log("test add Link2")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{"4", "2"}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
		t.Log(route.GetNextHop("7"))
	} else {
		t.Log("not update network topology")
	}

	t.Log("test add Link3")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{"6"}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
		t.Log(route.GetNextHop("7"))

	} else {
		t.Log("not update network topology")
	}

	t.Log("test add repeat Link")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{"6"}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
		t.Log(route.GetNextHop("7"))

	} else {
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log("not update network topology")
		t.Log(route.GetNextHop("7"))
	}

	t.Log("test delete Link")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
		t.Log(route.GetNextHop("4"))

	} else {
		t.Log("not update network topology")
	}

	t.Log("test delete repeat Link")
	if route.UpdateNetworkTopology(&Link{srcNode: "7", dstNodes: []string{}}) {
		route.UpdateNextHop()
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log(route.nextHop)
	} else {
		printNetworkTopologyList(route.netTopology.list, t)
		t.Log("not update network topology")
		t.Log(route.GetNextHop("6"))
	}

}
