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

package db

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/bocheninc/L0/components/utils"
)

var testConfig = &Config{
	DbPath:            "/tmp/rocksdb-test/",
	Columnfamilies:    []string{"account", "balance", "ledger", "col3", "col2"},
	KeepLogFileNumber: 10,
	MaxLogFileSize:    10485760,
	LogLevel:          "warn",
}

func TestNewDB(t *testing.T) {
	NewDB(testConfig)
}
func TestReadAndWrite(t *testing.T) {
	db := NewDB(testConfig)
	// Put
	err := db.Put("default", []byte("foo"), []byte("bar"))
	if err != nil {
		t.Fatalf("faild to put, err: [%s]", err)
	}
	// Get
	value, err1 := db.Get("default", []byte("foo"))
	if err1 != nil {
		t.Fatalf("faild to get, err: [%s]", err1)
	}
	if !bytes.Equal(value, []byte("bar")) {
		t.Fatal("value not equal")
	}
}

func TestDelete(t *testing.T) {
	db := NewDB(testConfig)

	err := db.Put("default", []byte("foo"), []byte("bar"))
	if err != nil {
		t.Fatalf("faild to put, err: [%s]", err)
	}
	db.Delete("default", []byte("foo"))
	value, err1 := db.Get("default", []byte("foo"))
	if err1 != nil {
		t.Fatalf("faild to delete, err: [%s]", err)
	}
	if value != nil {
		t.Fatalf("faild to put")
	}
}

func TestBulkRead(t *testing.T) {
	prefix := "pre_"
	db := NewDB(testConfig)
	num := 100000
	for i := 0; i < num; i++ {
		keyStr := fmt.Sprintf("%s%d", prefix, i)
		key := []byte(keyStr)
		value := []byte("fdsfadsfasfsfdsfasf")
		err := db.Put("default", key, value)

		if err != nil {
			t.Fatalf("faild to put, err: [%s]", err)
		}
	}

	begin := utils.CurrentTimestamp()
	resCh := make(chan map[string][]byte)
	go db.GetByPrefix([]byte(prefix), resCh)
	for {
		if r, ok := <-resCh; ok {
			fmt.Println(r)
		} else {
			break
		}
	}
	end := utils.CurrentTimestamp()
	gap := end - begin
	fmt.Println("time gap:", gap)
}

func TestWriteBatch(t *testing.T) {
	db := NewDB(testConfig)

	var writeBatchs []*WriteBatch

	for i := 0; i < 100000; i++ {
		writeBatchs = append(writeBatchs, NewWriteBatch("balance", OperationPut, []byte("key"+strconv.Itoa(i)), []byte("value"+strconv.Itoa(i))))
	}
	fmt.Println("start writeBatch...")

	var cnt int
	for i := 0; i < 500; i++ {
		fmt.Println("times: ", cnt)
		db.AtomicWrite(writeBatchs)
		cnt++
	}

}
