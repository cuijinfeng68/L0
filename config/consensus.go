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
	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus/consenter"
	"github.com/bocheninc/L0/core/consensus/lbft"
	"github.com/bocheninc/L0/core/consensus/nbft"
	"github.com/bocheninc/L0/core/consensus/noops"
)

func ConsenterOptions() *consenter.Options {
	option := consenter.NewDefaultOptions()
	option.Plugin = getString("consensus.plugin", option.Plugin)
	option.Noops = NoopsOptions()
	option.Nbft = NbftOptions()
	option.Lbft = LbftOptions()
	return option
}

func NoopsOptions() *noops.Options {
	option := noops.NewDefaultOptions()
	option.BlockSize = getInt("consensus.noops.blockSize", option.BlockSize)
	option.BlockInterval = getDuration("consensus.noops.blockInterval", option.BlockInterval)
	return option
}

func NbftOptions() *nbft.Options {
	option := nbft.NewDefaultOptions()
	option.Chain = getString("blockchain.id", option.Chain)
	option.ID = utils.BytesToHex(crypto.Ripemd160(crypto.Ripemd160([]byte(getString("consensus.nbft.id", option.ID) + option.Chain))))
	option.N = getInt("consensus.nbft.N", option.N)
	option.Q = getInt("consensus.nbft.Q", option.Q)
	option.BlockSize = getInt("consensus.nbft.blockSize", option.BlockSize)
	option.BlockInterval = getDuration("consensus.nbft.blockInterval", option.BlockInterval)
	option.BlockTimeout = getDuration("consensus.nbft.blockTimeout", option.BlockTimeout)
	option.BlockDelay = getDuration("consensus.nbft.blockDelay", option.BlockDelay)
	return option
}

func LbftOptions() *lbft.Options {
	option := lbft.NewDefaultOptions()
	option.Chain = getString("blockchain.id", option.Chain)
	option.ID = option.Chain + ":" + utils.BytesToHex(crypto.Ripemd160([]byte(getString("consensus.lbft.id", option.ID)+option.Chain)))
	option.N = getInt("consensus.lbft.N", option.N)
	option.Q = getInt("consensus.lbft.Q", option.Q)
	option.K = getInt("consensus.lbft.K", option.K)
	option.BlockSize = getInt("consensus.lbft.blockSize", option.BlockSize)
	option.BlockInterval = getDuration("consensus.lbft.blockInterval", option.BlockInterval)
	option.BlockTimeout = getDuration("consensus.lbft.blockTimeout", option.BlockTimeout)
	option.BlockDelay = getDuration("consensus.bft.blockDelay", option.BlockDelay)
	option.ViewChange = getDuration("consensus.lbft.viewChange", option.ViewChange)
	option.ResendViewChange = getDuration("consensus.lbft.resendViewChange", option.ViewChange)
	option.ViewChangePeriod = getDuration("consensus.lbft.viewChangePeriod", option.ViewChangePeriod)
	option.NullRequest = getDuration("consensus.lbft.nullRequest", option.NullRequest)
	option.BufferSize = getInt("consensus.lbft.bufferSize", option.BufferSize)
	option.MaxConcurrentNumFrom = getInt("consensus.lbft.maxConcurrentNumFrom", option.MaxConcurrentNumFrom)
	option.MaxConcurrentNumTo = getInt("consensus.lbft.maxConcurrentNumTo", option.MaxConcurrentNumTo)
	return option
}
