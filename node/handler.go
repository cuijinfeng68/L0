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
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/config"
	"github.com/bocheninc/L0/core/accounts/keystore"
	"github.com/bocheninc/L0/core/blockchain"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/merge"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/msgnet"
	jrpc "github.com/bocheninc/L0/rpc"
)

// ProtocolManager manages the protocol
type ProtocolManager struct {
	*blockchain.Blockchain
	consenter consensus.Consenter
	// msg-net

	statusData StatusData

	peers []*peer
	// syncer
	msgnet msgnet.Stack
	merger *merge.Helper

	*ledger.Ledger
	*keystore.KeyStore
	*p2p.Server

	msgCh chan *p2p.Msg
}

// NewProtocolManager returns a new sub protocol manager.
func NewProtocolManager(db *db.BlockchainDB, netConfig *p2p.Config,
	blockchain *blockchain.Blockchain, consenter consensus.Consenter,
	ledger *ledger.Ledger, ks *keystore.KeyStore,
	mergeConfig *merge.Config, logDir string) *ProtocolManager {
	manager := &ProtocolManager{
		Blockchain: blockchain,
		consenter:  consenter,

		Server:   p2p.NewServer(db, netConfig),
		Ledger:   ledger,
		KeyStore: ks,
		msgCh:    make(chan *p2p.Msg, 100),
	}

	manager.Server.Protocols = append(manager.Server.Protocols, p2p.Protocol{
		Name:    params.ProtocolName,
		Version: params.ProtocolVersion,
		Run: func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
			// add peer -> sub protocol handleshake -> handle message
			return manager.handle(peer, rw)
		},
		BaseCmd: baseMsg,
	})

	manager.msgnet = msgnet.NewMsgnet(manager.peerAddress(), netConfig.RouteAddress, manager.handleMsgnetMessage, logDir)
	manager.merger = merge.NewHelper(ledger, blockchain, manager, mergeConfig)

	go jrpc.StartServer(config.JrpcConfig(), manager)
	return manager
}

// Start starts a protocol server
func (pm *ProtocolManager) Start() {
	pm.Server.Start()
	pm.merger.Start()

	go pm.consensusReadLoop()
	go pm.broadcastLoop()

	pm.init()
}

// Sign signs data with nodekey
func (pm ProtocolManager) Sign(data []byte) (*crypto.Signature, error) {
	return pm.Server.Sign(data)
}

// init initializes protocol manager
func (pm *ProtocolManager) init() {
	pm.statusData = StatusData{
		Version:     params.VersionMajor,
		StartHeight: pm.Blockchain.CurrentHeight(),
	}
}

func (pm *ProtocolManager) handle(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	// handleshake
	pm.handleShake(rw)
	msg, err := rw.ReadMsg()
	// log.Debugf("status msg %v, error %v", msg, err)
	if err != nil {
		return err
	}

	if msg.Cmd == statusMsg {
		pm.OnStatus(msg, p)
	} else {
		return err
	}

	return pm.handleMsg(p, rw)
}

func (pm *ProtocolManager) handleMsg(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	for {
		m, err := rw.ReadMsg()
		log.Debugf("ProtocolManager handle message %s", msgMap[m.Cmd])
		if err != nil {
			return err
		}
		switch m.Cmd {
		case statusMsg:
			return fmt.Errorf("should not appear status message")
		// case getBlocksMsg:
		// 	pm.OnGetBlocks(m, p)
		// case invMsg:
		// 	pm.OnInv(m, p)
		case txMsg:
			pm.OnTx(m, p)
		// case blockMsg:
		// 	pm.OnBlock(m, p)
		// case getdataMsg:
		// 	pm.OnGetData(m, p)
		case consensusMsg:
			pm.OnConsensus(m, p)
		case broadcastAckMergeTxsMsg:
			pm.merger.HandleLocalMsg(m)
		default:
			log.Error("Unknown message")
		}
	}
}

func (pm *ProtocolManager) handleShake(rw p2p.MsgReadWriter) {
	rw.WriteMsg(*p2p.NewMsg(statusMsg, utils.Serialize(pm.statusData)))
}

// Relay relays inventory to remote peers
func (pm *ProtocolManager) Relay(inv types.IInventory) {
	var (
		// inventory InvVect
		msg *p2p.Msg
	)
	// log.Debugf("ProtocolManager Relay inventory, hash: %s", inv.Hash())

	switch inv.(type) {
	case *types.Transaction:
		var tx types.Transaction
		tx.Deserialize(inv.Serialize())

		if tx.GetType() == types.TypeMerged {
			msg = p2p.NewMsg(broadcastAckMergeTxsMsg, inv.Serialize())
			break
		}

		if pm.Blockchain.ProcessTransaction(&tx) {
			// inventory.Type = InvTypeTx
			// inventory.Hashes = []crypto.Hash{inv.Hash()}
			msg = p2p.NewMsg(txMsg, inv.Serialize())
		}
	case *types.Block:
		if pm.Blockchain.ProcessBlock(inv.(*types.Block)) {
			pm.statusData.StartHeight++
			// inventory.Type = InvTypeBlock
			// inventory.Hashes = []crypto.Hash{inv.Hash()}
			// log.Debugf("Relay inventory %v", inventory)
			// msg = p2p.NewMsg(invMsg, utils.Serialize(inventory))
		}
	}
	if msg != nil {
		// log.Debugf("relay message %v", *msg)
		pm.msgCh <- msg
	}
}

func (pm *ProtocolManager) consensusReadLoop() {
	for {
		select {
		case consensusData := <-pm.consenter.BroadcastConsensusChannel():
			to := consensusData.To()
			if bytes.Equal(coordinate.HexToChainCoordinate(to), params.ChainID) {
				log.Debugf("Broadcast Consensus Message from %v to %v", params.ChainID, coordinate.HexToChainCoordinate(to))
				pm.msgCh <- p2p.NewMsg(consensusMsg, consensusData.Payload())
			} else {
				// broadcast message to msg-net
				data := msgnet.Message{}
				data.Cmd = msgnet.ChainConsensusMsg
				data.Payload = consensusData.Payload()
				res := pm.SendMsgnetMessage(pm.peerAddress(), to, data)
				log.Debugf("Broadcast consensus message to msg-net, result: %t", res)
			}
		case tx := <-pm.consenter.BroadcastTransactionChannel():
			msg := p2p.NewMsg(txMsg, tx.Serialize())
			pm.msgCh <- msg
		}
	}
}

func (pm *ProtocolManager) broadcastLoop() {
	for {
		select {
		case msg := <-pm.msgCh:
			pm.Broadcast(msg)
		}
	}
}

// OnStatus handles statusMsg
func (pm *ProtocolManager) OnStatus(m p2p.Msg, p *p2p.Peer) {
	// swich status with remote peer
	// add status to peer instance
	// if local peer startheight behind remote, start sync
	statusData := StatusData{}
	utils.Deserialize(m.Payload, &statusData)
	peer := newPeer(p, statusData)
	pm.peers = append(pm.peers, peer)
	log.Debugf("Status Msg %d %d", pm.statusData.StartHeight, peer.Status.StartHeight)
	if pm.statusData.StartHeight < peer.Status.StartHeight {
		// getBlocks := GetBlocks{
		// 	Version:       pm.statusData.Version,
		// 	LocatorHashes: []crypto.Hash{pm.Blockchain.CurrentBlockHash()},
		// 	HashStop:      crypto.Hash{},
		// }
		// p2p.SendMessage(p.Conn, p2p.NewMsg(getBlocksMsg, utils.Serialize(getBlocks)))
	}
}

// OnTx processes tx message
func (pm *ProtocolManager) OnTx(m p2p.Msg, p *p2p.Peer) {
	//TODO: broadcast after validation
	tx := new(types.Transaction)
	tx.Deserialize(m.Payload)
	// p.AddFilter(m.CheckSum[:])
	log.Debugf("Tx Msg %s", tx.Hash())
	if pm.Blockchain.ProcessTransaction(tx) {
		// pm.msgCh <- &m
	}
}

// OnGetBlocks processes getblocks message
func (pm *ProtocolManager) OnGetBlocks(m p2p.Msg, peer *p2p.Peer) {
	var (
		getblocks GetBlocks
		hash      crypto.Hash
		hashes    []crypto.Hash
		inventory InvVect
		err       error
	)

	utils.Deserialize(m.Payload, getblocks)
	//
	for _, h := range getblocks.LocatorHashes {
		// validate locator
		hash = h
	}

	for {
		hash, err = pm.GetNextBlockHash(hash)
		log.Debugf("GetNextBlockHash hash %s, error %v", hash, err)
		if err != nil || hash.Equal(crypto.Hash{}) {
			break
		} else {
			hashes = append(hashes, hash)
		}
	}

	if len(hashes) > 0 {
		inventory.Type = InvTypeBlock
		inventory.Hashes = hashes

		p2p.SendMessage(peer.Conn, p2p.NewMsg(invMsg, utils.Serialize(inventory)))
	}
}

// OnBlock processes block message
func (pm *ProtocolManager) OnBlock(m p2p.Msg, p *p2p.Peer) {
	//TODO: broadcast after validation
	blk := new(types.Block)
	blk.Deserialize(m.Payload)
	log.Debugf("Block Msg %s", blk.Hash())
	// p.AddFilter(m.CheckSum[:])
	if pm.Blockchain.ProcessBlock(blk) {
		pm.statusData.StartHeight++
		// pm.msgCh <- p2p.NewMsg(invMsg, blk.Hash().Bytes())
	}
}

// OnInv processes inventory message
func (pm *ProtocolManager) OnInv(m p2p.Msg, peer *p2p.Peer) {
	// TODO: parse tx inv
	var (
		inventory InvVect
		getdata   GetData
		hashes    []crypto.Hash
	)

	utils.Deserialize(m.Payload, &inventory)

	switch inventory.Type {
	case InvTypeTx:
		// log.Debugf("Inv Tx %v", inventory.Hashes)
		for _, h := range inventory.Hashes {
			if tx, _ := pm.GetTransaction(h); tx == nil {
				hashes = append(hashes, h)
			}
		}

		getdata.InvList = []InvVect{
			InvVect{
				Type: InvTypeTx,
			},
		}
	case InvTypeBlock:
		// log.Debugf("Inv Block %v", inventory.Hashes)
		for _, h := range inventory.Hashes {
			if block, _ := pm.GetBlockByHash(h.Bytes()); block == nil {
				hashes = append(hashes, h)
			}
		}
		getdata.InvList = []InvVect{
			InvVect{
				Type: InvTypeBlock,
			},
		}
	}

	if len(hashes) > 0 {
		getdata.InvList[0].Hashes = hashes
		msg := p2p.NewMsg(getdataMsg, utils.Serialize(getdata))
		p2p.SendMessage(peer.Conn, msg)
	}
}

// OnGetData processes getdata message
func (pm *ProtocolManager) OnGetData(m p2p.Msg, peer *p2p.Peer) {
	var (
		getdata GetData
	)

	utils.Deserialize(m.Payload, &getdata)
	log.Debugf("OnGetData message %v", getdata)

	for _, inventory := range getdata.InvList {
		switch inventory.Type {
		case InvTypeBlock:
			for _, h := range inventory.Hashes {
				if block, _ := pm.GetBlockByHash(h.Bytes()); block != nil {
					log.Debugf("GetBlock from local, %s", block.Hash())
					msg := p2p.NewMsg(blockMsg, block.Serialize())
					p2p.SendMessage(peer.Conn, msg)
				}
			}
		case InvTypeTx:
			for _, h := range inventory.Hashes {
				if tx, _ := pm.GetTransaction(h); tx != nil {
					log.Debugf("GetTransaction from local, %s", tx.Hash())
					msg := p2p.NewMsg(txMsg, tx.Serialize())
					p2p.SendMessage(peer.Conn, msg)
				}
			}
		}
	}
}

// OnConsensus processes consensus message
func (pm *ProtocolManager) OnConsensus(m p2p.Msg, peer *p2p.Peer) {
	log.Debugf("Req receive consensus message %v", m.Cmd)
	pm.consenter.RecvConsensus(m.Payload) //(p.ID.String(), []byte(""), m.Payload)
}

func (pm *ProtocolManager) peerAddress() string {
	return fmt.Sprintf("%s:%s", coordinate.NewChainCoordinate(params.ChainID), pm.GetLocalPeer().ID)
}

// SendMsgnetMessage sends message to msg-net
func (pm *ProtocolManager) SendMsgnetMessage(src, dst string, msg msgnet.Message) bool {
	h := crypto.Sha256(append(msg.Serialize(), src+dst...))
	sig, err := pm.Sign(h[:])

	if err != nil {
		log.Errorln(err.Error())
		return false
	}

	if pm.msgnet != nil {
		return pm.msgnet.Send(dst, msg.Serialize(), sig[:])
	}

	return false
}

func (pm *ProtocolManager) handleMsgnetMessage(src, dst string, payload, signature []byte) error {
	sig := crypto.Signature{}
	copy(sig[:], signature)

	if !sig.Validate() {
		return errors.New("msg-net signature error")
	}

	h := crypto.Sha256(append(payload, src+dst...))
	pub, err := sig.RecoverPublicKey(h[:])
	if pub == nil || err != nil {
		log.Debug("PubilcKey verify error")
		return errors.New("PubilcKey verify error")
	}

	msg := msgnet.Message{}
	msg.Deserialize(payload)
	log.Debugf("recv msg-net message type %d ", msg.Cmd)

	switch msg.Cmd {
	case msgnet.ChainConsensusMsg:
		chainID, peerID := parseID(src)
		// consensus.OnNewMessage(peerID.String(), chainID, m.Payload)
		log.Debugf("recv consensus msg from  %v:%v \n", chainID, peerID)
		pm.consenter.RecvConsensus(msg.Payload)
	case msgnet.ChainTxMsg:
		tx := &types.Transaction{}
		tx.Deserialize(msg.Payload)
		pm.Blockchain.ProcessTransaction(tx)
		log.Debugln("recv transaction msg")
	case msgnet.ChainMergeTxsMsg:
		fallthrough
	case msgnet.ChainAckMergeTxsMsg:
		fallthrough
	case msgnet.ChainAckedMergeTxsMsg:
		chainID, peerID := parseID(src)
		pm.merger.HandleNetMsg(msg.Cmd, chainID.String(), peerID.String(), msg)
		log.Debugf("mergeRecv cmd : %v transaction msg from message net %v:%v ,src: %v\n", msg.Cmd, chainID, peerID, src)
	default:
		log.Debug("not know msgnet.type...")
	}
	return nil
}

// parseID returns chainID and PeerID
func parseID(peerAddress string) (coordinate.ChainCoordinate, p2p.PeerID) {
	id := strings.Split(peerAddress, ":")
	chainID := coordinate.HexToChainCoordinate(id[0])
	if len(id) == 2 {
		peerid, _ := hex.DecodeString(id[1])
		return chainID, p2p.PeerID(peerid)
	}
	return chainID, nil
}
