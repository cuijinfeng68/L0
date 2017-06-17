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

package rpc

import (
	"math/big"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/types"
)

//LedgerInterface ledger interface
type LedgerInterface interface {
	Height() (uint32, error)
	GetBalance(addr accounts.Address) (*big.Int, uint32, error)
	GetBalanceNonce(addr accounts.Address) (*big.Int, uint32)
	GetTransaction(txHash crypto.Hash) (*types.Transaction, error)
	GetBlockByHash(blockHashBytes []byte) (*types.Block, error)
	GetBlockByNumber(number uint32) (*types.Block, error)
	GetLastBlockHash() (crypto.Hash, error)
	GetTxsByBlockHash(blockHashBytes []byte, transactionType uint32) (types.Transactions, error)
	GetTxsByBlockNumber(blockNumber uint32, transactionType uint32) (types.Transactions, error)
	GetTxsByMergeTxHash(mergeTxHash crypto.Hash) (types.Transactions, error)
	GetAtmoicTxsStatistics() int
	GetAcrossTxsStatistics() (int, int)
	GetBlockAtmoicTxsStatistics() int
	GetBlockAcrossTxsStatistics() (int, int, int, int)
}

//Ledger ledger rpc api
type Ledger struct {
	ledger LedgerInterface
}

//NewLedger initialization
func NewLedger(legderInterface LedgerInterface) *Ledger {
	return &Ledger{ledger: legderInterface}
}

//GetTxsByBlockNumberArgs get txs by block number args
type GetTxsByBlockNumberArgs struct {
	BlockNumber uint32
	TxType      uint32
}

//GetTxsByBlockHashArgs get txs by block hash args
type GetTxsByBlockHashArgs struct {
	BlockHash string
	TxType    uint32
}

//Height get blockchain height
func (l *Ledger) Height(ignore string, reply *uint32) error {
	height, err := l.ledger.Height()
	if err != nil {
		return err
	}
	*reply = height
	return nil
}

//GetBalance returns balance by account address
func (l *Ledger) GetBalance(addr string, reply *state.Balance) error {
	amount, nonce, err := l.ledger.GetBalance(accounts.HexToAddress(addr))
	if err != nil {
		return err
	}
	*reply = state.Balance{Amount: amount, Nonce: nonce}
	return nil
}

//GetBalanceInTxPool return nonce
func (l *Ledger) GetBalanceInTxPool(addr string, reply *state.Balance) error {
	amount, nonce := l.ledger.GetBalanceNonce(accounts.HexToAddress(addr))
	nonce = nonce - 1
	log.Debug("amount: ", amount, " nonce: ", nonce)
	*reply = state.Balance{Amount: amount, Nonce: nonce}
	return nil
}

//GetTxByHash returns transaction by tx hash []byte
func (l *Ledger) GetTxByHash(txHashBytes string, reply *types.Transaction) error {
	tx, err := l.ledger.GetTransaction(crypto.HexToHash(txHashBytes))
	if err != nil {
		return err
	}
	*reply = *tx
	return nil
}

// GetBlockByHash returns the block detail by hash
func (l *Ledger) GetBlockByHash(blockHashBytes string, reply *types.Block) error {
	block, err := l.ledger.GetBlockByHash(crypto.HexToHash(blockHashBytes).Bytes())
	if err != nil {
		return err
	}
	*reply = *block
	return nil
}

//GetBlockByNumber get block by block number
func (l *Ledger) GetBlockByNumber(number uint32, reply *types.Block) error {
	block, err := l.ledger.GetBlockByNumber(number)
	if err != nil {
		return err
	}
	*reply = *block
	return nil
}

//GetLastBlockHash returns the last Block hash
func (l *Ledger) GetLastBlockHash(ignore string, reply *crypto.Hash) error {
	blockHash, err := l.ledger.GetLastBlockHash()
	if err != nil {
		return err
	}
	*reply = blockHash
	return nil
}

//GetTxsByBlockHash get txs by block hash
func (l *Ledger) GetTxsByBlockHash(args GetTxsByBlockHashArgs, reply *types.Transactions) error {
	txs, err := l.ledger.GetTxsByBlockHash(crypto.HexToHash(args.BlockHash).Bytes(), args.TxType)
	if err != nil {
		return err
	}
	*reply = txs
	return nil
}

//GetTxsByBlockNumber get txs by block number
func (l *Ledger) GetTxsByBlockNumber(args GetTxsByBlockNumberArgs, reply *types.Transactions) error {

	txs, err := l.ledger.GetTxsByBlockNumber(args.BlockNumber, args.TxType)
	if err != nil {
		return err
	}
	*reply = txs
	return nil
}

//GetTxsByMergeTxHash return cross chain transactions by merge transaction
func (l *Ledger) GetTxsByMergeTxHash(mergeTxHash string, reply *types.Transactions) error {
	txs, err := l.ledger.GetTxsByMergeTxHash(crypto.HexToHash(mergeTxHash))
	if err != nil {
		return err
	}
	*reply = txs
	return nil
}

//DeserializeTx deserializes transaction by transaction serialize string
func (l *Ledger) DeserializeTx(hexString string, reply *types.Transaction) error {
	tx := new(types.Transaction)
	if err := tx.Deserialize(utils.HexToBytes(hexString)); err != nil {
		return err
	}
	*reply = *tx
	return nil
}
