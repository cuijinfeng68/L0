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

package merge

import (
	"math/big"
	"testing"

	"bytes"

	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var data []byte

func getChainCoordinate(i int) coordinate.ChainCoordinate {

	return coordinate.NewChainCoordinate([]byte{byte(0), byte(1), byte(i)})
}

func getTestTxs(src, dst int) types.Transactions {
	testTxs := make([]*types.Transaction, 0)
	for i := 0; i < 3; i++ {
		tx := types.NewTransaction(
			getChainCoordinate(src),
			getChainCoordinate(dst),
			types.TypeAtomic,
			uint32(1),
			accounts.ChainCoordinateToAddress(getChainCoordinate(i)),
			big.NewInt(int64(i*100)),
			big.NewInt(1000),
			utils.CurrentTimestamp(),
		)
		testTxs = append(testTxs, tx)
	}
	return testTxs
}

func TestSerialize(t *testing.T) {
	testTxs := getTestTxs(0, 1)
	testUp := NewUploadPayload(5, 10, testTxs, testTxs)
	data = testUp.Serialize()
}

func TestDeserialize(t *testing.T) {
	testUp := new(UploadPayload)
	testUp.Deserialize(data)

	if !bytes.Equal(testUp.Serialize(), data) {
		t.Errorf("Deserialize error! %v != %v", testUp.Serialize(), data)
	}
}
