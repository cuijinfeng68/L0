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

package node

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/bocheninc/L0/components/crypto"

	"github.com/bocheninc/L0/components/utils"
)

func TestStatusPayload(t *testing.T) {
	var (
		status = StatusData{
			Version:     33,
			StartHeight: 20,
		}
	)

	statusBytes := utils.Serialize(status)

	status2 := StatusData{}
	utils.Deserialize(statusBytes, &status2)
	if !reflect.DeepEqual(status, status2) {
		t.Errorf("status not equal")
	}

}

func TestInvVectPayload(t *testing.T) {
	var (
		inventory = InvVect{
			Type: InvTypeBlock,
			Hashes: []crypto.Hash{
				crypto.Sha256([]byte("1")),
				crypto.Sha256([]byte("2"))},
		}
	)

	inventoryBytes := utils.Serialize(inventory)

	inventoryBlock := InvVect{}
	utils.Deserialize(inventoryBytes, &inventoryBlock)

	if !bytes.Equal(inventoryBytes, utils.Serialize(inventoryBlock)) {
		t.Errorf("Deserialize error")
	}
}
