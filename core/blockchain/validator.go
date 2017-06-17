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

package blockchain

import (
	"bytes"
	"container/list"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

type validatorAccount struct {
	txs    *list.List
	txMap  map[crypto.Hash]*list.Element
	amount *big.Int
	nonce  uint32
	sync.RWMutex
}

type validatorFilter struct {
	sync.Mutex
	delTxsChan    chan types.Transactions
	txCacheFilter map[crypto.Hash]bool
}

type Validator struct {
	sync.Mutex
	isValid        bool
	txPool         *LinkedList
	ledger         *ledger.Ledger
	accounts       map[string]*validatorAccount
	txsCacheFilter *validatorFilter
	delTxsChan     chan types.Transactions
}

func newValidatorFilter() *validatorFilter {
	return &validatorFilter{
		txCacheFilter: make(map[crypto.Hash]bool),
		delTxsChan:    make(chan types.Transactions, 100),
	}
}

func (vf *validatorFilter) addTxCacheFilter(txHash crypto.Hash) {
	vf.Lock()
	defer vf.Unlock()

	vf.txCacheFilter[txHash] = true
}

func (vf *validatorFilter) removeTxCacheFilter(txHash crypto.Hash) {
	vf.Lock()
	defer vf.Unlock()

	delete(vf.txCacheFilter, txHash)
}

func (vf *validatorFilter) hasTxInCacheFilter(txHash crypto.Hash) bool {
	vf.Lock()
	defer vf.Unlock()

	_, ok := vf.txCacheFilter[txHash]

	return ok
}

func (vf *validatorFilter) removeTxsCacheFilter(txs types.Transactions) {
	for _, tx := range txs {
		vf.removeTxCacheFilter(tx.Hash())
	}
}

func newValidatorAccount(address accounts.Address, leger *ledger.Ledger) *validatorAccount {
	amount, nonce, _ := leger.GetBalance(address)
	return &validatorAccount{
		amount: amount,
		nonce:  nonce + uint32(1),
		txs:    list.New(),
		txMap:  make(map[crypto.Hash]*list.Element),
	}
}

func (va *validatorAccount) addAmount(tx *types.Transaction) {
	va.Lock()
	defer va.Unlock()
	va.amount = va.amount.Add(va.amount, tx.Amount())
	log.Info("RemoveTxInVerify va.amount: ", va.amount)
}

func (va *validatorAccount) addTransaction(tx *types.Transaction) bool {
	va.Lock()
	defer va.Unlock()

	addr := tx.Sender()
	isOK := true
	amount := (&big.Int{}).Sub(va.amount, tx.Amount())
	nonce := va.nonce

	switch tx.GetType() {
	case types.TypeMerged:
	case types.TypeIssue:
		if nonce != tx.Nonce() {
			isOK = false
		}
	case types.TypeAcrossChain:
		fallthrough
	case types.TypeDistribut:
		fallthrough
	case types.TypeBackfront:
		fallthrough
	case types.TypeAtomic:
		if nonce != tx.Nonce() || amount.Sign() < 0 {
			isOK = false
		}
	case types.TypeSmartContract:
		//TODO
	default:
		log.Errorf("add: unknow tx's type, tx_hash: %v, tx_type: %v", tx.Hash().String(), tx.GetType())
	}

	if isOK {
		ele := va.txs.PushBack(tx)
		va.txMap[tx.Hash()] = ele
		va.amount.Set(amount)
		if tx.GetType() != types.TypeMerged {
			va.nonce++
		}

		log.Debugf("add: new tx, tx_hash: %v, tx_sender: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v, va.amount: %v, va.nonce: %v",
			tx.Hash().String(), addr.String(), tx.GetType(), tx.Amount(), tx.Nonce(), va.amount, va.nonce)
		return true
	}

	log.Debugf("can't add: new tx, tx_hash: %v, tx_sender: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v, va.amount: %v, va.nonce: %v",
		tx.Hash().String(), addr.String(), tx.GetType(), tx.Amount(), tx.Nonce(), va.amount, va.nonce)
	return false
}

func (va *validatorAccount) removeTransaction(tx *types.Transaction) bool {
	va.Lock()
	defer va.Unlock()

	if va.txs.Len() > 0 {
		ele := va.txs.Front()
		data := ele.Value.(*types.Transaction)
		if bytes.Equal(data.Hash().Bytes(), tx.Hash().Bytes()) {
			delete(va.txMap, tx.Hash())
			va.txs.Remove(ele)
			return true
		} else {
			log.Errorf("trx order different between consensus and leger, txs_list_first: %v, cur_tx: %v",
				data.Hash().String(), tx.Hash().String())
			// TODO  Delete all the transactions before this transaction
			// TODO And update amount of sender
			// TODO or this node stop receive tx after 20s, then reset cache
		}
	}

	return false
}

func (va *validatorAccount) checkTransaction(tx *types.Transaction) (bool, error) {
	va.Lock()
	defer va.Unlock()

	if ele, ok := va.txMap[tx.Hash()]; ok {
		otx := ele.Value.(*types.Transaction)
		res := otx.Amount().Cmp(tx.Amount())
		if res > 0 {
			va.amount = va.amount.Add(va.amount, (&big.Int{}).Sub(otx.Amount(), tx.Amount()))
		} else if res < 0 {
			amount := (&big.Int{}).Set(va.amount)
			amount = amount.Add(amount, (&big.Int{}).Sub(otx.Amount(), tx.Amount()))
			if amount.Sign() >= 0 {
				va.amount.Set(amount)
			} else {
				for be := va.txs.Back(); be != nil; be = be.Prev() {
					if be.Value.(*types.Transaction).Nonce() < otx.Nonce() {
						return true, errors.New("Tx amount is big")
					}

					if amount.Sign() < 0 {
						amount = amount.Add(amount, be.Value.(*types.Transaction).Amount())
					} else {
						var next *list.Element
						va.nonce = be.Value.(*types.Transaction).Nonce()
						for ne := be.Next(); ne != nil; ne = next {
							next = ne.Next()
							va.txs.Remove(ne)
							delete(va.txMap, ne.Value.(*types.Transaction).Hash())
						}
						break
					}

				}
				va.amount.Set(amount)
			}
		} else {

			//log.Debugf("innoment checkTranaction, otx_hash: %v, otx_nonce: %v, otx.amount: %v, " +
			//	"tx_hash: %v, tx_nonce: %v, tx_amount", otx.Hash().String(), otx.Nonce(), otx.Amount(),
			//	tx.Hash().String(), tx.Nonce(), tx.Amount())

			log.Debugf("innoment checkTranaction, otx_hash: %v, otx: %v "+
				"tx_hash: %v, otx: %v", otx.Hash().String(), otx,
				tx.Hash().String(), tx)
			return false, nil
		}

		va.txs.InsertAfter(tx, ele)
		va.txs.Remove(ele)
		va.txMap[tx.Hash()] = ele
		delete(va.txMap, otx.Hash())
		log.Debugf("checkTransaction changeTx, tx_hash: %v", tx.Hash().String())
	}

	return false, nil
}

func (vr *Validator) checkIssueTransaction(tx *types.Transaction) bool {
	address := tx.Sender()
	addressHex := utils.BytesToHex(address.Bytes())
	for _, addr := range params.PublicAddress {
		if strings.Compare(addressHex, addr) == 0 {
			return true
		}
	}

	return false
}

func (vr *Validator) checkTransaction(tx *types.Transaction) bool {
	isOK := true

	if !(strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 || (strings.Compare(tx.ToChain(), params.ChainID.String()) == 0)) {
		log.Errorf("invalid transaction, fromCahin or toChain == params.ChainID")
		return false
	}

	switch tx.GetType() {
	case types.TypeAtomic:
		//TODO fromChain==toChain
		if strings.Compare(tx.FromChain(), tx.ToChain()) != 0 {
			log.Errorf("add: fail[should fromchain == tochain], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeAcrossChain:
		//TODO the len of fromchain == the len of tochain
		if !(len(tx.FromChain()) == len(tx.ToChain()) && strings.Compare(tx.FromChain(), tx.ToChain()) != 0) {
			log.Errorf("add: fail[should(chain same floor, and different)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeDistribut:
		//TODO |fromChain - toChain| = 1 and sender_addr == receive_addr
		address := tx.Sender()
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChainParent := coordinate.HexToChainCoordinate(tx.ToChain()).ParentCoorinate()
		if !bytes.Equal(fromChain, toChainParent) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("add: fail[should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeBackfront:
		address := tx.Sender()
		fromChainParent := coordinate.HexToChainCoordinate(tx.FromChain()).ParentCoorinate()
		toChain := coordinate.HexToChainCoordinate(tx.ToChain())
		if !bytes.Equal(fromChainParent, toChain) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("add: fail[should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeMerged:
	//TODO nothing to do
	case types.TypeIssue:
		//TODO the first floor and meet issue account
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChain := coordinate.HexToChainCoordinate(tx.FromChain())

		if !(len(fromChain) == len(toChain) && strings.Compare(fromChain.String(), "00") == 0) {
			log.Errorf("add: fail[should(the first floor)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
			isOK = false
		}

		if ok := vr.checkIssueTransaction(tx); !ok {
			log.Errorf("add: valid issue tx public key fail, tx: %v", tx.Hash().String())
			isOK = false
		}

	}

	return isOK
}

func NewValidator(ledger *ledger.Ledger) *Validator {
	validator := &Validator{
		isValid:        true,
		txPool:         NewLinkedList(),
		ledger:         ledger,
		accounts:       make(map[string]*validatorAccount),
		txsCacheFilter: newValidatorFilter(),
		delTxsChan:     make(chan types.Transactions, 100),
	}
	go validator.Loop()
	return validator
}

// startValidator - start validator
func (vr *Validator) startValidator() {
	vr.isValid = true
}

// stopValidator - stop validator
func (vr *Validator) stopValidator() {
	vr.isValid = false
}

func (vr *Validator) getTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	itx := vr.txPool.Has(txHash.String())
	if itx != nil {
		tx := itx.(*types.Transaction)
		return tx, true
	}

	return nil, false
}

func (vr *Validator) getBalanceNonce(addr accounts.Address) (*big.Int, uint32) {
	senderAccount := vr.getSenderAccount(addr)
	senderAccount.Lock()
	defer senderAccount.Unlock()

	return senderAccount.amount, senderAccount.nonce
}

func (vr *Validator) getSenderAccount(address accounts.Address) *validatorAccount {
	vr.Lock()
	account, ok := vr.accounts[address.String()]
	vr.Unlock()

	if !ok {
		vr.Lock()
		vr.accounts[address.String()] = newValidatorAccount(address, vr.ledger)
		account = vr.accounts[address.String()]
		vr.Unlock()
	}

	return account
}

func (vr *Validator) updateRecipientAccount(tx *types.Transaction) {
	vr.Lock()
	defer vr.Unlock()

	if account, ok := vr.accounts[tx.Recipient().String()]; ok {
		account.amount.Add(account.amount, tx.Amount())
	}
}

func (vr *Validator) hasTransaction(tx *types.Transaction) bool {
	exist := vr.txPool.Contains(tx.Hash().String())

	return exist
}

func (vr *Validator) checkExceptionTransaction(tx *types.Transaction) (bool, error) {
	address := tx.Sender()
	senderAccount := vr.getSenderAccount(address)
	exist, err := senderAccount.checkTransaction(tx)

	return exist, err
}

func (vr *Validator) removeTxsForAccount(txs types.Transactions) {
	for _, tx := range txs {
		if strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 {
			address := tx.Sender()
			senderAccount := vr.getSenderAccount(address)
			if ok := senderAccount.removeTransaction(tx); ok {
				//vr.txPool.Remove(tx)
			}
		}

		// TODO: to update Recipient
		vr.updateRecipientAccount(tx)
	}
}

func (vr *Validator) Loop() {
	for {
		select {
		case txs := <-vr.delTxsChan:
			vr.removeTxsForAccount(txs)
		case txs := <-vr.txsCacheFilter.delTxsChan:
			vr.txsCacheFilter.removeTxsCacheFilter(txs)
		}
	}
}

func (vr *Validator) TxsLenInTxPool() int {
	return vr.txPool.Len()
}

func (vr *Validator) IterElementInTxPool(function func(*types.Transaction) bool) {
	//vr.txPool.IterElement(function)

	t1 := time.Now()
	vr.txPool.IterElement(func(element IElement) bool {
		if vr.isValid {
			txHash := element.(*types.Transaction).Hash()
			if vr.txsCacheFilter.hasTxInCacheFilter(txHash) {
				return false
			}
			vr.txsCacheFilter.addTxCacheFilter(txHash)
		}
		return function(element.(*types.Transaction))
	})

	elapsed := time.Since(t1)
	log.Info(" < --- > IterElementInTxPool elapsed: ", elapsed)

}

func (vr *Validator) VerifyTxInTxPool(tx *types.Transaction) bool {
	if vr.isValid == false {
		ok := vr.checkTransaction(tx)
		if ok {
			vr.txPool.Add(tx)
			log.Debugf("added new tx, tx_hash: %v", tx.Hash().String())
			return true
		}

		log.Debugf("can't add new tx, tx_hash: %v", tx.Hash().String())
		return false
	}

	address, err := tx.Verfiy()
	if err != nil {
		log.Debugf("varify fail, tx_hash: ", tx.Hash().String())
		return false
	}

	ok := vr.checkTransaction(tx)
	if ok {
		senderAccount := vr.getSenderAccount(address)
		vr.Lock()
		if ok = senderAccount.addTransaction(tx); ok {
			vr.txPool.Add(tx)
		}
		vr.Unlock()
	}

	return ok
}

func (vr *Validator) VerifyTxsInConsensus(txs types.Transactions, role bool) types.Transactions {
	if vr.isValid == false || role == true {
		return txs
	}

	t1 := time.Now()
	for _, tx := range txs {
		if ok := vr.hasTransaction(tx); ok {
			continue
		} else if ok, err := vr.checkExceptionTransaction(tx); ok {
			if err != nil {
				log.Errorf("VerifyTxsInConsensus invalid transaction, tx_hash: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v",
					tx.Hash().String(), tx.GetType(), tx.Amount(), tx.Nonce())
				return make(types.Transactions, 0)
			}
		} else {
			ok := vr.VerifyTxInTxPool(tx)
			if !ok {
				if ok = vr.hasTransaction(tx); !ok {
					log.Errorf("VerifyTxsInConsensus can't add transaction, tx_hash: %v, tx_type: %v, tx_amount: %v, tx_nonce: %v",
						tx.Hash().String(), tx.GetType(), tx.Amount(), tx.Nonce())
					return make(types.Transactions, 0)
				}
			}
		}
	}

	elapsed := time.Since(t1)
	log.Info(" < --- > VerifyTxsInConsensus elapsed: ", elapsed, " len: ", len(txs), " txPool len: ", vr.txPool.Len())

	return txs
}

func (vr *Validator) RemoveTxInVerify(txs types.Transactions) {
	if vr.isValid == false {
		elements := []IElement{}
		for _, tx := range txs {
			elements = append(elements, tx)
			//TxBufferPool.Put(tx)
		}
		vr.txPool.Removes(elements)
		return
	}

	t1 := time.Now()

	elements := []IElement{}
	for _, tx := range txs {
		elements = append(elements, tx)
	}
	vr.txPool.Removes(elements)
	vr.delTxsChan <- txs
	vr.txsCacheFilter.delTxsChan <- txs

	elapsed := time.Since(t1)
	log.Info(" < --- > RemoveTxInVerify elapsed: ", elapsed, " len: ", len(txs), " txPool len: ", vr.txPool.Len())
}
