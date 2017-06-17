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

package keystore

import (
	"io/ioutil"
	"testing"

	"github.com/bocheninc/L0/core/accounts"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func TestKeyEncryptDecrypt(t *testing.T) {
	keyjson, err := ioutil.ReadFile("testdata/very-light-scrypt.json")
	if err != nil {
		t.Fatal(err)
	}
	password := "foo"
	address := accounts.HexToAddress("d0e1dc3b3c93480388542daa45f64781c31fef6d")

	for i := 0; i < 3; i++ {
		if _, err := DecryptKey(keyjson, password+"bad"); err == nil {
			t.Errorf("test %d: json key decrypted with bad password", i)
		}

		key, err := DecryptKey(keyjson, password)
		if err != nil {
			t.Errorf("test %d: json key failed to decrypt: %v", i, err)
		}
		if key.Address != address {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Address, address)
		}
		password += "new data appended"
		if keyjson, err = EncryptKey(key, password, veryLightScryptN, veryLightScryptP); err != nil {
			t.Errorf("test %d: failed to recrypt key %v", i, err)
		}
	}
}
