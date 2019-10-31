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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
)


type broadcaster struct {
	comm consensus_pbft.NetworkStack
	identify consensusInterface.ValidatorIdentifyInterface
	f                uint32
	broadcastTimeout time.Duration
	msgChans         map[pbftTypes.ReplicaID]chan *sendRequest
	closed           sync.WaitGroup
	closedCh         chan struct{}
}

type sendRequest struct {
	msg  *message.Message
	done chan bool
}

func newBroadcaster(self pbftTypes.ReplicaID, N uint32, f uint32, broadcastTimeout time.Duration,
	identify consensusInterface.ValidatorIdentifyInterface, c consensus_pbft.NetworkStack) *broadcaster {
	queueSize := 10 // XXX increase after testing

	chans := make(map[pbftTypes.ReplicaID]chan *sendRequest)
	b := &broadcaster{
		comm:             c,
		identify: 		  identify,
		f:                f,
		broadcastTimeout: broadcastTimeout,
		msgChans:         chans,
		closedCh:         make(chan struct{}),
	}
	for i := uint32(0); i < N; i++ {
		if pbftTypes.ReplicaID(i) == self {
			continue
		}
		chans[pbftTypes.ReplicaID(i)] = make(chan *sendRequest, queueSize)
	}

	// We do not start the go routines in the above loop to avoid concurrent map read/writes
	for i := uint32(0); i < N; i++ {
		if pbftTypes.ReplicaID(i) == self {
			continue
		}
		go b.drainer(pbftTypes.ReplicaID(i))
	}

	return b
}

func (b *broadcaster) Close() {
	close(b.closedCh)
	b.closed.Wait()
}

func (b *broadcaster) Wait() {
	b.closed.Wait()
}

func (b *broadcaster) drainerSend(dest pbftTypes.ReplicaID, send *sendRequest, successLastTime bool) bool {
	// Note, successLastTime is purely used to avoid flooding the log with unnecessary warning messages when a network problem is encountered
	singletons.Log.Info("broadcaster drainerSend:","dest",dest)
	defer func() {
		b.closed.Done()
	}()
	nodeID, err := b.identify.GetValidatorNodeId(dest)
	if err != nil {
		if successLastTime {
			singletons.Log.Warnf("could not get handle for replica %d", dest)
		}
		send.done <- false
		return false
	}

	err = b.comm.Unicast(send.msg, nodeID)
	if err != nil {
		if successLastTime {
			singletons.Log.Warnf("could not send to replica %d: %v", dest, err)
		}
		send.done <- false
		return false
	}

	send.done <- true
	return true

}

func (b *broadcaster) drainer(dest pbftTypes.ReplicaID) {
	successLastTime := false
	destChan, exsit := b.msgChans[dest] // Avoid doing the map lookup every send
	if !exsit {
		singletons.Log.Warnf("could not get message channel for replica %d", dest)
		return
	}

	for {
		select {
		case send := <-destChan:
			successLastTime = b.drainerSend(dest, send, successLastTime)
		case <-b.closedCh:
			for {
				// Drain the message channel to free calling waiters before we shut down
				select {
				case send := <-destChan:
					send.done <- false
					b.closed.Done()
				default:
					return
				}
			}
		}
	}
}

func (b *broadcaster) unicastOne(msg *message.Message, dest pbftTypes.ReplicaID, wait chan bool) {
	select {
	case b.msgChans[dest] <- &sendRequest{
		msg:  msg,
		done: wait,
	}:
	default:
		// If this channel is full, we must discard the message and flag it as done
		singletons.Log.Error("unicastOne default")
		wait <- false
		b.closed.Done()
	}
}

func (b *broadcaster) send(msg *message.Message, dest *pbftTypes.ReplicaID) error {
	select {
	case <-b.closedCh:
		return fmt.Errorf("broadcaster closed")
	default:
	}

	var destCount uint32
	var required uint32
	if dest != nil {
		destCount = 1
		required = 1
	} else {
		destCount = uint32(len(b.msgChans))
		required = destCount - b.f
	}

	wait := make(chan bool, destCount)

	if dest != nil {
		b.closed.Add(1)
		b.unicastOne(msg, *dest, wait)
	} else {
		b.closed.Add(len(b.msgChans))
		for i := range b.msgChans {
			b.unicastOne(msg, i, wait)
		}
	}

	succeeded := uint32(0)
	timer := time.NewTimer(b.broadcastTimeout)

	// This loop will try to send, until one of:
	// a) the required number of sends succeed
	// b) all sends complete regardless of success
	// c) the timeout expires and the required number of sends have returned
outer:
	for i := uint32(0); i < destCount; i++ {
		select {
		case success := <-wait:
			if success {
				succeeded++
				if succeeded >= required {
					break outer
				}
			}
		case <-timer.C:
			for i := i; i < required; i++ {
				<-wait
			}
			break outer
		}
	}

	return nil
}

func (b *broadcaster) Unicast(msg *message.Message, dest pbftTypes.ReplicaID) error {
	return b.send(msg, &dest)
}

func (b *broadcaster) Broadcast(msg *message.Message) error {
	return b.send(msg, nil)
}
