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

// the vm execute context

package vm

import (
	"errors"

	"encoding/hex"
	"math/big"

	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/types"
)

const (
	contractCodeKey = "__CONTRACT_CODE_KEY__"
)

// CTX the vm execute context
type CTX struct {
	Payload       string
	ContractAddr  string
	ContractSpec  *types.ContractSpec
	Transaction   *types.Transaction
	L0Handler     contract.ISmartConstract
	StateQueue    *stateQueue
	TransferQueue *transferQueue
}

// NewCTX create a real invoke ctx
func NewCTX(tx *types.Transaction, cs *types.ContractSpec, l0Handler contract.ISmartConstract) *CTX {
	ctx := new(CTX)
	ctx.Payload = getContractCode(cs, l0Handler)
	ctx.ContractAddr = hex.EncodeToString(cs.ContractAddr)
	ctx.Transaction = tx
	ctx.ContractSpec = cs
	ctx.L0Handler = l0Handler
	ctx.StateQueue = newStateQueue()
	ctx.TransferQueue = newTransferQueue()

	return ctx
}

func (ctx *CTX) transfer(recipientAddr string, amount int64, txType uint32) error {
	if amount <= 0 {
		return errors.New("amount must above 0")
	}

	contractAddr := ctx.ContractAddr
	var contractBalances int64
	if v, ok := ctx.TransferQueue.balancesMap[contractAddr]; ok {
		contractBalances = v
	} else {
		bls, err := ctx.L0Handler.GetBalances(contractAddr)
		if err != nil {
			return errors.New("get balances error")
		}
		contractBalances = bls.Int64()
	}

	if contractBalances < amount {
		return errors.New("balances not enough")
	}

	var recipientBalances int64
	if v, ok := ctx.TransferQueue.balancesMap[recipientAddr]; ok {
		recipientBalances = v
	} else {
		bls, err := ctx.L0Handler.GetBalances(recipientAddr)
		if err != nil {
			return errors.New("get balances error")
		}
		recipientBalances = bls.Int64()
	}

	ctx.TransferQueue.balancesMap[contractAddr] = contractBalances - amount
	ctx.TransferQueue.balancesMap[recipientAddr] = recipientBalances + amount

	ctx.TransferQueue.offer(&transferOpfunc{txType, contractAddr, recipientAddr, big.NewInt(amount)})

	return nil
}

func (ctx *CTX) payload() string {
	return ctx.Payload
}

func (ctx *CTX) currentBlockHeight() uint32 {
	return ctx.L0Handler.CurrentBlockHeight()
}

func (ctx *CTX) getBalances(addr string) (*big.Int, error) {
	if v, ok := ctx.TransferQueue.balancesMap[addr]; ok {
		return big.NewInt(v), nil
	}

	return ctx.L0Handler.GetBalances(addr)
}

func (ctx *CTX) putState(key string, value []byte) error {
	if err := checkStateKey(key); err != nil {
		return err
	}

	ctx.StateQueue.stateMap[key] = value
	ctx.StateQueue.offer(&stateOpfunc{stateOpTypePut, key, value})
	return nil
}

func (ctx *CTX) getState(key string) ([]byte, error) {
	if err := checkStateKey(key); err != nil {
		return nil, err
	}

	if v, ok := ctx.StateQueue.stateMap[key]; ok {
		return v, nil
	}

	return ctx.L0Handler.GetState(key)
}

func (ctx *CTX) delState(key string) error {
	if err := checkStateKey(key); err != nil {
		return err
	}

	ctx.StateQueue.stateMap[key] = nil
	ctx.StateQueue.offer(&stateOpfunc{stateOpTypeDelete, key, nil})
	return nil
}

func (ctx *CTX) commit() {
	for {
		txOP := ctx.TransferQueue.poll()
		if txOP == nil {
			break
		}
		ctx.L0Handler.AddTransfer(txOP.from, txOP.to, txOP.amount, txOP.txType)
	}

	for {
		stateOP := ctx.StateQueue.poll()
		if stateOP == nil {
			break
		}

		if stateOP.optype == stateOpTypePut {
			ctx.L0Handler.AddState(stateOP.key, stateOP.value)
		} else if stateOP.optype == stateOpTypeDelete {
			ctx.L0Handler.DelState(stateOP.key)
		}
	}

	ctx.L0Handler.SmartContractCommitted()
}

func getContractCode(cs *types.ContractSpec, l0Handler contract.ISmartConstract) string {
	code := cs.ContractCode
	if code != nil && len(code) > 0 {
		l0Handler.AddState(contractCodeKey, code)
		return string(code)
	}

	code, err := l0Handler.GetState(contractCodeKey)
	if code != nil && err == nil {
		return string(code)
	}

	return ""
}

func checkStateKey(key string) error {
	if contractCodeKey == key {
		return errors.New("state key illegal:" + key)
	}

	return nil
}
