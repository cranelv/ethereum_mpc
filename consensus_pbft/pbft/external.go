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
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
)

// --------------------------------------------------------------
//
// external contains all of the functions which
// are intended to be called from outside of the pbft package
//
// --------------------------------------------------------------

// Event types

// stateUpdatedEvent is sent when state transfer completes
type stateUpdatedEvent struct {
	chkpt  *checkpointMessage
	stateInfo  *message.StateInfo
}

// executedEvent is sent when a requested execution completes
type executedEvent struct {
	tag interface{}
}

// commitedEvent is sent when a requested commit completes
type committedEvent struct {
	tag    interface{}
	stateInfo *message.StateInfo
}

// rolledBackEvent is sent when a requested rollback completes
type rolledBackEvent struct{}

type externalEventReceiver struct {
	manager events.Manager
}

// RecvMsg is called by the stack when a new message is received
func (eer *externalEventReceiver) RecvMsg(ocMsg *message.Message, senderId *pbftTypes.PeerID) error {
	eer.manager.Queue() <- batchMessageEvent{
		msg:    ocMsg,
		sender: senderId,
	}
	return nil
}

// Executed is called whenever Execute completes, no-op for noops as it uses the legacy synchronous api
func (eer *externalEventReceiver) Executed(tag interface{}) {
	eer.manager.Queue() <- executedEvent{tag}
}

// Committed is called whenever Commit completes, no-op for noops as it uses the legacy synchronous api
func (eer *externalEventReceiver) Committed(tag interface{}, stateInfo  *message.StateInfo) {
	eer.manager.Queue() <- committedEvent{tag, stateInfo}
}

// RolledBack is called whenever a Rollback completes, no-op for noops as it uses the legacy synchronous api
func (eer *externalEventReceiver) RolledBack(tag interface{}) {
	eer.manager.Queue() <- rolledBackEvent{}
}

// StateUpdated is a signal from the stack that it has fast-forwarded its state
func (eer *externalEventReceiver) StateUpdated(tag interface{}, stateInfo *message.StateInfo) {
	eer.manager.Queue() <- stateUpdatedEvent{
		chkpt:  tag.(*checkpointMessage),
		stateInfo: stateInfo,
	}
}
