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
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"
	
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
)

// =============================================================================
// init
// =============================================================================

const (
	// UnreasonableTimeout is an ugly thing, we need to create timers, then stop them before they expire, so use a large timeout
	UnreasonableTimeout = 100 * time.Hour
)

// =============================================================================
// custom interfaces and structure definitions
// =============================================================================

// Event Types

// workEvent is a temporary type, to inject work
type workEvent func()

// viewChangeTimerEvent is sent when the view change timer expires
type viewChangeTimerEvent struct{}

// execDoneEvent is sent when an execution completes
type execDoneEvent struct{}

// pbftMessageEvent is sent when a consensus messages is received to be sent to pbft
type pbftMessageEvent pbftMessage

// viewChangedEvent is sent when the view change timer expires
type viewChangedEvent struct{}

// viewChangeResendTimerEvent is sent when the view change resend timer expires
type viewChangeResendTimerEvent struct{}

// returnRequestBatchEvent is sent by pbft when we are forwarded a request
type returnRequestBatchEvent *message.RequestBatch

// nullRequestEvent provides "keep-alive" null requests
type nullRequestEvent struct{}

// Unless otherwise noted, all methods consume the PBFT thread, and should therefore
// not rely on PBFT accomplishing any work while that thread is being held
type innerStack interface {
	broadcast(msgPayload []byte)
	unicast(msgPayload []byte, receiverID pbftTypes.ReplicaID) (err error)
	execute(seqNo uint64, reqBatch *message.RequestBatch) // This is invoked on a separate thread
	getState() []byte
	getLastSeqNo() (uint64, error)
	skipTo(seqNo uint64, snapshotID []byte, peers []pbftTypes.ReplicaID)

	sign(msg []byte) ([]byte, error)
	verify(senderID pbftTypes.ReplicaID, signature []byte, message []byte) error

	invalidateState()
	validateState()

	consensus_pbft.StatePersistor
}

// This structure is used for incoming PBFT bound messages
type pbftMessage struct {
	sender pbftTypes.ReplicaID
	msg    message.MessageInterface
}

type checkpointMessage struct {
	seqNo uint64
	snapshotId    []byte
}

type stateUpdateTarget struct {
	checkpointMessage
	replicas []pbftTypes.ReplicaID
}
/*
其中watermark的作用是：
我们想象一种情况，主节点是坏的，它在给请求编号时故意选择了一个很大的编号，以至于超出了序号的范围，
所以我们需要设置一个低水位（low water mark）h和高水位（high water mark）H，
让主节点分配的编号在h和H之间，不能肆意分配。
三阶段到此结束，客户端等待f+1个相同的副本结果作为最后结果。

当一个节点执行完某条请求后，可以广播一条消息，当全网有2f+1个节点都执行完这条请求后就可以删除它的日志了。
但是每条消息都进行广播来确认是否删除是低效的，所以可以k条消息放一起确认，每当k条请求执行完后，就广播一条消息，
当2f+1个节点都执行完这个请求后，就可以将这k条请求的日志删除了。

这就是建立检查点checkpoint，当节点i执行完k条请求后，就生成一个checkpoint，并广播checkpoint消息：<CHECKPOINT, n, d, i>，
n是最近一个影响状态的请求序号，d是状态的摘要。当有2f+1个检查点消息时，就证明这个检查点是正确的，形成一个stable checkpoint。
具有这个stable checkpoint的节点就可以将所有序号小于等于n的pre-prepare,prepare,commit消息，以及之前的检查点和检查点消息删除。
但是由于节点的执行速度不同，要使不同的节点的检查点维持在一个范围内，最快的几点与最慢的节点之间最多差L个检查点。
这里又用到了水线watermark，用检查点协议来更新水线的高低值（h和H），这两个高低值限定了可以被接受的消息。
水线的低值h与最近稳定检查点的序列号相同，而水线的高值H=h+L，L需要足够大才能使副本不至于为了等待稳定检查点而停顿。

 */
type pbftCore struct {
	// internal data
	internalLock sync.Mutex
	executing    bool // signals that application is executing

	idleChan   chan struct{} // Used to detect idleness for testing
	injectChan chan func()   // Used as a hack to inject work onto the PBFT thread, to be removed eventually

	consumer innerStack

	// PBFT data
	activeView    bool              // view change happening
	byzantine     bool              // whether this node is intentionally acting as Byzantine; useful for debugging on the testnet
	f             uint32               // max. number of faults we can tolerate
	N             uint32               // max.number of validators in the network
	h             uint64            // low watermark
	id            pbftTypes.ReplicaID            // replica ID; PBFT `i`
	K             uint64            // checkpoint period
	logMultiplier uint64            // use this value to calculate log size : k*logMultiplier
	L             uint64            // log size
	lastExec      uint64            // last request we executed
	replicaCount  int               // number of replicas; PBFT `|R|`
	seqNo         uint64            // PBFT "n", strictly monotonic increasing sequence number
	view          uint64            // current view
	chkpts        map[uint64]string // state checkpoints; map lastExec to global hash
	pset          map[uint64]*message.ViewChange_PQ
	qset          map[qidx]*message.ViewChange_PQ

	skipInProgress    bool               // Set when we have detected a fall behind scenario until we pick a new starting point
	stateTransferring bool               // Set when state transfer is executing
	highStateTarget   *stateUpdateTarget // Set to the highest weak checkpoint cert we have observed
	hChkpts           map[pbftTypes.ReplicaID]uint64  // highest checkpoint sequence number observed for each replica

	currentExec           *uint64                  // currently executing request
	timerActive           bool                     // is the timer running?
	vcResendTimer         events.Timer             // timer triggering resend of a view change
	newViewTimer          events.Timer             // timeout triggering a view change
	requestTimeout        time.Duration            // progress timeout for requests
	vcResendTimeout       time.Duration            // timeout before resending view change
	newViewTimeout        time.Duration            // progress timeout for new views
	newViewTimerReason    string                   // what triggered the timer
	lastNewViewTimeout    time.Duration            // last timeout we used during this view change
	broadcastTimeout      time.Duration            // progress timeout for broadcast
	outstandingReqBatches map[pbftTypes.MessageDigest]*message.RequestBatch // track whether we are waiting for request batches to execute

	nullRequestTimer   events.Timer  // timeout triggering a null request
	nullRequestTimeout time.Duration // duration for this timeout
	viewChangePeriod   uint64        // period between automatic view changes
	viewChangeSeqNo    uint64        // next seqNo to perform view change

	missingReqBatches map[pbftTypes.MessageDigest]bool // for all the assigned, non-checkpointed request batches we might be missing during view-change

	// implementation of PBFT `in`
	reqBatchStore   map[pbftTypes.MessageDigest]*message.RequestBatch // track request batches
	certStore       map[msgID]*msgCert       // track quorum certificates for requests
	checkpointStore map[message.Checkpoint]bool      // track checkpoints as set
	viewChangeStore map[vcidx]*message.ViewChange    // track view-change messages
	newViewStore    map[uint64]*message.NewView      // track last new-view we received or sent
}

type qidx struct {
	d pbftTypes.MessageDigest
	n uint64
}

type msgID struct { // our index through certStore
	v uint64
	n uint64
}

type msgCert struct {
	digest      pbftTypes.MessageDigest
	prePrepare  *message.PrePrepare
	sentPrepare bool
	prepare     []*message.Prepare
	sentCommit  bool
	commit      []*message.Commit
}

type vcidx struct {
	v  uint64
	id pbftTypes.ReplicaID
}

type sortableUint64Slice []uint64

func (a sortableUint64Slice) Len() int {
	return len(a)
}
func (a sortableUint64Slice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a sortableUint64Slice) Less(i, j int) bool {
	return a[i] < a[j]
}

// =============================================================================
// constructors
// =============================================================================

func newPbftCore(id pbftTypes.ReplicaID, config *params.Config, consumer innerStack, etf events.TimerFactory) *pbftCore {
	instance := &pbftCore{}
	instance.id = id
	instance.consumer = consumer
	//three timer

	//==========================================================================
	//newViewTimer对应于viewChangeTimerEvent{}，当这个timer在一定时间没有close时，
	//就会触发一个viewchange事件
	//==========================================================================
	instance.newViewTimer = etf.CreateTimer()
	//==========================================================================
	//vcResendTimer对应viewChangeResendTimerEvent，发出viewchange过时时会
	//触发一个将viewchange从新发送
	//==========================================================================
	instance.vcResendTimer = etf.CreateTimer()
	//==========================================================================
	//nullRequestTimer对应nullRequestEvent，如果主节点长期没有发送preprepare消息，
	//也就是分配了seq的reqBatch。它timeout就认为主节点挂掉了然后发送viewchange消息
	//==========================================================================
	instance.nullRequestTimer = etf.CreateTimer()

	instance.N = config.N//max consenter count
	instance.f = config.F//max faulty
	if instance.f*3+1 > instance.N {
		panic(fmt.Sprintf("need at least %d enough replicas to tolerate %d byzantine faults, but only %d replicas configured", instance.f*3+1, instance.f, instance.N))
	}

	instance.K = config.K//检查点时间段赋值

	//计算日志的大小值赋值,日志倍增器
	instance.logMultiplier = config.LogMultiplier
	if instance.logMultiplier < 2 {
		panic("Log multiplier must be greater than or equal to 2")
	}
	//日志大小计算
	instance.L = instance.logMultiplier * instance.K // log size
	//自动视图改变的时间段
	instance.viewChangePeriod = config.ViewChangePeriod

	//这个节点是否故意充当拜占庭;testnet用于调试
	instance.byzantine = config.Byzantine

	//请求过程超时
	instance.requestTimeout = config.Request
	//重发视图改变之前超时
	instance.vcResendTimeout = config.Resendviewchange
	//新的视图超时
	instance.newViewTimeout = config.Viewchange
	//超时持续
	instance.nullRequestTimeout = config.Nullrequest
	//广播过程超时
	instance.broadcastTimeout = config.Broadcast
	//查看view发生
	instance.activeView = true
	instance.replicaCount = int(instance.N)

	singletons.Log.Infof("PBFT type = %T", instance.consumer)
	singletons.Log.Infof("PBFT Max number of validating peers (N) = %v", instance.N)
	singletons.Log.Infof("PBFT Max number of failing peers (f) = %v", instance.f)
	singletons.Log.Infof("PBFT byzantine flag = %v", instance.byzantine)
	singletons.Log.Infof("PBFT request timeout = %v", instance.requestTimeout)
	singletons.Log.Infof("PBFT view change timeout = %v", instance.newViewTimeout)
	singletons.Log.Infof("PBFT Checkpoint period (K) = %v", instance.K)
	singletons.Log.Infof("PBFT broadcast timeout = %v", instance.broadcastTimeout)
	singletons.Log.Infof("PBFT Log multiplier = %v", instance.logMultiplier)
	singletons.Log.Infof("PBFT log size (L) = %v", instance.L)
	if instance.nullRequestTimeout > 0 {
		singletons.Log.Infof("PBFT null requests timeout = %v", instance.nullRequestTimeout)
	} else {
		singletons.Log.Infof("PBFT null requests disabled")
	}
	if instance.viewChangePeriod > 0 {
		singletons.Log.Infof("PBFT view change period = %v", instance.viewChangePeriod)
	} else {
		singletons.Log.Infof("PBFT automatic view change disabled")
	}

	// init the logs
	//跟踪法定证书请求
	instance.certStore = make(map[msgID]*msgCert)
	//跟踪请求批次
	instance.reqBatchStore = make(map[pbftTypes.MessageDigest]*message.RequestBatch)
	//跟踪检查点设置
	instance.checkpointStore = make(map[message.Checkpoint]bool)
	//检查点状态; 映射lastExec到全局hash
	instance.chkpts = make(map[uint64]string)
	//跟踪视view change消息
	instance.viewChangeStore = make(map[vcidx]*message.ViewChange)
	instance.pset = make(map[uint64]*message.ViewChange_PQ)
	instance.qset = make(map[qidx]*message.ViewChange_PQ)
	//跟踪我们接收后者发送的最后一个新视图
	instance.newViewStore = make(map[uint64]*message.NewView)

	// initialize state transfer
	//观察每一个replica最高薄弱点序列数
	instance.hChkpts = make(map[pbftTypes.ReplicaID]uint64)

	instance.chkpts[0] = "XXX GENESIS"
	// 在我们使用视图改变期间最后超时
	instance.lastNewViewTimeout = instance.newViewTimeout
	//跟踪我们是否正在等待请求批处理执行
	instance.outstandingReqBatches = make(map[pbftTypes.MessageDigest]*message.RequestBatch)
	//对于所有已经分配我们可能错过的在视图改变期间的非检查点的请求批次
	instance.missingReqBatches = make(map[pbftTypes.MessageDigest]bool)

	//将变量的值恢复到初始状态
	instance.restoreState()

	// 执行视图改变的下一个序号
	instance.viewChangeSeqNo = ^uint64(0) // infinity
	instance.updateViewChangeSeqNo()

	return instance
}

// close tears down resources opened by newPbftCore
func (instance *pbftCore) close() {
	instance.newViewTimer.Halt()
	instance.nullRequestTimer.Halt()
}

// allow the view-change protocol to kick-off when the timer expires
func (instance *pbftCore) ProcessEvent(e events.Event) events.Event {
	var err error
	singletons.Log.Debugf("Replica %d processing event", instance.id)
	switch et := e.(type) {
	case viewChangeTimerEvent:
		singletons.Log.Infof("Replica %d view change timer expired, sending view change: %s", instance.id, instance.newViewTimerReason)
		instance.timerActive = false
		instance.sendViewChange()
	case *pbftMessage:
		return pbftMessageEvent(*et)
	case pbftMessageEvent:
		msg := et
		singletons.Log.Debugf("Replica %d received incoming message from %v", instance.id, msg.sender)
		next, err := instance.recvMsg(msg.msg, msg.sender)
		if err != nil {
			break
		}
		return next
	case *message.RequestBatch:
		singletons.Log.Infof("Replica %d processing RequestBatch event", instance.id)
		err = instance.recvRequestBatch(et)
	case *message.PrePrepare:
		singletons.Log.Infof("Replica %d processing PrePrepare event", instance.id)
		err = instance.recvPrePrepare(et)
	case *message.Prepare:
		singletons.Log.Infof("Replica %d processing Prepare event", instance.id)
		err = instance.recvPrepare(et)
	case *message.Commit:
		singletons.Log.Infof("Replica %d processing Commit event", instance.id)
		err = instance.recvCommit(et)
	case *message.Checkpoint:
		singletons.Log.Infof("Replica %d processing Checkpoint event", instance.id)
		return instance.recvCheckpoint(et)
	case *message.ViewChange:
		singletons.Log.Infof("Replica %d processing ViewChange event", instance.id)
		return instance.recvViewChange(et)
	case *message.NewView:
		singletons.Log.Infof("Replica %d processing NewView event", instance.id)
		return instance.recvNewView(et)
	case *message.FetchRequestBatch:
		singletons.Log.Infof("Replica %d processing FetchRequestBatch event", instance.id)
		err = instance.recvFetchRequestBatch(et)
	case returnRequestBatchEvent:
		singletons.Log.Infof("Replica %d processing returnRequestBatchEvent event", instance.id)
		return instance.recvReturnRequestBatch(et)
	case stateUpdatedEvent:
		update := et.chkpt
		instance.stateTransferring = false
		// If state transfer did not complete successfully, or if it did not reach our low watermark, do it again
		if et.stateInfo == nil || update.seqNo < instance.h {
			if et.stateInfo == nil {
				singletons.Log.Warnf("Replica %d attempted state transfer target was not reachable (%v)", instance.id, et.chkpt)
			} else {
				singletons.Log.Warnf("Replica %d recovered to seqNo %d but our low watermark has moved to %d", instance.id, update.seqNo, instance.h)
			}
			if instance.highStateTarget == nil {
				singletons.Log.Debugf("Replica %d has no state targets, cannot resume state transfer yet", instance.id)
			} else if update.seqNo < instance.highStateTarget.seqNo {
				singletons.Log.Debugf("Replica %d has state target for %d, transferring", instance.id, instance.highStateTarget.seqNo)
				instance.retryStateTransfer(nil)
			} else {
				singletons.Log.Debugf("Replica %d has no state target above %d, highest is %d", instance.id, update.seqNo, instance.highStateTarget.seqNo)
			}
			return nil
		}
		singletons.Log.Infof("Replica %d application caught up via state transfer, lastExec now %d", instance.id, update.seqNo)
		instance.lastExec = update.seqNo
		instance.moveWatermarks(instance.lastExec) // The watermark movement handles moving this to a checkpoint boundary
		instance.skipInProgress = false
		instance.consumer.validateState()
		instance.Checkpoint(update.seqNo, update.snapshotId)
		instance.executeOutstanding()
	case execDoneEvent:
		instance.execDoneSync()
		if instance.skipInProgress {
			instance.retryStateTransfer(nil)
		}
		// We will delay new view processing sometimes
		return instance.processNewView()
	case nullRequestEvent:
		instance.nullRequestHandler()
	case workEvent:
		et() // Used to allow the caller to steal use of the main thread, to be removed
	case viewChangeQuorumEvent:
		singletons.Log.Debugf("Replica %d received view change quorum, processing new view", instance.id)
		if instance.primary(instance.view) == instance.id {
			return instance.sendNewView()
		}
		return instance.processNewView()
	case viewChangedEvent:
		// No-op, processed by plugins if needed
	case viewChangeResendTimerEvent:
		if instance.activeView {
			singletons.Log.Warnf("Replica %d had its view change resend timer expire but it's in an active view, this is benign but may indicate a bug", instance.id)
			return nil
		}
		singletons.Log.Debugf("Replica %d view change resend timer expired before view change quorum was reached, resending", instance.id)
		instance.view-- // sending the view change increments this
		return instance.sendViewChange()
	default:
		singletons.Log.Warnf("Replica %d received an unknown message type %T \n", instance.id, et)
	}

	if err != nil {
		singletons.Log.Warnf(err.Error())
	}

	return nil
}

// =============================================================================
// helper functions for PBFT
// =============================================================================

// Given a certain view n, what is the expected primary?
func (instance *pbftCore) primary(n uint64) pbftTypes.ReplicaID {
	return pbftTypes.ReplicaID(n % uint64(instance.replicaCount))
}

// Is the sequence number between watermarks?
func (instance *pbftCore) inW(n uint64) bool {
	return n-instance.h > 0 && n-instance.h <= instance.L
}

// Is the view right? And is the sequence number between watermarks?
func (instance *pbftCore) inWV(v uint64, n uint64) bool {
	return instance.view == v && instance.inW(n)
}

// Given a digest/view/seq, is there an entry in the certLog?
// If so, return it. If not, create it.
func (instance *pbftCore) getCert(v uint64, n uint64) (cert *msgCert) {
	idx := msgID{v, n}
	cert, ok := instance.certStore[idx]
	if ok {
		return
	}

	cert = &msgCert{}
	instance.certStore[idx] = cert
	return
}

// =============================================================================
// preprepare/prepare/commit quorum checks
// =============================================================================

// intersectionQuorum returns the number of replicas that have to
// agree to guarantee that at least one correct replica is shared by
// two intersection quora
func (instance *pbftCore) intersectionQuorum() uint32 {
	return (instance.N + instance.f + 2) / 2
}

// allCorrectReplicasQuorum returns the number of correct replicas (N-f)
func (instance *pbftCore) allCorrectReplicasQuorum() uint32 {
	return (instance.N - instance.f)
}

func (instance *pbftCore) prePrepared(digest pbftTypes.MessageDigest, v uint64, n uint64) bool {
	_, mInLog := instance.reqBatchStore[digest]

	if digest != "" && !mInLog {
		return false
	}

	if q, ok := instance.qset[qidx{digest, n}]; ok && q.View == v {
		return true
	}

	cert := instance.certStore[msgID{v, n}]
	if cert != nil {
		p := cert.prePrepare
		if p != nil && p.View == v && p.SequenceNumber == n && p.BatchDigest == digest {
			return true
		}
	}
	singletons.Log.Debugf("Replica %d does not have view=%d/seqNo=%d pre-prepared",
		instance.id, v, n)
	return false
}

func (instance *pbftCore) prepared(digest pbftTypes.MessageDigest, v uint64, n uint64) bool {
	if !instance.prePrepared(digest, v, n) {
		return false
	}

	if p, ok := instance.pset[n]; ok && p.View == v && p.BatchDigest == digest {
		return true
	}

	quorum := uint32(0)
	cert := instance.certStore[msgID{v, n}]
	if cert == nil {
		return false
	}

	for _, p := range cert.prepare {
		if p.View == v && p.SequenceNumber == n && p.BatchDigest == digest {
			quorum++
		}
	}

	singletons.Log.Debugf("Replica %d prepare count for view=%d/seqNo=%d: %d",
		instance.id, v, n, quorum)

	return quorum >= instance.intersectionQuorum()-1
}

func (instance *pbftCore) committed(digest pbftTypes.MessageDigest, v uint64, n uint64) bool {
	if !instance.prepared(digest, v, n) {
		return false
	}

	quorum := uint32(0)
	cert := instance.certStore[msgID{v, n}]
	if cert == nil {
		return false
	}

	for _, p := range cert.commit {
		if p.View == v && p.SequenceNumber == n {
			quorum++
		}
	}

	singletons.Log.Debugf("Replica %d commit count for view=%d/seqNo=%d: %d",
		instance.id, v, n, quorum)

	return quorum >= instance.intersectionQuorum()
}

// =============================================================================
// receive methods
// =============================================================================

func (instance *pbftCore) nullRequestHandler() {
	if !instance.activeView {
		return
	}

	if instance.primary(instance.view) != instance.id {
		// backup expected a null request, but primary never sent one
		singletons.Log.Infof("Replica %d null request timer expired, sending view change", instance.id)
		instance.sendViewChange()
	} else {
		// time for the primary to send a null request
		// pre-prepare with null digest
		singletons.Log.Infof("Primary %d null request timer expired, sending null request", instance.id)
		instance.sendPrePrepare(nil, "")
	}
}

func (instance *pbftCore) recvMsg(msg message.MessageInterface, senderID pbftTypes.ReplicaID) (interface{}, error) {
	// todo: check senderId
	switch msg.(type) {
	case *message.RequestBatch:
		return msg,nil
	case *message.ReturnRequestBatch:
		return msg,nil
	default:
		if msg.Sender() != senderID{
			return nil, fmt.Errorf("Sender ID included in message (%v) doesn't match ID corresponding to the receiving stream (%v)", msg.Sender(), senderID)
		}
	}
	return msg,nil
    /*
	if reqBatch := msg.GetRequestBatch(); reqBatch != nil {
		return reqBatch, nil
	} else if preprep := msg.GetPrePrepare(); preprep != nil {
		if senderID != preprep.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in pre-prepare message (%v) doesn't match ID corresponding to the receiving stream (%v)", preprep.ReplicaId, senderID)
		}
		return preprep, nil
	} else if prep := msg.GetPrepare(); prep != nil {
		if senderID != prep.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in prepare message (%v) doesn't match ID corresponding to the receiving stream (%v)", prep.ReplicaId, senderID)
		}
		return prep, nil
	} else if commit := msg.GetCommit(); commit != nil {
		if senderID != commit.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in commit message (%v) doesn't match ID corresponding to the receiving stream (%v)", commit.ReplicaId, senderID)
		}
		return commit, nil
	} else if chkpt := msg.GetCheckpoint(); chkpt != nil {
		if senderID != chkpt.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in checkpoint message (%v) doesn't match ID corresponding to the receiving stream (%v)", chkpt.ReplicaId, senderID)
		}
		return chkpt, nil
	} else if vc := msg.GetViewChange(); vc != nil {
		if senderID != vc.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in view-change message (%v) doesn't match ID corresponding to the receiving stream (%v)", vc.ReplicaId, senderID)
		}
		return vc, nil
	} else if nv := msg.GetNewView(); nv != nil {
		if senderID != nv.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in new-view message (%v) doesn't match ID corresponding to the receiving stream (%v)", nv.ReplicaId, senderID)
		}
		return nv, nil
	} else if fr := msg.GetFetchRequestBatch(); fr != nil {
		if senderID != fr.ReplicaId {
			return nil, fmt.Errorf("Sender ID included in fetch-request-batch message (%v) doesn't match ID corresponding to the receiving stream (%v)", fr.ReplicaId, senderID)
		}
		return fr, nil
	} else if reqBatch := msg.GetReturnRequestBatch(); reqBatch != nil {
		// it's ok for sender ID and replica ID to differ; we're sending the original request message
		return returnRequestBatchEvent(reqBatch), nil
	}
	return nil, fmt.Errorf("Invalid message: %v", msg)
	*/
}

func (instance *pbftCore) recvRequestBatch(reqBatch *message.RequestBatch) error {
	digest := message.Digest(reqBatch)
	singletons.Log.Debugf("Replica %d received request batch %s", instance.id, digest)

	instance.reqBatchStore[digest] = reqBatch
	instance.outstandingReqBatches[digest] = reqBatch
	instance.persistRequestBatch(digest)
	if instance.activeView {
		instance.softStartTimer(instance.requestTimeout, fmt.Sprintf("new request batch %s", digest))
	}
	if instance.primary(instance.view) == instance.id && instance.activeView {
		instance.nullRequestTimer.Stop()
		instance.sendPrePrepare(reqBatch, digest)
	} else {
		singletons.Log.Debugf("Replica %d is backup, not sending pre-prepare for request batch %s", instance.id, digest)
	}
	return nil
}

func (instance *pbftCore) sendPrePrepare(reqBatch *message.RequestBatch, digest pbftTypes.MessageDigest) {
	singletons.Log.Debugf("Replica %d is primary, issuing pre-prepare for request batch %s", instance.id, digest)

	n := instance.seqNo + 1
	for _, cert := range instance.certStore { // check for other PRE-PREPARE for same digest, but different seqNo
		if p := cert.prePrepare; p != nil {
			if p.View == instance.view && p.SequenceNumber != n && p.BatchDigest == digest && digest != "" {
				singletons.Log.Infof("Other pre-prepare found with same digest but different seqNo: %d instead of %d", p.SequenceNumber, n)
				return
			}
		}
	}

	if !instance.inWV(instance.view, n) || n > instance.h+instance.L/2 {
		// We don't have the necessary stable certificates to advance our watermarks
		singletons.Log.Warnf("Primary %d not sending pre-prepare for batch %s - out of sequence numbers", instance.id, digest)
		return
	}

	if n > instance.viewChangeSeqNo {
		singletons.Log.Infof("Primary %d about to switch to next primary, not sending pre-prepare with seqno=%d", instance.id, n)
		return
	}

	singletons.Log.Debugf("Primary %d broadcasting pre-prepare for view=%d/seqNo=%d and digest %s", instance.id, instance.view, n, digest)
	instance.seqNo = n
	preprep := &message.PrePrepare{
		View:           instance.view,
		SequenceNumber: n,
		BatchDigest:    digest,
		RequestBatch:   reqBatch,
		ReplicaId:      instance.id,
	}
	cert := instance.getCert(instance.view, n)
	cert.prePrepare = preprep
	cert.digest = digest
	instance.persistQSet()
	instance.innerBroadcast(preprep)
	instance.maybeSendCommit(digest, instance.view, n)
}

func (instance *pbftCore) resubmitRequestBatches() {
	if instance.primary(instance.view) != instance.id {
		return
	}

	var submissionOrder []*message.RequestBatch

outer:
	for d, reqBatch := range instance.outstandingReqBatches {
		for _, cert := range instance.certStore {
			if cert.digest == d {
				singletons.Log.Debugf("Replica %d already has certificate for request batch %s - not going to resubmit", instance.id, d)
				continue outer
			}
		}
		singletons.Log.Debugf("Replica %d has detected request batch %s must be resubmitted", instance.id, d)
		submissionOrder = append(submissionOrder, reqBatch)
	}

	if len(submissionOrder) == 0 {
		return
	}

	for _, reqBatch := range submissionOrder {
		// This is a request batch that has not been pre-prepared yet
		// Trigger request batch processing again
		instance.recvRequestBatch(reqBatch)
	}
}

func (instance *pbftCore) recvPrePrepare(preprep *message.PrePrepare) error {
	singletons.Log.Debugf("Replica %d received pre-prepare from replica %d for view=%d/seqNo=%d",
		instance.id, preprep.ReplicaId, preprep.View, preprep.SequenceNumber)

	if !instance.activeView {
		singletons.Log.Debugf("Replica %d ignoring pre-prepare as we are in a view change", instance.id)
		return nil
	}

	if instance.primary(instance.view) != preprep.ReplicaId {
		singletons.Log.Warnf("Pre-prepare from other than primary: got %d, should be %d", preprep.ReplicaId, instance.primary(instance.view))
		return nil
	}

	if !instance.inWV(preprep.View, preprep.SequenceNumber) {
		if preprep.SequenceNumber != instance.h && !instance.skipInProgress {
			singletons.Log.Warnf("Replica %d pre-prepare view different, or sequence number outside watermarks: preprep.View %d, expected.View %d, seqNo %d, low-mark %d", instance.id, preprep.View, instance.primary(instance.view), preprep.SequenceNumber, instance.h)
		} else {
			// This is perfectly normal
			singletons.Log.Debugf("Replica %d pre-prepare view different, or sequence number outside watermarks: preprep.View %d, expected.View %d, seqNo %d, low-mark %d", instance.id, preprep.View, instance.primary(instance.view), preprep.SequenceNumber, instance.h)
		}

		return nil
	}

	if preprep.SequenceNumber > instance.viewChangeSeqNo {
		singletons.Log.Infof("Replica %d received pre-prepare for %d, which should be from the next primary", instance.id, preprep.SequenceNumber)
		instance.sendViewChange()
		return nil
	}

	cert := instance.getCert(preprep.View, preprep.SequenceNumber)
	if cert.digest != "" && cert.digest != preprep.BatchDigest {
		singletons.Log.Warnf("Pre-prepare found for same view/seqNo but different digest: received %s, stored %s", preprep.BatchDigest, cert.digest)
		instance.sendViewChange()
		return nil
	}

	cert.prePrepare = preprep
	cert.digest = preprep.BatchDigest

	// Store the request batch if, for whatever reason, we haven't received it from an earlier broadcast
	if _, ok := instance.reqBatchStore[preprep.BatchDigest]; !ok && preprep.BatchDigest != "" {
		digest := message.Digest(preprep.RequestBatch)
		if digest != preprep.BatchDigest {
			singletons.Log.Warnf("Pre-prepare and request digest do not match: request %s, digest %s", digest, preprep.BatchDigest)
			return nil
		}
		instance.reqBatchStore[digest] = preprep.RequestBatch
		singletons.Log.Debugf("Replica %d storing request batch %s in outstanding request batch store", instance.id, digest)
		instance.outstandingReqBatches[digest] = preprep.RequestBatch
		instance.persistRequestBatch(digest)
	}

	instance.softStartTimer(instance.requestTimeout, fmt.Sprintf("new pre-prepare for request batch %s", preprep.BatchDigest))
	instance.nullRequestTimer.Stop()

	if instance.primary(instance.view) != instance.id && instance.prePrepared(preprep.BatchDigest, preprep.View, preprep.SequenceNumber) && !cert.sentPrepare {
		singletons.Log.Debugf("Backup %d broadcasting prepare for view=%d/seqNo=%d", instance.id, preprep.View, preprep.SequenceNumber)
		prep := &message.Prepare{
			View:           preprep.View,
			SequenceNumber: preprep.SequenceNumber,
			BatchDigest:    preprep.BatchDigest,
			ReplicaId:      instance.id,
		}
		cert.sentPrepare = true
		instance.persistQSet()
		instance.recvPrepare(prep)
		return instance.innerBroadcast(prep)
	}

	return nil
}

func (instance *pbftCore) recvPrepare(prep *message.Prepare) error {
	singletons.Log.Debugf("Replica %d received prepare from replica %d for view=%d/seqNo=%d",
		instance.id, prep.ReplicaId, prep.View, prep.SequenceNumber)

	if instance.primary(prep.View) == prep.ReplicaId {
		singletons.Log.Warnf("Replica %d received prepare from primary, ignoring", instance.id)
		return nil
	}

	if !instance.inWV(prep.View, prep.SequenceNumber) {
		if prep.SequenceNumber != instance.h && !instance.skipInProgress {
			singletons.Log.Warnf("Replica %d ignoring prepare for view=%d/seqNo=%d: not in-wv, in view %d, low water mark %d", instance.id, prep.View, prep.SequenceNumber, instance.view, instance.h)
		} else {
			// This is perfectly normal
			singletons.Log.Debugf("Replica %d ignoring prepare for view=%d/seqNo=%d: not in-wv, in view %d, low water mark %d", instance.id, prep.View, prep.SequenceNumber, instance.view, instance.h)
		}
		return nil
	}

	cert := instance.getCert(prep.View, prep.SequenceNumber)

	for _, prevPrep := range cert.prepare {
		if prevPrep.ReplicaId == prep.ReplicaId {
			singletons.Log.Warnf("Ignoring duplicate prepare from %d", prep.ReplicaId)
			return nil
		}
	}
	cert.prepare = append(cert.prepare, prep)
	instance.persistPSet()

	return instance.maybeSendCommit(prep.BatchDigest, prep.View, prep.SequenceNumber)
}

//
func (instance *pbftCore) maybeSendCommit(digest pbftTypes.MessageDigest, v uint64, n uint64) error {
	cert := instance.getCert(v, n)
	if instance.prepared(digest, v, n) && !cert.sentCommit {
		singletons.Log.Debugf("Replica %d broadcasting commit for view=%d/seqNo=%d",
			instance.id, v, n)
		commit := &message.Commit{
			View:           v,
			SequenceNumber: n,
			BatchDigest:    digest,
			ReplicaId:      instance.id,
		}
		cert.sentCommit = true
		instance.recvCommit(commit)
		return instance.innerBroadcast(commit)
	}
	return nil
}

func (instance *pbftCore) recvCommit(commit *message.Commit) error {
	singletons.Log.Debugf("Replica %d received commit from replica %d for view=%d/seqNo=%d",
		instance.id, commit.ReplicaId, commit.View, commit.SequenceNumber)

	if !instance.inWV(commit.View, commit.SequenceNumber) {
		if commit.SequenceNumber != instance.h && !instance.skipInProgress {
			singletons.Log.Warnf("Replica %d ignoring commit for view=%d/seqNo=%d: not in-wv, in view %d, low water mark %d", instance.id, commit.View, commit.SequenceNumber, instance.view, instance.h)
		} else {
			// This is perfectly normal
			singletons.Log.Debugf("Replica %d ignoring commit for view=%d/seqNo=%d: not in-wv, in view %d, low water mark %d", instance.id, commit.View, commit.SequenceNumber, instance.view, instance.h)
		}
		return nil
	}

	cert := instance.getCert(commit.View, commit.SequenceNumber)
	for _, prevCommit := range cert.commit {
		if prevCommit.ReplicaId == commit.ReplicaId {
			singletons.Log.Warnf("Ignoring duplicate commit from %d", commit.ReplicaId)
			return nil
		}
	}
	cert.commit = append(cert.commit, commit)

	if instance.committed(commit.BatchDigest, commit.View, commit.SequenceNumber) {
		instance.stopTimer()
		instance.lastNewViewTimeout = instance.newViewTimeout
		delete(instance.outstandingReqBatches, commit.BatchDigest)

		instance.executeOutstanding()

		if commit.SequenceNumber == instance.viewChangeSeqNo {
			singletons.Log.Infof("Replica %d cycling view for seqNo=%d", instance.id, commit.SequenceNumber)
			instance.sendViewChange()
		}
	}

	return nil
}

func (instance *pbftCore) updateHighStateTarget(target *stateUpdateTarget) {
	if instance.highStateTarget != nil && instance.highStateTarget.seqNo >= target.seqNo {
		singletons.Log.Debugf("Replica %d not updating state target to seqNo %d, has target for seqNo %d", instance.id, target.seqNo, instance.highStateTarget.seqNo)
		return
	}

	instance.highStateTarget = target
}

func (instance *pbftCore) stateTransfer(optional *stateUpdateTarget) {
	if !instance.skipInProgress {
		singletons.Log.Debugf("Replica %d is out of sync, pending state transfer", instance.id)
		instance.skipInProgress = true
		instance.consumer.invalidateState()
	}

	instance.retryStateTransfer(optional)
}

func (instance *pbftCore) retryStateTransfer(optional *stateUpdateTarget) {
	if instance.currentExec != nil {
		singletons.Log.Debugf("Replica %d is currently mid-execution, it must wait for the execution to complete before performing state transfer", instance.id)
		return
	}

	if instance.stateTransferring {
		singletons.Log.Debugf("Replica %d is currently mid state transfer, it must wait for this state transfer to complete before initiating a new one", instance.id)
		return
	}

	target := optional
	if target == nil {
		if instance.highStateTarget == nil {
			singletons.Log.Debugf("Replica %d has no targets to attempt state transfer to, delaying", instance.id)
			return
		}
		target = instance.highStateTarget
	}

	instance.stateTransferring = true

	singletons.Log.Debugf("Replica %d is initiating state transfer to seqNo %d", instance.id, target.seqNo)
	instance.consumer.skipTo(target.seqNo, target.snapshotId, target.replicas)

}

func (instance *pbftCore) executeOutstanding() {
	if instance.currentExec != nil {
		singletons.Log.Debugf("Replica %d not attempting to executeOutstanding because it is currently executing %d", instance.id, *instance.currentExec)
		return
	}
	singletons.Log.Debugf("Replica %d attempting to executeOutstanding", instance.id)

	for idx := range instance.certStore {
		if instance.executeOne(idx) {
			break
		}
	}

	singletons.Log.Debugf("Replica %d certstore %+v", instance.id, instance.certStore)

	instance.startTimerIfOutstandingRequests()
}

func (instance *pbftCore) executeOne(idx msgID) bool {
	cert := instance.certStore[idx]

	if idx.n != instance.lastExec+1 || cert == nil || cert.prePrepare == nil {
		return false
	}

	if instance.skipInProgress {
		singletons.Log.Debugf("Replica %d currently picking a starting point to resume, will not execute", instance.id)
		return false
	}

	// we now have the right sequence number that doesn't create holes

	digest := cert.digest
	reqBatch := instance.reqBatchStore[digest]

	if !instance.committed(digest, idx.v, idx.n) {
		return false
	}

	// we have a commit certificate for this request batch
	currentExec := idx.n
	instance.currentExec = &currentExec

	// null request
	if digest == "" {
		singletons.Log.Infof("Replica %d executing/committing null request for view=%d/seqNo=%d",
			instance.id, idx.v, idx.n)
		instance.execDoneSync()
	} else {
		singletons.Log.Infof("Replica %d executing/committing request batch for view=%d/seqNo=%d and digest %s",
			instance.id, idx.v, idx.n, digest)
		// synchronously execute, it is the other side's responsibility to execute in the background if needed
		instance.consumer.execute(idx.n, reqBatch)
	}
	return true
}

func (instance *pbftCore) Checkpoint(seqNo uint64, id []byte) {
	if seqNo%instance.K != 0 {
		singletons.Log.Errorf("Attempted to checkpoint a sequence number (%d) which is not a multiple of the checkpoint interval (%d)", seqNo, instance.K)
		return
	}

	idAsString := base64.StdEncoding.EncodeToString(id)

	singletons.Log.Debugf("Replica %d preparing checkpoint for view=%d/seqNo=%d and b64 id of %s",
		instance.id, instance.view, seqNo, idAsString)

	chkpt := &message.Checkpoint{
		SequenceNumber: seqNo,
		ReplicaId:      instance.id,
		Id:             idAsString,
	}
	instance.chkpts[seqNo] = idAsString

	instance.persistCheckpoint(seqNo, id)
	instance.recvCheckpoint(chkpt)
	instance.innerBroadcast(chkpt)
}

func (instance *pbftCore) execDoneSync() {
	if instance.currentExec != nil {
		singletons.Log.Infof("Replica %d finished execution %d, trying next", instance.id, *instance.currentExec)
		instance.lastExec = *instance.currentExec
		if instance.lastExec%instance.K == 0 {
			instance.Checkpoint(instance.lastExec, instance.consumer.getState())
		}

	} else {
		// XXX This masks a bug, this should not be called when currentExec is nil
		singletons.Log.Warnf("Replica %d had execDoneSync called, flagging ourselves as out of date", instance.id)
		instance.skipInProgress = true
	}
	instance.currentExec = nil

	instance.executeOutstanding()
}

func (instance *pbftCore) moveWatermarks(n uint64) {
	// round down n to previous low watermark
	h := n / instance.K * instance.K

	for idx, cert := range instance.certStore {
		if idx.n <= h {
			singletons.Log.Debugf("Replica %d cleaning quorum certificate for view=%d/seqNo=%d",
				instance.id, idx.v, idx.n)
			instance.persistDelRequestBatch(cert.digest)
			delete(instance.reqBatchStore, cert.digest)
			delete(instance.certStore, idx)
		}
	}

	for testChkpt := range instance.checkpointStore {
		if testChkpt.SequenceNumber <= h {
			singletons.Log.Debugf("Replica %d cleaning checkpoint message from replica %d, seqNo %d, b64 snapshot id %s",
				instance.id, testChkpt.ReplicaId, testChkpt.SequenceNumber, testChkpt.Id)
			delete(instance.checkpointStore, testChkpt)
		}
	}

	for n := range instance.pset {
		if n <= h {
			delete(instance.pset, n)
		}
	}

	for idx := range instance.qset {
		if idx.n <= h {
			delete(instance.qset, idx)
		}
	}

	for n := range instance.chkpts {
		if n < h {
			delete(instance.chkpts, n)
			instance.persistDelCheckpoint(n)
		}
	}

	instance.h = h

	singletons.Log.Debugf("Replica %d updated low watermark to %d",
		instance.id, instance.h)

	instance.resubmitRequestBatches()
}

func (instance *pbftCore) weakCheckpointSetOutOfRange(chkpt *message.Checkpoint) bool {
	H := instance.h + instance.L

	// Track the last observed checkpoint sequence number if it exceeds our high watermark, keyed by replica to prevent unbounded growth
	if chkpt.SequenceNumber < H {
		// For non-byzantine nodes, the checkpoint sequence number increases monotonically
		delete(instance.hChkpts, chkpt.ReplicaId)
	} else {
		// We do not track the highest one, as a byzantine node could pick an arbitrarilly high sequence number
		// and even if it recovered to be non-byzantine, we would still believe it to be far ahead
		instance.hChkpts[chkpt.ReplicaId] = chkpt.SequenceNumber

		// If f+1 other replicas have reported checkpoints that were (at one time) outside our watermarks
		// we need to check to see if we have fallen behind.
		if len(instance.hChkpts) >= int(instance.f)+1 {
			chkptSeqNumArray := make([]uint64, len(instance.hChkpts))
			index := 0
			for replicaID, hChkpt := range instance.hChkpts {
				chkptSeqNumArray[index] = hChkpt
				index++
				if hChkpt < H {
					delete(instance.hChkpts, replicaID)
				}
			}
			sort.Sort(sortableUint64Slice(chkptSeqNumArray))

			// If f+1 nodes have issued checkpoints above our high water mark, then
			// we will never record 2f+1 checkpoints for that sequence number, we are out of date
			// (This is because all_replicas - missed - me = 3f+1 - f - 1 = 2f)
			if m := chkptSeqNumArray[len(chkptSeqNumArray)-(int(instance.f)+1)]; m > H {
				singletons.Log.Warnf("Replica %d is out of date, f+1 nodes agree checkpoint with seqNo %d exists but our high water mark is %d", instance.id, chkpt.SequenceNumber, H)
				instance.reqBatchStore = make(map[pbftTypes.MessageDigest]*message.RequestBatch) // Discard all our requests, as we will never know which were executed, to be addressed in #394
				instance.persistDelAllRequestBatches()
				instance.moveWatermarks(m)
				instance.outstandingReqBatches = make(map[pbftTypes.MessageDigest]*message.RequestBatch)
				instance.skipInProgress = true
				instance.consumer.invalidateState()
				instance.stopTimer()

				// TODO, reprocess the already gathered checkpoints, this will make recovery faster, though it is presently correct

				return true
			}
		}
	}

	return false
}

func (instance *pbftCore) witnessCheckpointWeakCert(chkpt *message.Checkpoint) {
	checkpointMembers := make([]pbftTypes.ReplicaID, instance.f+1) // Only ever invoked for the first weak cert, so guaranteed to be f+1
	i := 0
	for testChkpt := range instance.checkpointStore {
		if testChkpt.SequenceNumber == chkpt.SequenceNumber && testChkpt.Id == chkpt.Id {
			checkpointMembers[i] = testChkpt.ReplicaId
			singletons.Log.Debugf("Replica %d adding replica %d (handle %v) to weak cert", instance.id, testChkpt.ReplicaId, checkpointMembers[i])
			i++
		}
	}

	snapshotID, err := base64.StdEncoding.DecodeString(chkpt.Id)
	if nil != err {
		err = fmt.Errorf("Replica %d received a weak checkpoint cert which could not be decoded (%s)", instance.id, chkpt.Id)
		singletons.Log.Error(err.Error())
		return
	}

	target := &stateUpdateTarget{
		checkpointMessage: checkpointMessage{
			seqNo: chkpt.SequenceNumber,
			snapshotId:    snapshotID,
		},
		replicas: checkpointMembers,
	}
	instance.updateHighStateTarget(target)

	if instance.skipInProgress {
		singletons.Log.Debugf("Replica %d is catching up and witnessed a weak certificate for checkpoint %d, weak cert attested to by %d of %d (%v)",
			instance.id, chkpt.SequenceNumber, i, instance.replicaCount, checkpointMembers)
		// The view should not be set to active, this should be handled by the yet unimplemented SUSPECT, see https://github.com/hyperledger/fabric/issues/1120
		instance.retryStateTransfer(target)
	}
}

func (instance *pbftCore) recvCheckpoint(chkpt *message.Checkpoint) events.Event {
	singletons.Log.Debugf("Replica %d received checkpoint from replica %d, seqNo %d, digest %s",
		instance.id, chkpt.ReplicaId, chkpt.SequenceNumber, chkpt.Id)

	if instance.weakCheckpointSetOutOfRange(chkpt) {
		return nil
	}

	if !instance.inW(chkpt.SequenceNumber) {
		if chkpt.SequenceNumber != instance.h && !instance.skipInProgress {
			// It is perfectly normal that we receive checkpoints for the watermark we just raised, as we raise it after 2f+1, leaving f replies left
			singletons.Log.Warnf("Checkpoint sequence number outside watermarks: seqNo %d, low-mark %d", chkpt.SequenceNumber, instance.h)
		} else {
			singletons.Log.Debugf("Checkpoint sequence number outside watermarks: seqNo %d, low-mark %d", chkpt.SequenceNumber, instance.h)
		}
		return nil
	}

	instance.checkpointStore[*chkpt] = true

	// Track how many different checkpoint values we have for the seqNo in question
	diffValues := make(map[string]struct{})
	diffValues[chkpt.Id] = struct{}{}

	matching := uint32(0)
	for testChkpt := range instance.checkpointStore {
		if testChkpt.SequenceNumber == chkpt.SequenceNumber {
			if testChkpt.Id == chkpt.Id {
				matching++
			} else {
				if _, ok := diffValues[testChkpt.Id]; !ok {
					diffValues[testChkpt.Id] = struct{}{}
				}
			}
		}
	}
	singletons.Log.Debugf("Replica %d found %d matching checkpoints for seqNo %d, digest %s",
		instance.id, matching, chkpt.SequenceNumber, chkpt.Id)

	// If f+2 different values have been observed, we'll never be able to get a stable cert for this seqNo
	if count := len(diffValues); count > int(instance.f)+1 {
		panic("Network unable to find stable certificate for seqNo %d (%d different values observed already)")
	}

	if matching == instance.f+1 {
		// We have a weak cert
		// If we have generated a checkpoint for this seqNo, make sure we have a match
		if ownChkptID, ok := instance.chkpts[chkpt.SequenceNumber]; ok {
			if ownChkptID != chkpt.Id {
				panic("Own checkpoint for seqNo %d (%s) different from weak checkpoint certificate (%s)")
			}
		}
		instance.witnessCheckpointWeakCert(chkpt)
	}

	if matching < instance.intersectionQuorum() {
		// We do not have a quorum yet
		return nil
	}

	// It is actually just fine if we do not have this checkpoint
	// and should not trigger a state transfer
	// Imagine we are executing sequence number k-1 and we are slow for some reason
	// then everyone else finishes executing k, and we receive a checkpoint quorum
	// which we will agree with very shortly, but do not move our watermarks until
	// we have reached this checkpoint
	// Note, this is not divergent from the paper, as the paper requires that
	// the quorum certificate must contain 2f+1 messages, including its own
	if _, ok := instance.chkpts[chkpt.SequenceNumber]; !ok {
		singletons.Log.Debugf("Replica %d found checkpoint quorum for seqNo %d, digest %s, but it has not reached this checkpoint itself yet",
			instance.id, chkpt.SequenceNumber, chkpt.Id)
		if instance.skipInProgress {
			logSafetyBound := instance.h + instance.L/2
			// As an optimization, if we are more than half way out of our log and in state transfer, move our watermarks so we don't lose track of the network
			// if needed, state transfer will restart on completion to a more recent point in time
			if chkpt.SequenceNumber >= logSafetyBound {
				singletons.Log.Debugf("Replica %d is in state transfer, but, the network seems to be moving on past %d, moving our watermarks to stay with it", instance.id, logSafetyBound)
				instance.moveWatermarks(chkpt.SequenceNumber)
			}
		}
		return nil
	}

	singletons.Log.Debugf("Replica %d found checkpoint quorum for seqNo %d, digest %s",
		instance.id, chkpt.SequenceNumber, chkpt.Id)

	instance.moveWatermarks(chkpt.SequenceNumber)

	return instance.processNewView()
}

// used in view-change to fetch missing assigned, non-checkpointed requests
func (instance *pbftCore) fetchRequestBatches() (err error) {
	for digest := range instance.missingReqBatches {
		msg := &message.FetchRequestBatch{
			BatchDigest: digest,
			ReplicaId:   instance.id,
		}

		instance.innerBroadcast(msg)
	}

	return
}

func (instance *pbftCore) recvFetchRequestBatch(fr *message.FetchRequestBatch) (err error) {
	digest := fr.BatchDigest
	if _, ok := instance.reqBatchStore[digest]; !ok {
		return nil // we don't have it either
	}

	reqBatch := instance.reqBatchStore[digest]
	msg := message.ReturnRequestBatch(*reqBatch)
	msgPacked, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("Error marshalling return-request-batch message: %v", err)
	}

	receiver := fr.ReplicaId
	err = instance.consumer.unicast(msgPacked, receiver)

	return
}

func (instance *pbftCore) recvReturnRequestBatch(reqBatch *message.RequestBatch) events.Event {
	digest := message.Digest(reqBatch)
	if _, ok := instance.missingReqBatches[digest]; !ok {
		return nil // either the wrong digest, or we got it already from someone else
	}
	instance.reqBatchStore[digest] = reqBatch
	delete(instance.missingReqBatches, digest)
	instance.persistRequestBatch(digest)
	return instance.processNewView()
}

// =============================================================================
// Misc. methods go here
// =============================================================================

// Marshals a Message and hands it to the Stack. If toSelf is true,
// the message is also dispatched to the local instance's RecvMsgSync.
func (instance *pbftCore) innerBroadcast(msg message.MessageInterface) error {
	msgRaw, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("Cannot marshal message %s", err)
	}

	doByzantine := false
	if instance.byzantine {
		rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))
		doIt := rand1.Intn(3) // go byzantine about 1/3 of the time
		if doIt == 1 {
			doByzantine = true
		}
	}

	// testing byzantine fault.
	if doByzantine {
		rand2 := rand.New(rand.NewSource(time.Now().UnixNano()))
		ignoreidx := rand2.Intn(int(instance.N))
		for i := 0; i < int(instance.N); i++ {
			if i != ignoreidx && pbftTypes.ReplicaID(i) != instance.id { //Pick a random replica and do not send message
				instance.consumer.unicast(msgRaw, pbftTypes.ReplicaID(i))
			} else {
				singletons.Log.Debugf("PBFT byzantine: not broadcasting to replica %v", i)
			}
		}
	} else {
		instance.consumer.broadcast(msgRaw)
	}
	return nil
}

func (instance *pbftCore) updateViewChangeSeqNo() {
	if instance.viewChangePeriod <= 0 {
		return
	}
	// Ensure the view change always occurs at a checkpoint boundary
	instance.viewChangeSeqNo = instance.seqNo + instance.viewChangePeriod*instance.K - instance.seqNo%instance.K
	singletons.Log.Debugf("Replica %d updating view change sequence number to %d", instance.id, instance.viewChangeSeqNo)
}

func (instance *pbftCore) startTimerIfOutstandingRequests() {
	if instance.skipInProgress || instance.currentExec != nil {
		// Do not start the view change timer if we are executing or state transferring, these take arbitrarilly long amounts of time
		return
	}

	if len(instance.outstandingReqBatches) > 0 {
		getOutstandingDigests := func() []pbftTypes.MessageDigest {
			var digests []pbftTypes.MessageDigest
			for digest := range instance.outstandingReqBatches {
				digests = append(digests, digest)
			}
			return digests
		}()
		instance.softStartTimer(instance.requestTimeout, fmt.Sprintf("outstanding request batches %v", getOutstandingDigests))
	} else if instance.nullRequestTimeout > 0 {
		timeout := instance.nullRequestTimeout
		if instance.primary(instance.view) != instance.id {
			// we're waiting for the primary to deliver a null request - give it a bit more time
			timeout += instance.requestTimeout
		}
		instance.nullRequestTimer.Reset(timeout, nullRequestEvent{})
	}
}

func (instance *pbftCore) softStartTimer(timeout time.Duration, reason string) {
	singletons.Log.Debugf("Replica %d soft starting new view timer for %s: %s", instance.id, timeout, reason)
	instance.newViewTimerReason = reason
	instance.timerActive = true
	instance.newViewTimer.SoftReset(timeout, viewChangeTimerEvent{})
}

func (instance *pbftCore) startTimer(timeout time.Duration, reason string) {
	singletons.Log.Debugf("Replica %d starting new view timer for %s: %s", instance.id, timeout, reason)
	instance.timerActive = true
	instance.newViewTimer.Reset(timeout, viewChangeTimerEvent{})
}

func (instance *pbftCore) stopTimer() {
	singletons.Log.Debugf("Replica %d stopping a running new view timer", instance.id)
	instance.timerActive = false
	instance.newViewTimer.Stop()
}
