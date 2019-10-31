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
	"testing"
	"time"
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	fmt "fmt"
	"github.com/ethereum/go-ethereum/log"
)

func (op *obcBatch) getPBFTCore() *pbftCore {
	return op.pbft
}

func obcBatchHelper(id pbftTypes.ReplicaID, config *params.Config,
	stack consensus_pbft.Stack) pbftConsumer {
	// It's not entirely obvious why the compiler likes the parent function, but not newObcBatch directly
	return newObcBatch(id, config, stack)
}

func TestNetworkBatch(t *testing.T) {
	log.InitLog(5)
	batchSize := 2
	validatorCount := uint32(4)
	net := makeConsumerNetwork(validatorCount, obcBatchHelper, func(ce *consumerEndpoint) {
		ce.consumer.(*obcBatch).batchSize = batchSize
	})
	defer net.stop()

	broadcaster := net.endpoints[generateBroadcaster(validatorCount)].getHandle()
	err := net.endpoints[1].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(1), broadcaster)
	if err != nil {
		t.Errorf("External request was not processed by backup: %v", err)
	}
	err = net.endpoints[2].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(2), broadcaster)
	if err != nil {
		t.Fatalf("External request was not processed by backup: %v", err)
	}

	net.process()
	net.process()

	if l := len(net.endpoints[0].(*consumerEndpoint).consumer.(*obcBatch).batchStore); l != 0 {
		t.Errorf("%d messages expected in primary's batchStore, found %v", 0,
			net.endpoints[0].(*consumerEndpoint).consumer.(*obcBatch).batchStore)
	}

	for _, ep := range net.endpoints {
		ce := ep.(*consumerEndpoint)
		block, err := ce.consumer.(*obcBatch).stack.GetState(1)
		if nil != err {
			t.Fatalf("Replica %d executed requests, expected a new block on the chain, but could not retrieve it : %s", ce.id, err)
		}
		numTrans := len(block.Payload.(*message.Block).Tasks)
		if numTrans != batchSize {
			t.Fatalf("Replica %d executed %d requests, expected %d",
				ce.id, numTrans, batchSize)
		}
	}
}

var inertState = &omniProto{
	GetBlockchainInfoImpl: func() *message.StateInfo {
		return &message.StateInfo{
			Hash: pbftTypes.MessageDigest("GENESIS"),
			Number:           1,
		}
	},
	GetBlockchainInfoBlobImpl: func() []byte {
		stateInfo := message.StateInfo{
			Hash: pbftTypes.MessageDigest("GENESIS"),
			Number:           1,
		}
		b, _ := singletons.Marshaler.Marshal(stateInfo)
		return b
	},
	GetNetworkInfoImpl : func() (self pbftTypes.Peer, network []pbftTypes.Peer, err error) {
		return nil,nil,nil
	},
	GetNetworkHandlesImpl : func() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error){
		return nil,nil,nil
	},
	InvalidateStateImpl: func() {},
	ValidateStateImpl:   func() {},
	UpdateStateImpl:     func(id interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID) {},
}

func TestClearOutstandingReqsOnStateRecovery(t *testing.T) {
	log.InitLog(5)
	omni := *inertState
	omni.UnicastImpl = func(msg *message.Message, receiverHandle *pbftTypes.PeerID) error { return nil }
	b := newObcBatch(0, loadConfig(), &omni)
	b.StateUpdated(&checkpointMessage{seqNo: 0, snapshotId: inertState.GetBlockchainInfoBlobImpl()}, inertState.GetBlockchainInfoImpl())

	defer b.Close()

	b.reqStore.storeOutstanding(&message.Request{})

	b.manager.Queue() <- stateUpdatedEvent{
		chkpt: &checkpointMessage{
			seqNo: 10,
		},
	}

	b.manager.Queue() <- nil

	if b.reqStore.outstandingRequests.Len() != 0 {
		t.Fatalf("Should not have any requests outstanding after completing state transfer")
	}
}

func TestOutstandingReqsIngestion(t *testing.T) {
	log.InitLog(5)
	bs := [3]*obcBatch{}
	peerId := stringToPeerId("vp1")
	for i := range bs {
		omni := *inertState
		omni.UnicastImpl = func(ocMsg *message.Message, peer *pbftTypes.PeerID) error { return nil }
		bs[i] = newObcBatch(pbftTypes.ReplicaID(i), loadConfig(), &omni)
		defer bs[i].Close()

		// Have vp1 only deliver messages
		if i == 1 {
			omni.UnicastImpl = func(ocMsg *message.Message, peer *pbftTypes.PeerID) error {
				dest, _ := omni.GetValidatorID(peer)
				if dest == 0 || dest == 2 {

					bs[dest].RecvMsg(ocMsg, &peerId)
				}
				return nil
			}
		}
	}
	for i := range bs {
		bs[i].StateUpdated(&checkpointMessage{seqNo: 0, snapshotId: inertState.GetBlockchainInfoBlobImpl()}, inertState.GetBlockchainInfoImpl())
	}

	err := bs[1].RecvMsg(createTxMsg(1), &peerId)
	if err != nil {
		t.Fatalf("External request was not processed by backup: %v", err)
	}

	for _, b := range bs {
		b.manager.Queue() <- nil
		b.broadcaster.Wait()
		b.manager.Queue() <- nil
	}

	for i, b := range bs {
		b.manager.Queue() <- nil
		count := b.reqStore.outstandingRequests.Len()
		if count != 1 {
			t.Errorf("Batch backup %d should have the request in its store", i)
		}
	}
}

func TestOutstandingReqsResubmission(t *testing.T) {
	log.InitLog(5)
	config := loadConfig()
	config.BatchSize = 2
	omni := *inertState
	b := newObcBatch(0, config, &omni)
	defer b.Close() // The broadcasting threads only cause problems here... but this test stalls without them

	transactionsBroadcast := 0
	omni.ExecuteImpl = func(tag interface{}, tasks []*message.Task) {
		transactionsBroadcast += len(tasks)
		fmt.Printf("\nExecuting %d transactions (%v)\n", len(tasks), tasks)
		nextExec := b.pbft.lastExec + 1
		b.pbft.currentExec = &nextExec
		b.manager.Inject(executedEvent{tag: tag})
	}

	omni.CommitImpl = func(tag interface{}, meta []byte) {
		b.manager.Inject(committedEvent{})
	}

	omni.UnicastImpl = func(ocMsg *message.Message, dest *pbftTypes.PeerID) error {
		return nil
	}

	b.StateUpdated(&checkpointMessage{seqNo: 0, snapshotId: inertState.GetBlockchainInfoBlobImpl()}, inertState.GetBlockchainInfoImpl())
	b.manager.Queue() <- nil // Make sure the state update finishes first

	reqs := make([]*message.Request, 8)
	for i := 0; i < len(reqs); i++ {
		reqs[i] = createPbftReq(int64(i), 0)
	}

	// Add four requests, with a batch size of 2
	b.reqStore.storeOutstanding(reqs[0])
	b.reqStore.storeOutstanding(reqs[1])
	b.reqStore.storeOutstanding(reqs[2])
	b.reqStore.storeOutstanding(reqs[3])

	executed := make(map[pbftTypes.MessageDigest]struct{})
	execute := func() {
		for d, reqBatch := range b.pbft.outstandingReqBatches {
			if _, ok := executed[d]; ok {
				continue
			}
			executed[d] = struct{}{}
			b.execute(b.pbft.lastExec+1, reqBatch)
		}
	}

	tmp := uint64(1)
	b.pbft.currentExec = &tmp
	events.SendEvent(b, committedEvent{})
	execute()

	if b.reqStore.outstandingRequests.Len() != 0 {
		t.Fatalf("All request batches should have been executed and deleted after exec")
	}

	// Simulate changing views, with a request in the qSet, and one outstanding which is not
	wreqsBatch := &message.RequestBatch{reqs[4]}
	prePrep := &message.PrePrepare{
		View:           0,
		SequenceNumber: b.pbft.lastExec + 1,
		BatchDigest:    "foo",
		RequestBatch:   wreqsBatch,
	}

	b.pbft.certStore[msgID{v: prePrep.View, n: prePrep.SequenceNumber}] = &msgCert{prePrepare: prePrep}

	// Add the request, which is already pre-prepared, to be outstanding, and one outstanding not pending, not prepared
	b.reqStore.storeOutstanding(reqs[4]) // req 6
	b.reqStore.storeOutstanding(reqs[5])
	b.reqStore.storeOutstanding(reqs[6])
	b.reqStore.storeOutstanding(reqs[7])

	events.SendEvent(b, viewChangedEvent{})
	execute()

	if b.reqStore.hasNonPending() {
		t.Errorf("All requests should have been resubmitted after view change")
	}

	// We should have one request in batch which has not been sent yet
	expected := 6
	if transactionsBroadcast != expected {
		t.Errorf("Expected %d transactions broadcast, got %d", expected, transactionsBroadcast)
	}

	events.SendEvent(b, batchTimerEvent{})
	execute()

	// If the already prepared request were to be resubmitted, we would get count 8 here
	expected = 7
	if transactionsBroadcast != expected {
		t.Errorf("Expected %d transactions broadcast, got %d", expected, transactionsBroadcast)
	}
}

func TestViewChangeOnPrimarySilence(t *testing.T) {
	log.InitLog(5)
	omni := *inertState
	omni.UnicastImpl = func(ocMsg *message.Message, peer *pbftTypes.PeerID) error { return nil } // For the checkpoint
	omni.SignImpl = func(msg []byte) ([]byte, error) { return msg, nil }
	omni.VerifyImpl = func(peerID pbftTypes.ReplicaID, signature []byte, message []byte) error { return nil }
	b := newObcBatch(1, loadConfig(), &omni)
	b.StateUpdated(&checkpointMessage{seqNo: 0, snapshotId: inertState.GetBlockchainInfoBlobImpl()}, inertState.GetBlockchainInfoImpl())
	b.pbft.requestTimeout = 50 * time.Millisecond
	defer b.Close()

	// Send a request, which will be ignored, triggering view change
	peerId := stringToPeerId("vp0")
	b.manager.Queue() <- batchMessageEvent{createTxMsg(1), &peerId}
	time.Sleep(time.Second)
	b.manager.Queue() <- nil

	if b.pbft.activeView {
		t.Fatalf("Should have caused a view change")
	}
}

func obcBatchSizeOneHelper(id pbftTypes.ReplicaID, config *params.Config, stack consensus_pbft.Stack) pbftConsumer {
	// It's not entirely obvious why the compiler likes the parent function, but not newObcClassic directly
	config.BatchSize = 1
	return newObcBatch(id, config, stack)
}

func TestClassicStateTransfer(t *testing.T) {
	log.InitLog(5)
	validatorCount := uint32(4)
	net := makeConsumerNetwork(validatorCount, obcBatchSizeOneHelper, func(ce *consumerEndpoint) {
		ce.consumer.(*obcBatch).pbft.K = 2
		ce.consumer.(*obcBatch).pbft.L = 4
	})
	defer net.stop()
	// net.debug = true

	filterMsg := true
	net.filterFn = func(src int, dst int, msg []byte) []byte {
		if filterMsg && dst == 3 { // 3 is byz
			return nil
		}
		return msg
	}

	// Advance the network one seqNo past so that Replica 3 will have to do statetransfer
	broadcaster := net.endpoints[generateBroadcaster(validatorCount)].getHandle()
	net.endpoints[1].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(1), broadcaster)
	net.process()

	// Move the seqNo to 9, at seqNo 6, Replica 3 will realize it's behind, transfer to seqNo 8, then execute seqNo 9
	filterMsg = false
	for n := 2; n <= 9; n++ {
		net.endpoints[1].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(int64(n)), broadcaster)
	}

	net.process()

	for _, ep := range net.endpoints {
		ce := ep.(*consumerEndpoint)
		obc := ce.consumer.(*obcBatch)
		_, err := obc.stack.GetState(9)
		if nil != err {
			t.Errorf("Replica %d executed requests, expected a new block on the chain, but could not retrieve it : %s", ce.id, err)
		}
		if !obc.pbft.activeView || obc.pbft.view != 0 {
			t.Errorf("Replica %d not active in view 0, is %v %d", ce.id, obc.pbft.activeView, obc.pbft.view)
		}
	}
}

func TestClassicBackToBackStateTransfer(t *testing.T) {
	log.InitLog(5)
	validatorCount := uint32(4)
	net := makeConsumerNetwork(validatorCount, obcBatchSizeOneHelper, func(ce *consumerEndpoint) {
		ce.consumer.(*obcBatch).pbft.K = 2
		ce.consumer.(*obcBatch).pbft.L = 4
		ce.consumer.(*obcBatch).pbft.requestTimeout = time.Hour // We do not want any view changes
	})
	defer net.stop()
	// net.debug = true

	filterMsg := true
	net.filterFn = func(src int, dst int, msg []byte) []byte {
		if filterMsg && dst == 3 { // 3 is byz
			return nil
		}
		return msg
	}

	// Get the group to advance past seqNo 1, leaving Replica 3 behind
	broadcaster := net.endpoints[generateBroadcaster(validatorCount)].getHandle()
	net.endpoints[1].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(1), broadcaster)
	net.process()

	// Now start including Replica 3, go to sequence number 10, Replica 3 will trigger state transfer
	// after seeing seqNo 8, then pass another target for seqNo 10 and 12, but transfer to 8, but the network
	// will have already moved on and be past to seqNo 13, outside of Replica 3's watermarks, but
	// Replica 3 will execute through seqNo 12
	filterMsg = false
	for n := 2; n <= 21; n++ {
		net.endpoints[1].(*consumerEndpoint).consumer.RecvMsg(createTxMsg(int64(n)), broadcaster)
	}

	net.process()

	for _, ep := range net.endpoints {
		ce := ep.(*consumerEndpoint)
		obc := ce.consumer.(*obcBatch)
		_, err := obc.stack.GetState(21)
		if nil != err {
			t.Errorf("Replica %d executed requests, expected a new block on the chain, but could not retrieve it : %s", ce.id, err)
		}
		if !obc.pbft.activeView || obc.pbft.view != 0 {
			t.Errorf("Replica %d not active in view 0, is %v %d", ce.id, obc.pbft.activeView, obc.pbft.view)
		}
	}
}

func TestClearBatchStoreOnViewChange(t *testing.T) {
	log.InitLog(5)
	omni := *inertState
	omni.UnicastImpl = func(ocMsg *message.Message, peer *pbftTypes.PeerID) error { return nil } // For the checkpoint
	b := newObcBatch(1, loadConfig(), &omni)
	b.StateUpdated(&checkpointMessage{seqNo: 0, snapshotId: inertState.GetBlockchainInfoBlobImpl()}, inertState.GetBlockchainInfoImpl())
	defer b.Close()

	b.batchStore = append(b.batchStore,&message.Request{})

	// Send a request, which will be ignored, triggering view change
	b.manager.Queue() <- viewChangedEvent{}
	b.manager.Queue() <- nil

	if len(b.batchStore) != 0 {
		t.Fatalf("Should have cleared the batch store on view change")
	}
}
