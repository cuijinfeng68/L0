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
	"testing"
)

// TestMsg tests the encode/Decode functions of Message
func TestMsg(t *testing.T) {
	msg := NewMsg(pingMsg, nil)
	buf := msg.Serialize()

	msg2 := new(Msg)
	msg2.Deserialize(buf)

	if msg2.Cmd != msg.Cmd {
		t.Errorf("error, %v - %v", msg, msg2)
	}
	if !bytes.Equal(msg2.Payload, msg.Payload) {
		t.Error("error")
	}

	t.Log(buf)

}

func TestReadWrite(t *testing.T) {
	msg := NewMsg(pingMsg, []byte("TEST"))
	buf := new(bytes.Buffer)
	msg.write(buf)

	m := &Msg{}
	m.read(buf)
	fmt.Printf("%v\n", msg)
	fmt.Printf("%v\n", m)
}
