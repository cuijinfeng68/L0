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

package vm

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"math/big"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/types"
	"github.com/yuin/gopher-lua"
)

func TestRealExecute(t *testing.T) {
	tx := &types.Transaction{}
	cs := &types.ContractSpec{ContractAddr: []byte("sender"), ContractParams: []string{"transfer", "receiver", "100"}}
	hd := &L0Handler{}

	//正式执行
	ctx := NewCTX(tx, cs, hd)

	begin := time.Now().UnixNano() / 1000000
	for i := 0; i < 1; i++ {
		_, err := RealExecute(ctx)
		if err != nil {
			log.Error(err)
		}
	}
	end := time.Now().UnixNano() / 1000000
	fmt.Println("run time:", end-begin)
}

type L0Handler struct {
}

func (hd *L0Handler) GetState(key string) ([]byte, error) {
	if "balances" == key {
		ltb := new(lua.LTable)
		ltb.RawSetString("sender", lua.LNumber(100))
		ltb.RawSetString("receiver", lua.LNumber(200))
		ltb.RawSetString("c", lua.LNumber(300))

		return lvalueToByte(ltb), nil
	} else if contractCodeKey == key {
		f, _ := os.Open("../tests/contract/l0coin.lua")
		defer f.Close()
		buf, _ := ioutil.ReadAll(f)

		return buf, nil
	}

	return nil, nil
}

func (hd *L0Handler) AddState(key string, value []byte) {

}

func (hd *L0Handler) DelState(key string) {

}

func (hd *L0Handler) GetBalances(addr string) (*big.Int, error) {
	return big.NewInt(100), nil
}

func (hd *L0Handler) CurrentBlockHeight() uint32 {
	return 100
}

func (hd *L0Handler) AddTransfer(fromAddr, toAddr string, amount *big.Int, txType uint32) {
	fmt.Printf("AddTransfer from:%s to:%s amount:%d txType:%d", fromAddr, toAddr, amount.Int64(), txType)
}

func (hd *L0Handler) SmartContractFailed() {

}

func (hd *L0Handler) SmartContractCommitted() {

}
