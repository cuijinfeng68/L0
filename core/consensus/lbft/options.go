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

package lbft

import "time"

//NewDefaultOptions Create nbft options with default value
func NewDefaultOptions() *Options {
	options := &Options{}
	options.Chain = "0"
	options.ID = "0"
	options.N = 4
	options.Q = 3
	options.K = 20
	options.BlockSize = 2000
	options.BlockTimeout = 8 * time.Second
	options.BlockInterval = 10 * time.Second
	options.BlockDelay = 10 * time.Second
	options.ViewChange = 5 * time.Second
	options.ResendViewChange = 5 * time.Second
	options.ViewChangePeriod = 0 * time.Second
	options.NullRequest = 4 * time.Second
	options.BufferSize = 100
	options.MaxConcurrentNumFrom = 1
	options.MaxConcurrentNumTo = 1
	return options
}

//Options Define nbft options
type Options struct {
	Chain                string
	ID                   string
	Primary              string
	AutoVote             bool
	N                    int
	Q                    int
	K                    int
	BlockSize            int
	BlockTimeout         time.Duration
	BlockInterval        time.Duration
	BlockDelay           time.Duration // BlockDelay > BlockInterval > BlockTimeout
	ViewChange           time.Duration
	ResendViewChange     time.Duration
	ViewChangePeriod     time.Duration
	NullRequest          time.Duration
	BufferSize           int
	MaxConcurrentNumFrom int
	MaxConcurrentNumTo   int
}
