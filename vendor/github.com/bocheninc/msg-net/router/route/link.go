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

//Link Node Link
type Link struct {
	srcNode  string
	dstNodes []string
}

//NewNodeLink initialization
func NewNodeLink(srcNode string, dstNodes []string) *Link {
	return &Link{srcNode: srcNode, dstNodes: dstNodes}
}

//GetSrcNode get srcNode
func (l *Link) GetSrcNode() string {
	return l.srcNode
}

//GetDstNodes get dstNodes
func (l *Link) GetDstNodes() []string {
	return l.dstNodes
}
