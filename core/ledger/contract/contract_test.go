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
	"testing"
	"time"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	testDb        = db.NewDB(db.DefaultConfig())
	testSCAddr    = "0xa032277be213f56221b6140998c03d860a60e2f8"
	testSender    = accounts.HexToAddress("0xa032277be213f56221b6140998c03d860a60e1f8")
	testReciepent = accounts.HexToAddress("0xa032277be213f56221b6140998c03d860a60e3f8")
)

var smartContract *SmartConstract

type TestLedger struct {
}

func newTestLedger() *TestLedger {
	return &TestLedger{}
}

func (ledger *TestLedger) GetTmpBalance(addr accounts.Address) (*big.Int, error) {
	return big.NewInt(int64(20)), nil
}
func (ledger *TestLedger) Height() (uint32, error) {
	return uint32(10), nil
}

func makeSmartContract() *SmartConstract {
	return NewSmartConstract(testDb, newTestLedger())
}

func TestInitEnv(t *testing.T) {
	smartContract = makeSmartContract()
}

func TestSmartConstract_StartConstract(t *testing.T) {
	ht, _ := smartContract.ledgerHandler.Height()
	smartContract.StartConstract(ht)
}

func TestSmartConstract_ExecTransaction(t *testing.T) {
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate([]byte("0")),
		coordinate.NewChainCoordinate([]byte("0")),
		types.TypeSmartContract,
		uint32(1),
		testSender,
		accounts.HexToAddress("00000000000000000000"),
		big.NewInt(10e11),
		big.NewInt(1),
		uint32(time.Now().Unix()),
	)

	smartContract.ExecTransaction(tx, testSCAddr)
}

func TestSmartConstract_AddState(t *testing.T) {
	smartContract.AddState("hello", []byte("world"))
	smartContract.AddState("Lucy", []byte("sweet"))
}

func TestSmartConstract_DelState(t *testing.T) {
	smartContract.DelState("hello")
}

func TestSmartConstract_GetState(t *testing.T) {
	value, err := smartContract.GetState("hello")
	t.Log(" hello value: ", string(value), " err: ", err)
	value, err = smartContract.GetState("Lucy")
	t.Log(" Lucy value: ", string(value), " err: ", err)
}

func TestSmartConstract_CurrentBlockHeight(t *testing.T) {
	ht, _ := smartContract.ledgerHandler.Height()
	t.Log(" block height: ", ht)
}

func TestSmartConstract_GetBalances(t *testing.T) {
	balance, _ := smartContract.ledgerHandler.GetTmpBalance(accounts.HexToAddress("123456789"))
	t.Log(" account balance: ", balance)
}

func TestSmartConstract_AddTransfer(t *testing.T) {
	smartContract.AddTransfer("11000000000000000000", "22000000000000000000", big.NewInt(int64(10)), uint32(types.TypeAtomic))
}

func TestSmartConstract_SmartContractCommitted(t *testing.T) {
	smartContract.SmartContractCommitted()
}

func TestSmartConstract_FinishContractTransaction(t *testing.T) {
	txs, err := smartContract.FinishContractTransaction()
	t.Log(" txs: ", txs, " err: ", err)
}

func TestSmartConstract_AddChangesForPersistence(t *testing.T) {
	var writeBatchs []*db.WriteBatch
	writeBatchs, _ = smartContract.AddChangesForPersistence(writeBatchs)
	t.Log(len(writeBatchs))
	if err := testDb.AtomicWrite(writeBatchs); err != nil {
		t.Error(err)
	}
}

func TestSmartConstract_StopContract(t *testing.T) {
	ht, _ := smartContract.ledgerHandler.Height()
	smartContract.StopContract(ht)
}

func TestSmartConstract_StartConstract2(t *testing.T) {
	ht, _ := smartContract.ledgerHandler.Height()
	smartContract.StartConstract(ht)
}

func TestSmartConstract_GetState2(t *testing.T) {
	_, err := smartContract.GetState("hello")
	utils.AssertEquals(t, err, errors.New("can't get date from db"))
	value, err := smartContract.GetState("Lucy")
	utils.AssertEquals(t, string(value), "sweet")
}
