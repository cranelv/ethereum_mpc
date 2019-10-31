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

package consensus_pbft

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
)

// ExecutionConsumer allows callbacks from asycnhronous execution and statetransfer
type ExecutionConsumer interface {
	Executed(tag interface{})                                // Called whenever Execute completes
	Committed(tag interface{}, state  *message.StateInfo)    // Called whenever Commit completes
	RolledBack(tag interface{})                              // Called whenever a Rollback completes
	StateUpdated(tag interface{}, state *message.StateInfo) // Called when state transfer completes, if target is nil, this indicates a failure and a new target should be supplied
}

// Consenter is used to receive messages from the network
// Every consensus plugin needs to implement this interface
type Consenter interface {
	RecvMsg(msg *message.Message, senderHandle *pbftTypes.PeerID) error // Called serially with incoming messages from gRPC
	ExecutionConsumer
}

// Inquirer is used to retrieve info about the validating network
type Inquirer interface {
	GetNetworkNodes() (self pbftTypes.Peer, network []pbftTypes.Peer, err error)
	GetNetworkNodeIDs() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error)
}

// Communicator is used to send messages to other validators
type Communicator interface {
	Broadcast(msg *message.Message, peerType pbftTypes.Peer_Type) error
	Unicast(msg *message.Message, receiverHandle *pbftTypes.PeerID) error
}

// NetworkStack is used to retrieve network info and send messages
type NetworkStack interface {
	Communicator
	Inquirer
}

// SecurityUtils is used to access the sign/verify methods from the crypto package
type SecurityUtils interface {
	Sign(msg []byte) ([]byte, error)
	Verify(peerID pbftTypes.ReplicaID, signature []byte, message []byte) error
}

// ReadOnlyLedger is used for interrogating the blockchain
type ReadOnlyLedger interface {
	GetState(height uint64) (state *message.StateInfo, err error)
//	GetBlockchainInfo() *pb.BlockchainInfo
	GetBlockchainInfoBlob() []byte
	GetBlockHeadMetadata() ([]byte, error)
}


// Executor is intended to eventually supplant the old Executor interface
// The problem with invoking the calls directly above, is that they must be coordinated
// with state transfer, to eliminate possible races and ledger corruption
type Executor interface {
	Start()                                                                     // Bring up the resources needed to use this interface
	Halt()                                                                      // Tear down the resources needed to use this interface
	Execute(tag interface{}, txs []*message.Task)                             // Executes a set of transactions, this may be called in succession
	Commit(tag interface{}, metadata []byte)                                    // Commits whatever transactions have been executed
	Rollback(tag interface{})                                                   // Rolls back whatever transactions have been executed
	UpdateState(tag interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID) // Attempts to synchronize state to a particular target, implicitly calls rollback if needed
}

// LedgerManager is used to manipulate the state of the ledger
type LedgerManager interface {
	InvalidateState() // Invalidate informs the ledger that it is out of date and should reject queries
	ValidateState()   // Validate informs the ledger that it is back up to date and should resume replying to queries
}

// StatePersistor is used to store consensus state which should survive a process crash
type StatePersistor interface {
	StoreState(key string, value []byte) error
	ReadState(key string) ([]byte, error)
	ReadStateSet(prefix string) (map[string][]byte, error)
	DelState(key string)
}

// Stack is the set of stack-facing methods available to the consensus plugin
type Stack interface {
	NetworkStack	// 网络消息发送和接收接口
	SecurityUtils	// Sign和Verify接口
	Executor		// 事件消息处理接口
	consensusInterface.NodeExecutor	// 交易处理接口
	LedgerManager	// 控制ledger的状态
	ReadOnlyLedger	// 操作blockchain
	StatePersistor	// 操作共识状态
}
