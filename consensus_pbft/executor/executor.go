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

package executor

import (
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/util/events"

	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
)

// PartialStack contains the ledger features required by the executor.Coordinator
type PartialStack interface {
	consensusInterface.NodeExecutor
	GetStateInfo() *message.StateInfo
}

type coordinatorImpl struct {
	manager         events.Manager              // Maintains event thread and sends events to the coordinator
	rawExecutor     PartialStack                // Does the real interaction with the ledger
	consumer        consensus_pbft.ExecutionConsumer // The consumer of this coordinator which receives the callbacks
	stc             consensusInterface.Coordinator	// State transfer instance
	batchInProgress bool                        // Are we mid execution batch
	skipInProgress  bool                        // Are we mid state transfer
}

// NewCoordinatorImpl creates a new executor.Coordinator
func NewImpl(consumer consensus_pbft.ExecutionConsumer, rawExecutor PartialStack) consensus_pbft.Executor {
	co := &coordinatorImpl{
		rawExecutor: rawExecutor,
		consumer:    consumer,
//		stc:         statetransfer.NewCoordinatorImpl(stps),
		manager:     events.NewManagerImpl(),
	}
	co.manager.SetReceiver(co)
	return co
}

// ProcessEvent is the main event loop for the executor.Coordinator
func (co *coordinatorImpl) ProcessEvent(event events.Event) events.Event {
	switch et := event.(type) {
	case executeEvent:
		singletons.Log.Debug("Executor is processing an executeEvent")
		if co.skipInProgress {
			singletons.Log.Error("FATAL programming error, attempted to execute a transaction during state transfer")
			return nil
		}

		if !co.batchInProgress {
			singletons.Log.Debug("Starting new transaction batch")
			co.batchInProgress = true
			err := co.rawExecutor.BeginTaskBatch(co)
			_ = err // TODO This should probably panic, see issue 752
		}

		co.rawExecutor.ExecTasks(co, et.tasks)

		co.consumer.Executed(et.tag)
	case commitEvent:
		singletons.Log.Debug("Executor is processing an commitEvent")
		if co.skipInProgress {
			singletons.Log.Error("Likely FATAL programming error, attempted to commit a transaction batch during state transfer")
			return nil
		}

		if !co.batchInProgress {
			singletons.Log.Error("Likely FATAL programming error, attemted to commit a transaction batch when one does not exist")
			return nil
		}

		_, err := co.rawExecutor.CommitTaskBatch(co, et.metadata)
		_ = err // TODO This should probably panic, see issue 752

		co.batchInProgress = false

		info := co.rawExecutor.GetStateInfo()

		singletons.Log.Debugf("Committed tasksInfo with hash %x to chain", info.Digest())

		co.consumer.Committed(et.tag, info)
	case rollbackEvent:
		singletons.Log.Debug("Executor is processing an rollbackEvent")
		if co.skipInProgress {
			singletons.Log.Error("Programming error, attempted to rollback a transaction batch during state transfer")
			return nil
		}

		if !co.batchInProgress {
			singletons.Log.Error("Programming error, attempted to rollback a transaction batch which had not started")
			return nil
		}

		err := co.rawExecutor.RollbackTaskBatch(co)
		_ = err // TODO This should probably panic, see issue 752

		co.batchInProgress = false

		co.consumer.RolledBack(et.tag)
	case stateUpdateEvent:
		singletons.Log.Debug("Executor is processing a stateUpdateEvent")
		if co.batchInProgress {
			err := co.rawExecutor.RollbackTaskBatch(co)
			_ = err // TODO This should probably panic, see issue 752
		}

		co.skipInProgress = true

		info := et.stateInfo
		for {
			err, recoverable := co.stc.SyncToTarget( info.Height(), info.Digest(), et.peers)
			if err == nil {
				singletons.Log.Debug("State transfer sync completed, returning")
				co.skipInProgress = false
				co.consumer.StateUpdated(et.tag, info)
				return nil
			}
			if !recoverable {
				singletons.Log.Warnf("State transfer failed irrecoverably, calling back to consumer: %s", err)
				co.consumer.StateUpdated(et.tag, nil)
				return nil
			}
			singletons.Log.Warnf("State transfer did not complete successfully but is recoverable, trying again: %s", err)
			et.peers = nil // Broaden the peers included in recover to all connected
		}
	default:
		singletons.Log.Error("Unknown event type %s", et)
	}

	return nil
}

// Commit commits whatever outstanding requests have been executed, it is an error to call this without pending executions
func (co *coordinatorImpl) Commit(tag interface{}, metadata []byte) {
	co.manager.Queue() <- commitEvent{tag, metadata}
}

// Execute adds additional executions to the current batch
func (co *coordinatorImpl) Execute(tag interface{}, txs []*message.Task) {
	co.manager.Queue() <- executeEvent{tag, txs}
}

// Rollback rolls back the executions from the current batch
func (co *coordinatorImpl) Rollback(tag interface{}) {
	co.manager.Queue() <- rollbackEvent{tag}
}

// UpdateState uses the state transfer subsystem to attempt to progress to a target
func (co *coordinatorImpl) UpdateState(tag interface{}, info *message.StateInfo, peers []*pbftTypes.PeerID) {
	co.manager.Queue() <- stateUpdateEvent{tag, info, peers}
}

// Start must be called before utilizing the Coordinator
func (co *coordinatorImpl) Start() {
	co.stc.Start()
	co.manager.Start()
}

// Halt should be called to clean up resources allocated by the Coordinator
func (co *coordinatorImpl) Halt() {
	co.stc.Stop()
	co.manager.Halt()
}

// Event types

type executeEvent struct {
	tag interface{}
	tasks []*message.Task
}

// Note, this cannot be a simple type alias, in case tag is nil
type rollbackEvent struct {
	tag interface{}
}

type commitEvent struct {
	tag      interface{}
	metadata []byte
}

type stateUpdateEvent struct {
	tag            interface{}
	stateInfo 	   *message.StateInfo
	peers          []*pbftTypes.PeerID
}
