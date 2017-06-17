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

package p2p

import (
	"net"
	"testing"

	"strconv"
	"time"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	pb "github.com/bocheninc/msg-net/protos"
)

var num = 10

var newMsg = func() common.IMsg {
	return &pb.Message{}
}

var handleMsgServer = func(conn net.Conn, send chan<- common.IMsg, msg common.IMsg) error {
	logger.Infoln("server received ", msg)
	msg.(*pb.Message).Payload = []byte("reply")
	send <- msg
	return nil
}

var handleMsgClient = func(conn net.Conn, send chan<- common.IMsg, msg common.IMsg) error {
	logger.Infoln("client received ", msg)
	return nil
}

//Server start and stop
func TestTcpStartAndStop(t *testing.T) {
	p := NewP2P("127.0.0.1:8000", newMsg, handleMsgServer)
	go p.Start()
	for i := 0; i < num; i++ {
		time.Sleep(time.Second)
		logger.Infoln(p.String())
	}

	p.Stop()
}

func TestTcpStartAndStop2(t *testing.T) {
	p := NewP2P("127.0.0.1:8001", newMsg, handleMsgServer)
	go p.Start()
	time.Sleep(time.Second)
	ps := []*P2P{}
	port := 8002
	for i := 0; i < num; i++ {
		p1 := NewP2P("127.0.0.1:"+strconv.Itoa(port), newMsg, handleMsgClient)
		go p1.Start()
		time.Sleep(time.Second)
		conn := p1.Connect("127.0.0.1:8001")
		time.Sleep(time.Second)
		if i%2 == 0 {
			p1.Disconnect(conn)
			ps = append(ps, p1)
		} else {
			p1.Stop()
		}
		time.Sleep(time.Second)
		port++
	}

	for _, p := range ps {
		p.Stop()
	}

	p.Stop()
}

func TestTcpStartAndStop4(t *testing.T) {
	p := NewP2P("127.0.0.1:8001", newMsg, handleMsgServer)
	go p.Start()
	time.Sleep(time.Second)
	ps := []*P2P{}
	port := 8002
	for i := 0; i < num; i++ {
		p1 := NewP2P("127.0.0.1:"+strconv.Itoa(port), newMsg, handleMsgClient)
		go p1.Start()
		time.Sleep(time.Second)
		conn := p.Connect("127.0.0.1:" + strconv.Itoa(port))
		time.Sleep(time.Second)
		if i%2 == 0 {
			p.Disconnect(conn)
			ps = append(ps, p1)
		} else {
			_ = conn
			p1.Stop()
		}
		time.Sleep(time.Second)
		port++
	}

	for _, p := range ps {
		p.Stop()
	}

	p.Stop()
}
