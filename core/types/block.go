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
	"errors"
	"io"
	"sync/atomic"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
)

// IInventory defines interface that broadcast data should implements
type IInventory interface {
	Hash() crypto.Hash
	Serialize() []byte
}

// Block represents an block in blockchain
type Block struct {
	Header       *BlockHeader `json:"header"`
	Transactions Transactions `json:"transactions"`

	// caches
	hash atomic.Value
}

// BlockHeader represents the header in block
type BlockHeader struct {
	PreviousHash  crypto.Hash `json:"previousHash" `
	TimeStamp     uint32      `json:"timeStamp"`
	Nonce         uint32      `json:"nonce" `
	TxsMerkleHash crypto.Hash `json:"transactionsMerkleHash" `
	Height        uint32      `json:"height" `
}

// NewBlockHeader returns a blockheader
func NewBlockHeader(prvHash crypto.Hash, timeStamp, height, nonce uint32, txsHash crypto.Hash) *BlockHeader {
	return &BlockHeader{
		prvHash,
		timeStamp,
		nonce,
		txsHash,
		height,
	}
}

// NewBlock returns an new block
func NewBlock(prvHash crypto.Hash,
	timeStamp, height, nonce uint32,
	txsHash crypto.Hash,
	Txs Transactions) *Block {
	return &Block{
		Header:       NewBlockHeader(prvHash, timeStamp, height, nonce, txsHash),
		Transactions: Txs,
	}
}

// Height returns the block height
func (b *Block) Height() uint32 { return b.Header.Height }

// Serialize returns the serialized bytes of a blockheader
func (h BlockHeader) Serialize() []byte {
	buf := new(bytes.Buffer)
	utils.VarEncode(buf, h)
	return buf.Bytes()
}

// Deserialize deserialize the input data to header
func (h *BlockHeader) Deserialize(data []byte) error {
	r := bytes.NewBuffer(data)
	utils.VarDecode(r, h)
	return nil
}

// Hash returns the hash of the blockheader
func (h *BlockHeader) Hash() crypto.Hash {
	return crypto.DoubleSha256(h.Serialize())
}

// PreviousHash returns the previous hash of the block
func (b *Block) PreviousHash() crypto.Hash {
	return b.Header.PreviousHash
}

// Hash returns the hash of the blockheader in block
func (b *Block) Hash() crypto.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(crypto.Hash)
	}
	v := b.Header.Hash()
	b.hash.Store(v)
	return v
}

// Serialize serializes the all data in block
func (b *Block) Serialize() []byte {
	return utils.Serialize(b)
}

// SerializeTxs serializes transactions
func SerializeTxs(r io.Writer, txs Transactions) {
	utils.VarEncode(r, (uint64)(txs.Len()))
	for i := 0; i < txs.Len(); i++ {
		utils.VarEncode(r, (uint64)(len(txs[i].Serialize())))
		r.Write(txs[i].Serialize())
	}
}

//GetTransactions get Transactions by Type
func (b *Block) GetTransactions(transactionType uint32) (Transactions, error) {
	var txs Transactions

	if transactionType < 0 || transactionType > 5 {
		return nil, errors.New("transaction type is not support")

	}

	for _, tx := range b.Transactions {
		if tx.GetType() == transactionType {
			txs = append(txs, tx)
		}
	}

	return txs, nil
}

// Deserialize deserializes bytes to Block
func (b *Block) Deserialize(data []byte) error {
	return utils.Deserialize(data, b)
}

// DeserializeTxs deserializes transactions
func DeserializeTxs(r io.Reader) Transactions {
	var (
		txs = make([]*Transaction, 0)
	)
	txsNum, _ := utils.ReadVarInt(r)
	for i := uint64(0); i < txsNum; i++ {
		txLen, _ := utils.ReadVarInt(r)
		buf := make([]byte, txLen)
		io.ReadFull(r, buf)
		tx := new(Transaction)
		tx.Deserialize(buf)
		txs = append(txs, tx)
	}
	return txs
}
