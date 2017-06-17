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

func TestVerifyNetWorkTopology(t *testing.T) {
	netTopology := &NetworkTopology{
		list: []*Link{
			&Link{srcNode: "1", dstNodes: []string{"2", "3", "5"}},
			&Link{srcNode: "2", dstNodes: []string{"1", "3", "7"}},
			&Link{srcNode: "3", dstNodes: []string{"1", "2", "5", "6"}},
			&Link{srcNode: "4", dstNodes: []string{"6", "7"}},
			&Link{srcNode: "5", dstNodes: []string{"1", "3"}},
			&Link{srcNode: "6", dstNodes: []string{"3", "4"}},

			&Link{srcNode: "7", dstNodes: []string{"4", "2"}},
		},
	}

	n := netTopology.verifyNetWorkTopology("1")

	t.Log(n.getLinkList())
}
