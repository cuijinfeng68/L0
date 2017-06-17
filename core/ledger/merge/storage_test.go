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

package merge

import (
	"math/big"
	"os"
	"reflect"
	"testing"

	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	testDb    = db.NewDB(db.DefaultConfig())
	txStorage = NewStorage(testDb)
	testTxs   = make([]*types.Transaction, 0)
)

func getChainCoordinate(i int) coordinate.ChainCoordinate {
	return coordinate.NewChainCoordinate([]byte{byte(0), byte(1), byte(i)})
}

func getTestTxs(src, dst int) types.Transactions {
	testTxs := make([]*types.Transaction, 0)
	for i := 0; i < 10; i++ {
		tx := types.NewTransaction(
			getChainCoordinate(src),
			getChainCoordinate(dst),
			types.TypeAtomic,
			uint32(1),
			accounts.ChainCoordinateToAddress(getChainCoordinate(i)),
			big.NewInt(int64(i*100)),
			big.NewInt(1000),
			utils.CurrentTimestamp(),
		)
		//sign tx
		keypair, _ := crypto.GenerateKey()
		s, _ := keypair.Sign(tx.Hash().Bytes())
		tx.WithSignature(s)
		testTxs = append(testTxs, tx)
		time.Sleep(time.Second)
	}
	return testTxs
}

func TestClassifiedTransaction(t *testing.T) {
	go func() {
		txs2 := getTestTxs(1, 2)
		testTxs = append(testTxs, txs2...)
	}()

	txs1 := getTestTxs(0, 1)
	testTxs = append(testTxs, txs1...)

	txs3 := getTestTxs(2, 3)
	testTxs = append(testTxs, txs3...)

	for _, v := range testTxs {
		t.Log(v.CreateTime())
	}

	if err := txStorage.ClassifiedTransaction(testTxs); err != nil {
		t.Error(err)
	}

}

func BenchmarkClassifiedTransaction(b *testing.B) {
	b.StopTimer()
	go func() {
		txs2 := getTestTxs(1, 2)
		testTxs = append(testTxs, txs2...)
	}()

	txs1 := getTestTxs(0, 1)
	testTxs = append(testTxs, txs1...)

	txs3 := getTestTxs(2, 3)
	testTxs = append(testTxs, txs3...)

	b.Log(len(testTxs))
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		if err := txStorage.ClassifiedTransaction(testTxs); err != nil {
			b.Error(err)
		}
	}
}

func TestGetMergedTransaction(t *testing.T) {
	txs, err := txStorage.GetMergedTransaction(uint32(9))
	if err != nil {
		t.Error(err)
	}

	t.Log("txs len: ", len(txs))

}

func TestPutTxsHashByMergeTxHash(t *testing.T) {
	mergeTXHash := crypto.HexToHash("c4cbf1af2881c3696858c9db72c5aa904ad7f21937b673d8e69c176ebc59649f")
	txsHashs := []crypto.Hash{crypto.HexToHash("66337165439a350226ba4f765713fcb731a1b4fc15fa2103ad38c52dc58c8194"),
		crypto.HexToHash("923cb10c40fcbe6af170de8ff5bb11a569172d5a75486a79ac3c2f32aa5cd306"),
		crypto.HexToHash("20a1830983bf89b1a3018b10a757ab11b3f064865e53f35663fbc34ff89c997a")}

	if err := txStorage.PutTxsHashByMergeTxHash(mergeTXHash, txsHashs); err != nil {
		t.Error(err)
	}

	testTxsHashs, err := txStorage.GetTxsByMergeTxHash(mergeTXHash)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < len(testTxsHashs); i++ {
		if !testTxsHashs[i].Equal(txsHashs[i]) {
			t.Error("error", testTxsHashs[i], txsHashs[i])
		}
	}

	if !reflect.DeepEqual(txsHashs, testTxsHashs) {
		t.Errorf("txsHashs : %v is not equal testTxsHashs : %v\n", txsHashs, testTxsHashs)
	}
	os.RemoveAll("/tmp/rocksdb-test1")
}
