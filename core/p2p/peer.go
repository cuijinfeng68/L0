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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/params"
	"github.com/willf/bloom"
)

var (
	scheme             = "encode"
	delimiter          = "&"
	filterN       uint = 100000
	falsePositive      = 0.0001
)

// PeerID represents the peer identity
type PeerID []byte

type peerMap struct {
	sync.RWMutex
	m map[net.Conn]*Peer
}

func newPeerMap() *peerMap {
	return &peerMap{
		m: make(map[net.Conn]*Peer),
	}
}

func (peers *peerMap) contains(pid PeerID) bool {
	peers.RLock()
	defer peers.RUnlock()

	for _, p := range peers.m {
		if bytes.Equal(p.ID, pid) {
			return true
		}
	}

	return false
}

func (peers *peerMap) get(c net.Conn) (*Peer, bool) {
	peers.RLock()
	defer peers.RUnlock()
	p, ok := peers.m[c]
	return p, ok
}

func (peers *peerMap) set(c net.Conn, p *Peer) {
	peers.Lock()
	defer peers.Unlock()

	peers.m[c] = p
}

func (peers *peerMap) remove(c net.Conn) {
	peers.Lock()
	defer peers.Unlock()

	delete(peers.m, c)
}

func (peers *peerMap) getPeersData(remotePeerID []byte) ([]byte, error) {
	peers.RLock()
	defer peers.RUnlock()

	data := ""
	for _, p := range peers.m {
		if p.Address == "" || strings.HasPrefix(p.Address, ":") {
			continue
		}

		if !bytes.Equal(remotePeerID, p.ID) {
			data += p.String() + delimiter
		}
	}
	return []byte(data), nil
}

func (peers *peerMap) count() int {
	peers.RLock()
	defer peers.RUnlock()

	return len(peers.m)
}

func (peers *peerMap) getPeers() []*Peer {
	peers.RLock()
	defer peers.RUnlock()

	var peerSlice []*Peer
	for _, p := range peers.m {
		if p.Address == "" || strings.HasPrefix(p.Address, ":") {
			continue
		}
		peerSlice = append(peerSlice, p)
	}

	return peerSlice
}

func (p PeerID) String() string {
	return hex.EncodeToString(p)
}

// Peer represents a peer in blockchain
type Peer struct {
	ID             PeerID
	LastActiveTime time.Time
	Address        string
	Conn           net.Conn

	filter  *bloom.BloomFilter
	running map[string]*protoRW
}

// NewPeer returns a new Peer with input id
func NewPeer(id []byte, conn net.Conn, addr string, protocols []Protocol) *Peer {
	protoMap := make(map[string]*protoRW)
	for _, proto := range protocols {
		protoMap[proto.Name] = &protoRW{Protocol: proto, in: make(chan Msg), w: conn}
	}

	return &Peer{
		ID:             PeerID(id),
		LastActiveTime: *new(time.Time),
		Conn:           conn,
		Address:        addr,
		filter:         bloom.NewWithEstimates(filterN, falsePositive),
		running:        protoMap,
	}
}

// String is the representation of a peer as a URL.
func (peer *Peer) String() string {
	u := url.URL{Scheme: scheme}
	u.User = url.User(peer.ID.String())
	u.Host = peer.Address

	_, _, _ = net.SplitHostPort(peer.Address)
	return u.String()
}

// AddFilter adds data to bloomfilter
func (peer *Peer) AddFilter(data []byte) {
	peer.filter.Add(data)
}

// TestFilter tests data
func (peer *Peer) TestFilter(data []byte) bool {
	return peer.filter.Test(data)
}

// GetPeerAddress returns local peer address info
func (peer *Peer) GetPeerAddress() string {
	return fmt.Sprintf("%s:%s", params.ChainID, peer.ID)
}

func (peer *Peer) getProto(cmd uint8) *protoRW {
	if peer != nil && peer.running != nil {
		// log.Debug("getProto ", peer.running)
		for _, rw := range peer.running {
			// log.Debug("getProto ", peer.running, rw != nil, cmd, rw.BaseCmd)
			if rw != nil && cmd > rw.BaseCmd {
				return rw
			}
			return nil
		}
	} else {
		log.Debugf("peer running not exist error %v", peer)
		return nil
	}
	return nil
}

func (peer *Peer) run() {
	//TODO: refactor this
	conn := peer.Conn
	peerManager := getPeerManager()
	for {
		m, err := readMsg(conn)
		if m == nil || err != nil {
			log.Errorf("peer read msg error %s", err)
			peerManager.delPeer <- conn
			break
		}
		//TODO: refactor this to synchronous
		// doHandshake -> doHandleshakeAck after this ... allow [ping, pong, peers, getpeers]
		if msgCmd, ok := msgMap[m.Cmd]; ok {
			log.Debugf("handle message %s, server address:%s", msgCmd, peer.Address)
		}
		// Update the ActiveTime when message reached
		peerManager.alivePeer <- conn

		switch m.Cmd {
		case pingMsg:
			respMsg := NewMsg(pongMsg, nil)
			respMsg.write(peer.Conn)
		case pongMsg:
			log.Debug("Received Pong Message")
		case peersMsg:
			peer.onPeers(m, peerManager)
		case getPeersMsg:
			peer.onGetPeers(m, conn, peerManager)
		default:
			// TODO: refactor this
			if p := peerManager.GetPeer(conn); p != nil || true {
				// pp, ok := pm.peers.get(conn)
				// log.Debugf("connection %v, peer %v, message- %v, peers:%v, peer: %v, ok: %v", conn, p, m.Cmd, pm.peers, pp, ok)
				proto := p.getProto(m.Cmd)
				if proto != nil {
					proto.in <- *m
				}
			} else {
				log.Error("unknown message", p)
				peerManager.delPeer <- conn
				break
			}
		}
		// log.Debugf("handle message over type:%d raddr:%s", m.Cmd, c.conn.RemoteAddr().String())
	}
}

func (peer Peer) onPeers(msg *Msg, pm *peerManager) {
	for _, peerURL := range strings.Split(string(msg.Payload), delimiter) {
		if peerURL == "" {
			continue
		}
		peer, _ := ParsePeer(peerURL)
		pm.dialTask <- peer
	}
}

func (peer Peer) onGetPeers(msg *Msg, w io.Writer, pm *peerManager) {
	peersData, err := pm.peers.getPeersData(msg.Payload)
	if err != nil {
		log.Errorf("PeerManager handle getPeersMsg error %v", err)
	}
	respMsg := NewMsg(peersMsg, peersData)
	respMsg.write(w)
}

// startProtocols starts all sub-protocols
func (peer *Peer) startProtocols() {
	log.Debug("Peer StartProtocols")
	go peer.run()
	for _, proto := range peer.running {
		// log.Debugf("Peer StartProtocols %v: %v,%v", proto.Name, peer.running, proto.Run)
		go func(proto *protoRW) {
			err := proto.Run(peer, proto)
			if err != nil {
				log.Errorf("Peer Handle Protocols error %v", err)
				//TODO: quit
				peerManager := getPeerManager()
				peerManager.delPeer <- peer.Conn
			}
		}(proto)
	}
}

// ParsePeer parses a peer designator.
func ParsePeer(rawurl string) (*Peer, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != scheme {
		return nil, fmt.Errorf("invalid URL scheme, want \"%s\"", scheme)
	}
	// Parse the PeerID from the user portion.
	if u.User == nil {
		return nil, errors.New("does not contain peer ID")
	}
	id, _ := hex.DecodeString(u.User.String())
	return NewPeer(id, nil, u.Host, nil), nil
}

func getPeerAddress(address string) string {
	ip, port, err := net.SplitHostPort(address)

	if ip == "" || err != nil {
		return net.JoinHostPort(GetLocalIP(), port)
	}
	return address
}
