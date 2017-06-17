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
	"net"
	"time"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/params"
)

const (
	columnFamily = "peer"
)

var (
	dbInstance *db.BlockchainDB
	pm         *peerManager
)

type peerManager struct {
	localPeer    *Peer
	peers        *peerMap
	handshakings *peerMap
	dialings     map[string]bool
	quit         chan struct{}
	addPeer      chan *Peer
	delPeer      chan net.Conn
	alivePeer    chan net.Conn
	// banPeer      chan net.Conn
	// blacklist    []string
	dialTaskDone chan string
	broadcastCh  chan *Msg
	clientConn   chan net.Conn
	dialTask     chan *Peer
}

// getPeerManager returns a peerManager
func getPeerManager() *peerManager {
	if pm == nil {
		pm = &peerManager{
			localPeer: NewPeer(
				config.PrivateKey.Public().Bytes(),
				nil, config.Address, nil),
			peers:        newPeerMap(),
			handshakings: newPeerMap(),
			dialings:     make(map[string]bool),
			quit:         make(chan struct{}, 1),
			addPeer:      make(chan *Peer, 1),
			delPeer:      make(chan net.Conn, 1),
			alivePeer:    make(chan net.Conn, 1),
			// banPeer:      make([]string, 1),
			// addBlackList: make(chan net.Conn, 1),
			broadcastCh:  make(chan *Msg, 10),
			clientConn:   make(chan net.Conn, 8),
			dialTask:     make(chan *Peer, 8),
			dialTaskDone: make(chan string),
		}
		// log.Debugf("local peerinfo %s", pm.localPeer)
	}
	return pm
}

// GetPeer returns a peer according the conn
func (pm *peerManager) GetPeer(conn net.Conn) *Peer {
	if p, ok := pm.peers.get(conn); ok {
		return p
	}
	return nil
}

// GetPeers returns all peers info
func (pm *peerManager) GetPeers() []*Peer {
	return pm.peers.getPeers()
}

// GetLocalPeer returns local peer info
func (pm *peerManager) GetLocalPeer() *Peer {
	return pm.localPeer
}

func (pm *peerManager) stop() {
	pm.savePeers()
	// close(pm.banPeer)
	close(pm.addPeer)
	close(pm.delPeer)
	close(pm.alivePeer)
	close(pm.dialTask)
	dbInstance.Close()
}

func (pm *peerManager) add(peer *Peer) {
	// if utils.Contain(string(peer.ID), pm.blacklist) {
	// 	log.Debugf("Forbid,peer id [%s] in black lispm.", peer.ID)
	// 	peer.Conn.Close()
	// }
	if pm.peers.contains(peer.ID) {
		log.Debugf("Peer [%s] already connected", peer.ID)
		peer.Conn.Close()
		return
	}

	peer.LastActiveTime = time.Now()
	pm.peers.set(peer.Conn, peer)
	log.Infof("Add Peer [%s] Success.", peer)

	// start all protocols
	peer.startProtocols()

	err := dbInstance.Put(columnFamily, peer.ID, []byte(peer.Address))
	if err != nil {
		log.Error(err.Error())
	}
}

func (pm *peerManager) del(conn net.Conn) {
	defer conn.Close()
	if peer, ok := pm.peers.get(conn); ok {
		log.Infof("Delete Peer [%s] Success.", peer)
		err := dbInstance.Delete(columnFamily, peer.ID)
		if err != nil {
			log.Error(err.Error())
		}
		pm.peers.remove(conn)
	}
	pm.handshakings.remove(conn)
}

// func (pm *peerManager) ban(conn net.Conn) {
// 	if peer, ok := pm.peers.get(conn); ok {
// 		pm.blacklist = append(pm.blacklist, string(peer.ID))
// 	}
// 	pm.delPeer <- conn
// }

// process peers option , usually run as goroutine
func (pm *peerManager) run() {
	pm.init()
	log.Infoln("PeerManager Start ...")
	log.Debugf("Local PeerInfo %s", pm.localPeer)

	go pm.connectLoop()
	go pm.broadcastLoop()

	ticker := time.NewTicker(time.Duration(int64(config.KeepAliveInterval)))
	for {
		select {
		case <-pm.quit:
			pm.stop()
		case peer := <-pm.addPeer:
			pm.add(peer)
		case conn := <-pm.delPeer:
			pm.del(conn)
		// case conn := <-pm.banPeer:
		// 	pm.ban(conn)
		case conn := <-pm.alivePeer:
			pm.updateActiveTime(conn)
		case <-ticker.C:
			pm.manage()
		}
	}
}

// init read peers from database and connect it
// if no peers data, connect bootstrap node
func (pm *peerManager) init() {
	if dbInstance == nil {
		log.Fatalln("Error,the database is not initialized.")
	}
	list, err := dbInstance.Get(columnFamily, []byte("peerList"))
	if err != nil {
		log.Errorln("Database get peers error :", err.Error())
	}
	if len(list) > 0 {
		peerList := bytes.Split(list, []byte{'&'})
		for _, peerID := range peerList {
			peerAddr, err := dbInstance.Get(columnFamily, peerID)
			if err != nil {
				log.Errorln(err.Error())
				continue
			}
			peer, _ := ParsePeer(string(peerAddr))
			pm.dialTask <- peer
		}
	} else {
		pm.getPeers()
	}
}

// connectLoop handles dial tasks
func (pm *peerManager) connectLoop() {
	for {
		select {
		case peer := <-pm.dialTask:
			pm.connect(peer)
		case peer := <-pm.dialTaskDone:
			delete(pm.dialings, peer)
		}
	}
}

// broadcastLoop handles brodcast tasks
func (pm *peerManager) broadcastLoop() {
	for {
		select {
		case msg := <-pm.broadcastCh:
			pm.broadcast(msg)
		}
	}
}

// connect connects peer,if success add to connections
func (pm *peerManager) connect(peer *Peer) {
	if bytes.Equal(pm.localPeer.ID, peer.ID) {
		log.Debugf("can ont connect self[%s]", peer.ID)
		return
	}

	if _, ok := pm.dialings[peer.String()]; ok ||
		pm.peers.contains(peer.ID) ||
		pm.handshakings.contains(peer.ID) {
		// log.Debugf("peer [%s] already connected", peer.ID)
		// log.Debugf("%s - %s - %s - %s", ok, pm.peers.contains(peer.ID), pm.handshakings.contains(peer.ID), peer)
		return
	}

	if pm.peers.count() >= config.MaxPeers {
		log.Debugf("connected peer more than max peers.")
		return
	}

	log.Debugf("peer manager try connect : %s", peer)

	// prevent connect a peer many times
	pm.dialings[peer.String()] = true

	go func() {
		defer func() {
			pm.dialTaskDone <- peer.String()
		}()

		for i := 0; i < config.ReconnectTimes; i++ {
			conn := Dial(peer.Address)
			if conn != nil {
				pm.clientConn <- conn
				return
			}

			log.Debugf("Reconnect Peer %v", peer.Address)
			time.Sleep(time.Duration(int64(config.ConnectTimeInterval)))
			if pm.peers.contains(peer.ID) || pm.handshakings.contains(peer.ID) {
				return
			}
		}
	}()

}

// keepAlive manages peers, send ping msg or reconnect
// make sure the minimum peers
func (pm *peerManager) manage() {
	log.Debugf("Peer Info [number: %d]", pm.peers.count())

	// add to test
	params.ConnNums = pm.peers.count()
	params.LocalIp = GetLocalIP()
	//end

	now := time.Now()
	for _, peer := range pm.peers.getPeers() {
		sec := now.Sub(peer.LastActiveTime)
		if int(sec) > config.KeepAliveInterval*config.KeepAliveTimes {
			log.Debugf("Peer Keep Alive Timeout %d > %d, lastActiveTime %v", int(sec), config.KeepAliveInterval*config.KeepAliveTimes, peer.LastActiveTime)
			pm.delPeer <- peer.Conn
			pm.dialTask <- peer
			continue
		}
		if int(sec) > config.KeepAliveInterval {
			msg := NewMsg(pingMsg, nil)
			if n, err := msg.write(peer.Conn); n == 0 || err != nil {
				log.Errorf("Send pingMsg error n: %d, err: %v", n, err)
			}
		}
	}

	if pm.peers.count() < config.MaxPeers {
		pm.getPeers()
	}
}

// getPeers sends getpeers msg to connected peers
// if the number of peers less than minPeers, try to connect bootstrap node
func (pm *peerManager) getPeers() {
	for _, bNode := range config.BootstrapNodes {
		peer, err := ParsePeer(bNode)
		if err != nil {
			log.Errorln(err.Error())
			continue
		}
		pm.dialTask <- peer
	}

	msg := NewMsg(getPeersMsg, pm.localPeer.ID[:])
	pm.broadcastCh <- msg
}

func (pm *peerManager) updateActiveTime(conn net.Conn) {
	// peer, ok := pm.peers.get(conn)
	// log.Debugf("keep alive %s peer %s ok %d", peer.LastActiveTime, peer, ok)
	if peer, ok := pm.peers.get(conn); ok {
		peer.LastActiveTime = time.Now()
		// log.Debugf("keep alive %s", peer.LastActiveTime)
	}
}

// savePeers saves peers to database
func (pm *peerManager) savePeers() {
	// TODO: lcnd exit call this
	log.Debugf("peer manager try to write %d records to database", pm.peers.count())
	if pm.peers.count() == 0 {
		log.Debugln("savePeerList: There is no peer in connections")
		return
	}

	peerList := make([][]byte, pm.peers.count())
	for _, peer := range pm.peers.getPeers() {
		if err := dbInstance.Put(columnFamily, peer.ID, []byte(peer.Address)); err != nil {
			log.Errorf("savePeerList: save peer [%s] to database error %v", peer.ID, err.Error())
			continue
		}
		peerList = append(peerList, peer.ID)
	}

	peers := bytes.Join(peerList, []byte{'&'})

	err := dbInstance.Put(columnFamily, []byte("peerList"), peers)
	if err != nil {
		log.Errorf("savePeerList: save peers to database error %v", err.Error())
	}
}

func (pm *peerManager) broadcast(msg *Msg) {
	if msg != nil {
		for _, peer := range pm.peers.getPeers() {
			log.Debugf("Peer Manager broadcast message %d to peer %s", msg.Cmd, peer.Address)
			// if msg.Cmd <= peersMsg || msg.Cmd == 23 || !peer.TestFilter(msg.CheckSum[:]) {
			if n, err := msg.write(peer.Conn); err != nil {
				log.Errorf("broadcast message write error %d - %v", n, err)
			}
			// } else {
			// 	log.Errorf("Peer Manager broadcast error %d %s", msg.Cmd, peer.Address)
			// }
		}
	} else {
		log.Errorf("broadcast message error, msg is nil %v", msg)
	}
}
