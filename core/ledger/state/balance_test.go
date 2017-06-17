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

package state

import (
	"bytes"
	"math/big"
	"testing"
)

var balanceBytes []byte

func TestSerialize(t *testing.T) {
	b := NewBalance(big.NewInt(0), uint32(0))

	balanceBytes = b.serialize()
	t.Log(balanceBytes)
}

func TestDeserialize(t *testing.T) {

	tmp := new(Balance)

	tmp.deserialize([]byte{0, 0})

	if !bytes.Equal(tmp.serialize(), balanceBytes) {
		t.Errorf("balance %v is not equal amount %v \n", tmp.Amount, balanceBytes)
	}
	t.Log(balanceBytes, tmp.Amount, tmp.Nonce)
}
