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
	"errors"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/types"
	"github.com/golang/protobuf/proto"
)

const (
	heightKey string = "blockLastHeight"
	atomic    uint32 = iota
	acrossChain
)

// Blockchain represents block
type Blockchain struct {
	dbHandler         *db.BlockchainDB
	columnFamily      string
	indexColumnFamily string
}

// NewBlockchain initialization
func NewBlockchain(db *db.BlockchainDB) *Blockchain {
	return &Blockchain{
		dbHandler:         db,
		columnFamily:      "block",
		indexColumnFamily: "index",
	}
}

// GetBlockByHash gets block by block hash
func (blockchain *Blockchain) GetBlockByHash(blockHash []byte) (*types.Block, error) {
	blockBytes, err := blockchain.dbHandler.Get(blockchain.columnFamily, blockHash)
	if err != nil {
		return nil, err
	}

	if len(blockBytes) == 0 {
		return nil, errors.New("not found block ")
	}

	block := &types.Block{}
	if err := block.Deserialize(blockBytes); err != nil {
		return nil, err
	}
	return block, nil
}

// GetBlockByNumber gets block by block height number
func (blockchain *Blockchain) GetBlockByNumber(blockNum uint32) (*types.Block, error) {
	blockHashBytes, err := blockchain.getBlockHashByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	return blockchain.GetBlockByHash(blockHashBytes)
}

// GetTransactionsByNumber by block height number
func (blockchain *Blockchain) GetTransactionsByNumber(blockNum uint32, transactionType uint32) (types.Transactions, error) {
	block, err := blockchain.GetBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	if transactionType == uint32(100) {
		return block.Transactions, nil
	}

	return block.GetTransactions(transactionType)
}

// GetTransactionsByHash by block hash
func (blockchain *Blockchain) GetTransactionsByHash(blockHash []byte, transactionType uint32) (types.Transactions, error) {
	block, err := blockchain.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if transactionType == uint32(100) {
		return block.Transactions, nil
	}
	return block.GetTransactions(transactionType)
}

// GetTransactionByTxHash gets transaction by transaction hash
func (blockchain *Blockchain) GetTransactionByTxHash(txHash []byte) (*types.Transaction, error) {
	bytes, err := blockchain.dbHandler.Get(blockchain.indexColumnFamily, txHash)
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, errors.New("not found transaction by txHash")
	}
	numbers, err := utils.DecodeUint32(bytes, 2)
	if err != nil {
		return nil, err
	}
	return blockchain.getTransactionByNumber(numbers[0], numbers[1])
}

// GetBlockchainHeight gets blockchain height
func (blockchain *Blockchain) GetBlockchainHeight() (uint32, error) {
	heightBytes, _ := blockchain.dbHandler.Get(blockchain.indexColumnFamily, []byte(heightKey))
	if len(heightBytes) == 0 {
		return 0, errors.New("failed to get the height")
	}
	height := utils.BytesToUint32(heightBytes)
	return height, nil
}

// AppendBlock appends a block
func (blockchain *Blockchain) AppendBlock(block *types.Block) []*db.WriteBatch {
	blockHashBytes := block.Hash().Bytes()
	blockHeightBytes := utils.Uint32ToBytes(block.Height())

	// storage
	var writeBatchs []*db.WriteBatch
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.columnFamily, db.OperationPut, blockHashBytes, block.Serialize()))        // block hash => block
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, blockHeightBytes, blockHashBytes))    // height => block hash
	writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, []byte(heightKey), blockHeightBytes)) // update block height

	//storage  tx hash
	for txIndex, tx := range block.Transactions {
		writeBatchs = append(writeBatchs, db.NewWriteBatch(blockchain.indexColumnFamily, db.OperationPut, tx.Hash().Bytes(), encodeUint32(block.Height(), uint32(txIndex)))) // tx hash => tx detail
	}

	return writeBatchs
}

func (blockchain *Blockchain) getBlockHashByNumber(blockNum uint32) ([]byte, error) {
	currentHeight, err := blockchain.GetBlockchainHeight()

	if err != nil {
		return nil, err
	}
	if blockNum > currentHeight {
		return nil, errors.New("exceeds the max height")
	}
	blockHashBytes, err := blockchain.dbHandler.Get(blockchain.indexColumnFamily, utils.Uint32ToBytes(blockNum))
	if err != nil {
		return nil, err
	}

	if len(blockHashBytes) == 0 {
		return nil, errors.New("not found block Hash")
	}
	return blockHashBytes, nil
}

func (blockchain *Blockchain) getTransactionByNumber(blockNum uint32, index uint32) (*types.Transaction, error) {
	block, err := blockchain.GetBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	return block.Transactions[index], nil
}

func encodeUint32(numbers ...uint32) []byte {
	b := proto.NewBuffer([]byte{})
	for _, number := range numbers {
		b.EncodeVarint(uint64(number))
	}
	return b.Bytes()
}

/*
func prependKeyPrefix(prefix byte, key []byte) []byte {
	modifiedKey := []byte{}
	modifiedKey = append(modifiedKey, prefix)
	modifiedKey = append(modifiedKey, key...)
	return modifiedKey
}
*/
