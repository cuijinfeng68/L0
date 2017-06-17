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

package merge

import "time"

//Config merge config struct
type Config struct {
	MaxPeers      int
	MergeDuration time.Duration
	ChainID       string
	PeerID        string
}

var (
	config *Config
)

// DefaultConfig returns default merge config
func DefaultConfig() *Config {
	return &Config{
		MaxPeers:      7,
		MergeDuration: 1 * time.Minute,
	}
}
