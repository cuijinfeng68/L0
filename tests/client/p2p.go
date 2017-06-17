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

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
)

var (
	priKey *crypto.PrivateKey
	conn   []net.Conn
)

const (
	pingMsg = iota + 1
	pongMsg
	handshakeMsg
	handshakeAckMsg
	statusMsg = 17
)

type Msg struct {
	Cmd      uint8
	Payload  []byte
	CheckSum [4]byte
}

type StatusData struct {
	Version     uint32
	StartHeight uint32
}

type ProtoHandshake struct {
	Name       string
	Version    string
	ID         []byte
	SrvAddress string
}

type EncHandshake struct {
	ID        []byte
	Signature *crypto.Signature
	Hash      *crypto.Hash
}

func listen(c net.Conn) {
	for {
		l, err := utils.ReadVarInt(c)
		if err != nil {
			panic(err)
		}
		data := make([]byte, l)
		io.ReadFull(c, data)
		m := new(Msg)
		err = utils.Deserialize(data, m)
		if err != nil {
			panic(err)
		}
		go processMsg(m, c)
	}
}

func processMsg(m *Msg, c net.Conn) {
	h := crypto.Sha256(m.Payload)
	if !bytes.Equal(m.CheckSum[:], h[0:4]) {
		println("Msg check error")
		return
	}

	respMsg := new(Msg)
	switch m.Cmd {
	case pingMsg:
		respMsg = NewMsg(pongMsg, nil)
	case handshakeMsg:
		proto := &ProtoHandshake{
			Name:       "l0-base-protocol",
			Version:    "0.0.1",
			ID:         priKey.Public().Bytes(),
			SrvAddress: "",
		}
		respMsg = NewMsg(handshakeMsg, utils.Serialize(*proto))
		fmt.Println("handshakeMsg")
	case handshakeAckMsg:
		h := crypto.Sha256([]byte("random string"))
		sign, _ := priKey.Sign(h[:])
		enc := &EncHandshake{
			Signature: sign,
			Hash:      &h,
			ID:        priKey.Public().Bytes(),
		}
		respMsg = NewMsg(handshakeAckMsg, utils.Serialize(*enc))
		fmt.Println("handshakeAckMsg")
	case statusMsg:
		status := &StatusData{
			Version:     uint32(0),
			StartHeight: uint32(0),
		}
		respMsg = NewMsg(statusMsg, utils.Serialize(*status))
		fmt.Println("statusMsg")
	default:
		return
	}
	sendMsg(respMsg, c)
}

func sendMsg(m *Msg, c net.Conn) {
	data := utils.Serialize(*m)
	data = append(utils.VarInt(uint64(len(data))), data...)
	c.Write(data)
}

func init() {
	// TCPSend(srvAddress)
}

// NewMsg returns a new msg
func NewMsg(MsgType uint8, payload []byte) *Msg {
	Msg := &Msg{
		Cmd:     MsgType,
		Payload: payload,
	}
	h := crypto.Sha256(payload)
	copy(Msg.CheckSum[:], h[0:4])
	return Msg
}

// TCPSend sends transaction with tcp
func TCPSend(srvAddress []string) {
	priKey, _ = crypto.GenerateKey()

	for _, address := range srvAddress {
		c, err := net.Dial("tcp", address)
		if err != nil || c == nil {
			panic(err)
		}
		go listen(c)
		fmt.Println("LocalAddr:", c.LocalAddr().String(), " RemoteAddr:", c.RemoteAddr().String())
		conn = append(conn, c)
	}
}

// Relay relays transaction to blockchain
func Relay(m *Msg) {
	data := utils.Serialize(*m)
	data = append(utils.VarInt(uint64(len(data))), data...)
	for _, c := range conn {
		c.Write(data)
	}
}
