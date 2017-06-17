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
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/merge"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/core/params"
	"github.com/spf13/viper"
)

const (
	defaultConfigFilename   = "lcnd.yaml"
	defaultLogFilename      = "lcnd.log"
	defaultChainDataDirname = "chaindata"
	defaultLogDirname       = "logs"
	defaultKeyStoreDirname  = "keystore"
	defaultNodeDirname      = "node"
	defaultNodeKeyFilename  = "nodekey"
	defaultMaxPeers         = 8
)

var (
	defaultConfig = &Config{
		NetConfig:   p2p.DefaultConfig(),
		DbConfig:    db.DefaultConfig(),
		MergeConfig: merge.DefaultConfig(),

		LogLevel: "debug",
		LogFile:  defaultLogFilename,
	}

	privkey *crypto.PrivateKey
)

// Config Represents the global config of lcnd
type Config struct {
	// dir
	DataDir     string
	LogDir      string
	NodeDir     string
	KeyStoreDir string

	// file
	PeersFile  string
	ConfigFile string

	// net
	NetConfig *p2p.Config

	//txMerger

	MergeConfig *merge.Config

	// log
	LogLevel string
	LogFile  string

	// db
	DbConfig    *db.Config
	NetDbConfig *db.Config

	// profile
	CPUFile string
}

// New returns a config according the config file
func New(cfgFile string) (cfg *Config, err error) {
	return loadConfig(cfgFile)
}

func loadConfig(cfgFile string) (conf *Config, err error) {
	var (
		cfg        *Config
		appDataDir string
	)

	cfg = defaultConfig

	if cfgFile != "" {
		if utils.FileExist(cfgFile) {
			viper.SetConfigFile(cfgFile)
		}
		if err := viper.ReadInConfig(); err != nil {
			log.Debugf("no config file, run as default config! viper.ReadInConfig error %s", err)
		} else {
			appDataDir = cfg.read()
		}
	}

	if appDataDir == "" {
		appDataDir = utils.AppDataDir()
		cfgFile = filepath.Join(appDataDir, defaultConfigFilename)
		if utils.FileExist(cfgFile) {
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Debug("no config file, run as default config!")
			} else {
				if dir := cfg.read(); dir != "" {
					if ok, _ := utils.IsDirExist(dir); ok {
						appDataDir = dir
					}
				}
			}
		}
	}

	utils.OpenDir(appDataDir)

	cfg.DataDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultChainDataDirname))
	cfg.LogDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultLogDirname))
	cfg.KeyStoreDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultKeyStoreDirname))
	cfg.NodeDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultNodeDirname))

	/*set chainid from config file just for test*/
	cfg.readParamConfig()

	cfg.DbConfig = DBConfig(cfg.DataDir)
	cfg.NetDbConfig = DBConfig(cfg.NodeDir)
	cfg.NetConfig = NetConfig(cfg.NodeDir)
	cfg.MergeConfig = MergeConfig(cfg.NodeDir)
	cfg.readLogConfig()

	return cfg, nil
}

func (cfg *Config) read() string {
	var (
		dataDir string
		cpuFile string
	)

	if cpuFile = viper.GetString("blockchain.cpuprofile"); cpuFile != "" {
		cfg.CPUFile = cpuFile
	}
	if dataDir = viper.GetString("blockchain.datadir"); dataDir != "" {
		return dataDir
	}

	return dataDir
}

/*set chainid from config file just for test*/
func (cfg *Config) readParamConfig() {
	str := getString("blockchain.id", "NET_NOT_SET")
	pk := getStringSlice("issueaddr.addr", []string{})
	params.ChainID = utils.HexToBytes(str)
	params.PublicAddress = pk
	params.Validator = viper.GetBool("blockchain.validator")
}

func (cfg *Config) readLogConfig() {
	var (
		logLevel, logFile string
	)
	if logLevel = viper.GetString("log.level"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if logFile = filepath.Join(cfg.LogDir, defaultLogFilename); logFile != "" {
		cfg.LogFile = logFile
	}
}
