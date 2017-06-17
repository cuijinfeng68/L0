// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of msg-net 
// 
// The msg-net is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// The msg-net is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package logger

import (
	"fmt"
	"testing"

	"github.com/bocheninc/msg-net/config"
)

func TestLoggerInit(t *testing.T) {
	fmt.Println("logger information :", String())

	config.Set("logger.Out", ".")
	config.Set("logger.Level", "debug")
	//config.Set("logger.formatter", "json")
	Init()

	fmt.Println("logger information :", String())

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	//Panic("panic")
	//Fatal("fatal")

	config.Set("logger.Out", ".")
	config.Set("logger.Level", "debug")
	config.Set("logger.formatter", "text")
	Init()
}

func TestLogger(t *testing.T) {
	fmt.Println("logger information :", String())
	Debugln("debug")
	Infoln("info")
	Warnln("warn")
	Errorln("error")
	//Panicln("panic")
	//Fatalln("fatal")

	Debugf("%s", "debugf")
	Infof("%s", "infof")
	Warnf("%s", "warnf")
	Errorf("%s", "errorf")
	//Panicf("%s", "panicf")
	//Fatalf("%s", "fatalf")
}
