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

package util

import (
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

//GetLocalIP gets local IP
func GetLocalIP() string {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

//IsDirExist determines whether the directory exists
func IsDirExist(path string) bool {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir()
	}
	return !os.IsNotExist(err)
}

//MkDir make directory
func MkDir(path string) error {
	err := os.Mkdir(path, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

//IsFileExist determines whether the file exists
func IsFileExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

//CallerInfo gets call location
func CallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		file = "???"
		line = 0
	}
	_, filename := path.Split(file)
	return "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "] "
}

//SysSignal checks exit signal
func SysSignal(function func()) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTSTP)
	for {
		select {
		case <-c:
			function()
		}
	}
}

//IsStrExist determines whether the string exists
func IsStrExist(str string, strs []string) bool {
	for _, v := range strs {
		if strings.EqualFold(v, str) {
			return true
		}
	}
	return false
}
