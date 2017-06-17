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

package utils

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"
)

// VarEncode encodes the input data
// TODO: val support all types
func VarEncode(w io.Writer, val interface{}) {
	// value - type - kind - Interface()
	s := reflect.ValueOf(val)

	recursiveEncode(w, s)
}

func recursiveEncode(w io.Writer, s reflect.Value) {
	switch s.Kind() {
	case reflect.Struct:
		numField := s.NumField()
		for i := 0; i < numField; i++ {
			recursiveEncode(w, s.Field(i))
		}
	case reflect.Ptr:
		ptrEncode(w, s)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		WriteVarInt(w, s.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		WriteVarInt(w, uint64(s.Int()))
	case reflect.Slice:
		sliceEncode(w, s)
	case reflect.Array:
		arrayEncode(w, s)
	case reflect.String:
		WriteVarInt(w, uint64(len(s.String())))
		w.Write([]byte(s.String()))
	default:
		break
	}
}

func ptrEncode(w io.Writer, s reflect.Value) {
	elemValue := s.Elem()
	// Check if the pointer is nil
	if !elemValue.IsValid() {
		WriteVarInt(w, 0)
		return
	}

	v := s.Interface()
	switch v.(type) {
	case *big.Int:
		bigVal := v.(*big.Int)
		WriteVarInt(w, (uint64)(len(bigVal.Bytes())))
		w.Write(bigVal.Bytes())
	default:
		recursiveEncode(w, elemValue)
	}
}

func sliceEncode(w io.Writer, s reflect.Value) {
	vType := s.Type().Elem()
	switch vType.Kind() {
	case reflect.Uint8:
		buf := s.Bytes()
		WriteVarInt(w, (uint64)(len(buf)))
		w.Write(buf)
	default:
		vlen := s.Len()
		WriteVarInt(w, (uint64)(vlen))
		for i := 0; i < vlen; i++ {
			recursiveEncode(w, s.Index(i))
		}
		break
	}
}

func arrayEncode(w io.Writer, v reflect.Value) {
	etp := v.Type().Elem()
	switch etp.Kind() {
	case reflect.Uint8:
		if !v.CanAddr() {
			cpy := reflect.New(v.Type()).Elem()
			cpy.Set(v)
			v = cpy
		}
		size := v.Len()
		slice := v.Slice(0, size).Bytes()

		WriteVarInt(w, (uint64)(size))
		w.Write(slice)
	default:
		// fmt.Println(v.Bytes())
	}

}

// VarDecode decodes the data to val, val mustbe pointer
func VarDecode(r io.Reader, val interface{}) error {
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("vardecode error, val mustbe pointer") //&InvalidVarDecodeError{reflect.TypeOf(v)}
	}
	return recursiveDecode(r, rv)
}

func recursiveDecode(r io.Reader, s reflect.Value) error {
	var (
		err error
		l   uint64
	)
	switch s.Kind() {
	case reflect.Ptr:
		err = ptrDecode(r, s)
	case reflect.Struct:
		for i := 0; i < s.NumField(); i++ {
			recursiveDecode(r, s.Field(i))
		}
	case reflect.Uint8, reflect.Uint32, reflect.Uint64:
		if l, err = ReadVarInt(r); l > 0 {
			s.SetUint(l)
		}
	case reflect.Int64:
		if l, err = ReadVarInt(r); l > 0 {
			s.SetInt((int64)(l))
		}
	case reflect.Slice:
		// only support []byte,
		// TODO: other types support
		err = sliceDecode(r, s)
	case reflect.Array:
		if l, err = ReadVarInt(r); l > 0 && int(l) == s.Len() {
			buf := make([]byte, l)
			n, rErr := io.ReadFull(r, buf)

			if n > 0 && rErr == nil {
				reflect.Copy(s, reflect.ValueOf(buf))
			}
			err = rErr
		} else {
			err = fmt.Errorf("array decode length error")
		}
	case reflect.String:
		if l, err = ReadVarInt(r); l > 0 && s.CanSet() {
			buf := make([]byte, l)
			io.ReadFull(r, buf)
			s.SetString(string(buf))
		}
	default:
		//fmt.Println("defualt", s)
	}
	return err
}

func sliceDecode(r io.Reader, s reflect.Value) error {
	var (
		l   uint64
		err error
	)
	vType := s.Type().Elem()
	l, err = ReadVarInt(r)

	if l > 0 {
		newVal := reflect.MakeSlice(s.Type(), (int)(l), int(l))

		switch vType.Kind() {
		case reflect.Uint8:
			buf := make([]byte, l)
			n, rErr := io.ReadFull(r, buf)
			if n == int(l) && rErr == nil && s.CanSet() {
				reflect.Copy(newVal, reflect.ValueOf(buf))
				s.Set(newVal)
			}
			err = rErr
		default:
			for i := 0; i < int(l); i++ {
				err = recursiveDecode(r, newVal.Index(i))
			}
			s.Set(newVal)
		}
	}

	return err
}

func ptrDecode(r io.Reader, s reflect.Value) error {
	var (
		l   uint64
		n   int
		err error
	)
	v := s.Interface()
	switch v.(type) {
	case *big.Int:
		bigVal := new(big.Int)
		l, err = ReadVarInt(r)
		if l > 0 {
			buf := make([]byte, l)
			n, err = io.ReadFull(r, buf)
			if n == int(l) && s.CanSet() {
				bigVal.SetBytes(buf)
				s.Set(reflect.ValueOf(bigVal))
			}
		}
		if l == 0 {
			s.Set(reflect.ValueOf(bigVal))
		}
	default:
		if s.Kind() == reflect.Ptr {
			if !s.IsNil() || !s.IsValid() {
				err = recursiveDecode(r, s.Elem())
			}
			val := reflect.New(s.Type().Elem())
			if err = recursiveDecode(r, val.Elem()); err == nil && s.CanSet() {
				s.Set(val)
			}
		}
	}
	return err
}

// Serialize serializes an object to bytes
func Serialize(obj interface{}) []byte {
	buf := new(bytes.Buffer)
	VarEncode(buf, obj)
	return buf.Bytes()
}

// Deserialize deserializes bytes to object
func Deserialize(data []byte, obj interface{}) error {
	buf := bytes.NewBuffer(data)
	return VarDecode(buf, obj)
}
