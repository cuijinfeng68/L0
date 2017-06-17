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

package router

import (
	"fmt"
	"net"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	pb "github.com/bocheninc/msg-net/protos"
	"github.com/looplab/fsm"
)

//NewHandler make new handler
func NewHandler(router *Router) *Handler {
	handler := &Handler{router: router}
	handler.Init()
	return handler
}

//Handler handler
type Handler struct {
	router *Router
	fsm    *fsm.FSM
}

//Init Initialization
func (h *Handler) Init() {
	if h.fsm != nil {
		return
	}
	h.fsm = fsm.NewFSM(
		"created",
		fsm.Events{
			{Name: "HELLO", Src: []string{"created"}, Dst: "established"},
			{Name: pb.Message_ROUTER_HELLO.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_ROUTER_HELLO_ACK.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_ROUTER_GET.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_ROUTER_GET_ACK.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_ROUTER_SYNC.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_ROUTER_CLOSE.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_PEER_HELLO.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_PEER_SYNC.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_PEER_CLOSE.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_KEEPALIVE.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_KEEPALIVE_ACK.String(), Src: []string{"established"}, Dst: "established"},
			{Name: pb.Message_CHAIN_MESSAGE.String(), Src: []string{"established"}, Dst: "established"},
		},
		fsm.Callbacks{
			// "enter_state":                                     func(e *fsm.Event) { h.enterState(e) },
			// "leave_state":                                     func(e *fsm.Event) { h.leaveState(e) },
			// "before_event":                                    func(e *fsm.Event) { h.beforeEvent(e) },
			// "after_event":                                     func(e *fsm.Event) { h.afterEvent(e) },
			"after_" + pb.Message_ROUTER_HELLO.String():     func(e *fsm.Event) { h.afterRouterHello(e) },
			"after_" + pb.Message_ROUTER_HELLO_ACK.String(): func(e *fsm.Event) { h.afterRouterHelloAck(e) },
			"after_" + pb.Message_ROUTER_GET.String():       func(e *fsm.Event) { h.afterRouterGet(e) },
			"after_" + pb.Message_ROUTER_GET_ACK.String():   func(e *fsm.Event) { h.afterRouterGetAck(e) },
			"after_" + pb.Message_ROUTER_SYNC.String():      func(e *fsm.Event) { h.afterRouterSync(e) },
			"after_" + pb.Message_ROUTER_CLOSE.String():     func(e *fsm.Event) { h.afterRouterClose(e) },
			"after_" + pb.Message_PEER_HELLO.String():       func(e *fsm.Event) { h.afterPeerHello(e) },
			"after_" + pb.Message_PEER_SYNC.String():        func(e *fsm.Event) { h.afterPeerSync(e) },
			"after_" + pb.Message_PEER_CLOSE.String():       func(e *fsm.Event) { h.afterPeerClose(e) },
			"after_" + pb.Message_KEEPALIVE.String():        func(e *fsm.Event) { h.afterKeepAlive(e) },
			"after_" + pb.Message_KEEPALIVE_ACK.String():    func(e *fsm.Event) { h.afterKeepAliveAck(e) },
			"after_" + pb.Message_CHAIN_MESSAGE.String():    func(e *fsm.Event) { h.afterChainMessage(e) },
		},
	)
}

//HandleMsg message handle
func (h *Handler) HandleMsg(conn net.Conn, sendChannel chan<- common.IMsg, iMsg common.IMsg) error {
	msg, ok := iMsg.(*pb.Message)
	if !ok {
		return fmt.Errorf("Received unexpected message type")
	}
	logger.Debugf("handling Message of type: %s in state %s", msg.Type, h.fsm.Current())
	if h.fsm.Cannot(msg.Type.String()) {
		return fmt.Errorf("cannot handle message (%s) with payload size (%d) while in state: %s", msg.Type.String(), len(msg.Payload), h.fsm.Current())
	}
	if err := h.fsm.Event(msg.Type.String(), msg, sendChannel, conn); err != nil {
		if noTransitionErr, ok := err.(*fsm.NoTransitionError); ok {
			if noTransitionErr.Err != nil {
				logger.Warnf("ignoring NoTransitionError: %s %v ", msg.Type.String(), noTransitionErr)
			}
		} else {
			return fmt.Errorf("failed to handle message (%s): current state: %s, error: %s", msg.Type.String(), h.fsm.Current(), err)
		}
	}
	return nil
}

// func (h *Handler) enterState(e *fsm.Event) {
// 	logger.Debugf("The bi-directional stream enter state,  %v ", e)
// }

// func (h *Handler) leaveState(e *fsm.Event) {
// 	logger.Debugf("The bi-directional stream leave state,  %v ", e)
// }

// func (h *Handler) beforeEvent(e *fsm.Event) {
// 	logger.Debugf("The bi-directional stream before event,  %v ", e)
// }

// func (h *Handler) afterEvent(e *fsm.Event) {
// 	logger.Debugf("The bi-directional stream after event,  %v ", e)
// }

func (h *Handler) afterRouterHello(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	sendChannel := e.Args[1].(chan<- common.IMsg)
	conn := e.Args[2].(net.Conn)
	//Recv
	router := &pb.Router{}
	if err := router.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}

	//Send
	router0 := &pb.Router{Id: h.router.id, Address: h.router.address}
	if h.router.routerExist(router.Address) {
		router0.Id = "unkown"
	} else {
		h.router.routerAdd(router.Address, router, conn)
		h.router.connKeepAliveAdd(conn, true)
	}
	bytes, err := router0.Serialize()
	if err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	sendChannel <- &pb.Message{Type: pb.Message_ROUTER_HELLO_ACK, Payload: bytes}

}

func (h *Handler) afterRouterHelloAck(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	sendChannel := e.Args[1].(chan<- common.IMsg)
	conn := e.Args[2].(net.Conn)

	//Recv
	router := &pb.Router{}
	if err := router.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	if router.Id == "unkown" {
		go h.router.server.Disconnect(conn)
	} else {
		h.router.routerAdd(router.Address, router, conn)
		h.router.connKeepAliveAdd(conn, true)
		sendChannel <- &pb.Message{Type: pb.Message_ROUTER_GET, Payload: nil}
	}
}

func (h *Handler) afterRouterGet(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	sendChannel := e.Args[1].(chan<- common.IMsg)

	//Send
	routers := &pb.Routers{}
	routers.Id = h.router.address
	h.router.routerIterFunc(func(key string, router *pb.Router) {
		routers.Routers = append(routers.Routers, &pb.Router{Id: router.Id, Address: router.Address})
	})
	bytes, err := routers.Serialize()
	if err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	sendChannel <- &pb.Message{Type: pb.Message_ROUTER_GET_ACK, Payload: bytes}
}

func (h *Handler) afterRouterGetAck(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)

	//Recv
	routers := &pb.Routers{}
	if err := routers.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	addresses := []string{}
	for _, router := range routers.Routers {
		addresses = append(addresses, router.Address)
	}
	h.router.Discovery(addresses)
}

func (h *Handler) afterRouterSync(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)

	if h.router.msgUniqueAdd(msg) {
		routers := &pb.Routers{}
		if err := routers.Deserialize(msg.Payload); err != nil {
			e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
			return
		}
		h.router.updateRouters(routers.Id, routers.Routers)
		h.router.broadcastMsg(msg)
	}
}

func (h *Handler) afterRouterClose(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)
	conn := e.Args[2].(net.Conn)

	//Recv
	router := &pb.Router{}
	if err := router.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	h.router.routerRemove(router.Address)
	h.router.connKeepAliveRemove(conn)
}

func (h *Handler) afterPeerHello(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	sendChannel := e.Args[1].(chan<- common.IMsg)
	conn := e.Args[2].(net.Conn)
	//Recv
	peer := &pb.Peer{}
	if err := peer.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	h.router.peerAdd(peer, conn)
	h.router.connKeepAliveAdd(conn, true)

	//Send
	router := &pb.Router{Id: h.router.id, Address: h.router.address}
	bytes, err := router.Serialize()
	if err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	sendChannel <- &pb.Message{Type: pb.Message_PEER_HELLO_ACK, Payload: bytes}
}

func (h *Handler) afterPeerSync(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)
	if h.router.msgUniqueAdd(msg) {
		peers := &pb.Peers{}
		if err := peers.Deserialize(msg.Payload); err != nil {
			e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
			return
		}
		h.router.updatePeers(peers.Id, peers.Peers)
		h.router.broadcastMsg(msg)
	}
}

func (h *Handler) afterPeerClose(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)
	conn := e.Args[2].(net.Conn)

	//Recv
	peer := &pb.Peer{}
	if err := peer.Deserialize(msg.Payload); err != nil {
		e.Cancel(fmt.Errorf("failed to handle message (%s) --- %s", msg.Type.String(), err))
		return
	}
	h.router.peerRemove(peer)
	h.router.connKeepAliveRemove(conn)
}

func (h *Handler) afterKeepAlive(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	//msg := e.Args[0].(*pb.Message)
	sendChannel := e.Args[1].(chan<- common.IMsg)

	sendChannel <- &pb.Message{Type: pb.Message_KEEPALIVE_ACK}
}

func (h *Handler) afterKeepAliveAck(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)
	//sendChannel := e.Args[1].(chan<- common.IMsg)

	_ = msg
}

func (h *Handler) afterChainMessage(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	msg := e.Args[0].(*pb.Message)

	if err := h.router.RouteMessage(msg); err != nil {
		e.Cancel(err)
	}
}
