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
	"context"
	"io"
	"net"
	"sync"

	"encoding/json"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
)

//NewClient Create a tcp client instance
func NewClient(address string, newMsg func() common.IMsg, handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error) *Client {
	client := &Client{address: address, newMsg: newMsg, handleMsg: handleMsg}
	return client
}

//Client  Define tcp client class and it can use connect to tcp server specify by address
type Client struct {
	address   string
	newMsg    func() common.IMsg                                    //function that create an IMsg instance which is used to recv data
	handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error //function that how to handle IMsg instance and send data

	conn   net.Conn
	cancel context.CancelFunc
	//ws             *sync.WaitGroup
	common.Handler //supply send and recv function
}

//IsConnected Connected to server or not
func (tc *Client) IsConnected() bool {
	return tc.conn != nil
}

//Connect Connect to tcp server and supply communication
func (tc *Client) Connect() net.Conn {
	if tc.IsConnected() {
		logger.Warnf("client %s already connected to server %s", tc.LocalAddr(), tc.RemoteAddr())
		return nil
	}

	logger.Debugf("client %s try to connect to server %s ...", tc.LocalAddr(), tc.RemoteAddr())
	if _, err := net.ResolveTCPAddr("tcp", tc.address); err != nil {
		logger.Errorf("client %s failed to connect to server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), err)
		return nil
	}
	conn, err := net.Dial("tcp", tc.RemoteAddr())
	if err != nil {
		logger.Errorf("client %s failed to connect to server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), err)
		return nil
	}
	tc.conn = conn
	logger.Infof("client %s information : %s", tc.LocalAddr(), tc.String())
	tc.handleConn()
	return conn
}

//Disconnect Disconnect to tcp server
func (tc *Client) Disconnect() {
	if !tc.IsConnected() {
		logger.Warnf("client %s already disconnected to server %s", tc.LocalAddr(), tc.RemoteAddr())
		return
	}
	logger.Debugf("client %s try to disconnect to server %s ...", tc.LocalAddr(), tc.RemoteAddr())
	tc.cancel()
	//tc.ws.Wait()
	tc.cancel = nil
	//tc.ws = nil
}

//String Get tcp client information
func (tc *Client) String() string {
	m := make(map[string]interface{})
	m["localAddr"] = tc.LocalAddr()
	m["remoteAddr"] = tc.RemoteAddr()
	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v", err)
	}
	return string(bytes)
}

//LocalAddr Get local address of client connection,  "unknow" if not connect to server
func (tc *Client) LocalAddr() string {
	if tc.conn != nil {
		return tc.conn.LocalAddr().String()
	}
	return "unknown"
}

//RemoteAddr Get remote address of client connection
func (tc *Client) RemoteAddr() string {
	if tc.conn != nil {
		return tc.conn.RemoteAddr().String()
	}
	return tc.address
}

func (tc *Client) handleConn() {
	tc.Handler.Init()
	ctx, cancel := context.WithCancel(context.Background())
	tc.cancel = cancel
	//tc.ws = &sync.WaitGroup{}
	go func(ctx context.Context) {
		// tc.ws.Add(1)
		// defer tc.ws.Done()
		ctx0, cancel0 := context.WithCancel(context.Background())
		ws0 := &sync.WaitGroup{}
		go func(ctx context.Context) {
			ws0.Add(1)
			defer ws0.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-tc.RecvChannel():
					if err := tc.handleMsg(tc.conn, tc.SendChannel(), msg); err != nil {
						logger.Errorf("clinet %s failed to handle msg from server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), err)
					}
				case msg := <-tc.SendChannel():
					if _, err := tc.Send(tc.conn, msg); err != nil {
						logger.Errorf("client %s failed to send msg to server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), err)
					} else {
						logger.Debugf("client %s send msg to server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), msg)
					}
				}
			}
		}(ctx0)

		for {
			select {
			case <-ctx.Done():
				cancel0()
				ws0.Wait()
				if err := tc.conn.Close(); err != nil {
					logger.Errorf("client %s failed to disconnect to server %s --- %v.", tc.LocalAddr(), tc.RemoteAddr(), err)
				} else {
					logger.Infof("client %s disconnected to server %s.", tc.LocalAddr(), tc.RemoteAddr())
				}
				tc.conn = nil
				return
			default:
			}
			msg := tc.newMsg()
			switch err := tc.Recv(tc.conn, msg); err {
			case io.EOF:
				cancel0()
				ws0.Wait()
				tc.conn.Close()
				logger.Infof("client %s received close from server %s.", tc.LocalAddr(), tc.RemoteAddr())
				tc.conn = nil
				tc.cancel = nil
				//tc.ws = nil
				return
			case nil:
				tc.RecvChannel() <- msg
				logger.Debugf("client %s received msg from server %s --- %v", tc.LocalAddr(), tc.RemoteAddr(), msg)
			default:
				if opErr, ok := err.(*net.OpError); ok && (opErr.Timeout() || opErr.Temporary()) {
					logger.Debugf("client %s failed to receive msg from server %s --- %v.", tc.LocalAddr(), tc.RemoteAddr(), err)
				} else {
					logger.Errorf("client %s failed to receive msg from server %s --- %v.", tc.LocalAddr(), tc.RemoteAddr(), err)
				}
			}
		}
	}(ctx)
}
