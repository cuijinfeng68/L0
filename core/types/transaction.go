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
	"errors"
	"math/big"
	"sync/atomic"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
)

var (
	// ErrEmptySignature represents no signature
	ErrEmptySignature = errors.New("Signature Empty Error")
)

// Transaction represents the basic transaction that contained in blocks
type Transaction struct {
	Data    txdata `json:"data"`
	Payload []byte `json:"payload"`

	hash   atomic.Value
	sender atomic.Value
}

type ContractSpec struct {
	ContractAddr []byte
	ContractCode []byte
	ContractParams []string
}

type txdata struct {
	FromChain  coordinate.ChainCoordinate `json:"fromChain"`
	ToChain    coordinate.ChainCoordinate `json:"toChain"`
	Type       uint32                     `json:"type"`
	Nonce      uint32                     `json:"nonce"`
	Sender     accounts.Address           `json:"sender"`
	Recipient  accounts.Address           `json:"recipient"`
	Amount     *big.Int                   `json:"amount"`
	Fee        *big.Int                   `json:"fee"`
	Signature  *crypto.Signature          `json:"signature"`
	CreateTime uint32                     `json:"createTime"`
}

// Transaction type
const (
	TypeAtomic      uint32 = iota // 链内交易
	TypeAcrossChain               // 跨链交易
	TypeMerged                    // 跨链合并交易
	TypeBackfront                 // 资金回笼交易
	TypeDistribut                 // 下发交易
	TypeIssue                     // 发行交易
	TypeSmartContract             // contract
)

// NewTransaction creates an new transaction with the parameters
func NewTransaction(
	fromChain coordinate.ChainCoordinate,
	toChain coordinate.ChainCoordinate,
	txType uint32, nonce uint32, sender, reciepent accounts.Address,
	amount, fee *big.Int, CreateTime uint32) *Transaction {
	tx := Transaction{
		Data: txdata{
			FromChain:  fromChain,
			ToChain:    toChain,
			Type:       txType,
			Nonce:      nonce,
			Sender:     sender,
			Recipient:  reciepent,
			Amount:     amount,
			Fee:        fee,
			CreateTime: CreateTime,
		},
	}
	return &tx
}

// Hash returns the hash of a transaction
func (tx *Transaction) Hash() crypto.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(crypto.Hash)
	}
	v := crypto.DoubleSha256(tx.Serialize())
	tx.hash.Store(v)
	return v
}

// SignHash returns the hash of a raw transaction before sign
func (tx *Transaction) SignHash() crypto.Hash {
	rawTx := NewTransaction(
		tx.Data.FromChain,
		tx.Data.ToChain,
		tx.Data.Type,
		tx.Data.Nonce,
		tx.Data.Sender,
		tx.Data.Recipient,
		tx.Data.Amount,
		tx.Data.Fee,
		tx.Data.CreateTime,
	)
	rawTx.Payload = tx.Payload
	return rawTx.Hash()
}

// Serialize returns the serialized bytes of a transaction
func (tx *Transaction) Serialize() []byte {
	return utils.Serialize(tx)
}

// Deserialize deserializes bytes to a transaction
func (tx *Transaction) Deserialize(data []byte) error {
	return utils.Deserialize(data, tx)
}

// Also can use this method verify signature
func (tx *Transaction) Verfiy() (accounts.Address, error) {
	var (
		a   accounts.Address
		err error
	)

	switch tx.GetType() {
	case TypeAtomic:
		fallthrough
	case TypeBackfront:
		fallthrough
	case TypeDistribut:
		fallthrough
	case TypeAcrossChain:
		fallthrough
	case TypeIssue:
		if tx.Data.Signature != nil {
			if sender := tx.sender.Load(); sender != nil {
				return sender.(accounts.Address), nil
			}
			p, err := tx.Data.Signature.RecoverPublicKey(tx.SignHash().Bytes())
			if err != nil {
				return a, err
			}
			a = accounts.PublicKeyToAddress(*p)
			tx.sender.Store(a)
		} else {
			err = ErrEmptySignature
		}

	case TypeMerged:
		a = accounts.ChainCoordinateToAddress(coordinate.HexToChainCoordinate(tx.FromChain()))
	}

	return a, err
}

// Sender returns the address of the sender.
func (tx *Transaction) Sender() accounts.Address {
	return tx.Data.Sender
}

// FromChain returns the chain coordinate of the sender
func (tx *Transaction) FromChain() string { return tx.Data.FromChain.String() }

// ToChain returns the chain coordinate of the recipient
func (tx *Transaction) ToChain() string { return tx.Data.ToChain.String() }

// Recipient returns the address of the recipient
func (tx *Transaction) Recipient() accounts.Address {
	return tx.Data.Recipient
}

// Amount returns the transfer amount of the transaction
func (tx *Transaction) Amount() *big.Int { return tx.Data.Amount }

// Nonce returns the nonce of the transaction
func (tx *Transaction) Nonce() uint32 { return tx.Data.Nonce }

// Fee returns the nonce of the transaction
func (tx *Transaction) Fee() *big.Int { return tx.Data.Fee }

// WithSignature returns a new transaction with the given signature.
func (tx *Transaction) WithSignature(sig *crypto.Signature) {
	//TODO: sender cache
	tx.Data.Signature = sig
}

//WithPayload returns a new transaction with the given data
func (tx *Transaction) WithPayload(data []byte) {
	tx.Payload = data
}

// CreateTime returns the create time of the transaction
func (tx *Transaction) CreateTime() uint32 {
	return tx.Data.CreateTime
}

// Compare implements interface consensus need
func (tx *Transaction) Compare(v interface{}) int {
	return 0
}

// GetType returns transaction type
func (tx *Transaction) GetType() uint32 { return tx.Data.Type }

// Transactions represents transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less compares nonce of the i'th and the j'th element in s
func (s Transactions) Less(i, j int) bool { return s[i].Data.Nonce < s[j].Data.Nonce }
