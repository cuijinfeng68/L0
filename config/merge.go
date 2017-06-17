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
	"github.com/bocheninc/L0/core/merge"
	"github.com/spf13/viper"
)

//MergeConfig returns merge configuration
func MergeConfig(nodeDir string) *merge.Config {
	var (
		config        = merge.DefaultConfig()
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

	//config.MaxPeers = getInt("net.maxPeers", config.MaxPeers)
	config.ChainID = getString("blockchain.id", "CHAINID-NOT_SET")
	config.MaxPeers = getInt("consensus.nbft.N", config.MaxPeers)
	config.PeerID = utils.BytesToHex(privkey.Public().Bytes())
	config.MergeDuration = getDuration("merge.mergeDuration", config.MergeDuration)

	return config
}
