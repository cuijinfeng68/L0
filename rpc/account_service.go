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

	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
)

type AccountInterface interface {
	NewAccount(passphrase string, accountType uint32) (accounts.Account, error)
	Accounts() ([]string, error)
	HasAddress(addr accounts.Address) bool
	Find(addr accounts.Address) *accounts.Account
	SignTx(a accounts.Account, tx *types.Transaction, pass string) (*types.Transaction, error)
}

// account
type Account struct {
	ai AccountInterface
}

func NewAccount(ai AccountInterface) *Account {
	return &Account{ai: ai}
}

type AccountNewArgs struct {
	AccountType uint32
	Passphrase  string
}

// NewAccount
func (a *Account) New(args *AccountNewArgs, reply *accounts.Address) error {
	newAccount, err := a.ai.NewAccount(args.Passphrase, args.AccountType)
	if err != nil {
		return err
	}
	*reply = newAccount.Address
	return nil
}

// List accounts
func (a *Account) List(param uint8, reply *[]string) error {
	addrs, err := a.ai.Accounts()
	if err != nil {
		return err
	}
	*reply = addrs
	return nil
}

// Exist
func (a *Account) Exist(addr string, reply *bool) error {
	*reply = a.ai.HasAddress(accounts.HexToAddress(addr))
	return nil
}

type SignTxArgs struct {
	OriginTx string
	Addr     string
	Pass     string
}

func (a *Account) Sign(args *SignTxArgs, reply *string) error {
	address := accounts.HexToAddress(args.Addr)
	if !a.ai.HasAddress(address) {
		return errors.New("address not exists")
	}
	account := a.ai.Find(address)
	tx := new(types.Transaction)
	txBytes := utils.HexToBytes(args.OriginTx)
	tx.Deserialize(txBytes)

	signTx, err := a.ai.SignTx(*account, tx, args.Pass)
	if err != nil {
		return err
	}

	*reply = utils.BytesToHex(signTx.Serialize())
	return nil
}
