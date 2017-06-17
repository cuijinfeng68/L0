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

package peer

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"strings"

	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	"github.com/bocheninc/msg-net/net/tcp"
	pb "github.com/bocheninc/msg-net/protos"
)

//NewPeer create Peer instance
func NewPeer(id string, addresses []string, function func(srcID, dstID string, payload []byte, signature []byte) error) *Peer {
	//params verify
	return &Peer{id: id, addresses: addresses, chainMessageHandle: function}
}

//Peer Define Peer class connected to Router
type Peer struct {
	id                 string
	addresses          []string
	chainMessageHandle func(srcID, dstID string, payload []byte, signature []byte) error

	client                *tcp.Client
	durationKeepAlive     time.Duration
	timerKeepAliveTimeout *time.Timer
	cancel                context.CancelFunc
	index                 int
}

//IsRunning Running or not
func (p *Peer) IsRunning() bool {
	return p.client != nil && p.client.IsConnected()
}

//Start Start peer service
func (p *Peer) Start() bool {
	if p.IsRunning() {
		logger.Warnf("peer %s is alreay running", p.id)
		return true
	}

	if len(p.addresses) == 0 {
		logger.Errorf("peer %s not specify addresses", p.id)
		return false
	}

	//keepalive
	p.durationKeepAlive = time.Second * 15
	if d, err := time.ParseDuration(config.GetString("router.timeout.keepalive")); err == nil {
		p.durationKeepAlive = d
	} else {
		logger.Warnf("failed to parse router.timeout.keepalive, set default timeout 5s --- %v", err)
	}

	var conn net.Conn
	for index, address := range p.addresses {
		p.index = index
		p.client = tcp.NewClient(address, func() common.IMsg {
			return &pb.Message{}
		}, p.handleMsg)

		if conn = p.client.Connect(); conn != nil {
			break
		}
		logger.Warnf("peer %s failed to start to server address %s", p.id, address)
	}
	p.timerKeepAliveTimeout = time.NewTimer(2 * p.durationKeepAlive)
	if conn == nil {
		logger.Errorf("peer %s failed to start", p.id)
		return false
	}

	peer := pb.Peer{Id: p.id}
	bytes, _ := peer.Serialize()
	p.client.SendChannel() <- &pb.Message{Type: pb.Message_PEER_HELLO, Payload: bytes}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.timerKeepAliveTimeout.C:
				if p.client != nil {
					p.client.Disconnect()
				}
				go func(ctx context.Context) {
					ctx0, cancel0 := context.WithCancel(ctx)
					_ = cancel0
					duration := time.Second * 5
					if d, err := time.ParseDuration(config.GetString("router.reconnect.interval")); err == nil {
						duration = d
					}
					max := 5
					if n := config.GetInt("router.reconnect.max"); n != 0 {
						max = n
					}
					for {
						select {
						case <-ctx0.Done():
							return
						default:
						}
						p.index = p.index + 1
						if p.index == len(p.addresses) {
							p.index = 0
						}
						for i := max; i > 0; i-- {
							logger.Warnf("peer %s connection timeout， reconnecting", p.id)
							address := p.addresses[p.index]
							p.client = tcp.NewClient(address, func() common.IMsg {
								return &pb.Message{}
							}, p.handleMsg)
							if conn := p.client.Connect(); conn != nil {
								peer := pb.Peer{Id: p.id}
								bytes, _ := peer.Serialize()
								p.client.SendChannel() <- &pb.Message{Type: pb.Message_PEER_HELLO, Payload: bytes}
								return
							}
							time.Sleep(duration)
						}
					}
				}(ctx)
			}
		}
	}(ctx)

	return true
}

//Send Send msg to Router
func (p *Peer) Send(id string, payload []byte, signature []byte) bool {
	if !p.IsRunning() {
		logger.Warnf("peer %s is alreay stopped", p.id)
		return false
	}
	if !strings.Contains(id, ":") {
		logger.Infof("broadcast all chain %s peers\n", id)
		id = id + ":"
	}
	chainMsg := pb.ChainMessage{SrcId: p.id, DstId: id, Payload: payload, Signature: signature}
	bytes, _ := chainMsg.Serialize()
	p.client.SendChannel() <- &pb.Message{Type: pb.Message_CHAIN_MESSAGE, Payload: bytes}

	return true
}

//Stop Stop peer service
func (p *Peer) Stop() {
	if !p.IsRunning() {
		logger.Warnf("peer %s is alreay stopped", p.id)
	}
	p.cancel()

	peer := pb.Peer{Id: p.id}
	bytes, _ := peer.Serialize()
	p.client.SendChannel() <- &pb.Message{Type: pb.Message_PEER_CLOSE, Payload: bytes}

	p.client.Disconnect()
	p.client = nil
}

//String Get Peer Infomation
func (p *Peer) String() string {
	m := make(map[string]interface{})
	m["id"] = p.id
	m["addresses"] = p.addresses
	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v\n", err)
	}
	return string(bytes)
}

func (p *Peer) handleMsg(conn net.Conn, channel chan<- common.IMsg, m common.IMsg) error {
	p.timerKeepAliveTimeout.Stop()

	msg := m.(*pb.Message)
	switch msg.Type {
	case pb.Message_ROUTER_CLOSE:
	case pb.Message_PEER_HELLO_ACK:
	case pb.Message_KEEPALIVE:
		p.client.SendChannel() <- &pb.Message{Type: pb.Message_KEEPALIVE_ACK, Payload: nil}
	case pb.Message_KEEPALIVE_ACK:
	case pb.Message_PEER_SYNC:
	case pb.Message_ROUTER_SYNC:
	case pb.Message_ROUTER_GET:
	case pb.Message_CHAIN_MESSAGE:
		chainMsg := &pb.ChainMessage{}
		if err := chainMsg.Deserialize(msg.Payload); err != nil {
			return err
		}
		if err := p.chainMessageHandle(chainMsg.SrcId, chainMsg.DstId, chainMsg.Payload, chainMsg.Signature); err != nil {
			return err
		}
	default:
		logger.Errorf("unsupport message type --- %v", msg.Type)
	}

	p.timerKeepAliveTimeout.Reset(2 * p.durationKeepAlive)
	return nil
}

//SetLogOut set log out path
func SetLogOut(dir string) {
	config.Set("logger.out", dir)
	logger.SetOut()
}
