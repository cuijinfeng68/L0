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

package peer

import (
	"testing"
	"time"

	"fmt"

	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/router"
)

func initTestConfig() {
	config.Set("router.discovery", "")
	config.Set("router.timeout.keepalive", "6s")
	config.Set("router.timeout.routers", "3s")
	config.Set("router.timeout.network.routers", "6s")
	config.Set("router.timeout.network.peers", "6s")
}

func chainMessageHandle(srcID, dstID string, payload []byte, signature []byte) error {

	fmt.Println("recv msg from peer", srcID, dstID, string(payload))
	return nil

}

func chainMessageHandle1(srcID, dstID string, payload []byte, signature []byte) error {

	fmt.Println("8001recv msg from peer", srcID, dstID, string(payload))
	return nil

}

var num = 5

/*
func TestPeer(t *testing.T) {
	initTestConfig()
	//SetLogOut("./loggggs")

	r := router.NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	p0 := NewPeer("8000", []string{"0.0.0.0:8000"}, chainMessageHandle)
	p0.Start()

	p1 := NewPeer("8002", []string{"0.0.0.0:8000"}, chainMessageHandle)
	p1.Start()

	time.Sleep(3 * time.Second)

	p0.Send("8002", []byte("8000-->8002"), nil)

	time.Sleep(3 * time.Second)

	p0.Stop()
	p1.Stop()

	time.Sleep(3 * time.Second)

	r.Stop()
}

/*
func TestPeer2(t *testing.T) {
	initTestConfig()

	r := router.NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	config.Set("router.discovery", "0.0.0.0:8000")
	rs := []*router.Router{}
	port := 8002
	for i := 0; i < num; i++ {
		r1 := router.NewRouter("00", "0.0.0.0:"+strconv.Itoa(port))
		go r1.Start()
		rs = append(rs, r1)
		time.Sleep(time.Second)
		port++
	}

	time.Sleep(3 * time.Second)

	p0 := NewPeer("8000", []string{"0.0.0.0:8000"}, chainMessageHandle)
	p0.Start()

	p1 := NewPeer("8002", []string{"0.0.0.0:8002"}, chainMessageHandle)
	p1.Start()

	time.Sleep(3 * time.Second)

	p0.Send("8002", []byte("8000-->8002"), nil)

	time.Sleep(3 * time.Second)

	p0.Stop()
	p1.Stop()

	time.Sleep(3 * time.Second)

	for _, r := range rs {
		r.Stop()
	}

	r.Stop()
}

func TestPeer3(t *testing.T) {
	initTestConfig()

	r := router.NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	config.Set("router.discovery", "0.0.0.0:8000")

	r2 := router.NewRouter("00", "0.0.0.0:8002")
	go r2.Start()
	time.Sleep(time.Second)

	r3 := router.NewRouter("00", "0.0.0.0:8004")
	go r3.Start()
	time.Sleep(time.Second)

	p0 := NewPeer("8000", []string{"0.0.0.0:8000", "0.0.0.0:8002", "0.0.0.0:8004"}, chainMessageHandle)
	p0.Start()

	time.Sleep(3 * time.Second)

	r.Stop()

	time.Sleep(30 * time.Second)

	r2.Stop()

	time.Sleep(30 * time.Second)

	p0.Stop()

	time.Sleep(3 * time.Second)

	r3.Stop()
}
*/
func TestBroadcastPeer(t *testing.T) {
	initTestConfig()

	r := router.NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	config.Set("router.discovery", "0.0.0.0:8000")

	r2 := router.NewRouter("00", "0.0.0.0:8001")
	go r2.Start()
	time.Sleep(time.Second)

	//ip don't use 0.0.0.0:8000
	p0 := NewPeer("00:8000", []string{"192.168.8.121:8000"}, chainMessageHandle)
	p0.Start()

	p1 := NewPeer("00:8001", []string{"192.168.8.121:8001"}, chainMessageHandle1)
	p1.Start()

	time.Sleep(2 * time.Second)
	p0.Send("00", []byte("8000-->8001"), nil)

	time.Sleep(5 * time.Second)

	p0.Stop()
	p1.Stop()
}
