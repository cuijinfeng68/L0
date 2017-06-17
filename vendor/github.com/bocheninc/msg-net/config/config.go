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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bocheninc/msg-net/util"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var config *viper.Viper

//ENVPREFIX identification
const ENVPREFIX = "msg-net"

func init() {
	config = viper.New()
	config.SetEnvPrefix(ENVPREFIX)
	config.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)

	paths := config.GetStringSlice("config.path")
	if len(paths) > 0 {
		for _, path := range paths {
			config.AddConfigPath(path)
		}
	} else {
		config.AddConfigPath(".")
		config.AddConfigPath("./config")
	}
	name := config.GetString("config.name")
	if name != "" {
		config.SetConfigName(name)
	} else {
		name = ENVPREFIX
		config.SetConfigName(ENVPREFIX)
	}
	t := config.GetString("config.type")
	if t != "" {
		config.SetConfigType(t)
	} else {
		t = ".yaml"
	}

	var flag = false
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		for _, p := range filepath.SplitList(gopath) {
			configpath := filepath.Join(p, "src/github.com/bocheninc/msg-net/config")
			if util.IsDirExist(configpath) && util.IsFileExist(filepath.Join(configpath, name+t)) {
				flag = true
				config.AddConfigPath(configpath)
			}
		}
		if flag {
			if err := config.ReadInConfig(); err != nil {
				//panic(fmt.Sprintf("failed to load config --- %v\n", err))
				defaultLoadConfig()
			}
		} else {
			defaultLoadConfig()
		}
		return
	}
	if err := config.ReadInConfig(); err != nil {
		defaultLoadConfig()
	}
}

func defaultLoadConfig() {
	SetDefault("logger.level", "info")
	SetDefault("logger.formatter", "text")
	SetDefault("logger.out", "")

	SetDefault("profiler.port", 6060)

	SetDefault("router.id", 0)
	SetDefault("router.address", "0.0.0.0:8000")
	SetDefault("router.addressAutoDetect", false)
	SetDefault("router.timeout.keepalive", time.Second*15)
	SetDefault("router.timeout.routers", time.Second*15)
	SetDefault("router.timeout.network.routers", time.Second*15)
	SetDefault("router.timeout.network.peers", time.Second*15)
	SetDefault("router.reconnect.interval", time.Second*10)
	SetDefault("router.reconnect.max", 5)

	SetDefault("report.on", false)
	SetDefault("report.interval", time.Second*60)
}

//ReadConfigFile loads configuration file
func ReadConfigFile(in string) {
	config.SetConfigFile(in)
	if err := config.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("failed to load config --- %v\n", err))
	}
}

//String returns summary
func String() string {
	m := make(map[string]interface{})
	m["config_file"] = config.ConfigFileUsed()
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

//SetDefault sets default value
func SetDefault(key string, value interface{}) { config.SetDefault(key, value) }

//Set sets value
func Set(key string, value interface{}) { config.Set(key, value) }

//IsSet checks whether the key attribute exists
func IsSet(key string) bool { return config.IsSet(key) }

//Get gets attribute by key - interface{}
func Get(key string) interface{} { return config.Get(key) }

//GetBool gets attribute by key - bool
func GetBool(key string) bool { return config.GetBool(key) }

//GetInt gets attribute by key - int
func GetInt(key string) int { return config.GetInt(key) }

//GetInt64 gets attribute by key - int64
func GetInt64(key string) int64 { return config.GetInt64(key) }

//GetFloat64 gets attribute by key - float64
func GetFloat64(key string) float64 { return config.GetFloat64(key) }

//GetString gets attribute by key - string
func GetString(key string) string { return config.GetString(key) }

//GetStringSlice gets attribute by key - []string
func GetStringSlice(key string) []string { return config.GetStringSlice(key) }

//GetStringMap gets attribute by key - map[string]interface{}
func GetStringMap(key string) map[string]interface{} { return config.GetStringMap(key) }

//GetStringMapString gets attribute by key - map[string]string
func GetStringMapString(key string) map[string]string { return config.GetStringMapString(key) }

//GetStringMapStringSlice gets attribute by key - map[string][]string
func GetStringMapStringSlice(key string) map[string][]string {
	return config.GetStringMapStringSlice(key)
}

//GetTime gets attribute by key - time.Time
func GetTime(key string) time.Time { return config.GetTime(key) }

//GetDuration gets attribute by key - time.Duration
func GetDuration(key string) time.Duration { return config.GetDuration(key) }

//BindPFlag Binding a flag key to the profile of the key
//	 serverCmd.Flags().Int("port", 1138, "Port to run Application server on")
//	 BindPFlag("port", serverCmd.Flags().Lookup("port"))
func BindPFlag(key string, flag *pflag.Flag) error { return config.BindPFlag(key, flag) }

//BindFlagValue Binding a value key to the profile of the value
//	 BindFlagValue("port", serverCmd.Flags().Lookup("port"))
func BindFlagValue(key string, flag viper.FlagValue) error { return config.BindFlagValue(key, flag) }
