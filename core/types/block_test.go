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

package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
)

var (
	testHashStr string
)

func TestBlockSerialize(t *testing.T) {
	var (
		testBlock = Block{
			Header: &BlockHeader{
				PreviousHash: crypto.DoubleSha256([]byte("xxxx")),
				TimeStamp:    uint32(time.Now().Unix()),
				Nonce:        uint32(100),
			},
		}
	)
	Txs := make([]*Transaction, 0)
	hashs := make([]crypto.Hash, 0)
	reciepent := accounts.HexToAddress("0xbf6080eaae18a6eb4d9d3b9ef08a8bdf02e3caa8")
	for i := 1; i < 3; i++ {
		tx := NewTransaction(
			coordinate.NewChainCoordinate([]byte{byte(i)}),
			coordinate.NewChainCoordinate([]byte{byte(i)}),
			TypeAtomic,
			uint32(i),
			reciepent,
			reciepent,
			big.NewInt(10000),
			big.NewInt(1000),
			utils.CurrentTimestamp(),
		)
		Txs = append(Txs, tx)
		hashs = append(hashs, tx.Hash())
	}

	testBlock.Transactions = Txs
	testBlock.Header.TxsMerkleHash = crypto.ComputeMerkleHash(hashs)[0]

	fmt.Println("Block", testBlock.Hash())
	fmt.Printf("Block Raw {'previousHash':%v, 'MerkleHash':%v,  'Nonce':%v, TimeStamp':%v, Txs:%v \n",
		testBlock.Header.PreviousHash.Bytes(),
		testBlock.Header.TxsMerkleHash.Bytes(),
		testBlock.Header.Nonce,
		testBlock.Header.TimeStamp,
		testBlock.Transactions,
	)
	fmt.Println("Block Header serialize()", testBlock.Header.Serialize())
	fmt.Println("Block AtomicTxs", testBlock.Transactions, len(testBlock.Transactions))
	fmt.Println("Block serialize()", hex.EncodeToString(testBlock.Serialize()))
	testHashStr = hex.EncodeToString(testBlock.Serialize())
}

func TestBlockDeserialize(t *testing.T) {
	testBlock := &Block{}
	data, _ := hex.DecodeString(testHashStr)
	fmt.Println("------------block deseriailze---------")
	testBlock.Deserialize(data)
	blkData := testBlock.Serialize()
	if !bytes.Equal(blkData, data) {
		t.Errorf("Block.Serialize error, %0x != %0x ", data, blkData)
	}

}
