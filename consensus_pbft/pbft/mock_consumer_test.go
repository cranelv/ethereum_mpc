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
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"

	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"fmt"
	"testing"
)
var HanoiTower [3][]int
func HanoiMove(from,to,num int) bool{
	len0 := len(HanoiTower[from])
	if(num == 1){
		value := HanoiTower[from][len0-1]
		HanoiTower[from] = HanoiTower[from][:len0-1]
		len1 := len(HanoiTower[to])
		if len1 >0 && HanoiTower[to][len1-1] < value{
			fmt.Printf("tower %d is error: %d < %d",to,HanoiTower[to][len1-1],value)
			return false
		}
		HanoiTower[to]= append(HanoiTower[to],value)
	}else{
		other := (from+to) ^ 3
		HanoiMove(from,other,num-1)
		HanoiMove(from,to,1)
		HanoiMove(other,to,num-1)
	}
	return true
}
func TestHanoiTower(t *testing.T)  {
	for i:=100;i>0;i--{
		HanoiTower[0] = append(HanoiTower[0],i)
	}
	HanoiMove(0,2,100)
	t.Log(HanoiTower[2])
}
type consumerEndpoint struct {
	*testEndpoint
	consumer     pbftConsumer
	execTasksResult func([]*message.Task) ([]*message.Result, error)
}

func (ce *consumerEndpoint) stop() {
	ce.consumer.Close()
}

func (ce *consumerEndpoint) isBusy() bool {
	pbft := ce.consumer.getPBFTCore()
	if pbft.timerActive || pbft.skipInProgress || pbft.currentExec != nil {
		ce.net.debugMsg("Reporting busy because of timer (%v) or skipInProgress (%v) or currentExec (%v)\n", pbft.timerActive, pbft.skipInProgress, pbft.currentExec)
		return true
	}

	select {
	case <-ce.consumer.idleChannel():
	default:
		ce.net.debugMsg("Reporting busy because consumer not idle\n")
		return true
	}

	select {
	case ce.consumer.getManager().Queue() <- nil:
		ce.net.debugMsg("Reporting busy because pbft not idle\n")
	default:
		return true
	}
	//}

	return false
}

func (ce *consumerEndpoint) deliver(msg []byte, senderHandle *pbftTypes.PeerID) {
	ce.consumer.RecvMsg(&message.Message{Type: message.Message_CONSENSUS, Payload: msg}, senderHandle)
}

type completeStack struct {
	*consumerEndpoint
	*noopSecurity
	*MockLedger
	mockPersist
	skipTarget chan struct{}
}

const MaxStateTransferTime int = 200

func (cs *completeStack) ValidateState()   {}
func (cs *completeStack) InvalidateState() {}
func (cs *completeStack) Start()           {}
func (cs *completeStack) Halt()            {}

func (cs *completeStack) UpdateState(tag interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID) {
	select {
	// This guarantees the first SkipTo call is the one that's queued, whereas a mutex can be raced for
	case cs.skipTarget <- struct{}{}:
		go func() {
			// State transfer takes time, not simulating this hides bugs
			time.Sleep(time.Duration((MaxStateTransferTime/2)+rand.Intn(MaxStateTransferTime/2)) * time.Millisecond)
			cs.simulateStateTransfer(target, peers)
			cs.consumer.StateUpdated(tag, cs.GetStateInfo())
			<-cs.skipTarget // Basically like releasing a mutex
		}()
	default:
		cs.net.debugMsg("Ignoring skipTo because one is already in progress\n")
	}
}

type pbftConsumer interface {
	innerStack
	consensus_pbft.Consenter
	getPBFTCore() *pbftCore
	getManager() events.Manager // TODO, remove, this is a temporary measure
	Close()
	idleChannel() <-chan struct{}
}

type consumerNetwork struct {
	*testnet
	identity consensusInterface.ValidatorIdentifyInterface
	mockLedgers []*MockLedger
}

func (cnet *consumerNetwork) GetLedgerByPeerID(peerID *pbftTypes.PeerID) (consensus_pbft.ReadOnlyLedger, bool) {
	id, err := cnet.identity.GetValidatorID(peerID)
	if nil != err {
		return nil, false
	}
	return cnet.mockLedgers[int(id)], true
}

func makeConsumerNetwork(N uint32, makeConsumer func(id pbftTypes.ReplicaID, config *params.Config, stack consensus_pbft.Stack) pbftConsumer, initFNs ...func(*consumerEndpoint)) *consumerNetwork {
	twl := consumerNetwork{mockLedgers: make([]*MockLedger, N)}

	endpointFunc := func(id pbftTypes.ReplicaID, net *testnet) endpoint {
		tep := makeTestEndpoint(id, net)
		ce := &consumerEndpoint{
			testEndpoint: tep,
		}

		ml := NewMockLedger(&twl)
		ml.ce = ce
		twl.mockLedgers[id] = ml

		cs := &completeStack{
			consumerEndpoint: ce,
			noopSecurity:     &noopSecurity{},
			MockLedger:       ml,
			skipTarget:       make(chan struct{}, 1),
		}

		ce.consumer = makeConsumer(id, loadConfig(), cs)
		ce.consumer.getPBFTCore().N = N
		ce.consumer.getPBFTCore().f = (N - 1) / 3

		for _, fn := range initFNs {
			fn(ce)
		}

		return ce
	}

	twl.testnet = makeTestnet(N, endpointFunc)
	return &twl
}
