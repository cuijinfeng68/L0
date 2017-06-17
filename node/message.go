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

package node

import (
	"github.com/bocheninc/L0/components/crypto"
)

// InvType represents the allowed types of inventory vectors
type InvType uint8

const (
	// InvTypeTx represents the transaction types
	InvTypeTx InvType = iota
	// InvTypeBlock represents the block types
	InvTypeBlock
)

// StatusData represents a status message
type StatusData struct {
	Version     uint32
	StartHeight uint32 //uint64
}

// GetBlocks represents a getblocks message
type GetBlocks struct {
	Version uint32

	//TODO: blockchain support locator
	LocatorHashes []crypto.Hash
	HashStop      crypto.Hash
}

// InvVect defines a inventory vector which is used to describe data,
type InvVect struct {
	Type   InvType
	Hashes []crypto.Hash
}

// GetData represents a getdata message
type GetData struct {
	InvList []InvVect
}
