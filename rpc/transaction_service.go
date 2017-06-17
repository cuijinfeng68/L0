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
	"errors"
	"math/big"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

type IBroadcast interface {
	Relay(inv types.IInventory)
}

type Transaction struct {
	pmHander IBroadcast
}

type TransactionCreateArgs struct {
	FromChain string
	ToChain   string
	Recipient string
	Amount    int64
	Fee       int64
	TxType    uint32
}

func NewTransaction(pmHandler IBroadcast) *Transaction {
	return &Transaction{pmHander: pmHandler}
}

func (t *Transaction) Create(args *TransactionCreateArgs, reply *string) error {
	fromChain := coordinate.HexToChainCoordinate(args.FromChain)
	toChain := coordinate.HexToChainCoordinate(args.ToChain)
	// nonce的值
	nonce := uint32(1)
	recipient := accounts.HexToAddress(args.Recipient)
	sender := accounts.Address{}
	amount := big.NewInt(args.Amount)
	fee := big.NewInt(args.Fee)

	tx := types.NewTransaction(fromChain, toChain, args.TxType, nonce, sender, recipient, amount, fee, utils.CurrentTimestamp())
	*reply = utils.BytesToHex(tx.Serialize())

	return nil
}

func (t *Transaction) Broadcast(txHex string, reply *crypto.Hash) error {
	if len(txHex) < 1 {
		return errors.New("Invalid Params: len(txSerializeData) must be >0 ")
	}

	tx := new(types.Transaction)
	tx.Deserialize(utils.HexToBytes(txHex))

	if tx.Amount().Sign() <= 0 {
		return errors.New("Invalid Amount in Tx, Amount must be >0")
	}

	if tx.Fee() == nil || tx.Fee().Sign() <= 0 {
		return errors.New("Invalid Fee in Tx, Fee must be >0")
	}

	_, err := tx.Verfiy()
	if err != nil {
		return errors.New("Invalid Tx, varify the signature of Tx failed")
	}

	t.pmHander.Relay(tx)
	*reply = tx.Hash()
	return nil
}
