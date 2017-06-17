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
	"testing"

	"github.com/bocheninc/L0/components/crypto"
)

func TestProtocolHandshake(t *testing.T) {
	proto := ProtoHandshake{
		Name:    "pro",
		Version: "0.0.1",
	}

	buf := proto.serialize()
	t.Log("buf: ", buf)

	p := ProtoHandshake{}
	p.deserialize(buf)
	t.Log("p->name:", p.Name, " p->version", p.Version, "p->ID: ", string(p.ID))
	if p.Name != proto.Name || p.Version != proto.Version {
		t.Error("Protocol serialize/deserialize Error")
	}

	if !bytes.Equal(proto.ID, p.ID) {
		t.Error("Protocol serialize/deserialize Error")
	}
}

func TestEncryptionHandshake(t *testing.T) {

	pri, _ := crypto.GenerateKey()
	h := crypto.Sha256([]byte("msg"))
	sign, _ := pri.Sign(h[:])
	encHandshake := EncHandshake{
		Signature: sign,
		Hash:      &h,
	}

	buf := encHandshake.serialize()

	enc := &EncHandshake{
		Signature: new(crypto.Signature),
		Hash:      new(crypto.Hash),
	}

	enc.deserialize(buf)

	t.Log(enc.Hash)
	if !bytes.Equal(enc.ID[:], encHandshake.ID[:]) {
		t.Error("error")
	}
	if !bytes.Equal(enc.Signature[:], encHandshake.Signature[:]) {
		t.Error("error")
	}
	if !bytes.Equal(enc.Hash.Bytes(), encHandshake.Hash.Bytes()) {
		t.Error("error")
	}
}
