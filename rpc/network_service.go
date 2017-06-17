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

package rpc

import (
	"github.com/bocheninc/L0/core/p2p"
)

type INetWorkInfo interface {
	GetPeers() []*p2p.Peer
	GetLocalPeer() *p2p.Peer
}

type Net struct {
	netServer INetWorkInfo
}

func NewNet(netServer INetWorkInfo) *Net {
	return &Net{
		netServer: netServer,
	}
}

func (n *Net) GetPeers(req string, reply *[]string) error {
	peers := n.netServer.GetPeers()
	for _, peer := range peers {
		*reply = append(*reply, peer.String())
	}
	localPeer := n.netServer.GetLocalPeer()
	*reply = append(*reply, localPeer.String())

	return nil
}

func (n *Net) GetLocalPeer(req string, reply *string) error {
	localPeer := n.netServer.GetLocalPeer()
	*reply = localPeer.String()
	return nil
}
