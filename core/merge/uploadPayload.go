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

import (
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/types"
)

// UploadPayload data to upload
type UploadPayload struct {
	PeerNum        uint32
	MergeDuration  uint32
	RepeatMergeTxs types.Transactions
	MergeTxs       types.Transactions
}

// NewUploadPayload returns a UploadPayload instance
func NewUploadPayload(peerNum, mergeDutation uint32, repeatMergeTxs, mergeTxs types.Transactions) *UploadPayload {
	return &UploadPayload{
		PeerNum:        peerNum,
		MergeDuration:  mergeDutation,
		RepeatMergeTxs: repeatMergeTxs,
		MergeTxs:       mergeTxs,
	}
}

// Serialize serializes updatePayload
func (up *UploadPayload) Serialize() []byte {
	return utils.Serialize(up)
}

// Deserialize deserializes updatePayload
func (up *UploadPayload) Deserialize(data []byte) error {
	return utils.Deserialize(data, up)
}
