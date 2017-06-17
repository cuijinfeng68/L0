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
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
)

// Config is the p2p network configuration
type Config struct {
	Address             string
	PrivateKey          *crypto.PrivateKey
	BootstrapNodes      []string
	MaxPeers            int
	ReconnectTimes      int
	ConnectTimeInterval int
	KeepAliveInterval   int
	KeepAliveTimes      int
	MinPeers            int
	Protocols           []Protocol
	RouteAddress        []string
}

var (
	defaultListenAddr = ":20166"
	config            *Config
)

//DefaultConfig defines the default network configuration
func DefaultConfig() *Config {
	return &Config{
		MaxPeers:            8,
		Address:             defaultListenAddr,
		ReconnectTimes:      3,
		ConnectTimeInterval: int(30 * time.Second),
		KeepAliveInterval:   int(15 * time.Second),
		KeepAliveTimes:      30,
		MinPeers:            3,
	}
}

// Server represents a p2p network server
type Server struct {
	Config
	*peerManager

	tcpServer *TCPServer
	quit      chan struct{}
}

// NewServer returns a new p2p server
func NewServer(db *db.BlockchainDB, cfg *Config) *Server {
	dbInstance = db
	config = cfg

	srv := &Server{
		tcpServer: newTCPServer(
			cfg.Address,
		),
		peerManager: getPeerManager(),
	}

	if db == nil || cfg == nil {
		log.Errorln("NewServer: database instance or config instance is nil.")
		return nil
	}

	log.Debugf("P2P Network Server database instance %v", db)
	log.Debugf("P2P Network Server config instance %v", cfg)

	return srv
}

// Start starts a p2p network run as goroutine
func (srv *Server) Start() {
	log.Infoln("P2P Network Server Starting ...")
	srv.init()

	go srv.run()
	srv.tcpServer.listen()
	go srv.peerManager.run()
}

// Sign signs data with node key
func (srv *Server) Sign(data []byte) (*crypto.Signature, error) {
	h := crypto.Sha256(data)

	if config.PrivateKey != nil {
		return config.PrivateKey.Sign(h[:])
	}

	return nil, fmt.Errorf("Node private key not config")
}

// Broadcast broadcasts message to remote peers
func (srv *Server) Broadcast(msg *Msg) {
	srv.peerManager.broadcastCh <- msg
}

func (srv *Server) init() {
	log.Infoln("Net Server initializing")

	if srv.peerManager == nil {
		srv.peerManager = getPeerManager()
	}

	srv.tcpServer.OnNewClient(srv.onNewPeer)
	srv.tcpServer.OnClientClose(srv.onPeerClose)
	// srv.tcpServer.OnNewMessage(srv.onMessage)
}

func (srv *Server) onNewPeer(c *Connection) {
	go func() {
		if err := srv.doHandshake(c); err != nil {
			log.Errorf("Handshake error %s", err)
			srv.onPeerClose(c)
			return
		}
	}()
}

func (srv *Server) onPeerClose(c *Connection) {
	srv.peerManager.delPeer <- c.conn
}

// func (srv *Server) onMessage(c *Connection, msg *Msg) {
// 	// msg.handle(c, srv)
// }

func (srv *Server) run() {
	for {
		select {
		case conn := <-srv.peerManager.clientConn:
			c := newConnection(conn, srv.tcpServer)
			// go c.listen()
			srv.onNewPeer(c)
		}
	}
}

// GetPeers returns all peers info
func (srv *Server) GetPeers() []*Peer {
	return srv.peerManager.GetPeers()
}

// GetLocalPeer returns local peer info
func (srv *Server) GetLocalPeer() *Peer {
	return srv.peerManager.GetLocalPeer()
}

func (srv *Server) doHandshake(c *Connection) error {
	if err := srv.doProtoHandshake(c); err != nil {
		return err
	}

	if err := srv.doEncHandshake(c); err != nil {
		return err
	}

	return nil
}

func (srv Server) doProtoHandshake(c *Connection) error {
	n, err := SendMessage(c.conn, NewMsg(handshakeMsg, GetProtoHandshake().serialize()))
	if n <= 0 || err != nil {
		return err
	}

	if err := srv.readProtoHandshake(c); err != nil {
		return err
	}
	return nil
}

func (srv Server) doEncHandshake(c *Connection) error {
	respMsg := NewMsg(handshakeAckMsg, GetEncHandshake().serialize())
	n, err := SendMessage(c.conn, respMsg)
	if n <= 0 || err != nil {
		return err
	}

	if err := srv.readEncHandshake(c); err != nil {
		return err
	}
	return nil
}

func (srv Server) readProtoHandshake(c *Connection) error {
	m, err := readMsg(c.conn)
	if m == nil && err != nil {
		return err
	}

	proto := &ProtoHandshake{}
	proto.deserialize(m.Payload)

	if !proto.matchProtocol(GetProtoHandshake()) {
		log.Debug("protocol error")
		srv.onPeerClose(c)
		return fmt.Errorf("protocol handshake error")
	}

	if srv.peers.contains(proto.ID) {
		log.Debugf("peer[%x] is already connected", proto.ID)
		srv.onPeerClose(c)
		return fmt.Errorf("peer[%x] is already connected", proto.ID)
	}
	peer := NewPeer(proto.ID, c.conn, proto.SrvAddress, srv.Protocols)
	if !bytes.Equal(proto.ID, peer.ID) {
		log.Errorf("PeerID not match %v != %v", proto.ID, peer.ID)
		return fmt.Errorf("PeerID not match %v != %v", proto.ID, peer.ID)
	}
	srv.handshakings.set(c.conn, peer)
	return nil
}

func (srv Server) readEncHandshake(c *Connection) error {
	m, err := readMsg(c.conn)
	if m == nil && err != nil {
		return err
	}

	respMsg := &Msg{}
	enc := &EncHandshake{}
	enc.deserialize(m.Payload)
	if !enc.matchProtocol(GetEncHandshake()) {
		log.Debugln("encryption verify error")
		srv.onPeerClose(c)
		return fmt.Errorf("Encryption Verify Error")
	}
	if p, ok := srv.handshakings.get(c.conn); ok {
		srv.handshakings.remove(c.conn)
		srv.peerManager.addPeer <- p

		respMsg = NewMsg(handshakeAckMsg, GetEncHandshake().serialize())
		respMsg.write(c.conn)
		return nil

	}
	return fmt.Errorf("handshaking can't find this connection")
}

func readMsg(r io.Reader) (*Msg, error) {
	msg := new(Msg)
	n, err := msg.read(r)
	if err != nil || n == 0 {
		log.Errorf("connection error %s", err)
		return nil, err
	}

	h := crypto.Sha256(msg.Payload)
	if !bytes.Equal(msg.CheckSum[:], h[0:4]) {
		return nil, fmt.Errorf("message checksum error")
	}

	return msg, nil
}
