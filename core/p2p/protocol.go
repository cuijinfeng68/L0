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
	"io"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
)

var (
	baseProtocolName    = "l0-base-protocol"
	baseProtocolVersion = "0.0.1"
	protoHandshake      *ProtoHandshake
	encHandshake        *EncHandshake
)

// Protocol raw structure
type Protocol struct {
	BaseCmd uint8
	Name    string
	Version string
	Run     func(p *Peer, rw MsgReadWriter) error
}

type protoRW struct {
	Protocol
	in chan Msg
	// exit
	w io.Writer
}

func (rw *protoRW) ReadMsg() (Msg, error) {
	select {
	case msg := <-rw.in:
		return msg, nil
	}
}

func (rw *protoRW) WriteMsg(msg Msg) (int, error) {
	return SendMessage(rw.w, &msg)
}

// ProtoHandshake is protocol handshake.  implement the interface of Protocol
type ProtoHandshake struct {
	Name       string
	Version    string
	ID         []byte
	SrvAddress string
}

// GetProtoHandshake returns protocol handshake
func GetProtoHandshake() *ProtoHandshake {
	if protoHandshake == nil {
		protoHandshake = &ProtoHandshake{
			Name:       baseProtocolName,
			Version:    baseProtocolVersion,
			ID:         getPeerID(),
			SrvAddress: getPeerAddress(config.Address),
		}
	}
	return protoHandshake
}

// GetEncHandshake returns enchandshake message
func GetEncHandshake() *EncHandshake {
	if encHandshake == nil {
		// TODO　Generate random string
		h := crypto.Sha256([]byte("random string"))
		sign, err := config.PrivateKey.Sign(h[:])
		if err != nil {
			log.Error(err.Error())
		}
		encHandshake = &EncHandshake{
			Signature: sign,
			Hash:      &h,
			ID:        getPeerID(),
		}
	}
	return encHandshake
}

// serialize ProtoHandshake instance to []byte
func (proto *ProtoHandshake) serialize() []byte {
	return utils.Serialize(*proto)
}

// deserialize buffer to ProtoHandshake instance
func (proto *ProtoHandshake) deserialize(data []byte) {
	utils.Deserialize(data, proto)
}

// matchProtocol returns the result of handshake
func (proto *ProtoHandshake) matchProtocol(i interface{}) bool {
	if p, ok := i.(*ProtoHandshake); ok {
		if p.Name == proto.Name || p.Version == proto.Version {
			return true
		}
	}
	return false
}

// EncHandshake is encryption handshake. implement the interface of Protocol
type EncHandshake struct {
	ID        []byte
	Signature *crypto.Signature
	Hash      *crypto.Hash
}

// matchProtocol returns the result of handshake
func (enc *EncHandshake) matchProtocol(i interface{}) bool {
	if e, ok := i.(*EncHandshake); ok {
		if enc != nil && enc.Hash != nil {
			_, err := e.Signature.RecoverPublicKey(enc.Hash.Bytes())
			if err == nil {
				return true
			}
			log.Errorf("enc handshake error %v", err.Error())
		}
		log.Errorf("enc handshake error, decode nil content %v", enc)
	}
	return false
}

// serialize EncHandshake instance to []byte
func (enc *EncHandshake) serialize() []byte {
	return utils.Serialize(enc)
}

// deserialize buffer to encHandshake instance
func (enc *EncHandshake) deserialize(data []byte) {
	utils.Deserialize(data, enc)
}

func getPeerID() []byte {
	return getPeerManager().localPeer.ID[:]
}
