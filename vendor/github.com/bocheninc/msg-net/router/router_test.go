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

package router

import (
	"strconv"
	"testing"
	"time"

	"github.com/bocheninc/msg-net/config"
)

var num = 6

func initTestConfig() {
	config.Set("router.discovery", "")
	config.Set("router.timeout.keepalive", "6s")
	config.Set("router.timeout.routers", "6s")
	config.Set("router.timeout.network.routers", "6s")
	config.Set("router.timeout.network.peers", "6s")
}

func TestRounterStartAndStop(t *testing.T) {
	initTestConfig()

	r := NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	rs := []*Router{}
	port := 8002
	for i := 0; i < num; i++ {
		r1 := NewRouter("00", "0.0.0.0:"+strconv.Itoa(port))
		go r1.Start()
		rs = append(rs, r1)
		time.Sleep(time.Second)
		port++
	}

	time.Sleep(time.Second)

	for _, r1 := range rs {
		r1.Stop()
	}

	r.Stop()
}

func TestRounterStartAndStop2(t *testing.T) {
	initTestConfig()

	r := NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	config.Set("router.discovery", "0.0.0.0:8000")

	rs := []*Router{}
	port := 8002
	for i := 0; i < num; i++ {
		r1 := NewRouter("00", "0.0.0.0:"+strconv.Itoa(port))
		go r1.Start()
		rs = append(rs, r1)
		time.Sleep(time.Second)
		port++
	}

	time.Sleep(3 * time.Second)

	for _, r1 := range rs {
		r1.Stop()
	}

	r.Stop()
}

func TestRounterStartAndStop3(t *testing.T) {
	initTestConfig()

	r := NewRouter("00", "0.0.0.0:8000")
	go r.Start()
	time.Sleep(time.Second)

	config.Set("router.discovery", "0.0.0.0:8000")

	rs := []*Router{}
	port := 8002
	for i := 0; i < num; i++ {
		r1 := NewRouter("00", "0.0.0.0:"+strconv.Itoa(port))
		go r1.Start()
		time.Sleep(time.Second)
		if i%2 == 0 {
			r1.Stop()
		} else {
			rs = append(rs, r1)
		}
		port++
	}

	time.Sleep(3 * time.Second)

	for _, r1 := range rs {
		r1.Stop()
	}

	r.Stop()
}
