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

package consenter

import (
	"strings"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/consensus/lbft"
	"github.com/bocheninc/L0/core/consensus/nbft"
	"github.com/bocheninc/L0/core/consensus/noops"
)

// NewConsenter Create consenter of plugin
func NewConsenter(option *Options, stack consensus.IStack) (consenter consensus.Consenter) {
	plugin := strings.ToLower(option.Plugin)
	if plugin == "lbft" {
		consenter = lbft.NewLbft(option.Lbft, stack)
	} else if plugin == "nbft" {
		consenter = nbft.NewNbft(option.Nbft, stack)
	} else {
		if plugin != "noops" {
			log.Warnf("Unspport consenter of plugin %s, use default plugin noops", plugin)
			plugin = "noops"
		}
		consenter = noops.NewNoops(option.Noops, stack)
	}
	log.Infof("Consenter %s : %s", plugin, consenter)
	go consenter.Start()
	return consenter
}

// NewDefaultOptions Create consenter options with default value
func NewDefaultOptions() *Options {
	options := &Options{
		Plugin: "noops",
		Noops:  noops.NewDefaultOptions(),
		Nbft:   nbft.NewDefaultOptions(),
		Lbft:   lbft.NewDefaultOptions(),
	}
	return options
}

// Options Define consenter options
type Options struct {
	Noops  *noops.Options
	Nbft   *nbft.Options
	Lbft   *lbft.Options
	Plugin string
}
