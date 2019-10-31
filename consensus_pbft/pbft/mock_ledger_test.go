/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pbft

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
)

type LedgerDirectory interface {
	GetLedgerByPeerID(peerID *pbftTypes.PeerID) (consensus_pbft.ReadOnlyLedger, bool)
}

type HashLedgerDirectory struct {
	remoteLedgers map[pbftTypes.PeerID]consensus_pbft.ReadOnlyLedger
}

func (hd *HashLedgerDirectory) GetLedgerByPeerID(peerID *pbftTypes.PeerID) (consensus_pbft.ReadOnlyLedger, bool) {
	ledger, ok := hd.remoteLedgers[*peerID]
	return ledger, ok
}

func (hd *HashLedgerDirectory) GetPeers() ([]pbftTypes.Peer, error) {
	_, network, err := hd.GetNetworkNodes()
	return network , err
}

func (hd *HashLedgerDirectory) GetPeerNode() (pbftTypes.Peer, error) {
	self, _, err := hd.GetNetworkNodes()
	return self, err
}
type testPeer struct {
	ID pbftTypes.PeerID
	Type pbftTypes.Peer_Type
}
func(pp* testPeer)GetPeerId() *pbftTypes.PeerID{
	peerID := pbftTypes.PeerID(pp.ID)
	return &peerID
}
func(pp* testPeer)GetType() pbftTypes.Peer_Type{
	return pp.Type
}

func (hd *HashLedgerDirectory) GetNetworkNodes() (self pbftTypes.Peer, network []pbftTypes.Peer, err error) {
	network = make([]pbftTypes.Peer, len(hd.remoteLedgers)+1)
	i := 0
	for peerID := range hd.remoteLedgers {
		network[i] = &testPeer{
			ID:peerID,
			Type: pbftTypes.Peer_VALIDATOR,
		}
		i++
	}
	peerId := stringToPeerId("self")
	network[i] = &testPeer{
		ID: peerId,
		Type: pbftTypes.Peer_VALIDATOR,
	}

	self = network[i]
	return
}

func (hd *HashLedgerDirectory) GetNetworkNodeIDs() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error) {
	oSelf, oNetwork, err := hd.GetNetworkNodes()
	if nil != err {
		return
	}

	self = oSelf.GetPeerId()
	network = make([]*pbftTypes.PeerID, len(oNetwork))
	for i, endpoint := range oNetwork {
		network[i] = endpoint.GetPeerId()
	}
	return
}

type MockLedger struct {
	cleanML       *MockLedger
	blocks        map[uint64]*message.StateInfo
	blockHeight   uint64
	remoteLedgers LedgerDirectory

	mutex *sync.Mutex

	txID          interface{}
	curBatch      []*message.Task
	curResults    []*message.Result
	preBatchState uint64

	ce *consumerEndpoint // To support the ExecTx stuff
}

func NewMockLedger(remoteLedgers LedgerDirectory) *MockLedger {
	mock := &MockLedger{}
	mock.mutex = &sync.Mutex{}
	mock.blocks = make(map[uint64]*message.StateInfo)
	mock.blockHeight = 1
	mock.blocks[0] = &message.StateInfo{Hash:"0x10021",Number:0}
	mock.remoteLedgers = remoteLedgers

	return mock
}

func (mock *MockLedger) BeginTaskBatch(id interface{}) error {
	if mock.txID != nil {
		return fmt.Errorf("Tx batch is already active")
	}
	mock.txID = id
	mock.curBatch = nil
	mock.curResults = nil
	return nil
}

func (mock *MockLedger) Execute(tag interface{}, txs []*message.Task) {
	go func() {
		if mock.txID == nil {
			mock.BeginTaskBatch(mock)
		}

		_, err := mock.ExecTasks(mock, txs)
		if err != nil {
			panic(err)
		}
		mock.ce.consumer.Executed(tag)
	}()
}

func (mock *MockLedger) Commit(tag interface{}, meta []byte) {
	go func() {
		_, err := mock.CommitTaskBatch(mock, meta)
		if err != nil {
			panic(err)
		}
		mock.ce.consumer.Committed(tag, mock.GetStateInfo())
	}()
}

func (mock *MockLedger) Rollback(tag interface{}) {
	go func() {
		mock.RollbackTaskBatch(mock)
		mock.ce.consumer.RolledBack(tag)
	}()
}

func (mock *MockLedger) ExecTasks(id interface{}, txs []*message.Task) ([]*message.Result, error) {
	if !reflect.DeepEqual(mock.txID, id) {
		return nil, fmt.Errorf("Invalid batch ID")
	}

	mock.curBatch = append(mock.curBatch, txs...)
	var err error
	var txResult []*message.Result
	if nil != mock.ce && nil != mock.ce.execTasksResult {
		txResult, err = mock.ce.execTasksResult(txs)
	} else {
		// This is basically a default fake default transaction execution
		if nil == txs {
			txs = []*message.Task{{Payload: []byte("DUMMY")}}
		}

		for _, task := range txs {
			if task.Payload == nil {
				task.Payload = []byte("DUMMY")
			}

			txResult = append(txResult, &message.Result{task.Type,task.TimeStamp,task.Payload})
		}

	}

	mock.curResults = append(mock.curResults, txResult...)

	return txResult, err
}

func (mock *MockLedger) CommitTaskBatch(id interface{}, metadata []byte) (*message.StateInfo, error) {
	block, err := mock.commonCommitTx(id, metadata, false)
	if nil == err {
		mock.txID = nil
		mock.curBatch = nil
		mock.curResults = nil
	}
	return &message.StateInfo{Payload:block}, err
}

func (mock *MockLedger) commonCommitTx(id interface{}, metadata []byte, preview bool) (*message.StateInfo, error) {
	if !reflect.DeepEqual(mock.txID, id) {
		return nil, fmt.Errorf("Invalid batch ID")
	}

	previousBlockHash := pbftTypes.MessageDigest("Genesis")
	if 0 < mock.blockHeight {
		previousBlock, _ := mock.GetState(mock.blockHeight - 1)
		previousBlockHash, _ = mock.HashBlock(previousBlock)
	}

	block := &message.Block{
		ConsensusMetadata: metadata,
		PreviousBlockHash: previousBlockHash,
		StateResults:         mock.curResults, // Use the current result output in the hash
		Tasks:      mock.curBatch,
	}

	if !preview {
		hash, _ := mock.HashBlock(&message.StateInfo{Number : mock.blockHeight,Payload:block})
		fmt.Printf("TEST LEDGER: Mock ledger is inserting block %d with hash %x\n", mock.blockHeight, hash)
		mock.blocks[mock.blockHeight] = &message.StateInfo{Number : mock.blockHeight,Hash:hash,Payload:block}
		mock.blockHeight++
	}

	return mock.blocks[mock.blockHeight], nil
}

func (mock *MockLedger) PreviewCommitTaskBatch(id interface{}, metadata []byte) ([]byte, error) {
	b, err := mock.commonCommitTx(id, metadata, true)
	if err != nil {
		return nil, err
	}
	return mock.getBlockInfoBlob(mock.blockHeight+1, b), nil
}

func (mock *MockLedger) RollbackTaskBatch(id interface{}) error {
	if !reflect.DeepEqual(mock.txID, id) {
		return fmt.Errorf("Invalid batch ID")
	}
	mock.curBatch = nil
	mock.curResults = nil
	mock.txID = nil
	return nil
}

func (mock *MockLedger) GetBlockchainSize() uint64 {
	mock.mutex.Lock()
	defer func() {
		mock.mutex.Unlock()
	}()
	return mock.blockHeight
}

func (mock *MockLedger) GetState(id uint64) (*message.StateInfo, error) {
	mock.mutex.Lock()
	defer func() {
		mock.mutex.Unlock()
	}()
	block, ok := mock.blocks[id]
	if !ok {
		return nil, fmt.Errorf("Block not found")
	}
	return block, nil
}

func (mock *MockLedger) HashBlock(block *message.StateInfo) (pbftTypes.MessageDigest, error) {
	block.Hash = singletons.Hasher.Hash(block)
	return block.Hash,nil
}

func (mock *MockLedger) GetStateInfo() *message.StateInfo {
	b, _ := mock.GetState(mock.blockHeight - 1)
	return  b
}

func (mock *MockLedger) GetBlockchainInfoBlob() []byte {
	b, _ := mock.GetState(mock.blockHeight - 1)
	return mock.getBlockInfoBlob(mock.blockHeight, b)
}

func (mock *MockLedger) getBlockInfoBlob(height uint64, block *message.StateInfo) []byte {
	h, _ := singletons.Marshaler.Marshal(block)
	return h
}

func (mock *MockLedger) GetBlockHeadMetadata() ([]byte, error) {
	b, ok := mock.blocks[mock.blockHeight-1]
	if !ok {
		return nil, fmt.Errorf("could not retrieve block from mock ledger")
	}
	block := b.Payload.(*message.Block)
	return block.ConsensusMetadata, nil
}

func (mock *MockLedger) simulateStateTransfer(info *message.StateInfo, peers []*pbftTypes.PeerID) {
	if mock.blockHeight >= info.Height() && info.Height()>0 {
		blockCursor := info.Height() - 1
		validHash := info.Digest()
		for {
			info, ok := mock.blocks[blockCursor]
			if !ok {
				break
			}
			hash, _ := mock.HashBlock(info)
			if hash == validHash {
				break
			}
			blockCursor--
			block := info.Payload.(*message.Block)
			validHash = block.PreviousBlockHash
			if blockCursor == ^uint64(0) {
				return
			}
		}
		panic(fmt.Sprintf("Asked to skip to a block (%d) which is lower than our current height of %d.  (Corrupt block at %d with hash %x)", info.Height, mock.blockHeight, blockCursor, validHash))
	}

	var remoteLedger consensus_pbft.ReadOnlyLedger
	if len(peers) > 0 {
		var ok bool
		remoteLedger, ok = mock.remoteLedgers.GetLedgerByPeerID(peers[0])
		if !ok {
			panic("Asked for results from a peer which does not exist")
		}
	} else {
//		panic("TODO, support state transfer from nil peers")
	}
	fmt.Printf("TEST LEDGER skipping to %+v", info)
	p := 0
	for n := mock.blockHeight; n < info.Height(); n++ {
		block, err := remoteLedger.GetState(n)

		if nil != err {
			n--
			fmt.Printf("TEST LEDGER: Block not ready yet")
			time.Sleep(100 * time.Millisecond)
			p++
			if p > 10 {
				panic("Tried to get a block 10 times, no luck")
			}
			continue
		}

		mock.blocks[n] = block
	}
	mock.blockHeight = info.Height()
}
