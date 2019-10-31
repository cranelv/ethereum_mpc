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
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
)

type inertTimer struct{}

func (it *inertTimer) Halt()                                                {}
func (it *inertTimer) Reset(duration time.Duration, event events.Event)     {}
func (it *inertTimer) SoftReset(duration time.Duration, event events.Event) {}
func (it *inertTimer) Stop()                                                {}

type inertTimerFactory struct{}

func (it *inertTimerFactory) CreateTimer() events.Timer {
	return &inertTimer{}
}

type noopSecurity struct{}

func (ns *noopSecurity) Sign(msg []byte) ([]byte, error) {
	return nil, nil
}

func (ns *noopSecurity) Verify(peerID pbftTypes.ReplicaID, signature []byte, message []byte) error {
	return nil
}

type mockPersist struct {
	store map[string][]byte
}

func (p *mockPersist) initialize() {
	if p.store == nil {
		p.store = make(map[string][]byte)
	}
}

func (p *mockPersist) ReadState(key string) ([]byte, error) {
	p.initialize()
	if val, ok := p.store[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("cannot find key %s", key)
}

func (p *mockPersist) ReadStateSet(prefix string) (map[string][]byte, error) {
	if p.store == nil {
		return nil, fmt.Errorf("no state yet")
	}
	ret := make(map[string][]byte)
	for k, v := range p.store {
		if len(k) >= len(prefix) && k[0:len(prefix)] == prefix {
			ret[k] = v
		}
	}
	return ret, nil
}

func (p *mockPersist) StoreState(key string, value []byte) error {
	p.initialize()
	p.store[key] = value
	return nil
}

func (p *mockPersist) DelState(key string) {
	p.initialize()
	delete(p.store, key)
}

func createRunningPbftWithManager(id pbftTypes.ReplicaID, config *params.Config, stack innerStack) (*pbftCore, events.Manager) {
	manager := events.NewManagerImpl()
	core := newPbftCore(id, loadConfig(), stack, events.NewTimerFactoryImpl(manager))
	manager.SetReceiver(core)
	manager.Start()
	return core, manager
}

func createTask(tag int64) (task *message.Task) {
	task = &message.Task{Type: 1,
		TimeStamp: uint64(tag),
		Payload:   []byte(fmt.Sprint(tag)),
	}
	return
}

func marshalTx(tx *message.Task) (txPacked []byte) {
	txPacked, _ = singletons.Marshaler.Marshal(tx)
	return
}

func createTxMsg(tag int64) (msg *message.Message) {
	req := createPbftReq(tag,pbftTypes.ReplicaID(tag))
	payload,_ := req.Marshal()
	msg = &message.Message{
		Type:    message.Message_CHAIN_TASKS,
		Payload: payload,
	}
	return
}

func createPbftReq(tag int64, replica pbftTypes.ReplicaID) (req *message.Request) {
	tx := createTask(tag)
	req = &message.Request{
		Timestamp: tx.TimeStamp,
		ReplicaId: replica,
	}
	req.Tasks = append(req.Tasks,tx)
	return
}

func createPbftReqBatch(tag int64, replica pbftTypes.ReplicaID) (reqBatch *message.RequestBatch) {
	reqBatch = &message.RequestBatch{}
	req := createPbftReq(tag, replica)
	*reqBatch = append(*reqBatch,req)
	return
}

func createPbftReqBatchMsg(tag int64, replica pbftTypes.ReplicaID) (msg *message.Message) {
	reqBatch := createPbftReqBatch(tag, replica)
	payload,_ := reqBatch.Marshal()
	msg = &message.Message{Payload: payload}
	return
}

func generateBroadcaster(validatorCount uint32) (requestBroadcaster int) {
	seed := rand.NewSource(time.Now().UnixNano())
	rndm := rand.New(seed)
	requestBroadcaster = rndm.Intn(int(validatorCount))
	return
}

type omniProto struct {
	consensusInterface.ValidatorIdentifyInterface
	// Stack methods
	GetNetworkInfoImpl         func() (self pbftTypes.Peer, network []pbftTypes.Peer, err error)
	GetNetworkHandlesImpl      func() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error)
	BroadcastImpl              func(msg *message.Message, peerType pbftTypes.Peer_Type) error
	UnicastImpl                func(msg *message.Message, receiverHandle *pbftTypes.PeerID) error
	SignImpl                   func(msg []byte) ([]byte, error)
	VerifyImpl                 func(peerID pbftTypes.ReplicaID, signature []byte, message []byte) error
	GetBlockImpl               func(id uint64) (block *message.StateInfo, err error)
	GetCurrentStateHashImpl    func() (stateHash []byte, err error)
	GetBlockchainSizeImpl      func() uint64
	GetBlockHeadMetadataImpl   func() ([]byte, error)
	GetBlockchainInfoImpl      func() *message.StateInfo
	GetBlockchainInfoBlobImpl  func() []byte
	HashBlockImpl              func(block *message.Block) ([]byte, error)
	VerifyBlockchainImpl       func(start, finish uint64) (uint64, error)
	PutBlockImpl               func(blockNumber uint64, block *message.StateInfo) error
	ApplyStateDeltaImpl        func(id interface{}, delta *message.StateInfo) error
	CommitStateDeltaImpl       func(id interface{}) error
	RollbackStateDeltaImpl     func(id interface{}) error
	EmptyStateImpl             func() error
	ExecuteImpl                func(id interface{}, tasks []*message.Task)
	CommitImpl                 func(id interface{}, meta []byte)
	RollbackImpl               func(id interface{})
	UpdateStateImpl            func(id interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID)
	BeginTxBatchImpl           func(id interface{}) error
	ExecTxsImpl                func(id interface{}, tasks []*message.Task) ([]*message.Result, error)
	CommitTxBatchImpl          func(id interface{}, metadata []byte) (*message.StateInfo, error)
	RollbackTxBatchImpl        func(id interface{}) error
	PreviewCommitTxBatchImpl   func(id interface{}, metadata []byte) ([]byte, error)
//	GetRemoteBlocksImpl        func(replicaID *pbftTypes.PeerID, start, finish uint64) (<-chan *pb.SyncBlocks, error)
//	GetRemoteStateSnapshotImpl func(replicaID *pbftTypes.PeerID) (<-chan *pb.SyncStateSnapshot, error)
//	GetRemoteStateDeltasImpl   func(replicaID *pbftTypes.PeerID, start, finish uint64) (<-chan *pb.SyncStateDeltas, error)
	ReadStateImpl              func(key string) ([]byte, error)
	ReadStateSetImpl           func(prefix string) (map[string][]byte, error)
	StoreStateImpl             func(key string, value []byte) error
	DelStateImpl               func(key string)
	ValidateStateImpl          func()
	InvalidateStateImpl        func()

	// Inner Stack methods
	broadcastImpl       func(msgPayload []byte)
	unicastImpl         func(msgPayload []byte, receiverID pbftTypes.ReplicaID) (err error)
	executeImpl         func(seqNo uint64, reqBatch *message.RequestBatch)
	getStateImpl        func() []byte
	skipToImpl          func(seqNo uint64, snapshotID []byte, peers []pbftTypes.ReplicaID)
	viewChangeImpl      func(curView uint64)
	signImpl            func(msg []byte) ([]byte, error)
	verifyImpl          func(senderID pbftTypes.ReplicaID, signature []byte, message []byte) error
	getLastSeqNoImpl    func() (uint64, error)
	validateStateImpl   func()
	invalidateStateImpl func()

	// Closable Consenter methods
	RecvMsgImpl func(ocMsg *message.Message, senderHandle *pbftTypes.PeerID) error
	CloseImpl   func()
	deliverImpl func([]byte, *pbftTypes.PeerID)

	// Orderer methods
	ValidateImpl func(seqNo uint64, id []byte) (commit bool, correctedID []byte, peerIDs []*pbftTypes.PeerID)
	SkipToImpl   func(seqNo uint64, id []byte, peers []*pbftTypes.PeerID)
}

func (op *omniProto) GetNetworkNodes() (self pbftTypes.Peer, network []pbftTypes.Peer, err error) {
	if nil != op.GetNetworkInfoImpl {
		return op.GetNetworkInfoImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) GetNetworkNodeIDs() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error) {
	if nil != op.GetNetworkHandlesImpl {
		return op.GetNetworkHandlesImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) Broadcast(msg *message.Message, peerType pbftTypes.Peer_Type) error {
	if nil != op.BroadcastImpl {
		return op.BroadcastImpl(msg, peerType)
	}

	panic("Unimplemented")
}
func (op *omniProto) Unicast(msg *message.Message, receiverHandle *pbftTypes.PeerID) error {
	if nil != op.UnicastImpl {
		return op.UnicastImpl(msg, receiverHandle)
	}

	panic("Unimplemented")
}
func (op *omniProto) Sign(msg []byte) ([]byte, error) {
	if nil != op.SignImpl {
		return op.SignImpl(msg)
	}

	panic("Unimplemented")
}
func (op *omniProto) Verify(peerID pbftTypes.ReplicaID, signature []byte, message []byte) error {
	if nil != op.VerifyImpl {
		return op.VerifyImpl(peerID, signature, message)
	}

	panic("Unimplemented")
}
func (op *omniProto) GetState(id uint64) (block *message.StateInfo, err error) {
	if nil != op.GetBlockImpl {
		return op.GetBlockImpl(id)
	}

	panic("Unimplemented")
}
func (op *omniProto) GetCurrentStateHash() (stateHash []byte, err error) {
	if nil != op.GetCurrentStateHashImpl {
		return op.GetCurrentStateHashImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) GetBlockchainSize() uint64 {
	if nil != op.GetBlockchainSizeImpl {
		return op.GetBlockchainSizeImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) GetBlockHeadMetadata() ([]byte, error) {
	if nil != op.GetBlockHeadMetadataImpl {
		return op.GetBlockHeadMetadataImpl()
	}

	return nil, nil
}
func (op *omniProto) GetBlockchainInfoBlob() []byte {
	if nil != op.GetBlockchainInfoBlobImpl {
		return op.GetBlockchainInfoBlobImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) GetBlockchainInfo() *message.StateInfo {
	if nil != op.GetBlockchainInfoImpl {
		return op.GetBlockchainInfoImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) HashBlock(block *message.Block) ([]byte, error) {
	if nil != op.HashBlockImpl {
		return op.HashBlockImpl(block)
	}

	panic("Unimplemented")
}
func (op *omniProto) VerifyBlockchain(start, finish uint64) (uint64, error) {
	if nil != op.VerifyBlockchainImpl {
		return op.VerifyBlockchainImpl(start, finish)
	}

	panic("Unimplemented")
}
func (op *omniProto) PutBlock(blockNumber uint64, block *message.StateInfo) error {
	if nil != op.PutBlockImpl {
		return op.PutBlockImpl(blockNumber, block)
	}

	panic("Unimplemented")
}
func (op *omniProto) ApplyStateDelta(id interface{}, delta *message.StateInfo) error {
	if nil != op.ApplyStateDeltaImpl {
		return op.ApplyStateDeltaImpl(id, delta)
	}

	panic("Unimplemented")
}
func (op *omniProto) CommitStateDelta(id interface{}) error {
	if nil != op.CommitStateDeltaImpl {
		return op.CommitStateDeltaImpl(id)
	}

	panic("Unimplemented")
}
func (op *omniProto) RollbackStateDelta(id interface{}) error {
	if nil != op.RollbackStateDeltaImpl {
		return op.RollbackStateDeltaImpl(id)
	}

	panic("Unimplemented")
}
func (op *omniProto) EmptyState() error {
	if nil != op.EmptyStateImpl {
		return op.EmptyStateImpl()
	}

	panic("Unimplemented")
}
func (op *omniProto) BeginTaskBatch(id interface{}) error {
	if nil != op.BeginTxBatchImpl {
		return op.BeginTxBatchImpl(id)
	}

	panic("Unimplemented")
}
func (op *omniProto) ExecTasks(id interface{}, txs []*message.Task) ([]*message.Result, error) {
	if nil != op.ExecTxsImpl {
		return op.ExecTxsImpl(id, txs)
	}

	panic("Unimplemented")
}
func (op *omniProto) CommitTaskBatch(id interface{}, metadata []byte) (*message.StateInfo, error) {
	if nil != op.CommitTxBatchImpl {
		return op.CommitTxBatchImpl(id, metadata)
	}

	panic("Unimplemented")
}
func (op *omniProto) RollbackTaskBatch(id interface{}) error {
	if nil != op.RollbackTxBatchImpl {
		return op.RollbackTxBatchImpl(id)
	}

	panic("Unimplemented")
}
func (op *omniProto) PreviewCommitTaskBatch(id interface{}, metadata []byte) ([]byte, error) {
	if nil != op.PreviewCommitTxBatchImpl {
		return op.PreviewCommitTxBatchImpl(id, metadata)
	}

	panic("Unimplemented")
}
/*
func (op *omniProto) GetRemoteBlocks(replicaID *pbftTypes.PeerID, start, finish uint64) (<-chan *pb.SyncBlocks, error) {
	if nil != op.GetRemoteBlocksImpl {
		return op.GetRemoteBlocksImpl(replicaID, start, finish)
	}

	panic("Unimplemented")
}
func (op *omniProto) GetRemoteStateSnapshot(replicaID *pbftTypes.PeerID) (<-chan *pb.SyncStateSnapshot, error) {
	if nil != op.GetRemoteStateSnapshotImpl {
		return op.GetRemoteStateSnapshotImpl(replicaID)
	}

	panic("Unimplemented")
}
func (op *omniProto) GetRemoteStateDeltas(replicaID *pbftTypes.PeerID, start, finish uint64) (<-chan *pb.SyncStateDeltas, error) {
	if nil != op.GetRemoteStateDeltasImpl {
		return op.GetRemoteStateDeltasImpl(replicaID, start, finish)
	}

	panic("Unimplemented")
}
*/
func (op *omniProto) broadcast(msgPayload []byte) {
	if nil != op.broadcastImpl {
		op.broadcastImpl(msgPayload)
		return
	}

	panic("Unimplemented")
}
func (op *omniProto) unicast(msgPayload []byte, receiverID pbftTypes.ReplicaID) (err error) {
	if nil != op.unicastImpl {
		return op.unicastImpl(msgPayload, receiverID)
	}

	panic("Unimplemented")
}
func (op *omniProto) execute(seqNo uint64, reqBatch *message.RequestBatch) {
	if nil != op.executeImpl {
		op.executeImpl(seqNo, reqBatch)
		return
	}

	panic("Unimplemented")
}
func (op *omniProto) skipTo(seqNo uint64, snapshotID []byte, peers []pbftTypes.ReplicaID) {
	if nil != op.skipToImpl {
		op.skipToImpl(seqNo, snapshotID, peers)
		return
	}

	panic("Unimplemented")
}
func (op *omniProto) viewChange(curView uint64) {
	if nil != op.viewChangeImpl {
		op.viewChangeImpl(curView)
		return
	}

	panic("Unimplemented")
}
func (op *omniProto) sign(msg []byte) ([]byte, error) {
	if nil != op.signImpl {
		return op.signImpl(msg)
	}

	panic("Unimplemented")
}
func (op *omniProto) verify(senderID pbftTypes.ReplicaID, signature []byte, message []byte) error {
	if nil != op.verifyImpl {
		return op.verifyImpl(senderID, signature, message)
	}

	panic("Unimplemented")
}

func (op *omniProto) RecvMsg(ocMsg *message.Message, senderHandle *pbftTypes.PeerID) error {
	if nil != op.RecvMsgImpl {
		return op.RecvMsgImpl(ocMsg, senderHandle)
	}

	panic("Unimplemented")
}

func (op *omniProto) getLastSeqNo() (uint64, error) {
	if op.getLastSeqNoImpl != nil {
		return op.getLastSeqNoImpl()
	}

	return 0, fmt.Errorf("getLastSeqNo is not implemented")
}

func (op *omniProto) Close() {
	if nil != op.CloseImpl {
		op.CloseImpl()
		return
	}

	panic("Unimplemented")
}

func (op *omniProto) Validate(seqNo uint64, id []byte) (commit bool, correctedID []byte, peerIDs []*pbftTypes.PeerID) {
	if nil != op.ValidateImpl {
		return op.ValidateImpl(seqNo, id)
	}

	panic("Unimplemented")

}

func (op *omniProto) SkipTo(seqNo uint64, meta []byte, id []*pbftTypes.PeerID) {
	if nil != op.SkipToImpl {
		op.SkipToImpl(seqNo, meta, id)
		return
	}

	panic("Unimplemented")
}

func (op *omniProto) deliver(msg []byte, target *pbftTypes.PeerID) {
	if nil != op.deliverImpl {
		op.deliverImpl(msg, target)
	}

	panic("Unimplemented")
}

func (op *omniProto) getState() []byte {
	if nil != op.getStateImpl {
		return op.getStateImpl()
	}

	panic("Unimplemented")
}

func (op *omniProto) ReadState(key string) ([]byte, error) {
	if nil != op.ReadStateImpl {
		return op.ReadStateImpl(key)
	}
	return nil, fmt.Errorf("unimplemented")
}

func (op *omniProto) ReadStateSet(prefix string) (map[string][]byte, error) {
	if nil != op.ReadStateImpl {
		return op.ReadStateSetImpl(prefix)
	}
	return nil, fmt.Errorf("unimplemented")
}

func (op *omniProto) DelState(key string) {
	if nil != op.DelStateImpl {
		op.DelStateImpl(key)
	}
}

func (op *omniProto) StoreState(key string, value []byte) error {
	if nil != op.StoreStateImpl {
		return op.StoreStateImpl(key, value)
	}
	return fmt.Errorf("unimplemented")
}

func (op *omniProto) ValidateState() {
	if nil != op.ValidateStateImpl {
		op.ValidateStateImpl()
		return
	}
	panic("unimplemented")
}

func (op *omniProto) InvalidateState() {
	if nil != op.InvalidateStateImpl {
		op.InvalidateStateImpl()
		return
	}
	panic("unimplemented")
}

func (op *omniProto) validateState() {
	if nil != op.validateStateImpl {
		op.validateStateImpl()
		return
	}
	panic("unimplemented")
}

func (op *omniProto) invalidateState() {
	if nil != op.invalidateStateImpl {
		op.invalidateStateImpl()
		return
	}
	panic("unimplemented")
}
func (op *omniProto) Commit(tag interface{}, meta []byte) {
	if nil != op.CommitImpl {
		op.CommitImpl(tag, meta)
		return
	}
	panic("unimplemented")
}
func (op *omniProto) UpdateState(tag interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID) {
	if nil != op.UpdateStateImpl {
		op.UpdateStateImpl(tag, target, peers)
		return
	}
	panic("unimplemented")
}
func (op *omniProto) Rollback(tag interface{}) {
	if nil != op.RollbackImpl {
		op.RollbackImpl(tag)
		return
	}
	panic("unimplemented")
}
func (op *omniProto) Execute(tag interface{}, tasks []*message.Task) {
	if nil != op.ExecuteImpl {
		op.ExecuteImpl(tag, tasks)
		return
	}
	panic("unimplemented")
}

// These methods are a temporary hack until the consensus API can be cleaned a little
func (op *omniProto) Start() {}
func (op *omniProto) Halt()  {}
