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
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
)

var testTx = getTestTransaction()
var testTxHex = fmt.Sprintf("%0x", testTx.Serialize())

func getTestTransaction() *Transaction {
	sender := accounts.HexToAddress("0xc9bc867a613381f35b4430a6cb712eff8bb50311")
	address := accounts.HexToAddress("0xc9bc867a613381f35b4430a6cb712eff8bb50310")
	fromChain := coordinate.NewChainCoordinate([]byte{0, 1, 3})
	toChain := coordinate.NewChainCoordinate([]byte{0, 1, 1})
	nonce := uint32(10000)
	tx := NewTransaction(fromChain, toChain, TypeAtomic, nonce, sender, address, big.NewInt(1100), big.NewInt(110), utils.CurrentTimestamp())

	tx.Payload = []byte("123456")
	return tx
}

func TestTxDeserialize(t *testing.T) {
	txBytes := utils.HexToBytes(testTxHex)
	tx := new(Transaction)
	tx.Deserialize(txBytes)

	if !bytes.Equal(tx.Serialize(), txBytes) {
		fmt.Println("Deserialize: ", tx, tx.Recipient(), tx.Amount(), tx.Fee())
		t.Errorf("Tx Deserialize error! %v != %v", tx.Serialize(), txBytes)
	}
}

func TestTxHash(t *testing.T) {
	var (
		testTxHex = "01fd102714c9bc867a613381f35b4430a6cb712eff8bb5031002044c016e"
	)
	tx := new(Transaction)
	txBytes := utils.HexToBytes(testTxHex)
	tx.Deserialize(txBytes)

}

func TestTxSender(t *testing.T) {
	var (
		priv, _ = crypto.GenerateKey()
		addr    = accounts.PublicKeyToAddress(*priv.Public())
	)
	tx := NewTransaction(
		nil, nil,
		TypeAtomic,
		1,
		addr,
		addr,
		big.NewInt(10e9),
		big.NewInt(10e8),
		uint32(1),
	)

	sig, _ := priv.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	tx2 := new(Transaction)
	tx2.Deserialize(tx.Serialize())

	if !bytes.Equal(tx.Serialize(), tx2.Serialize()) {
		t.Errorf("Deserialize error with Signature, %0x != %0x", tx.Serialize(), tx2.Serialize())
	}
}
