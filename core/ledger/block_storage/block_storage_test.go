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

package block_storage

import (
	"bytes"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	testDb = db.NewDB(db.DefaultConfig())

	lastid    uint32
	sender    = accounts.HexToAddress("0xa122277be213f56221b6140998c03d860a60e1f8")
	reciepent = accounts.HexToAddress("0x27c649b7c4f66cfaedb99d6b38527db4deda6f41")
	amount    = big.NewInt(521000)
	fee       = big.NewInt(200)

	testTxHash crypto.Hash

	blockHashByte []byte
)

func addGenesisblock(b *Blockchain) error {
	blockHeader := new(types.BlockHeader)
	blockHeader.TimeStamp = utils.CurrentTimestamp()
	blockHeader.Nonce = uint32(100)
	blockHeader.Height = 0

	genesisBlock := new(types.Block)
	genesisBlock.Header = blockHeader
	writeBatchs := b.AppendBlock(genesisBlock)
	if err := b.dbHandler.AtomicWrite(writeBatchs); err != nil {
		return err
	}
	return nil
}

func TestAppendBlock(t *testing.T) {

	b := NewBlockchain(testDb)
	if err := addGenesisblock(b); err != nil {
		t.Error(err)
	}

	var previousHash crypto.Hash
	for i := 1; i < 3; i++ {
		header := new(types.BlockHeader)
		header.TimeStamp = uint32(time.Now().Unix())
		header.Nonce = rand.Uint32()
		header.Height = uint32(i)

		header.PreviousHash = previousHash

		nb := new(types.Block)
		nb.Header = header

		// transaction
		var hashSlice []crypto.Hash
		for j := 0; j < 2; j++ {
			tx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(i + j)}), coordinate.NewChainCoordinate([]byte{byte(i + j)}), types.TypeAtomic, rand.Uint32(), reciepent, amount, fee, utils.CurrentTimestamp())

			//sing tx
			keypair, _ := crypto.GenerateKey()
			s, _ := keypair.Sign(tx.Hash().Bytes())
			tx.WithSignature(s)

			nb.Transactions = append(nb.Transactions, tx)

			hashSlice = append(hashSlice, tx.Hash())

			testTxHash = tx.Hash()
		}

		merkleHash := crypto.GetMerkleHash(hashSlice) //  utils.ComputeMerkleHash(hashSlice)
		nb.Header.TxsMerkleHash = merkleHash

		writeBatchs := b.AppendBlock(nb)

		if err := b.dbHandler.AtomicWrite(writeBatchs); err != nil {
			t.Error(err)
		}

		previousHash = nb.Hash()
		t.Log(len(nb.Transactions))
		blockHashByte = nb.Serialize()
	}
	return
}

func TestGetBlockchainHeight(t *testing.T) {
	b := NewBlockchain(testDb)
	t.Log(b.GetBlockchainHeight())
	block, err := b.GetBlockByNumber(1)
	if err != nil {
		t.Error(err)
	}
	t.Log(block.Transactions[0].Amount())
}

func TestGetTransactionsByNumber(t *testing.T) {
	b := NewBlockchain(testDb)
	txs, err := b.GetTransactionsByNumber(1, types.TypeAtomic)
	if err != nil {
		t.Error(err)
	}
	t.Log("transactions len:", len(txs))

	block, err := b.GetBlockByNumber(2)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(block.Serialize(), blockHashByte) {
		t.Errorf("Block.Serialize error, %0x != %0x ", block.Serialize(), blockHashByte)
	}
}

func TestGetTransactionByTxHash(t *testing.T) {
	b := NewBlockchain(testDb)

	tx, _ := b.GetTransactionByTxHash(testTxHash.Bytes())
	t.Log(tx.Amount())

	os.RemoveAll("/tmp/rocksdb-test")
}
