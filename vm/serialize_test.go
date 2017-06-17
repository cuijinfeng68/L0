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
	"testing"

	"bytes"

	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func TestLValueConvert(t *testing.T) {
	ls := lua.LString("hello")
	data := lvalueToByte(ls)
	buf := bytes.NewBuffer(data)
	v, err := byteToLValue(buf)
	if err != nil || string(v.(lua.LString)) != "hello" {
		t.Error("convert string error")
	}

	lb := lua.LBool(true)
	data = lvalueToByte(lb)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil || bool(v.(lua.LBool)) != true {
		t.Error("convert bool error")
	}

	ln := lua.LNumber(float64(123456789.4321))
	data = lvalueToByte(ln)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil || float64(v.(lua.LNumber)) != 123456789.4321 {
		t.Error("convert number error")
	}

	ltb := new(lua.LTable)
	ltb.RawSetString("str", ls)
	ltb.RawSetInt(1, lb)
	ltb.RawSet(lb, lb)

	lctb := new(lua.LTable)
	ltb.RawSetInt(10, ls)

	ltb.RawSet(lctb, lctb)

	data = lvalueToByte(ltb)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil {
		t.Error("convert table error")
	}
	ntb := v.(*lua.LTable)
	ntb.ForEach(func(key lua.LValue, value lua.LValue) {
		fmt.Println(key, " : ", value)
	})

}
