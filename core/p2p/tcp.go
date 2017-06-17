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

package p2p

import (
	"net"

	"strings"

	"github.com/bocheninc/L0/components/log"
)

// TCPServer represents a tcp server
type TCPServer struct {
	address string
	core    int

	onNewClient   func(c *Connection)
	onNewMessage  func(c *Connection, msg *Msg)
	onClientClose func(c *Connection)
}

// Connection represents a tcp client
type Connection struct {
	conn   net.Conn
	server *TCPServer
}

// newTCPServer returns a tcp server
func newTCPServer(addr string) *TCPServer {
	srv := new(TCPServer)
	srv.address = addr
	srv.core = 1

	return srv
}

// newConnection returns a new tcp client
func newConnection(conn net.Conn, srv *TCPServer) *Connection {
	return &Connection{conn, srv}
}

// listen listens all tcp.bind and start accept connections.
func (srv *TCPServer) listen() (err error) {
	var (
		bind     string
		listener *net.TCPListener
		addr     *net.TCPAddr
	)
	addrs := strings.Split(srv.address, ",")

	for _, bind = range addrs {
		if addr, err = net.ResolveTCPAddr("tcp4", bind); err != nil {
			log.Errorf("net.ResolveTCPAddr(\"tcp4\", \"%s\") error(%v)", bind, err)
			return
		}
		if listener, err = net.ListenTCP("tcp4", addr); err != nil {
			log.Errorf("net.ListenTCP(\"tcp4\", \"%s\") error(%v)", bind, err)
			return
		}
		// split N core accept
		for i := 0; i < srv.core; i++ {
			go srv.accept(listener)
		}
	}
	return
}

// func (c *Connection) listen() {
// 	for {
// 		msg := new(Msg)
// 		n, err := msg.read(c.conn)

// 		if err != nil && n == 0 {
// 			log.Errorf("connection error %s", err)
// 			c.conn.Close()
// 			c.server.onClientClose(c)
// 			break
// 		}
// 		c.server.onNewMessage(c, msg)
// 	}

// }

func (c *Connection) send(msg []byte) {
	c.conn.Write(msg)
}

func (c *Connection) close() {
	c.conn.Close()
	c.server.onClientClose(c)
}

// Dial connects to a endpoint
func Dial(addr string) net.Conn {
	conn, _ := net.Dial("tcp4", addr)
	return conn
}

// accept accepts connections on the listener and serves requests
// for each incoming connection.
func (srv *TCPServer) accept(lis *net.TCPListener) {
	var (
		conn *net.TCPConn
		err  error
	)
	for {
		if conn, err = lis.AcceptTCP(); err != nil {
			// if listener close then return
			log.Errorf("listener.Accept(\"%s\") error(%v)", lis.Addr().String(), err)
			return
		}

		// handle requests
		log.Debugf("Accept connection %s, %v", conn.RemoteAddr(), conn)
		c := newConnection(conn, srv)
		// go c.listen()
		srv.onNewClient(c)
	}
}

// OnNewClient called when new client connect
func (srv *TCPServer) OnNewClient(callback func(c *Connection)) {
	srv.onNewClient = callback
}

// OnNewMessage called when received a new message from client
func (srv *TCPServer) OnNewMessage(callback func(*Connection, *Msg)) {
	srv.onNewMessage = callback
}

// OnClientClose called when client closed
func (srv *TCPServer) OnClientClose(callback func(c *Connection)) {
	srv.onClientClose = callback
}
