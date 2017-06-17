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
	"strings"

	"github.com/bocheninc/msg-net/util"
)

//NetworkTopology topological graph
type NetworkTopology struct {
	list []*Link
}

func newNetworkTopology() *NetworkTopology {
	return &NetworkTopology{}
}

func (n *NetworkTopology) linkIsExist(link *Link) bool {

	for _, tmpLink := range n.list {
		if strings.EqualFold(tmpLink.srcNode, link.srcNode) {
			if len(tmpLink.dstNodes) == len(link.dstNodes) {
				for _, dstNode := range tmpLink.dstNodes {
					if util.IsStrExist(dstNode, link.dstNodes) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (n *NetworkTopology) srcNodeIsExist(srcNode string) bool {

	for _, tmpLink := range n.list {
		if strings.EqualFold(tmpLink.srcNode, srcNode) {
			return true
		}
	}
	return false
}

func (n *NetworkTopology) deleteNilLink() {

	for _, tmpLink := range n.list {
		if len(tmpLink.dstNodes) == 0 {
			for k, tLink := range n.list {
				if tLink == tmpLink {
					n.list = append(n.list[:k], n.list[k+1:]...)
				}
			}
		}
	}
}

func (n *NetworkTopology) deleteLink(link *Link) {
	for k, tmpLink := range n.list {
		if strings.EqualFold(tmpLink.srcNode, link.srcNode) {
			n.list = append(n.list[:k], n.list[k+1:]...)
		}
	}
	//delete srcnode from dstNodes
	for k, tmpLink := range n.list {
		for key, dstNode := range tmpLink.dstNodes {
			if strings.EqualFold(dstNode, link.srcNode) {
				n.list[k].dstNodes = append(n.list[k].dstNodes[:key], n.list[k].dstNodes[key+1:]...)
			}
		}
	}

}

func (n *NetworkTopology) addLink(link *Link) {

	n.list = append(n.list, link)
	for k, tmpLink := range n.list {
		for _, tmpDstNode := range link.dstNodes {
			if strings.EqualFold(tmpDstNode, tmpLink.srcNode) {
				n.list[k].dstNodes = append(n.list[k].dstNodes, link.srcNode)
			}
		}
	}

	for _, dstNode := range link.dstNodes {
		if !n.srcNodeIsExist(dstNode) {
			n.list = append(n.list, &Link{srcNode: dstNode, dstNodes: []string{link.srcNode}})
		}
	}
}

func (n *NetworkTopology) getLinkList() []Link {
	tmplist := []Link{}
	for _, v := range n.list {
		tmplist = append(tmplist, *v)
	}
	return tmplist
}

func (n *NetworkTopology) verifyNetWorkTopology(localNode string) *NetworkTopology {

	tmpNetworktopology := &NetworkTopology{}
	link := n.getLink(localNode)
	cache := []string{}
	cache = append(cache, localNode)
	for _, dstNode := range link.dstNodes {
		cache = append(cache, dstNode)
	}
	return n.checkLink(cache, tmpNetworktopology, link, localNode)
}

func (n *NetworkTopology) getLink(srcNode string) *Link {
	for _, tmpLink := range n.list {
		if strings.EqualFold(tmpLink.srcNode, srcNode) {
			return tmpLink
		}
	}
	return nil
}

func (n *NetworkTopology) checkLink(cache []string, nt *NetworkTopology, link *Link, localNode string) *NetworkTopology {
	if nt.linkIsExist(link) {
		return nt
	}
	nt.list = append(nt.list, link)

	for _, dstNode := range link.dstNodes {
		if localNode == link.srcNode {
			n.checkLink(cache, nt, n.getLink(dstNode), localNode)
		}
		if !util.IsStrExist(dstNode, cache) {
			cache = append(cache, dstNode)
			n.checkLink(cache, nt, n.getLink(dstNode), localNode)

		}
	}
	return nt
}
