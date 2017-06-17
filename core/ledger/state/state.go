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

package state

import (
	"errors"

	"math/big"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
)

// State represents the account state
type State struct {
	dbHandler     *db.BlockchainDB
	balancePrefix []byte
	columnFamily  string
	tmpBalance    map[string]*Balance
}

const (
	// OperationPlus is plus operation in state
	OperationPlus uint32 = iota
	// OperationSub is sub operation in state
	OperationSub
)

// NewState returns a new State
func NewState(db *db.BlockchainDB) *State {
	return &State{
		dbHandler:     db,
		balancePrefix: []byte("bl_"),
		columnFamily:  "balance",
		tmpBalance:    make(map[string]*Balance),
	}
}

// UpdateBalance updates the account balance
func (state *State) UpdateBalance(a accounts.Address, balance *Balance, fee *big.Int, operation uint32) ([]*db.WriteBatch, error) {
	var writeBatchs []*db.WriteBatch
	tmpBalance, err := state.GetTmpBalance(a)
	if err != nil {
		return nil, err
	}

	switch operation {
	case OperationPlus:
		if !state.checkBalance(tmpBalance.Amount, balance.Amount, big.NewInt(0), OperationPlus) {
			return nil, ErrNegativeBalance
		}
		tmpBalance.Amount.Add(tmpBalance.Amount, balance.Amount)
	case OperationSub:

		if !state.checkBalance(tmpBalance.Amount, balance.Amount, fee, OperationSub) {
			return nil, ErrNegativeBalance
		}
		tmpBalance.Amount.Sub(tmpBalance.Amount.Sub(tmpBalance.Amount, fee), balance.Amount)

		state.tmpBalance[a.String()].Nonce = balance.Nonce
	default:
		return nil, errors.New("unknown operation")
	}

	key := append(state.balancePrefix, a.Bytes()...)

	writeBatchs = append(writeBatchs, db.NewWriteBatch(state.columnFamily, db.OperationPut, key, tmpBalance.serialize()))

	return writeBatchs, nil
}

// GetBalance returns balance by account
func (state *State) GetBalance(a accounts.Address) (*big.Int, uint32, error) {
	key := append(state.balancePrefix, a.Bytes()...)
	balanceBytes, err := state.dbHandler.Get(state.columnFamily, key)

	if err != nil {
		return big.NewInt(0), 0, err
	}
	if len(balanceBytes) == 0 {
		return big.NewInt(0), 0, nil
	}
	balance := new(Balance)
	balance.deserialize(balanceBytes)
	log.Info("balanceBytes: ", balanceBytes, "Amount: ", balance.Amount)
	return balance.Amount, balance.Nonce, nil
}

// Init initializes a account
func (state *State) Init(a accounts.Address) error {
	key := append(state.balancePrefix, a.Bytes()...)
	value := big.NewInt(0).Bytes()
	err := state.dbHandler.Put(state.columnFamily, key, value)
	return err
}

// Transfer updates the sender->recipient account balance
func (state *State) Transfer(sender, recipient accounts.Address, fee *big.Int, balance *Balance, txType uint32) ([]*db.WriteBatch, error) {
	var writeBatchs []*db.WriteBatch

	senderBalance, err := state.GetTmpBalance(sender)
	if err != nil {
		return nil, err
	}

	//sender=recipient Amount deducting fee
	if sender.Equal(recipient) {
		if !state.checkBalance(senderBalance.Amount, big.NewInt(0), fee, OperationSub) && txType != types.TypeIssue {
			return nil, ErrNegativeBalance
		}
		senderBalance.Amount.Sub(senderBalance.Amount, fee)
		senderBalance.Nonce = balance.Nonce
		writeBatchs = append(writeBatchs, db.NewWriteBatch(state.columnFamily, db.OperationPut, append(state.balancePrefix, sender.Bytes()...),
			senderBalance.serialize()))
		return writeBatchs, nil
	}

	if !state.checkBalance(senderBalance.Amount, balance.Amount, fee, OperationSub) && txType != types.TypeIssue {
		return nil, ErrNegativeBalance
	}
	senderBalance.Amount.Sub(senderBalance.Amount.Sub(senderBalance.Amount, fee), balance.Amount)

	senderBalance.Nonce = balance.Nonce

	recipientBalance, err := state.GetTmpBalance(recipient)
	if err != nil {
		return nil, err
	}

	if !state.checkBalance(recipientBalance.Amount, balance.Amount, big.NewInt(0), OperationPlus) && txType != types.TypeIssue {
		return nil, ErrNegativeBalance
	}
	recipientBalance.Amount.Add(recipientBalance.Amount, balance.Amount)
	writeBatchs = append(writeBatchs, db.NewWriteBatch(state.columnFamily, db.OperationPut, append(state.balancePrefix, sender.Bytes()...),
		senderBalance.serialize()))
	writeBatchs = append(writeBatchs, db.NewWriteBatch(state.columnFamily, db.OperationPut, append(state.balancePrefix, recipient.Bytes()...),
		recipientBalance.serialize()))

	return writeBatchs, nil
}

func (state *State) GetTmpBalance(addr accounts.Address) (*Balance, error) {
	balance, ok := state.tmpBalance[addr.String()]
	if !ok {
		Amount, Nonce, err := state.GetBalance(addr)
		if err != nil {
			return nil, err
		}
		state.tmpBalance[addr.String()] = NewBalance(Amount, Nonce)
		return state.tmpBalance[addr.String()], nil
	}
	return balance, nil
}

//AtomicWrite atomic writeBatchs
func (state *State) AtomicWrite(writeBatchs []*db.WriteBatch) error {
	if err := state.dbHandler.AtomicWrite(writeBatchs); err != nil {
		return err
	}
	//clear map
	state.tmpBalance = make(map[string]*Balance)
	return nil
}

//checkBalance check negative Balance,flag = 1 add, flag = 2 sub
func (state *State) checkBalance(balance, change, fee *big.Int, operation uint32) bool {
	tmpBalance := new(big.Int)
	tmpChange := new(big.Int)
	tmpFee := new(big.Int)

	tmpBalance.Set(balance)
	tmpChange.Set(change)
	tmpFee.Set(fee)

	switch operation {
	case OperationPlus:
		tmpBalance.Add(tmpBalance, tmpChange)
	case OperationSub:
		tmpBalance.Sub(tmpBalance.Sub(tmpBalance, fee), tmpChange)
	}
	if tmpBalance.Sign() < 0 {
		return false
	}
	return true
}
