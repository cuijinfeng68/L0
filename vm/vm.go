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

// Package vm the contract execute environment
package vm

import (
	"errors"

	"bytes"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/yuin/gopher-lua"
)

var zeroAddr = accounts.Address{}

var conf *Config

func init() {
	conf = DefaultConfig()
}

// PreExecute execute contract but not commit change(balances and state)
func PreExecute(ctx *CTX) (bool, error) {
	return execContract(ctx)
}

// RealExecute execute contract and commit change(balances and state)
func RealExecute(ctx *CTX) (bool, error) {
	ok, err := execContract(ctx)
	if err != nil || !ok {
		return ok, err
	}

	//commit all change
	ctx.commit()

	return true, nil
}

// execContract start a lua vm and execute smart contract script
func execContract(ctx *CTX) (bool, error) {
	payload := ctx.payload()
	if len(payload) == 0 || len(payload) > conf.ExecLimitMaxScriptSize {
		return false, errors.New("contract script code size illegal, max size is:" + string(conf.ExecLimitMaxScriptSize) + " byte")
	}

	L := newState()
	defer L.Close()

	L.PreloadModule("L0", genModelLoader(ctx))
	err := L.DoString(payload)
	if err != nil {
		return false, err
	}

	if bytes.Equal(ctx.Transaction.Recipient().Bytes(), zeroAddr.Bytes()) {
		return callLuaFunc(L, "L0Init")
	} else {
		params := ctx.ContractSpec.ContractParams
		return callLuaFunc(L, "L0Invoke", params...)
	}
}

func genModelLoader(ctx *CTX) lua.LGFunction {
	expt := exporter(ctx)

	return func(L *lua.LState) int {
		mod := L.SetFuncs(L.NewTable(), expt) // register functions to the table
		L.Push(mod)
		return 1
	}
}

// newState create a lua vm
func newState() *lua.LState {
	opt := lua.Options{
		SkipOpenLibs:        true,
		CallStackSize:       conf.VMCallStackSize,
		RegistrySize:        conf.VMRegistrySize,
		MaxAllowOpCodeCount: conf.ExecLimitMaxOpcodeCount,
	}
	L := lua.NewState(opt)

	// forbid: lua.IoLibName, lua.OsLibName, lua.DebugLibName, lua.ChannelLibName, lua.CoroutineLibName
	openLib(L, lua.LoadLibName, lua.OpenPackage)
	openLib(L, lua.BaseLibName, lua.OpenBase)
	openLib(L, lua.TabLibName, lua.OpenTable)
	openLib(L, lua.StringLibName, lua.OpenString)
	openLib(L, lua.MathLibName, lua.OpenMath)

	return L
}

// openLib loads the built-in libraries. It is equivalent to running OpenLoad,
// then OpenBase, then iterating over the other OpenXXX functions in any order.
func openLib(L *lua.LState, libName string, libFunc lua.LGFunction) {
	L.Push(L.NewFunction(libFunc))
	L.Push(lua.LString(libName))
	L.Call(1, 0)
}

// call lua function(L0Init, L0Invoke)
func callLuaFunc(L *lua.LState, funcName string, params ...string) (bool, error) {
	p := lua.P{
		Fn:      L.GetGlobal(funcName),
		NRet:    1,
		Protect: true,
	}

	var err error
	l := len(params)
	if l == 0 {
		err = L.CallByParam(p, lua.LNil, lua.LNil)
	} else if l == 1 {
		err = L.CallByParam(p, lua.LString(params[0]), lua.LNil)
	} else {
		tb := new(lua.LTable)
		for i := 1; i < l; i++ {
			tb.RawSet(lua.LNumber(i), lua.LString(params[i]))
		}
		err = L.CallByParam(p, lua.LString(params[0]), tb)
	}
	if err != nil {
		return false, err
	}

	ret := L.CheckBool(-1) // returned value
	L.Pop(1)               // remove received value

	return ret, nil
}
