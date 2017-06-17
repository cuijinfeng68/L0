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

package ledger

import (
	"fmt"
	"math/big"
	"strings"

	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger/block_storage"
	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/ledger/merge"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/vm"
)

var (
	ledgerInstance *Ledger
)

// Ledger represents the ledger in blockchain
type Ledger struct {
	block    *block_storage.Blockchain
	state    *state.State
	storage  *merge.Storage
	contract *contract.SmartConstract

	sync.Mutex
	atmoicTxsStatistics     int
	acrossTxsStatistics     map[string]int
	blockAtmoicTxStatistics int
	blockAcrossTxStatistics map[string]int
}

// NewLedger returns the ledger instance
func NewLedger(db *db.BlockchainDB) *Ledger {
	if ledgerInstance == nil {
		ledgerInstance = &Ledger{
			block:                   block_storage.NewBlockchain(db),
			state:                   state.NewState(db),
			storage:                 merge.NewStorage(db),
			atmoicTxsStatistics:     0,
			acrossTxsStatistics:     make(map[string]int),
			blockAtmoicTxStatistics: 0,
			blockAcrossTxStatistics: make(map[string]int),
		}
		_, err := ledgerInstance.Height()
		if err != nil {
			ledgerInstance.init()
		}
	}

	ledgerInstance.contract = contract.NewSmartConstract(db, ledgerInstance)
	return ledgerInstance
}

// VerifyChain verifys the blockchain data
func (ledger *Ledger) VerifyChain() {
	height, err := ledger.Height()
	if err != nil {
		panic(err)
	}

	currentBlock, err := ledger.GetBlockByNumber(height)
	for i := height; i >= 1; i-- {
		previousBlock, err := ledger.GetBlockByNumber(i - 1) // storage
		if previousBlock != nil && err != nil {
			log.Debug("get block err")
			panic(err)
		}

		// verify previous block
		if !previousBlock.Hash().Equal(currentBlock.Header.PreviousHash) {
			panic(fmt.Errorf("block [%d], veifychain breaks", i))
		}
		currentBlock = previousBlock
	}
}

// GetGenesisBlock returns the genesis block of the ledger
func (ledger *Ledger) GetGenesisBlock() *types.Block {

	genesisBlock, err := ledger.GetBlockByNumber(0)
	if err != nil {
		panic(err)
	}
	return genesisBlock
}

// AppendBlock appends a new block to the ledger,flag = true pack up block ,flag = false sync block
func (ledger *Ledger) AppendBlock(block *types.Block, flag bool) error {
	var err error
	var txWriteBatchs []*db.WriteBatch

	txWriteBatchs, block.Transactions, err = ledger.executeTransaction(block.Transactions)
	if err != nil {
		return err
	}

	block.Header.TxsMerkleHash = merkleRootHash(block.Transactions)
	writeBatchs := ledger.block.AppendBlock(block)

	writeBatchs = append(writeBatchs, txWriteBatchs...)

	if err := ledger.state.AtomicWrite(writeBatchs); err != nil {
		return nil
	}

	if flag {
		var txs types.Transactions
		for _, tx := range block.Transactions {
			if (tx.GetType() == types.TypeMerged && !ledger.checkCoordinate(tx)) || tx.GetType() == types.TypeAcrossChain {
				txs = append(txs, tx)
			}
		}
		if err := ledger.storage.ClassifiedTransaction(txs); err != nil {
			return err
		}
		log.Infoln("blockHeight: ", block.Height(), "need merge Txs len : ", len(txs), "all Txs len: ", len(block.Transactions))
	}

	return nil
}

// GetBlockByNumber gets the block by the given number
func (ledger *Ledger) GetBlockByNumber(number uint32) (*types.Block, error) {

	return ledger.block.GetBlockByNumber(number)
}

// GetBlockByHash returns the block detail by hash
func (ledger *Ledger) GetBlockByHash(blockHashBytes []byte) (*types.Block, error) {

	return ledger.block.GetBlockByHash(blockHashBytes)
}

// Height returns height of ledger, return -1 if not exist
func (ledger *Ledger) Height() (uint32, error) {

	return ledger.block.GetBlockchainHeight()
}

//GetLastBlockHash returns last block hash
func (ledger *Ledger) GetLastBlockHash() (crypto.Hash, error) {
	height, err := ledger.block.GetBlockchainHeight()
	if err != nil {
		return crypto.Hash{}, err
	}
	lastBlock, err := ledger.block.GetBlockByNumber(height)
	if err != nil {
		return crypto.Hash{}, err
	}

	return lastBlock.Hash(), nil
}

// GetTxsByBlockHash returns transactions  by block hash and transactionType
func (ledger *Ledger) GetTxsByBlockHash(blockHashBytes []byte, transactionType uint32) (types.Transactions, error) {

	return ledger.block.GetTransactionsByHash(blockHashBytes, transactionType)
}

//GetTxsByBlockNumber returns transactions by blcokNumber and transactionType
func (ledger *Ledger) GetTxsByBlockNumber(blockNumber uint32, transactionType uint32) (types.Transactions, error) {

	return ledger.block.GetTransactionsByNumber(blockNumber, transactionType)
}

//GetTxByTxHash returns transaction by tx hash []byte
func (ledger *Ledger) GetTxByTxHash(txHashBytes []byte) (*types.Transaction, error) {

	return ledger.block.GetTransactionByTxHash(txHashBytes)
}

// GetBalance returns balance by account
func (ledger *Ledger) GetBalance(addr accounts.Address) (*big.Int, uint32, error) {

	return ledger.state.GetBalance(addr)
}

//GetMergedTransaction returns merged transaction within a specified period of time
func (ledger *Ledger) GetMergedTransaction(duration uint32) (types.Transactions, error) {

	t1 := time.Now()
	txs, err := ledger.storage.GetMergedTransaction(duration)
	if err != nil {
		return nil, err
	}
	delay1 := time.Since(t1)
	log.Debug("getMerge delay :", delay1)
	return txs, nil
}

//PutTxsHashByMergeTxHash put transactions hashs by merge transaction hash
func (ledger *Ledger) PutTxsHashByMergeTxHash(mergeTxHash crypto.Hash, txsHashs []crypto.Hash) error {
	return ledger.storage.PutTxsHashByMergeTxHash(mergeTxHash, txsHashs)
}

//GetTxsByMergeTxHash gets transactions
func (ledger *Ledger) GetTxsByMergeTxHash(mergeTxHash crypto.Hash) (types.Transactions, error) {
	txsHashs, err := ledger.storage.GetTxsByMergeTxHash(mergeTxHash)
	if err != nil {
		return nil, err
	}

	txs := types.Transactions{}
	for _, v := range txsHashs {
		tx, err := ledger.GetTxByTxHash(v.Bytes())
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

// init generates the genesis block
func (ledger *Ledger) init() error {
	blockHeader := new(types.BlockHeader)
	blockHeader.TimeStamp = uint32(0)
	blockHeader.Nonce = uint32(100)
	blockHeader.Height = 0

	genesisBlock := new(types.Block)
	genesisBlock.Header = blockHeader
	writeBatchs := ledger.block.AppendBlock(genesisBlock)

	return ledger.state.AtomicWrite(writeBatchs)
}

func (ledger *Ledger) commitedTranaction(tx *types.Transaction, writeBatchs []*db.WriteBatch) ([]*db.WriteBatch, error) {
	ledger.Lock()
	defer ledger.Unlock()
	var err error
	ledger.blockAtmoicTxStatistics = 0
	ledger.blockAcrossTxStatistics = make(map[string]int)
	switch tx.GetType() {
	case types.TypeIssue:
		if writeBatchs, err = ledger.executeIssueTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	case types.TypeAtomic:
		if writeBatchs, err = ledger.executeAtomicTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	case types.TypeAcrossChain:
		ledger.blockAtmoicTxStatistics++
		ledger.atmoicTxsStatistics++
		if writeBatchs, err = ledger.executeACrossChainTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	case types.TypeMerged:
		if writeBatchs, err = ledger.executeMergedTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	case types.TypeBackfront:
		if writeBatchs, err = ledger.executeBackfrontTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	case types.TypeDistribut:
		if writeBatchs, err = ledger.executeDistriTx(writeBatchs, tx); err != nil {
			return nil, err
		}
	}

	return writeBatchs, err
}

func (ledger *Ledger) executeTransaction(Txs types.Transactions) ([]*db.WriteBatch, types.Transactions, error) {
	var err error
	var writeBatchs []*db.WriteBatch
	var ctxs types.Transactions
	var txs types.Transactions

	bh, _ := ledger.Height()
	ledger.contract.StartConstract(bh)
	for _, tx := range Txs {
		if tx.GetType() == types.TypeSmartContract {
			txs, writeBatchs, err = ledger.executeSmartContractTx(writeBatchs, tx)
			if err != nil {
				return nil, nil, err
			} else {
				ctxs = append(ctxs, txs...)
			}
		}

		writeBatchs, err = ledger.commitedTranaction(tx, writeBatchs)
		if err != nil {
			return nil, nil, err
		}
	}

	Txs = append(Txs, ctxs...)
	writeBatchs, err = ledger.contract.AddChangesForPersistence(writeBatchs)
	if err != nil {
		return nil, nil, err
	}
	ledger.contract.StopContract(bh)
	return writeBatchs, Txs, nil
}

func (ledger *Ledger) executeIssueTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	sender := tx.Sender()
	atomicTxWriteBatchs, err := ledger.state.Transfer(sender, tx.Recipient(), tx.Fee(), state.NewBalance(tx.Amount(), tx.Nonce()), types.TypeIssue)
	if err != nil {
		return writeBatchs, err
	}
	writeBatchs = append(writeBatchs, atomicTxWriteBatchs...)

	return writeBatchs, nil
}

func (ledger *Ledger) executeAtomicTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	sender := tx.Sender()
	atomicTxWriteBatchs, err := ledger.state.Transfer(sender, tx.Recipient(), tx.Fee(), state.NewBalance(tx.Amount(), tx.Nonce()), types.TypeAtomic)
	if err != nil {
		if err == state.ErrNegativeBalance {
			//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
			return writeBatchs, nil
		}
		return writeBatchs, err
	}
	writeBatchs = append(writeBatchs, atomicTxWriteBatchs...)

	return writeBatchs, nil
}

func (ledger *Ledger) executeACrossChainTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	chainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes()
	if bytes.Equal(chainID, params.ChainID) {
		ledger.addAcrossTxsCnt("send:" + tx.ToChain())
		sender := tx.Sender()
		TxWriteBatch, err := ledger.state.UpdateBalance(sender, state.NewBalance(tx.Amount(), tx.Nonce()), tx.Fee(), state.OperationSub)
		if err != nil {
			if err == state.ErrNegativeBalance {
				//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
				return writeBatchs, nil
			}
			return writeBatchs, err
		}
		writeBatchs = append(writeBatchs, TxWriteBatch...)
	} else {
		ledger.addAcrossTxsCnt("recv:" + tx.FromChain())
		mergedTxWriteBatchs, err := ledger.state.UpdateBalance(tx.Recipient(), state.NewBalance(tx.Amount(), tx.Nonce()), tx.Fee(), state.OperationPlus)
		if err != nil {
			if err == state.ErrNegativeBalance {
				//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
				return writeBatchs, nil
			}
			return writeBatchs, err
		}

		writeBatchs = append(writeBatchs, mergedTxWriteBatchs...)
	}
	return writeBatchs, nil
}

func (ledger *Ledger) executeMergedTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	//mergeTx not continue merge
	if tx.GetType() == types.TypeMerged && ledger.checkCoordinate(tx) {
		sender := tx.Data.Signature.Bytes()
		senderAddress := accounts.NewAddress(sender)
		TxWriteBatchs, err := ledger.state.Transfer(senderAddress, tx.Recipient(), tx.Fee(), state.NewBalance(tx.Amount(), tx.Nonce()), tx.GetType())
		if err != nil {
			if err == state.ErrNegativeBalance {
				//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
				return writeBatchs, nil
			}
			return writeBatchs, err
		}
		writeBatchs = append(writeBatchs, TxWriteBatchs...)
		return writeBatchs, nil
	}

	return ledger.executeACrossChainTx(writeBatchs, tx)
}

func (ledger *Ledger) executeDistriTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	chainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes()
	if bytes.Equal(chainID, params.ChainID) {
		chainAddress := accounts.ChainCoordinateToAddress(coordinate.HexToChainCoordinate(tx.ToChain()))
		TxWriteBatch, err := ledger.state.UpdateBalance(chainAddress, state.NewBalance(tx.Amount(), uint32(0)), big.NewInt(0), state.OperationPlus)
		if err != nil {
			if err == state.ErrNegativeBalance {
				//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
				return writeBatchs, nil
			}
			return writeBatchs, err
		}
		writeBatchs = append(writeBatchs, TxWriteBatch...)
	}
	return ledger.executeACrossChainTx(writeBatchs, tx)
}

func (ledger *Ledger) executeBackfrontTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) ([]*db.WriteBatch, error) {
	//Backfront transaction
	chainID := coordinate.HexToChainCoordinate(tx.ToChain()).Bytes()
	if bytes.Equal(chainID, params.ChainID) {
		chainAddress := accounts.ChainCoordinateToAddress(coordinate.HexToChainCoordinate(tx.ToChain()))
		TxWriteBatch, err := ledger.state.UpdateBalance(chainAddress, state.NewBalance(tx.Amount(), uint32(0)), big.NewInt(0), state.OperationSub)
		if err != nil {
			if err == state.ErrNegativeBalance {
				//log.Debugf("execute transaction: %s, err:%s\n", tx.Hash().String(), err)
				return writeBatchs, nil
			}
			return writeBatchs, err
		}
		writeBatchs = append(writeBatchs, TxWriteBatch...)
	}
	return ledger.executeACrossChainTx(writeBatchs, tx)
}

func (ledger *Ledger) executeSmartContractTx(writeBatchs []*db.WriteBatch, tx *types.Transaction) (types.Transactions, []*db.WriteBatch, error) {
	contractSpec := new(types.ContractSpec)
	utils.Deserialize(tx.Payload, contractSpec)
	ledger.contract.ExecTransaction(tx, string(contractSpec.ContractAddr))
	ctx := vm.NewCTX(tx, contractSpec, ledger.contract)
	_, err := vm.RealExecute(ctx)
	if err != nil {
		log.Errorf("contract execute failed ......")
		return nil, nil, errors.New("contract execute failed ......")
	}

	smartContractTxs, err := ledger.contract.FinishContractTransaction()
	if err != nil {
		log.Error("FinishContractTransaction: ", err)
		return nil, nil, err
	}

	for _, tx := range smartContractTxs {
		writeBatchs, err = ledger.commitedTranaction(tx, writeBatchs)
		if err != nil {
			return nil, nil, err
		}
	}

	return smartContractTxs, writeBatchs, nil
}

func (ledger *Ledger) checkCoordinate(tx *types.Transaction) bool {
	fromChainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes()
	toChainID := coordinate.HexToChainCoordinate(tx.ToChain()).Bytes()
	if bytes.Equal(fromChainID, toChainID) {
		return true
	}
	return false
}

func (ledger *Ledger) GetTmpBalance(addr accounts.Address) (*big.Int, error) {
	balance, err := ledger.state.GetTmpBalance(addr)
	if err != nil {
		log.Error("can't get balance from db")
	}

	return balance.Amount, err
}

func merkleRootHash(txs []*types.Transaction) crypto.Hash {
	if len(txs) > 0 {
		hashs := make([]crypto.Hash, 0)
		for _, tx := range txs {
			hashs = append(hashs, tx.Hash())
		}
		return crypto.ComputeMerkleHash(hashs)[0]
	}
	return crypto.Hash{}
}

func (ledger *Ledger) GetAtmoicTxsStatistics() int {
	return ledger.atmoicTxsStatistics
}

func (ledger *Ledger) GetAcrossTxsStatistics() (int, int) {
	ledger.Lock()
	defer ledger.Unlock()
	var allSendAcrossTxCnt, allRecvAcrossTxCnt int
	for k, v := range ledger.acrossTxsStatistics {
		if strings.Contains(k, "send:") {
			allSendAcrossTxCnt = allSendAcrossTxCnt + v
		} else {
			allRecvAcrossTxCnt = allRecvAcrossTxCnt + v
		}
	}
	return allSendAcrossTxCnt, allRecvAcrossTxCnt
}

func (ledger *Ledger) GetBlockAtmoicTxsStatistics() int {
	return ledger.blockAtmoicTxStatistics
}

func (ledger *Ledger) GetBlockAcrossTxsStatistics() (int, int, int, int) {
	ledger.Lock()
	defer ledger.Unlock()
	var sendAcrossTxCnt, recvAcrossTxCnt, sendAcrossChainCnt, recvAcrossChainCnt int
	for k, v := range ledger.blockAcrossTxStatistics {
		if strings.Contains(k, "send:") {
			sendAcrossTxCnt = sendAcrossTxCnt + v
			sendAcrossChainCnt++
		} else {
			recvAcrossTxCnt = recvAcrossTxCnt + v
			recvAcrossChainCnt++
		}
	}
	return sendAcrossTxCnt, recvAcrossTxCnt, sendAcrossChainCnt, recvAcrossChainCnt
}

func (ledger *Ledger) addAcrossTxsCnt(key string) {
	ledger.blockAcrossTxStatistics[key]++
	ledger.acrossTxsStatistics[key]++
}
