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

package consensus

// ITransaction Interface for consensus content, consensus input object
type ITransaction interface {
	Serialize() []byte
	Deserialize(payload []byte) error
	FromChain() string
	ToChain() string
	CreateTime() uint32
	Nonce() uint32
}

// ITxPool Interface for tx containter
type ITxPool interface {
	IterTransaction(func(ITransaction) bool)
	Removes([]ITransaction)
	Len() int
}

// CommittedTxs Consensus output object
type CommittedTxs struct {
	SeqNos       []uint64
	Time         uint32
	Transactions []ITransaction
}

// IBroadcast Interface for consensus broadcast content
type IBroadcast interface {
	To() string
	Payload() []byte
}

// Consenter Interface for plugin consenser
type Consenter interface {
	Start()
	Stop()
	RecvConsensus([]byte)
	BroadcastConsensusChannel() <-chan IBroadcast
	BroadcastTransactionChannel() <-chan ITransaction
	CommittedTxsChannel() <-chan *CommittedTxs
}

// IStack Interface for other function for plugin consenser
type IStack interface {
	NewTransaction() ITransaction
	VerifyTxsInConsensus(txs []ITransaction, primary bool) []ITransaction
	GetLastSeqNo() uint64
	ITxPool
}
