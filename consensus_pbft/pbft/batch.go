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
	"time"
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
)

type obcBatch struct {
	obcGeneric
	externalEventReceiver
	pbft        *pbftCore
	broadcaster *broadcaster

	batchSize        int
	batchStore       []*message.Request
	batchTimer       events.Timer
	batchTimerActive bool
	batchTimeout     time.Duration

	manager events.Manager // TODO, remove eventually, the event manager

	incomingChan chan *batchMessage // Queues messages for processing by main thread
	idleChan     chan struct{}      // Idle channel, to be removed

	reqStore *requestStore // Holds the outstanding and pending requests

	deduplicator *deduplicator

	persistForward
}

type batchMessage struct {
	msg    *message.Message
	sender *pbftTypes.PeerID
}

// Event types

// batchMessageEvent is sent when a consensus message is received that is then to be sent to pbft
type batchMessageEvent batchMessage

// batchTimerEvent is sent when the batch timer expires
type batchTimerEvent struct{}
//==============================================================================
// 在newobcbatch时，会初始化得到一个pbftcore的一个实例，这个是算法的核心模块。
// 并此时会启动一个batchTimer（这个batchTimer是一个计时器，
// 当batchTimer timeout后会触发一个sendbatch操作，这个只有primary节点才会去做）。
// 当然此时会创建一个事件处理机制，这个事件处理机制是各个模块沟通的一个bridge。
// ==============================================================================
//用方法newObcBatch初始化一个obcbatch对象。这个batch对象的作用就是用来做request缓存，
//提高transaction的执行效率，如果每来一个请求就去做一次共识，那代价会很高。缓存存储在batchStore里。

func newObcBatch(id pbftTypes.ReplicaID, config *params.Config,
	stack consensus_pbft.Stack) *obcBatch {

	op := &obcBatch{
		obcGeneric: obcGeneric{stack: stack},
	}
	op.NetValidator.net = stack
	op.NetValidator.indexer = op
	op.persistForward.persistor = stack

	singletons.Log.Debugf("Replica %d obtaining startup information", id)

	op.manager = events.NewManagerImpl() // TODO, this is hacky, eventually rip it out
	op.manager.SetReceiver(op)
	etf := events.NewTimerFactoryImpl(op.manager)
	op.pbft = newPbftCore(id, config, op, etf)
	op.manager.Start()
	blockchainInfoBlob := stack.GetBlockchainInfoBlob()
	op.externalEventReceiver.manager = op.manager
	op.broadcaster = newBroadcaster(id, op.pbft.N, op.pbft.f, op.pbft.broadcastTimeout,op, stack)
	op.manager.Queue() <- workEvent(func() {
		op.pbft.stateTransfer(&stateUpdateTarget{
			checkpointMessage: checkpointMessage{
				seqNo: op.pbft.lastExec,
				snapshotId:    blockchainInfoBlob,
			},
		})
	})

	op.batchSize = config.BatchSize
	op.batchStore = nil
	op.batchTimeout = config.BatchTimeout

	singletons.Log.Infof("PBFT Batch size = %d", op.batchSize)
	singletons.Log.Infof("PBFT Batch timeout = %v", op.batchTimeout)

	if op.batchTimeout >= op.pbft.requestTimeout {
		op.pbft.requestTimeout = 3 * op.batchTimeout / 2
		singletons.Log.Warnf("Configured request timeout must be greater than batch timeout, setting to %v", op.pbft.requestTimeout)
	}

	if op.pbft.requestTimeout >= op.pbft.nullRequestTimeout && op.pbft.nullRequestTimeout != 0 {
		op.pbft.nullRequestTimeout = 3 * op.pbft.requestTimeout / 2
		singletons.Log.Warnf("Configured null request timeout must be greater than request timeout, setting to %v", op.pbft.nullRequestTimeout)
	}

	op.incomingChan = make(chan *batchMessage)

	op.batchTimer = etf.CreateTimer()

	op.reqStore = newRequestStore()

	op.deduplicator = newDeduplicator()

	op.idleChan = make(chan struct{})
	close(op.idleChan) // TODO remove eventually

	return op
}

// Close tells us to release resources we are holding
func (op *obcBatch) Close() {
	op.batchTimer.Halt()
	op.pbft.close()
}

func (op *obcBatch) submitToLeader(req *message.Request) events.Event {
	// Broadcast the request to the network, in case we're in the wrong view
	op.broadcastMsg(req)
	op.logAddTaskFromRequest(req)
	op.reqStore.storeOutstanding(req)
	op.startTimerIfOutstandingRequests()
	if op.pbft.primary(op.pbft.view) == op.pbft.id && op.pbft.activeView {
		return op.leaderProcReq(req)
	}
	return nil
}

func (op *obcBatch) broadcastMsg(msg message.MessageInterface) {
	msgPayload, _ := msg.Marshal()
	ocMsg := op.wrapMessage(msgPayload)
	op.broadcaster.Broadcast(ocMsg)
}

// send a message to a specific replica
func (op *obcBatch) unicastMsg(msg message.MessageInterface, receiverID pbftTypes.ReplicaID) {
	msgPayload, _ := msg.Marshal()
	ocMsg := op.wrapMessage(msgPayload)
	op.broadcaster.Unicast(ocMsg, receiverID)
}

// =============================================================================
// innerStack interface (functions called by pbft-core)
// =============================================================================

// multicast a message to all replicas
func (op *obcBatch) broadcast(msgPayload []byte) {
	op.broadcaster.Broadcast(op.wrapMessage(msgPayload))
}

// send a message to a specific replica
func (op *obcBatch) unicast(msgPayload []byte, receiverID pbftTypes.ReplicaID) (err error) {
	return op.broadcaster.Unicast(op.wrapMessage(msgPayload), receiverID)
}

func (op *obcBatch) sign(msg []byte) ([]byte, error) {
	return op.stack.Sign(msg)
}

// verify message signature
func (op *obcBatch) verify(senderID pbftTypes.ReplicaID, signature []byte, message []byte) error {
	return op.stack.Verify(senderID, signature, message)
}

// execute an opaque request which corresponds to an OBC Transaction
func (op *obcBatch) execute(seqNo uint64, reqBatch *message.RequestBatch) {
	var tasks []*message.Task
	for _, msg := range *reqBatch {
		if outstanding, pending := op.reqStore.remove(msg); !outstanding || !pending {
			singletons.Log.Debugf("Batch replica %d missing transactions outstanding=%v, pending=%v", op.pbft.id, outstanding, pending)
		}
		tasks = append(tasks,msg.Tasks...)
		op.deduplicator.Execute(msg)
	}
	meta, _ := singletons.Marshaler.Marshal(seqNo)
	singletons.Log.Debugf("Batch replica %d received exec for seqNo %d containing %d transactions", op.pbft.id, seqNo, len(tasks))
	op.stack.Execute(meta, tasks) // This executes in the background, we will receive an executedEvent once it completes
}

// =============================================================================
// functions specific to batch mode
// =============================================================================

func (op *obcBatch) leaderProcReq(req *message.Request) events.Event {
	// XXX check req sig
	digest := message.Digest(req)
	singletons.Log.Debugf("Batch primary %d queueing new request %s", op.pbft.id, string(digest))
	op.batchStore = append(op.batchStore, req)
	op.reqStore.storePending(req)

	if !op.batchTimerActive {
		op.startBatchTimer()
	}

	if len(op.batchStore) >= op.batchSize {
		return op.sendBatch()
	}

	return nil
}

func (op *obcBatch) sendBatch() events.Event {
	op.stopBatchTimer()
	if len(op.batchStore) == 0 {
		singletons.Log.Error("Told to send an empty batch store for ordering, ignoring")
		return nil
	}

	reqBatch := message.RequestBatch(op.batchStore)
	op.batchStore = nil
	singletons.Log.Infof("Creating batch with %d requests", len(reqBatch))
	return &reqBatch
}

func (op *obcBatch) taskToReq(tasks []*message.Task) *message.Request {
	now := time.Now()
	req := &message.Request{
		Timestamp: uint64(now.UnixNano()),
		ReplicaId: op.pbft.id,
		Tasks:   tasks,
	}
	// XXX sign req
	return req
}

func (op *obcBatch) processMessage(ocMsg *message.Message, senderID *pbftTypes.PeerID) events.Event {
	if ocMsg.Type != message.Message_CHAIN_TASKS && ocMsg.Type != message.Message_CONSENSUS {
		singletons.Log.Errorf("Unexpected message type: %s", ocMsg.Type)
		return nil
	}
	msg,err := ocMsg.Unmarshal()
	if err != nil {
		singletons.Log.Errorf("Error unmarshaling message: %s", err)
		return nil
	}
	if ocMsg.Type == message.Message_CHAIN_TASKS {
		req := msg.(*message.Request)
		return op.submitToLeader(req)
	}

	if ocMsg.Type != message.Message_CONSENSUS {
		singletons.Log.Errorf("Unexpected message type: %s", ocMsg.Type)
		return nil
	}



	if req,ok := msg.(*message.Request); ok {
		if !op.deduplicator.IsNew(req) {
			singletons.Log.Warnf("Replica %d ignoring request as it is too old", op.pbft.id)
			return nil
		}

		op.logAddTaskFromRequest(req)
		op.reqStore.storeOutstanding(req)
		if (op.pbft.primary(op.pbft.view) == op.pbft.id) && op.pbft.activeView {
			return op.leaderProcReq(req)
		}
		op.startTimerIfOutstandingRequests()
		return nil
	} else {
		replicaId,_ := op.GetValidatorID(senderID)
		return pbftMessageEvent{
			msg:    msg,
			sender: replicaId,
		}
	}

	singletons.Log.Errorf("Unknown request: %+v", msg)

	return nil
}

func (op *obcBatch) logAddTaskFromRequest(req *message.Request) {
	// This is potentially a very large expensive debug statement, guard
	task := req.Tasks
	singletons.Log.Debugf("Replica %d adding request from %d with task %d into outstandingReqs", op.pbft.id, req.ReplicaId, task[0].Type)
}

func (op *obcBatch) resubmitOutstandingReqs() events.Event {
	op.startTimerIfOutstandingRequests()

	// If we are the primary, and know of outstanding requests, submit them for inclusion in the next batch until
	// we run out of requests, or a new batch message is triggered (this path will re-enter after execution)
	// Do not enter while an execution is in progress to prevent duplicating a request
	if op.pbft.primary(op.pbft.view) == op.pbft.id && op.pbft.activeView && op.pbft.currentExec == nil {
		needed := op.batchSize - len(op.batchStore)

		for op.reqStore.hasNonPending() {
			outstanding := op.reqStore.getNextNonPending(needed)

			// If we have enough outstanding requests, this will trigger a batch
			for _, nreq := range outstanding {
				if msg := op.leaderProcReq(nreq); msg != nil {
					op.manager.Inject(msg)
				}
			}
		}
	}
	return nil
}

// allow the primary to send a batch when the timer expires
func (op *obcBatch) ProcessEvent(event events.Event) events.Event {
	singletons.Log.Debugf("Replica %d batch main thread looping", op.pbft.id)
	switch et := event.(type) {
	case batchMessageEvent:
		singletons.Log.Debugf("batchMessageEvent : Replica %d batch main thread ", op.pbft.id)
		ocMsg := et
		return op.processMessage(ocMsg.msg, ocMsg.sender)
	case executedEvent:
		singletons.Log.Debugf("executedEvent : Replica %d batch main thread ", op.pbft.id)
		op.stack.Commit(nil, et.tag.([]byte))
	case committedEvent:
		singletons.Log.Debugf("committedEvent : Replica %d batch main thread ", op.pbft.id)
		return execDoneEvent{}
	case execDoneEvent:
		singletons.Log.Debugf("execDoneEvent : Replica %d batch main thread ", op.pbft.id)
		if res := op.pbft.ProcessEvent(event); res != nil {
			// This may trigger a view change, if so, process it, we will resubmit on new view
			return res
		}
		return op.resubmitOutstandingReqs()
	case batchTimerEvent:
		singletons.Log.Debugf("batchTimerEvent : Replica %d batch main thread ", op.pbft.id)
		if op.pbft.activeView && (len(op.batchStore) > 0) {
			return op.sendBatch()
		}
	case *message.Commit:
		// TODO, this is extremely hacky, but should go away when batch and core are merged
		singletons.Log.Debugf("Commit : Replica %d batch main thread ", op.pbft.id)
		res := op.pbft.ProcessEvent(event)
		op.startTimerIfOutstandingRequests()
		return res
	case viewChangedEvent:
		singletons.Log.Debugf("viewChangedEvent : Replica %d batch main thread ", op.pbft.id)
		op.batchStore = nil
		// Outstanding reqs doesn't make sense for batch, as all the requests in a batch may be processed
		// in a different batch, but PBFT core can't see through the opaque structure to see this
		// so, on view change, clear it out
		op.pbft.outstandingReqBatches = make(map[pbftTypes.MessageDigest]*message.RequestBatch)

		singletons.Log.Debugf("Replica %d batch thread recognizing new view", op.pbft.id)
		if op.batchTimerActive {
			op.stopBatchTimer()
		}

		if op.pbft.skipInProgress {
			// If we're the new primary, but we're in state transfer, we can't trust ourself not to duplicate things
			op.reqStore.outstandingRequests.empty()
		}

		op.reqStore.pendingRequests.empty()
		for i := op.pbft.h + 1; i <= op.pbft.h+op.pbft.L; i++ {
			if i <= op.pbft.lastExec {
				continue
			}

			cert, ok := op.pbft.certStore[msgID{v: op.pbft.view, n: i}]
			if !ok || cert.prePrepare == nil {
				continue
			}

			if cert.prePrepare.BatchDigest == "" {
				// a null request
				continue
			}

			if cert.prePrepare.RequestBatch == nil {
				singletons.Log.Warnf("Replica %d found a non-null prePrepare with no request batch, ignoring")
				continue
			}

			op.reqStore.storePendings(cert.prePrepare.RequestBatch)
		}

		return op.resubmitOutstandingReqs()
	case stateUpdatedEvent:
		singletons.Log.Debugf("stateUpdatedEvent : Replica %d batch main thread ", op.pbft.id)
		// When the state is updated, clear any outstanding requests, they may have been processed while we were gone
		op.reqStore = newRequestStore()
		return op.pbft.ProcessEvent(event)
	default:
		singletons.Log.Debugf("default : Replica %d batch main thread ", op.pbft.id)
		return op.pbft.ProcessEvent(event)
	}

	return nil
}

func (op *obcBatch) startBatchTimer() {
	op.batchTimer.Reset(op.batchTimeout, batchTimerEvent{})
	singletons.Log.Debugf("Replica %d started the batch timer", op.pbft.id)
	op.batchTimerActive = true
}

func (op *obcBatch) stopBatchTimer() {
	op.batchTimer.Stop()
	singletons.Log.Debugf("Replica %d stopped the batch timer", op.pbft.id)
	op.batchTimerActive = false
}

// Wraps a payload into a batch message, packs it and wraps it into
// a Fabric message. Called by broadcast before transmission.
func (op *obcBatch) wrapMessage(msgPayload []byte) *message.Message{
	ocMsg := &message.Message{
		Type:    message.Message_CONSENSUS,
		Timestamp: uint64(time.Now().Nanosecond()),
		Payload: msgPayload,
	}
	self,_,_ := op.net.GetNetworkNodeIDs()
	if self != nil {
		ocMsg.Sender = *self
	}
	return ocMsg
}

// Retrieve the idle channel, only used for testing
func (op *obcBatch) idleChannel() <-chan struct{} {
	return op.idleChan
}

// TODO, temporary
func (op *obcBatch) getManager() events.Manager {
	return op.manager
}

func (op *obcBatch) startTimerIfOutstandingRequests() {
	if op.pbft.skipInProgress || op.pbft.currentExec != nil || !op.pbft.activeView {
		// Do not start view change timer if some background event is in progress
		singletons.Log.Debugf("Replica %d not starting timer because skip in progress or current exec or in view change", op.pbft.id)
		return
	}

	if !op.reqStore.hasNonPending() {
		// Only start a timer if we are aware of outstanding requests
		singletons.Log.Debugf("Replica %d not starting timer because all outstanding requests are pending", op.pbft.id)
		return
	}
	op.pbft.softStartTimer(op.pbft.requestTimeout, "Batch outstanding requests")
}
