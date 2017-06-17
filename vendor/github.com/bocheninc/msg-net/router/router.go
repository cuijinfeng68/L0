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

//Package router 提供与tracker，router， peer间通信服务
package router

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"sync"

	"strings"

	"bytes"

	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	"github.com/bocheninc/msg-net/net/p2p"
	pb "github.com/bocheninc/msg-net/protos"
	"github.com/bocheninc/msg-net/router/route"
)

//NewRouter make new router struct
func NewRouter(id, address string) *Router {
	r := &Router{id: id, address: address}
	return r
}

//Router 路由对象，提供路由协议
type Router struct {
	id      string
	address string

	handler    *Handler
	server     *p2p.P2P
	cancelFunc context.CancelFunc
	//ws         *sync.WaitGroup

	connRouters map[string]net.Conn
	routers     map[string]*pb.Router
	rwRouters   sync.RWMutex
	allRouters  *route.Route
	peers       map[string]net.Conn
	rwPeers     sync.RWMutex
	allPeers    *Peers

	msgUnique map[string]time.Time
	rwMsg     sync.RWMutex

	connKeepAlive map[net.Conn]time.Time
	rwKeepAlive   sync.RWMutex

	timerKeepAlive         *time.Timer
	durationKeepAlive      time.Duration
	timerRouters           *time.Timer
	durationRouters        time.Duration
	timerNetworkRouters    *time.Timer
	durationNetworkRouters time.Duration
	timerNetworkPeers      *time.Timer
	durationNetworkPeers   time.Duration
}

//IsRunning Running or not for supply services
func (r *Router) IsRunning() bool {
	return r.cancelFunc != nil
}

//Start Start server for supply services
func (r *Router) Start() {
	if r.IsRunning() {
		logger.Warnf("router %s is already running.", r.address)
		return
	}

	r.handler = NewHandler(r)
	r.server = p2p.NewP2P(r.address, func() common.IMsg {
		return &pb.Message{}
	}, r.handleMsg)

	done := make(chan struct{})

	go StartReport(r)

	//sever start
	go func() {

		r.server.Start()
		close(done)
	}()
	//wait for server start
	for {
		select {
		case <-done:
			return
		default:
		}
		time.Sleep(time.Millisecond)
		if r.server.IsRunning() {
			break
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel
	//r.ws = &sync.WaitGroup{}
	r.connRouters = make(map[string]net.Conn)
	r.routers = make(map[string]*pb.Router)
	r.allRouters = route.NewRoute(r.address)
	r.peers = make(map[string]net.Conn)
	r.allPeers = NewPeers()
	r.msgUnique = make(map[string]time.Time)
	r.connKeepAlive = make(map[net.Conn]time.Time)
	r.handler.fsm.Event("HELLO")

	//keepalive timeout
	r.durationKeepAlive = time.Second * 5
	if d, err := time.ParseDuration(config.GetString("router.timeout.keepalive")); err == nil {
		r.durationKeepAlive = d
	} else {
		logger.Warnf("failed to parse router.timeout.keepalive, set default timeout 5s --- %v", err)
	}
	r.timerKeepAlive = time.NewTimer(r.durationKeepAlive)
	//routers timeout
	r.durationRouters = time.Second * 5
	if d, err := time.ParseDuration(config.GetString("router.timeout.routers")); err == nil {
		r.durationRouters = d
	} else {
		logger.Warnf("failed to parse router.timeout.routers, set default timeout 5s --- %v", err)
	}
	r.timerRouters = time.NewTimer(r.durationRouters)
	//network routers timeout
	r.durationNetworkRouters = time.Second * 5
	if d, err := time.ParseDuration(config.GetString("router.timeout.network.routers")); err == nil {
		r.durationNetworkRouters = d
	} else {
		logger.Warnf("failed to parse router.timeout.network.routers, set default timeout 5s --- %v", err)
	}
	r.timerNetworkRouters = time.NewTimer(r.durationNetworkRouters)
	//network peers timeout
	r.durationNetworkPeers = time.Second * 5
	if d, err := time.ParseDuration(config.GetString("router.timeout.network.peers")); err == nil {
		r.durationNetworkPeers = d
	} else {
		logger.Warnf("failed to parse router.timeout.peers, set default timeout 5s --- %v", err)
	}
	r.timerNetworkPeers = time.NewTimer(r.durationNetworkPeers)

	//connect to discovery routers
	addresses := config.GetStringSlice("router.discovery")
	r.Discovery(addresses)

	// r.ws.Add(1)
	// defer r.ws.Done()
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			r.timerKeepAlive.Stop()
			r.timerRouters.Stop()
			r.timerNetworkPeers.Stop()
			r.timerNetworkRouters.Stop()
			ticker.Stop()
			return
		case <-r.timerKeepAlive.C:
			r.connKeepAliveUpdate(ctx, 2*r.durationKeepAlive)
			r.broadcastMsg(&pb.Message{Type: pb.Message_KEEPALIVE})
			r.timerKeepAlive.Reset(r.durationKeepAlive)
		case <-r.timerRouters.C:
			logger.Debugf("p2p information : %s", r.server.String())
			logger.Infof("router information : %s", r.String())
			r.broadcastRouters()
		case <-r.timerNetworkPeers.C:
			r.broadcastNetworkPeers()
		case <-r.timerNetworkRouters.C:
			r.broadcastNetworkPeers()
		case <-ticker.C:
			r.msgUniqueUpdate(5 * time.Second)
		}
	}
}

//Stop stop server
func (r *Router) Stop() {
	if !r.IsRunning() {
		logger.Warnf("router %s is already stopped.", r.address)
		return
	}

	router := &pb.Router{Id: r.id, Address: r.address}
	bytes, _ := router.Serialize()
	msg := &pb.Message{Type: pb.Message_ROUTER_CLOSE, Payload: bytes}
	r.broadcastMsg(msg)

	r.server.Stop()
	//r.server = nil

	r.cancelFunc()
	//r.ws.Wait()
}

//Discovery discoveries other router and connects them
func (r *Router) Discovery(addresses []string) {
	unDiscovery := []string{}
	for _, address := range addresses {
		if address == r.address {
			continue
		}
		if r.routerExist(address) {
			continue
		}
		if conn := r.server.Connect(address); conn != nil {
			//send hello messge
			router := &pb.Router{Id: r.id, Address: r.address}
			payload, _ := router.Serialize()
			msg := &pb.Message{Type: pb.Message_ROUTER_HELLO, Payload: payload}
			(&common.Handler{}).Send(conn, msg)
		} else {
			unDiscovery = append(unDiscovery, address)
		}
	}
	if len(unDiscovery) > 0 {
		time.AfterFunc(r.durationKeepAlive, func() {
			r.Discovery(unDiscovery)
		})
	}
}

//RouteMessage router message
func (r *Router) RouteMessage(msg *pb.Message) error {
	if bytes.Contains(msg.Metadata, []byte(r.address)) {
		return nil
	}
	msg.Metadata = append(msg.Metadata, []byte(r.address)...)

	chainMsg := &pb.ChainMessage{}
	if err := chainMsg.Deserialize(msg.Payload); err != nil {
		return err
	}

	dstID := chainMsg.DstId
	logger.Debugf("router %s route message %s to dstID %s", r.address, chainMsg.SrcId, dstID)
	keys := r.allPeers.GetKeys(dstID)
	if len(keys) == 0 {
		logger.Errorf("router %s route message  %s to dstID %s failed ", r.address, chainMsg.SrcId, dstID)
	}
	for _, key := range keys {
		if key == r.address {
			r.peerIterFunc(func(peer *pb.Peer, conn net.Conn) {
				if (strings.HasSuffix(dstID, ":") && strings.HasPrefix(peer.Id, dstID)) || peer.Id == dstID {
					logger.Debugf("router %s route message %s to dstID %s (%s) successfully", r.address, chainMsg.SrcId, dstID, peer.Id)
					(&common.Handler{}).Send(conn, msg)
				}
			})
		} else {
			nextKey, err := r.allRouters.GetNextHop(key)
			if err != nil {
				logger.Warnf("get next hop err: %s ", err)
			} else {
				logger.Debugf("router %s route message %s to dstID %s in next %s", r.address, chainMsg.SrcId, dstID, nextKey)

				r.rwRouters.RLock()
				if conn, ok := r.connRouters[nextKey]; ok {
					(&common.Handler{}).Send(conn, msg)
				}
				r.rwRouters.RUnlock()
			}

		}

	}
	return nil
}

func (r *Router) handleMsg(conn net.Conn, channel chan<- common.IMsg, msg common.IMsg) error {
	r.connKeepAliveAdd(conn, false)
	return r.handler.HandleMsg(conn, channel, msg)
}

//String returns summary
func (r *Router) String() string {
	m := make(map[string]interface{})
	m["id"] = r.id
	m["address"] = r.address

	f := make([]interface{}, 0)
	r.routerIterFunc(func(address string, router *pb.Router) {
		f = append(f, router.Address)
	})
	m["routers"] = f
	m["routers_cnt"] = len(f)

	v := make([]interface{}, 0)
	r.peerIterFunc(func(peer *pb.Peer, conn net.Conn) {
		v = append(v, peer.Id)
	})
	m["peers"] = v
	m["peers_cnt"] = len(v)

	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v\n", err)
	}
	return string(bytes)
}

func (r *Router) routerAdd(key string, router *pb.Router, conn net.Conn) {
	logger.Infoln("add new router :", key)
	r.rwRouters.Lock()

	r.routers[key] = router
	r.connRouters[key] = conn

	r.rwRouters.Unlock()

	r.broadcastNetworkRouters()
}

func (r *Router) routerRemove(key string) {
	logger.Infoln("remove router :", key)
	r.rwRouters.Lock()

	delete(r.routers, key)
	delete(r.connRouters, key)

	r.rwRouters.Unlock()

	r.broadcastNetworkRouters()
}

func (r *Router) routerIterFunc(function func(string, *pb.Router)) {
	r.rwRouters.RLock()
	defer r.rwRouters.RUnlock()
	for key, router := range r.routers {
		function(key, router)
	}
}

func (r *Router) routerConnIterFunc(function func(string, net.Conn)) {
	r.rwRouters.RLock()
	defer r.rwRouters.RUnlock()
	for key, conn := range r.connRouters {
		function(key, conn)
	}
}

func (r *Router) routerExist(key string) bool {
	if key == "" {
		return false
	}
	r.rwRouters.RLock()
	defer r.rwRouters.RUnlock()
	_, ok := r.routers[key]
	return ok
}

func (r *Router) peerAdd(peer *pb.Peer, conn net.Conn) {
	r.rwPeers.Lock()

	bytes, _ := json.Marshal(peer)
	r.peers[string(bytes)] = conn

	r.rwPeers.Unlock()

	r.broadcastNetworkPeers()
}

func (r *Router) peerRemove(peer *pb.Peer) {
	r.rwPeers.Lock()

	bytes, _ := json.Marshal(peer)
	delete(r.peers, string(bytes))

	r.rwPeers.Unlock()

	r.broadcastNetworkPeers()
}

func (r *Router) peerIterFunc(function func(*pb.Peer, net.Conn)) {
	r.rwPeers.RLock()
	defer r.rwPeers.RUnlock()
	for key, conn := range r.peers {
		peer := &pb.Peer{}
		json.Unmarshal([]byte(key), peer)
		function(peer, conn)
	}
}

func (r *Router) isPeer(conn net.Conn) *pb.Peer {
	r.rwPeers.RLock()
	defer r.rwPeers.RUnlock()
	for key, tconn := range r.peers {
		if tconn == conn {
			peer := &pb.Peer{}
			json.Unmarshal([]byte(key), peer)
			return peer
		}
	}
	return nil
}

func (r *Router) msgUniqueAdd(msg *pb.Message) bool {
	r.rwMsg.Lock()
	defer r.rwMsg.Unlock()
	bytes, _ := json.Marshal(msg)
	if _, ok := r.msgUnique[string(bytes)]; ok {
		return false
	}
	r.msgUnique[string(bytes)] = time.Now()
	return true
}

func (r *Router) msgUniqueUpdate(duration time.Duration) {
	r.rwMsg.Lock()
	defer r.rwMsg.Unlock()
	keys := []string{}
	for key, t := range r.msgUnique {
		if time.Now().Sub(t) > duration {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		delete(r.msgUnique, key)
	}
}

func (r *Router) connKeepAliveAdd(conn net.Conn, first bool) {
	r.rwKeepAlive.Lock()
	defer r.rwKeepAlive.Unlock()
	if first {
		r.connKeepAlive[conn] = time.Now()
	} else if _, ok := r.connKeepAlive[conn]; ok {
		r.connKeepAlive[conn] = time.Now()
	}
}

func (r *Router) connKeepAliveRemove(conn net.Conn) {
	r.rwKeepAlive.Lock()
	defer r.rwKeepAlive.Unlock()
	delete(r.connKeepAlive, conn)
}

func (r *Router) connKeepAliveUpdate(ctx context.Context, duration time.Duration) {
	r.rwKeepAlive.Lock()
	defer r.rwKeepAlive.Unlock()
	conns := []net.Conn{}
	for conn, t := range r.connKeepAlive {
		if time.Now().Sub(t) > duration {
			conns = append(conns, conn)
		}
	}
	for _, conn := range conns {
		if peer := r.isPeer(conn); peer != nil {
			r.peerRemove(peer)
		} else {
			var key string
			r.routerConnIterFunc(func(tkey string, tconn net.Conn) {
				if conn == tconn {
					key = tkey
				}
			})

			if key != "" {
				r.routerRemove(key)
			}
			localAddr := conn.LocalAddr().String()
			remoteAddr := conn.RemoteAddr().String()
			if strings.Split(localAddr, ":")[1] != strings.Split(r.address, ":")[1] {
				go func(localAddr, remoteAddr string) {
					ctx0, cancel0 := context.WithCancel(ctx)
					_ = cancel0
					duration := time.Second * 5
					if d, err := time.ParseDuration(config.GetString("router.reconnect.interval")); err == nil {
						duration = d
					}
					for {
						select {
						case <-ctx0.Done():
							return
						default:
						}
						logger.Warnf("connection %s -> %s timeout， reconnecting key %s..", localAddr, remoteAddr, key)
						if r.routerExist(r.address) {
							break
						}
						if conn := r.server.Connect(remoteAddr); conn != nil {
							//发送HELLO消息
							router := &pb.Router{Id: r.id, Address: r.address}
							payload, _ := router.Serialize()
							(&common.Handler{}).Send(conn, &pb.Message{Type: pb.Message_ROUTER_HELLO, Payload: payload})
							break
						}
						time.Sleep(duration)
					}
				}(localAddr, remoteAddr)
			}
		}
		delete(r.connKeepAlive, conn)
		r.server.Disconnect(conn)
	}
}

func (r *Router) broadcastRouters() {
	r.timerRouters.Stop()
	msg := &pb.Message{Type: pb.Message_ROUTER_GET, Payload: nil}
	r.broadcastMsg(msg)
	r.timerRouters.Reset(r.durationRouters)
}

func (r *Router) broadcastNetworkRouters() {
	r.timerNetworkRouters.Stop()
	routers := &pb.Routers{}
	routers.Id = r.address
	r.routerIterFunc(func(address string, router *pb.Router) {
		routers.Routers = append(routers.Routers, router)
	})
	bytes, _ := routers.Serialize()
	msg := &pb.Message{Type: pb.Message_ROUTER_SYNC, Payload: bytes}
	msg.Metadata = append(msg.Metadata, []byte(time.Now().String()+":"+r.address)...)

	r.updateRouters(routers.Id, routers.Routers)
	r.msgUniqueAdd(msg)
	r.broadcastMsg(msg)
	r.timerNetworkRouters.Reset(r.durationNetworkRouters)
}

func (r *Router) broadcastNetworkPeers() {
	r.timerNetworkPeers.Stop()
	peers := &pb.Peers{}
	peers.Id = r.address
	r.peerIterFunc(func(peer *pb.Peer, conn net.Conn) {
		peers.Peers = append(peers.Peers, &pb.Peer{Id: peer.Id})
	})
	bytes, _ := peers.Serialize()
	msg := &pb.Message{Type: pb.Message_PEER_SYNC, Payload: bytes}
	msg.Metadata = append(msg.Metadata, []byte(time.Now().String()+":"+r.address)...)

	r.updatePeers(peers.Id, peers.Peers)
	r.msgUniqueAdd(msg)
	r.broadcastMsg(msg)
	r.timerNetworkPeers.Reset(r.durationNetworkPeers)
}

func (r *Router) broadcastMsg(msg *pb.Message) {
	r.server.BroadCastToClient(msg, func(conn net.Conn, msg common.IMsg) error {
		(&common.Handler{}).Send(conn, msg)
		return nil
	})
	r.server.BroadCastToServer(msg, func(conn net.Conn, msg common.IMsg) error {
		(&common.Handler{}).Send(conn, msg)
		return nil
	})
}

func (r *Router) updatePeers(key string, peers []*pb.Peer) {
	r.allPeers.Update(key, peers)
}

func (r *Router) updateRouters(key string, routers []*pb.Router) {
	addresses := []string{}
	for _, router := range routers {
		addresses = append(addresses, router.Address)
	}
	if r.allRouters.UpdateNetworkTopology(route.NewNodeLink(key, addresses)) {
		r.allRouters.UpdateNextHop()
	}
}
