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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
)

type mockMsg struct {
	msg  *message.Message
	dest *pbftTypes.PeerID
}

type mockComm struct {
	self  pbftTypes.ReplicaID
	n     uint64
	msgCh chan mockMsg
}

func (m *mockComm) Unicast(msg *message.Message, dest *pbftTypes.PeerID) error {
	m.msgCh <- mockMsg{msg, dest}
	return nil
}

func (m *mockComm) Broadcast(msg *message.Message, t pbftTypes.Peer_Type) error {
	return nil
}

func (m *mockComm) GetNetworkNodes() (pbftTypes.Peer, []pbftTypes.Peer, error) {
	return nil, nil, nil
}

func (m *mockComm) GetNetworkNodeIDs() (*pbftTypes.PeerID, []*pbftTypes.PeerID, error) {
	var h []*pbftTypes.PeerID
	for n := uint64(0); n < m.n; n++ {
		peerId := stringToPeerId(fmt.Sprintf("vp%d", n))
		h = append(h, &peerId)
	}
	return h[m.self], h, nil
}
func newIdentify(m* mockComm) consensusInterface.ValidatorIdentifyInterface {
	identify := &PbftIdentify{}
	_,peers,_ := m.GetNetworkNodeIDs()
	for i,peerId := range peers{
		peer := PbftPeer{&discover.Node{ID:discover.NodeID(*peerId)},pbftTypes.Peer_VALIDATOR}
		identify.validatorInfo = append(identify.validatorInfo, PbftInfo{pbftTypes.ReplicaID(i),&peer})
	}
	return identify
}
func TestBroadcast(t *testing.T) {
	log.InitLog(5)
	m := &mockComm{
		self:  1,
		n:     4,
		msgCh: make(chan mockMsg, 4),
	}
	sent := make(map[pbftTypes.PeerID]int)
	go func() {
		for msg := range m.msgCh {
			sent[*msg.dest]++
		}
	}()
	identify := newIdentify(m)
	b := newBroadcaster(m.self, 4, 1, time.Second,identify, m)

	msg := &message.Message{Payload: []byte("hi")}
	b.Broadcast(msg)
	time.Sleep(100 * time.Millisecond)
	b.Close()

	sentCount := 0
	for _, q := range sent {
		if q == 1 {
			sentCount++
		}
	}

	if sentCount < 2 {
		t.Errorf("broadcast did not send to all peers: %v", sent)
	}
}

type mockStuckComm struct {
	mockComm
	done chan struct{}
}

func (m *mockStuckComm) Unicast(msg *message.Message, dest *pbftTypes.PeerID) error {
	ret := m.mockComm.Unicast(msg, dest)
	if *dest == stringToPeerId("vp0") {
		select {
		case <-time.After(2 * time.Second):
			return fmt.Errorf("timeout")
		case <-m.done:
			return fmt.Errorf("closed")
		}
	}
	return ret
}

func TestBroadcastStuck(t *testing.T) {
	log.InitLog(5 )
	m := &mockStuckComm{
		mockComm: mockComm{
			self:  1,
			n:     4,
			msgCh: make(chan mockMsg),
		},
		done: make(chan struct{}),
	}
	sent := make(map[string][]*pbftTypes.PeerID)
	go func() {
		for msg := range m.msgCh {
			key := string(msg.msg.Payload)
			singletons.Log.Info(key)
			sent[key] = append(sent[key], msg.dest)
		}
	}()
	identify := newIdentify(&m.mockComm)
	b := newBroadcaster(m.self, 4, 1, time.Second,identify, m)

	maxc := 20
	for c := 0; c < maxc; c++ {
		b.Broadcast(&message.Message{Payload: []byte(fmt.Sprintf("%d", c))})
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Fatal("blocked")
		}
	}()
	time.Sleep(100 * time.Millisecond)
	close(m.done)
	b.Close()
	close(done)

	sendDone := 0
	for _, q := range sent {
		if len(q) >= 2 {
			sendDone++
		}
	}
	if sendDone != maxc {
		t.Errorf("expected %d sent messages: %v", maxc, sent)
	}
}

func TestBroadcastUnicast(t *testing.T) {
	m := &mockComm{
		self:  1,
		n:     4,
		msgCh: make(chan mockMsg, 4),
	}
	sent := make(map[pbftTypes.PeerID]int)
	go func() {
		for msg := range m.msgCh {
			sent[*msg.dest]++
		}
	}()
	identify := newIdentify(m)
	b := newBroadcaster(m.self, 4, 1, time.Second,identify, m)

	msg := &message.Message{Payload: []byte("hi")}
	b.Unicast(msg, 0)
	time.Sleep(100 * time.Millisecond)
	b.Close()

	sentCount := 0
	for _, q := range sent {
		if q == 1 {
			sentCount++
		}
	}

	if sentCount != 1 {
		t.Errorf("broadcast did not send to dest peer: %v", sent)
	}
}

type mockFailComm struct {
	mockComm
	done chan struct{}
}

func (m *mockFailComm) Unicast(msg *message.Message, dest *pbftTypes.PeerID) error {
	return fmt.Errorf("always fails on purpose")
}

func TestBroadcastAllFail(t *testing.T) {
	m := &mockFailComm{
		mockComm: mockComm{
			self:  1,
			n:     4,
			msgCh: make(chan mockMsg),
		},
		done: make(chan struct{}),
	}
	identify := newIdentify(&m.mockComm)
	b := newBroadcaster(m.self, 4, 1, time.Second,identify, m)

	maxc := 20
	for c := 0; c < maxc; c++ {
		b.Broadcast(&message.Message{Payload: []byte(fmt.Sprintf("%d", c))})
	}

	done := make(chan struct{})
	go func() {
		close(m.done)
		b.Close() // If the broadcasts are still trying (despite all the failures), this call blocks until the timeout
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Fatal("Could not successfully close broadcaster, after 1 second")
	}
}

func TestBroadcastTimeout(t *testing.T) {
	log.InitLog(5)
	expectTime := 10 * time.Second
	deltaTime := 50 * time.Millisecond
	m := &mockIndefinitelyStuckComm{
		mockComm: mockComm{
			self:  1,
			n:     4,
			msgCh: make(chan mockMsg),
		},
		done: make(chan struct{}),
	}
	identify := newIdentify(&m.mockComm)
	b := newBroadcaster(m.self, 4, 1, expectTime,identify, m)
	broadcastDone := make(chan time.Time)

	beginTime := time.Now()
	go func() {
		b.Broadcast(&message.Message{Payload: []byte(fmt.Sprintf("%d", 1))})
		broadcastDone <- time.Now()
	}()

	checkTime := expectTime + deltaTime
	select {
	case endTime := <-broadcastDone:
		t.Log("Broadcast consume time: ", endTime.Sub(beginTime))
		close(broadcastDone)
		close(m.done)
		return
	case <-time.After(checkTime):
		close(broadcastDone)
		close(m.done)
		t.Fatalf("Broadcast timeout after %v, expected %v", checkTime, expectTime)
	}
}

type mockIndefinitelyStuckComm struct {
	mockComm
	done chan struct{}
}

func (m *mockIndefinitelyStuckComm) Unicast(msg *message.Message, dest *pbftTypes.PeerID) error {
	if *dest == stringToPeerId("vp0") {
		<-m.done
	}
	return fmt.Errorf("Always failing, on purpose, with vp0 stuck")
}

func TestBroadcastIndefinitelyStuck(t *testing.T) {
	m := &mockIndefinitelyStuckComm{
		mockComm: mockComm{
			self:  1,
			n:     4,
			msgCh: make(chan mockMsg),
		},
		done: make(chan struct{}),
	}
	identify := newIdentify(&m.mockComm)
	b := newBroadcaster(m.self, 4, 1, time.Second,identify, m)

	broadcastDone := make(chan struct{})

	go func() {
		maxc := 3
		for c := 0; c < maxc; c++ {
			b.Broadcast(&message.Message{Payload: []byte(fmt.Sprintf("%d", c))})
		}
		close(broadcastDone)
	}()

	select {
	case <-broadcastDone:
		// Success
	case <-time.After(10 * time.Second):
		t.Errorf("Got blocked for too long")
	}

	close(m.done)
	b.Close()
}
