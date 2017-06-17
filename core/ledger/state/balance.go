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

package state

import (
	"math/big"

	"github.com/bocheninc/L0/components/utils"
)

// Balance contain amount nonce
type Balance struct {
	Amount *big.Int
	Nonce  uint32
}

// NewBalance initialization
func NewBalance(amount *big.Int, nonce uint32) *Balance {
	return &Balance{Amount: amount, Nonce: nonce}
}

func (b *Balance) serialize() []byte {
	return utils.Serialize(b)
}

func (b *Balance) deserialize(balanceBytes []byte) {
	utils.Deserialize(balanceBytes, b)
}
