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

package common

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

//Deadline Nonblocking network timeout
var Deadline time.Duration
var channelCap int
var maxMsgSize uint64 = 1024 * 1024 * 10

func init() {
	Deadline = 10 * time.Second
	channelCap = 100
}

//IMsg Message serialization interface
type IMsg interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

//Handler Message send and receive interface
type Handler struct {
	recvChannel chan IMsg
	sendChannel chan IMsg
}

//Init Initialization
func (h *Handler) Init() {
	h.recvChannel = make(chan IMsg, channelCap)
	h.sendChannel = make(chan IMsg, channelCap)
}

//RecvChannel Message receive channel
func (h *Handler) RecvChannel() chan IMsg {
	return h.recvChannel
}

//SendChannel Message send channel
func (h *Handler) SendChannel() chan IMsg {
	return h.sendChannel
}

// Send sends message
func (h *Handler) Send(conn net.Conn, m IMsg) (int, error) {
	var buf bytes.Buffer
	bytes, err := m.Serialize()
	if err != nil {
		return 0, err
	}
	//head
	preBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(preBytes, uint64(len(bytes)))
	preNum, err := buf.Write(preBytes)
	if err != nil {
		return 0, err
	}
	if _, err := buf.Write(bytes); err != nil {
		return 0, err
	}
	//message data
	num, err := conn.Write(buf.Bytes())
	if err != nil {
		return 0, err
	}
	return num - preNum, nil
}

//Recv receives message
func (h *Handler) Recv(conn net.Conn, m IMsg) error {
	preBytes := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(Deadline))
	if n, err := io.ReadFull(conn, preBytes); err != nil {
		return err
	} else if n != 8 {
		return fmt.Errorf("missing (8 == %v)", n)
	}

	num := binary.BigEndian.Uint64(preBytes)
	if num > maxMsgSize {
		return fmt.Errorf("message too big: %v", num)
	}
	//fmt.Println("------> ", num)
	bytes := make([]byte, num)
	conn.SetReadDeadline(time.Now().Add(Deadline))
	if n, err := io.ReadFull(conn, bytes); err != nil {
		//if _, err := conn.Read(bytes); err != nil {
		return err
	} else if uint64(n) != num {
		return fmt.Errorf("missing (%v == %v)", num, n)
	}
	return m.Deserialize(bytes)
}

//IsLocalAddress determines whether the local network address
func IsLocalAddress(address string) bool {
	if !IsValidAddress(address) {
		return false
	}
	return IsLocalIP(strings.Split(address, ":")[0])
}

//IsLocalIP determines whether the local network IP
func IsLocalIP(ip string) bool {
	if !IsValidIP(ip) {
		return false
	}
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if ipnet.IP.String() == ip {
					return true
				}
			}
		}
	}
	return false
}

//IsValidAddress determines whether a valid network address
func IsValidAddress(address string) bool {
	strs := strings.Split(address, ":")
	if len(strs) != 2 {
		return false
	}
	if !IsValidIP(strs[0]) {
		return false
	}
	if _, err := strconv.Atoi(strs[1]); err != nil {
		return false
	}
	return true
}

//IsValidIP Determine whether a valid network IP
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

//NameByIP gets network card name based on local IP
func NameByIP(ip string) string {
	if !IsValidIP(ip) {
		return ""
	}
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		addrs, _ := inter.Addrs()
		for _, address := range addrs {
			// check the address type and if it is not a loopback then display it
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					if ipnet.IP.String() == ip {
						return inter.Name
					}
				}
			}
		}
	}
	return ""
}

//NameByIndex gets network card name base on index
func NameByIndex(index int) string {
	if inter, err := net.InterfaceByIndex(index); err == nil {
		return inter.Name
	}
	return ""
}

//IPByName gets IP base on network name
func IPByName(name string) string {
	name = strings.ToLower(name)
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if strings.ToLower(inter.Name) == name {
			addrs, _ := inter.Addrs()
			for _, address := range addrs {
				// check the address type and if it is not a loopback then display it
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
}

//IPByIndex gets IP base on index
func IPByIndex(index int) string {
	if inter, err := net.InterfaceByIndex(index); err == nil {
		addrs, _ := inter.Addrs()
		for _, address := range addrs {
			// check the address type and if it is not a loopback then display it
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return ""
}

//ChooseInterface choose local interface
func ChooseInterface() string {
	indexs := []int{}
	fmt.Printf("%c[1m", 0x1B)
	fmt.Println("Available bridged network interfaces:")
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		ip := IPByIndex(inter.Index)
		fmt.Println("  ", inter.Index, inter.Name, inter.HardwareAddr, ip)
		if ip != "" {
			indexs = append(indexs, inter.Index)
		}
	}
	fmt.Println("When choosing an interface, it is usually the one that is being used to connect to the internet.")
	fmt.Printf("%c[0m", 0x1B)

	bytes, _ := json.Marshal(indexs)
	if len(indexs) == 1 {
		index := indexs[0]
		fmt.Println("Which interface should the network bridge to", string(bytes), "?", index)
		return IPByIndex(index)
	}
	for {
		fmt.Print("Which interface should the network bridge to", string(bytes), "? ")
		var index int
		fmt.Scanln(&index)
		if ip := IPByIndex(index); ip != "" {
			return ip
		}
		fmt.Println("Please input an index with ip")
	}
}
