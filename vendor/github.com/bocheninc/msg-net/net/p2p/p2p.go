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

//Package p2p supply p2p nework objects
package p2p

import (
	"encoding/json"
	"net"

	"sync"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	"github.com/bocheninc/msg-net/net/tcp"
)

//NewP2P create p2p instance
func NewP2P(address string, newMsg func() common.IMsg, handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error) *P2P {
	p2p := &P2P{address: address, newMsg: newMsg, handleMsg: handleMsg}
	return p2p
}

//P2P Define p2p class, supply peer to peer network
type P2P struct {
	address   string
	newMsg    func() common.IMsg
	handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error

	server  *tcp.Server
	clients map[net.Conn]*tcp.Client
	sync.RWMutex
}

//IsRunning Running or not for supply services
func (p *P2P) IsRunning() bool {
	return p.server != nil && p.server.IsRunning()
}

//Start Start server for supply services
func (p *P2P) Start() {
	if p.IsRunning() {
		logger.Warnf("server %s is already runing.", p.address)
		return
	}

	p.clients = nil
	p.server = nil
	p.clients = make(map[net.Conn]*tcp.Client)
	p.server = tcp.NewServer(p.address, p.newMsg, p.handleMsg)
	p.server.Start()
}

//Stop Stop server for supply services
func (p *P2P) Stop() {
	if !p.IsRunning() {
		logger.Warnf("server %s is already stopped.", p.address)
		return
	}

	p.server.Stop()
	//p.server = nil
	p.iterFunc(func(conn net.Conn, tc *tcp.Client) {
		go tc.Disconnect()
	})
	p.clients = nil
}

//Connect Connect to tcp server
func (p *P2P) Connect(address string) net.Conn {
	clinet := tcp.NewClient(address, p.newMsg, p.handleMsg)
	if conn := clinet.Connect(); conn != nil {
		p.add(conn, clinet)
		return conn
	}
	return nil
}

//Disconnect Close connection
func (p *P2P) Disconnect(conn net.Conn) {
	if tc := p.remove(conn); tc != nil {
		tc.Disconnect()
	} else {
		p.server.Disconnect(conn)
	}
}

//BroadCastToServer Broadcast msg
func (p *P2P) BroadCastToServer(msg common.IMsg, function func(net.Conn, common.IMsg) error) {
	p.iterFunc(func(conn net.Conn, tc *tcp.Client) {
		if err := function(conn, msg); err != nil {
			logger.Errorf("server %s failed to broadcast msg to %s  --- %v", p.address, conn.RemoteAddr().String(), err)
		}
	})
}

//BroadCastToClient Broadcast msg
func (p *P2P) BroadCastToClient(msg common.IMsg, function func(net.Conn, common.IMsg) error) {
	p.server.BroadCast(msg, function)
}

//String Get tcp server information
func (p *P2P) String() string {
	m := make(map[string]interface{})

	m["address"] = p.address
	f := make([]interface{}, 0)
	p.BroadCastToClient(nil, func(conn net.Conn, msg common.IMsg) error {
		f = append(f, conn.RemoteAddr().String())
		return nil
	})
	m["clients"] = f
	m["client_cnt"] = len(f)

	v := make([]interface{}, 0)
	p.BroadCastToServer(nil, func(conn net.Conn, msg common.IMsg) error {
		v = append(v, conn.RemoteAddr().String())
		return nil
	})
	m["servers"] = v
	m["servers_cnt"] = len(v)

	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v", err)
	}
	return string(bytes)
}

func (p *P2P) add(conn net.Conn, cc *tcp.Client) net.Conn {
	p.Lock()
	defer p.Unlock()
	p.clients[conn] = cc
	return conn

}

func (p *P2P) remove(conn net.Conn) *tcp.Client {
	p.Lock()
	defer p.Unlock()
	tc, ok := p.clients[conn]
	if ok {
		delete(p.clients, conn)
	}
	return tc
}

func (p *P2P) iterFunc(function func(conn net.Conn, cc *tcp.Client)) {
	p.Lock()
	defer p.Unlock()
	cs := []net.Conn{}
	for conn, cc := range p.clients {
		if !cc.IsConnected() {
			cs = append(cs, conn)
		}
	}
	for _, conn := range cs {
		delete(p.clients, conn)
	}

	for conn, cc := range p.clients {
		function(conn, cc)
	}
}
