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

package contract

import (
	"errors"
	"math/big"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
)

var DeployAddr = []byte("00000000000000000000")

type ILedgerSmartContract interface {
	GetTmpBalance(addr accounts.Address) (*big.Int, error)
	Height() (uint32, error)
}

type ISmartConstract interface {
	GetState(key string) ([]byte, error)
	AddState(key string, value []byte)
	DelState(key string)
	GetBalances(addr string) (*big.Int, error)
	CurrentBlockHeight() uint32
	AddTransfer(fromAddr, toAddr string, amount *big.Int, txType uint32)
	SmartContractFailed()
	SmartContractCommitted()
}

// State represents the account state
type SmartConstract struct {
	dbHandler     *db.BlockchainDB
	balancePrefix []byte
	columnFamily  string
	ledgerHandler ILedgerSmartContract
	stateExtra    *StateExtra

	height           uint32
	scAddr           string
	committed        bool
	currentTx        *types.Transaction
	smartContractTxs types.Transactions
}

// NewState returns a new State
func NewSmartConstract(db *db.BlockchainDB, ledgerHandler ILedgerSmartContract) *SmartConstract {
	return &SmartConstract{
		dbHandler:     db,
		balancePrefix: []byte("sc_"),
		columnFamily:  "scontract",
		ledgerHandler: ledgerHandler,
		stateExtra:    NewStateExtra(),
	}
}

// StartConstract start constract
func (sctx *SmartConstract) StartConstract(blockHeight uint32) {
	log.Debugf("startConstract() for blockHeight [%d]", blockHeight)
	if !sctx.InProgress() {
		log.Errorf("A tx [%d] is already in progress. Received call for begin of another smartcontract [%d]", sctx.height, blockHeight)
	}
	sctx.height = blockHeight
}

// StopContract start contract
func (sctx *SmartConstract) StopContract(blockHeight uint32) {
	log.Debugf("stopConstract() for blockHeight [%d]", blockHeight)
	if sctx.height != blockHeight {
		log.Errorf("Different blockHeight in contract-begin [%s] and contract-finish [%s]", sctx.height, blockHeight)
	}

	sctx.height = 0
	sctx.stateExtra = NewStateExtra()
}

// ExecTransaction exec transaction
func (sctx *SmartConstract) ExecTransaction(tx *types.Transaction, scAddr string) {
	sctx.committed = false
	sctx.currentTx = tx
	sctx.scAddr = scAddr
	sctx.smartContractTxs = make(types.Transactions, 0)
}

// GetState get value
func (sctx *SmartConstract) GetState(key string) ([]byte, error) {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	value := sctx.stateExtra.get(sctx.scAddr, key)
	if len(value) == 0 {
		var err error
		scAddrkey := EnSmartContractKey(sctx.scAddr, key)
		value, err = sctx.dbHandler.Get(sctx.columnFamily, []byte(scAddrkey))
		if err != nil || len(value) == 0 {
			return nil, errors.New("can't get date from db")
		}
	}

	return value, nil
}

// AddState put key-value into cache
func (sctx *SmartConstract) AddState(key string, value []byte) {
	log.Debugf("PutState smartcontract=[%s], key=[%s], value=[%#v]", sctx.scAddr, key, value)
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	sctx.stateExtra.set(sctx.scAddr, key, value)
}

// DelState remove key-value
func (sctx *SmartConstract) DelState(key string) {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	sctx.stateExtra.delete(sctx.scAddr, key)
}

// GetBalances get balance
func (sctx *SmartConstract) GetBalances(addr string) (*big.Int, error) {
	return sctx.ledgerHandler.GetTmpBalance(accounts.HexToAddress(addr))
}

// CurrentBlockHeight get currentBlockHeight
func (sctx *SmartConstract) CurrentBlockHeight() uint32 {
	height, err := sctx.ledgerHandler.Height()
	if err == nil {
		log.Errorf("can't read blockchain height")
	}

	return height
}

// SmartContractFailed execute smartContract fail
func (sctx *SmartConstract) SmartContractFailed() {
	sctx.committed = false
	log.Errorf("VM can't put state into L0")
}

// SmartContractCommitted execute smartContract successfully
func (sctx *SmartConstract) SmartContractCommitted() {
	sctx.committed = true
}

// AddTransfer add transfer to make new transaction
func (sctx *SmartConstract) AddTransfer(fromAddr, toAddr string, amount *big.Int, txType uint32) {
	tx := types.NewTransaction(sctx.currentTx.Data.FromChain, sctx.currentTx.Data.ToChain, txType,
		sctx.currentTx.Data.Nonce, accounts.HexToAddress(fromAddr), accounts.HexToAddress(toAddr),
		amount, sctx.currentTx.Data.Fee, sctx.currentTx.Data.CreateTime)

	sctx.smartContractTxs = append(sctx.smartContractTxs, tx)
}

// InProgress
func (sctx *SmartConstract) InProgress() bool {
	return true
}

// FinishContractTransaction finish contract transaction
func (sctx *SmartConstract) FinishContractTransaction() (types.Transactions, error) {
	if !sctx.committed {
		return nil, errors.New("Execute VM Fail ....")
	}

	return sctx.smartContractTxs, nil
}

// AddChangesForPersistence put cache data into db
func (sctx *SmartConstract) AddChangesForPersistence(writeBatch []*db.WriteBatch) ([]*db.WriteBatch, error) {
	updateContractStateDelta := sctx.stateExtra.getUpdatedContractStateDelta()
	for _, smartContract := range updateContractStateDelta {
		updates := smartContract.getUpdatedKVs()
		for _, value := range updates {
			if value.optype == db.OperationDelete {
				log.Debugf("Contract Del: %s", value.key)
				writeBatch = append(writeBatch, db.NewWriteBatch(sctx.columnFamily, db.OperationDelete, []byte(value.key), value.value))
			} else if value.optype == db.OperationPut {
				log.Debugf("Contract Put: %s", value.key)
				writeBatch = append(writeBatch, db.NewWriteBatch(sctx.columnFamily, db.OperationPut, []byte(value.key), value.value))
			} else {
				log.Errorf("invalid method ...")
			}
		}
	}

	return writeBatch, nil
}
