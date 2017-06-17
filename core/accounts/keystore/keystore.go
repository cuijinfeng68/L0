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
	crand "crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
)

var (
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

var columnFamily = "account"
var KeyStoreScheme = "keystore"
var ksInstance *KeyStore
var ksPInstance *KeyStore
var once sync.Once

// KeyStore definition
type KeyStore struct {
	storage keyStore
	db      *db.BlockchainDB
	ksDir   string
}

// NewKeyStore new a KeyStore instance
func NewKeyStore(db *db.BlockchainDB, keydir string, scryptN, scryptP int) *KeyStore {
	once.Do(func() {
		keydir, err := filepath.Abs(keydir)
		if err != nil {
			panic(err)
		}
		ksInstance = &KeyStore{storage: &keyStorePassphrase{keydir, scryptN, scryptP}}
		ksInstance.db = db
		ksInstance.ksDir = keydir
	})
	return ksInstance
}

// NewPlaintextKeyStore new a PlaintextKeyStore instance
func NewPlaintextKeyStore(db *db.BlockchainDB, keydir string) *KeyStore {
	once.Do(func() {
		keydir, err := filepath.Abs(keydir)
		if err != nil {
			panic(err)
		}
		ksPInstance = &KeyStore{storage: &keyStorePlain{keydir}}
		ksPInstance.db = db
		ksPInstance.ksDir = keydir
	})
	return ksPInstance
}

// HasAddress returns if current node has the specified addr
func (ks *KeyStore) HasAddress(addr accounts.Address) bool {
	a, _ := ks.db.Get(columnFamily, addr.Bytes())
	if len(a) == 0 {
		return false
	}
	return true
}

func (ks *KeyStore) Find(addr accounts.Address) *accounts.Account {
	var account accounts.Account
	a, _ := ks.db.Get(columnFamily, addr.Bytes())
	if len(a) == 0 {
		return &account
	}
	account.Deserialize(a)
	return &account
}

func (ks *KeyStore) Accounts() ([]string, error) {
	var res []string
	err := filepath.Walk(ks.ksDir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		hexStr := strings.Split(f.Name(), "--")[2]
		address := accounts.HexToAddress(hexStr) //accounts.NewAddress([]byte(hexStr[2]))
		res = append(res, address.String())

		return nil
	})
	if err != nil {
		return res, err
	}
	return res, err
}

// NewAccount creates a new account
func (ks *KeyStore) NewAccount(passphrase string, accountType uint32) (accounts.Account, error) {
	_, a, err := storeNewKey(ks.storage, crand.Reader, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}
	a.AccountType = accountType
	err = ks.db.Put(columnFamily, a.Address.Bytes(), a.Serialize())
	if err != nil {
		return accounts.Account{}, err
	}
	return a, nil
}

// Delete removes the speciified account
func (ks *KeyStore) Delete(a accounts.Account, passphrase string) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if key != nil {
		crypto.ZeroKey((*crypto.PrivateKey)(key.PrivateKey))
	}
	if err != nil {
		return err
	}
	err = os.Remove(a.URL.Path)
	if err != nil {
		return err
	}
	err = ks.db.Delete(columnFamily, a.Address.Bytes())
	return err
}

// Update update the specified account
func (ks *KeyStore) Update(a accounts.Account, passphrase, newPassphrase string) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}
	return ks.storage.StoreKey(a.URL.Path, key, newPassphrase)
}

// SignTx sign the specified transaction
func (ks *KeyStore) SignTx(a accounts.Account, tx *types.Transaction, pass string) (*types.Transaction, error) {
	_, key, err := ks.getDecryptedKey(a, pass)
	if err != nil {
		return nil, err
	}

	priv := key.PrivateKey
	sig, err1 := priv.Sign(tx.Hash().Bytes())
	if err1 != nil {
		return nil, err1
	}
	tx.WithSignature(sig)
	return tx, nil
}

// SignHashWithPassphrase signs hash if the private key matching the given address
// can be decrypted with the given passphrase. The produced signature is in the
// [R || S || V] format where V is 0 or 1.
func (ks *KeyStore) SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) (signature []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return []byte{}, err
	}
	defer crypto.ZeroKey(key.PrivateKey)
	sig, err := key.PrivateKey.Sign(hash)
	if err != nil {
		return []byte{}, err
	}
	return sig.Bytes(), nil
}

func (ks *KeyStore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *Key, error) {
	addr := accounts.PublicKeyToAddress(*a.PublicKey)
	key, err := ks.storage.GetKey(addr, a.URL.Path, auth)
	return a, key, err
}
