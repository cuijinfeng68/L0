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

package tcp

import (
	"net"
	"strconv"
	"testing"

	"time"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	pb "github.com/bocheninc/msg-net/protos"
)

var num = 60

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

//测试TCP服务启动，与退出  -- 无客户端
func TestTcpStartAndStop(t *testing.T) {
	s := NewServer("127.0.0.1:8000", newMsg, handleMsgServer)
	go s.Start()
	for i := 0; i < num; i++ {
		time.Sleep(time.Second)
		logger.Infoln(s.String())
	}

	s.Stop()
}

//测试TCP服务启动，与退出 -- 客户端被动关闭
func TestTcpStartAndStop2(t *testing.T) {
	s := NewServer("127.0.0.1:8001", newMsg, handleMsgServer)
	go s.Start()
	time.Sleep(time.Second)

	for i := 0; i < num; i++ {
		c := NewClient("127.0.0.1:8001", newMsg, handleMsgClient)
		c.Connect()
		time.Sleep(time.Second)
	}

	s.Stop()
}

//测试TCP服务启动，与退出 -- 客户端主动关闭
func TestTcpStartAndStopnum(t *testing.T) {
	s := NewServer("127.0.0.1:8002", newMsg, handleMsgServer)
	go s.Start()
	time.Sleep(time.Second)

	for i := 0; i < num; i++ {
		c := NewClient("127.0.0.1:8002", newMsg, handleMsgClient)
		c.Connect()
		time.Sleep(time.Second)
		c.Disconnect()
	}

	s.Stop()
}

//测试TCP服务启动，与退出 -- 部分客户端主动关闭
func TestTcpStartAndStop4(t *testing.T) {
	s := NewServer("127.0.0.1:8003", newMsg, handleMsgServer)
	go s.Start()
	time.Sleep(time.Second)

	for i := 0; i < num; i++ {
		c := NewClient("127.0.0.1:8003", newMsg, handleMsgClient)
		c.Connect()
		time.Sleep(time.Second)
		if i%2 == 0 {
			c.Disconnect()
		}
	}

	s.Stop()
}

//测试TCP服务启动，与退出 -- 客户端发送消息
func TestTcpStartAndStop5(t *testing.T) {
	s := NewServer("127.0.0.1:8004", newMsg, handleMsgServer)
	go s.Start()
	time.Sleep(time.Second)

	c := NewClient("127.0.0.1:8004", newMsg, handleMsgClient)
	c.Connect()
	for i := 0; i < num; i++ {
		msg := &pb.Message{}
		msg.Payload = []byte(strconv.Itoa(i))
		c.Handler.Send(c.conn, msg)
	}

	time.Sleep(time.Second)
	s.Stop()
}
