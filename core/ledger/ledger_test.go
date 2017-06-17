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
	"math/big"
	"os"
	"testing"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

var (
	testDb = db.NewDB(db.DefaultConfig())

	issueReciepent     = accounts.HexToAddress("0xa032277be213f56221b6140998c03d860a60e1f8")
	atmoicReciepent    = accounts.HexToAddress("0xa132277be213f56221b6140998c03d860a60e1f8")
	acrossReciepent    = accounts.HexToAddress("0xa232277be213f56221b6140998c03d860a60e1f8")
	distributReciepent = accounts.HexToAddress("0xa332277be213f56221b6140998c03d860a60e1f8")
	backfrontReciepent = accounts.HexToAddress("0xa432277be213f56221b6140998c03d860a60e1f8")

	issueAmount = big.NewInt(100)
	Amount      = big.NewInt(1)
	fee         = big.NewInt(0)
	li          = NewLedger(testDb)
)

func TestExecuteIssueTx(t *testing.T) {
	params.ChainID = []byte{byte(0)}
	var wrriteBash []*db.WriteBatch

	issueTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		issueReciepent,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature)

	wrriteBash, err := li.executeIssueTx(wrriteBash, issueTx)
	if err != nil {
		t.Error(err)
	}
	li.state.AtomicWrite(wrriteBash)

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(issueReciepent))

}

func TestExecuteAtmoicTx(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(0)}

	atmoicTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*atmoicTxKeypair.Public())

	atmoicTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeAtomic,
		uint32(1),
		addr,
		atmoicReciepent,
		Amount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := atmoicTxKeypair.Sign(atmoicTx.Hash().Bytes())
	atmoicTx.WithSignature(signature)
	atmoicSender := atmoicTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		atmoicSender,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, atmoicTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(atmoicSender))
	t.Log(li.GetBalance(atmoicReciepent))

}

func TestExecuteAcossTx1(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(0)}

	acrossTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*acrossTxKeypair.Public())

	acrossTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeAcrossChain,
		uint32(1),
		addr,
		acrossReciepent,
		Amount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := acrossTxKeypair.Sign(acrossTx.Hash().Bytes())
	acrossTx.WithSignature(signature)
	acrossSender := acrossTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())
	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		acrossSender,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, acrossTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(acrossSender))
	t.Log(li.GetBalance(acrossReciepent))
}

func TestExecuteAcossTx2(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(0)}

	acrossTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*acrossTxKeypair.Public())

	acrossTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(1)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeAcrossChain,
		uint32(1),
		addr,
		acrossReciepent,
		Amount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := acrossTxKeypair.Sign(acrossTx.Hash().Bytes())
	acrossTx.WithSignature(signature)
	acrossSender := acrossTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		acrossReciepent,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, acrossTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(acrossSender))
	t.Log(li.GetBalance(acrossReciepent))
}

func TestExecuteMergedTx(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(0)}

	from := coordinate.NewChainCoordinate([]byte{byte(0)})
	to := coordinate.NewChainCoordinate([]byte{byte(0)})
	sender := coordinate.NewChainCoordinate([]byte{byte(0), byte(0)})
	reciepent := coordinate.NewChainCoordinate([]byte{byte(0), byte(1)})

	mergedTx := types.NewTransaction(from,
		to,
		types.TypeMerged,
		uint32(0),
		accounts.ChainCoordinateToAddress(sender),
		accounts.ChainCoordinateToAddress(reciepent),
		Amount,
		fee,
		utils.CurrentTimestamp())

	senderAddress := accounts.ChainCoordinateToAddress(sender)
	sig := &crypto.Signature{}
	copy(sig[:], senderAddress[:])
	mergedTx.WithSignature(sig)

	issueTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		senderAddress,
		issueAmount,
		fee,
		utils.CurrentTimestamp())

	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, mergedTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	issueSenderaddress := issueTx.Sender()

	t.Log(li.GetBalance(issueSenderaddress))
	t.Log(li.GetBalance(accounts.ChainCoordinateToAddress(sender)))
	t.Log(li.GetBalance(accounts.ChainCoordinateToAddress(reciepent)))
}

func TestExecuteDistributTx(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(0)}

	distributTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*distributTxKeypair.Public())

	distributTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeDistribut,
		uint32(1),
		addr,
		distributReciepent,
		Amount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := distributTxKeypair.Sign(distributTx.Hash().Bytes())
	distributTx.WithSignature(signature)
	distributAddress := distributTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())
	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		distributAddress,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, distributTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(accounts.ChainCoordinateToAddress(coordinate.NewChainCoordinate([]byte{byte(1)}))))
	t.Log(li.GetBalance(distributAddress))
	t.Log(li.GetBalance(distributReciepent))
}

func TestExecuteBackfrontTx(t *testing.T) {
	var writeBash []*db.WriteBatch
	params.ChainID = []byte{byte(1)}

	backfrontTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*backfrontTxKeypair.Public())

	backfrontTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeBackfront,
		uint32(1),
		addr,
		backfrontReciepent,
		Amount,
		fee,
		utils.CurrentTimestamp())
	signature, _ := backfrontTxKeypair.Sign(backfrontTx.Hash().Bytes())
	backfrontTx.WithSignature(signature)
	backfrontAddrress := backfrontTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(1),
		addr,
		backfrontAddrress,
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	err := li.executeTransaction(types.Transactions{issueTx, backfrontTx}, writeBash)
	if err != nil {
		t.Error(err)
	}

	sender := issueTx.Sender()
	t.Log(li.GetBalance(sender))
	t.Log(li.GetBalance(accounts.ChainCoordinateToAddress(coordinate.NewChainCoordinate([]byte{byte(0)}))))
	t.Log(li.GetBalance(backfrontAddrress))
	t.Log(li.GetBalance(backfrontReciepent))

	os.RemoveAll("/tmp/rocksdb-test1")

}
