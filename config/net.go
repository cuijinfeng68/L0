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

package config

import (
	"path/filepath"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/spf13/viper"
)

// NetConfig returns a p2p network configuration
func NetConfig(nodeDir string) *p2p.Config {
	var (
		config        = p2p.DefaultConfig()
		privkey       *crypto.PrivateKey
		hexPrivateKey string
		nodeKeyFile   = filepath.Join(nodeDir, defaultNodeKeyFilename)
		err           error
	)

	if !utils.FileExist(nodeKeyFile) {
		// no configuration and node, generate a new key and store it
		privkey, _ = crypto.GenerateKey()
		privkey.SaveECDSA(nodeKeyFile)
	} else {
		privkey, err = crypto.LoadECDSA(nodeKeyFile)
		if err != nil {
			privkey, _ = crypto.GenerateKey()
			privkey.SaveECDSA(nodeKeyFile)
		}
	}

	if hexPrivateKey = viper.GetString("net.privateKey"); hexPrivateKey != "" {
		privkey, _ = crypto.HexToECDSA(hexPrivateKey)
		privkey.SaveECDSA(nodeKeyFile)
	}

	config.Address = getString("net.listenAddr", config.Address)
	config.BootstrapNodes = getStringSlice("net.bootstrapNodes", config.BootstrapNodes)
	config.PrivateKey = privkey
	config.MaxPeers = getInt("net.maxPeers", config.MaxPeers)
	config.ReconnectTimes = getInt("net.reconnectTimes", config.ReconnectTimes)
	config.ConnectTimeInterval = getInt("net.connectTimeInterval", config.ConnectTimeInterval)
	config.KeepAliveInterval = getInt("net.keepAliveInterval", config.KeepAliveInterval)
	config.KeepAliveTimes = getInt("net.keepAliveTimes", config.KeepAliveTimes)
	config.MinPeers = getInt("net.minPeers", config.MinPeers)
	config.RouteAddress = getStringSlice("net.msgnet.routeAddress", config.RouteAddress)

	return config
}
