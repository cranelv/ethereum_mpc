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

package helper

import (
	"fmt"


	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/executor"
//	"github.com/ethereum/go-ethereum/consensus_pbft/persist"
	"github.com/ethereum/go-ethereum/consensus_pbft/peer"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/helper/persist"
)

// Helper contains the reference to the peer's MessageHandlerCoordinator
type Helper struct {
	consenter    consensus_pbft.Consenter
	coordinator  peer.MessageHandlerCoordinator
	identify 	consensusInterface.ValidatorIdentifyInterface
	secOn        bool
	valid        bool // Whether we believe the state is up to date
	Node	    consensusInterface.NodeInterface
	curBatch     []*types.Transaction       // TODO, remove after issue 579
//	curBatchErrs []*types.TransactionResult // TODO, remove after issue 579
	persist.Helper

	executor consensus_pbft.Executor
}

// NewHelper constructs the consensus helper object
func NewHelper(mhc peer.MessageHandlerCoordinator) *Helper {
	h := &Helper{
		coordinator: mhc,
		secOn:       false,//viper.GetBool("security.enabled"),
		Node:   mhc.GetChainNode(),
		valid:       true, // Assume our state is consistent until we are told otherwise, actual consensus (pbft) will invalidate this immediately, but noops will not
	}

	h.executor = executor.NewImpl(h, mhc)
	return h
}

func (h *Helper) setConsenter(c consensus_pbft.Consenter) {
	h.consenter = c
	h.executor.Start() // The consenter may be expecting a callback from the executor because of state transfer completing, it will miss this if we start the executor too early
}

// GetNetworkInfo returns the PeerEndpoints of the current validator and the entire validating network
func (h *Helper) GetNetworkNodes() (self pbftTypes.Peer, network []pbftTypes.Peer, err error) {
	ep := h.coordinator.(pbftTypes.Peer)
	self = ep

	peers, err := h.coordinator.GetPeers()
	if err != nil {
		return self, network, fmt.Errorf("Couldn't retrieve list of peers: %v", err)
	}
//	peers := peersMsg()
	for _, endpoint := range peers {
		if true { //endpoint.Type == peer.PeerEndpoint_VALIDATOR {
			network = append(network, endpoint)
		}
	}
	network = append(network, self)

	return
}

// GetNetworkHandles returns the PeerIDs of the current validator and the entire validating network
func (h *Helper) GetNetworkNodeIDs() (self *pbftTypes.PeerID, network []*pbftTypes.PeerID, err error) {
	selfEP, networkEP, err := h.GetNetworkNodes()
	if err != nil {
		return self, network, fmt.Errorf("Couldn't retrieve validating network's endpoints: %v", err)
	}

	self = selfEP.GetPeerId()

	for _, endpoint := range networkEP {
		network = append(network, endpoint.GetPeerId())
	}
	network = append(network, self)

	return
}

// Broadcast sends a message to all validating peers
func (h *Helper) Broadcast(msg *message.Message, peerType consensusInterface.PeerEndpoint_Type) error {
	errors := h.coordinator.Broadcast(msg, peerType)
	if len(errors) > 0 {
		return fmt.Errorf("Couldn't broadcast successfully")
	}
	return nil
}

// Unicast sends a message to a specified receiver
func (h *Helper) Unicast(msg *message.Message, receiver *pbftTypes.PeerID) error {
	return h.coordinator.Unicast(msg, receiver)
}

// Sign a message with this validator's signing key
func (h *Helper) Sign(msg []byte) ([]byte, error) {
//	if h.secOn {
//		return h.secHelper.Sign(msg)
//	}
	log.Debug("Security is disabled")
	return msg, nil
}

// Verify that the given signature is valid under the given replicaID's verification key
// If replicaID is nil, use this validator's verification key
// If the signature is valid, the function should return nil
func (h *Helper) Verify(replicaID pbftTypes.ReplicaID, signature []byte, message []byte) error {
	if !h.secOn {
		log.Debug("Security is disabled")
		return nil
	}

	log.Debug("Verify message from: %v", replicaID)
	_, network, err := h.GetNetworkNodeIDs()
	if err != nil {
		return fmt.Errorf("Couldn't retrieve validating network's endpoints: %v", err)
	}

	// check that the sender is a valid replica
	// if so, call crypto verify() with that endpoint's pkiID
	for _, endpoint := range network {
		log.Debug("Endpoint name: %v", endpoint)
		endId,_ := h.identify.GetValidatorID(endpoint)
		if replicaID == endId {
			return h.Node.Verify(endpoint, signature, message)
		}
	}
	return fmt.Errorf("Could not verify message from %s (unknown peer)", replicaID)
}

// BeginTxBatch gets invoked when the next round
// of transaction-batch execution begins
func (h *Helper) BeginTaskBatch(id interface{}) error {
/*	ledger, err := ledger.GetLedger()
	if err != nil {
		return fmt.Errorf("Failed to get the ledger: %v", err)
	}
	if err := ledger.BeginTxBatch(id); err != nil {
		return fmt.Errorf("Failed to begin transaction with the ledger: %v", err)
	}
	h.curBatch = nil     // TODO, remove after issue 579
	h.curBatchErrs = nil // TODO, remove after issue 579
*/
	return nil
}

// ExecTxs executes all the transactions listed in the txs array
// one-by-one. If all the executions are successful, it returns
// the candidate global state hash, and nil error array.
func (h *Helper) ExecTasks(id interface{}, txs []*message.Task) ([]*message.Result, error) {
	// TODO id is currently ignored, fix once the underlying implementation accepts id

	// The secHelper is set during creat ChaincodeSupport, so we don't need this step
	// cxt := context.WithValue(context.Background(), "security", h.coordinator.GetSecHelper())
	// TODO return directly once underlying implementation no longer returns []error
/*
	succeededTxs, res, ccevents, txerrs, err := chaincode.ExecuteTransactions(context.Background(), chaincode.DefaultChain, txs)

	h.curBatch = append(h.curBatch, succeededTxs...) // TODO, remove after issue 579

	//copy errs to result
	txresults := make([]*pb.TransactionResult, len(txerrs))

	//process errors for each transaction
	for i, e := range txerrs {
		//NOTE- it'll be nice if we can have error values. For now success == 0, error == 1
		if txerrs[i] != nil {
			txresults[i] = &pb.TransactionResult{Txid: txs[i].Hash(), Error: e.Error(), ErrorCode: 1, ChaincodeEvent: ccevents[i]}
		} else {
			txresults[i] = &pb.TransactionResult{Txid: txs[i].Hash(), ChaincodeEvent: ccevents[i]}
		}
	}
	h.curBatchErrs = append(h.curBatchErrs, txresults...) // TODO, remove after issue 579

	return res, err
*/
	return nil,nil
}

// CommitTxBatch gets invoked when the current transaction-batch needs
// to be committed. This function returns successfully iff the
// transactions details and state changes (that may have happened
// during execution of this transaction-batch) have been committed to
// permanent storage.
func (h *Helper) CommitTaskBatch(id interface{}, metadata []byte) (*message.StateInfo, error) {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the ledger: %v", err)
	}
	// TODO fix this one the ledger has been fixed to implement
	if err := ledger.CommitTxBatch(id, h.curBatch, h.curBatchErrs, metadata); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction to the ledger: %v", err)
	}

	size := ledger.GetBlockchainSize()
	defer func() {
		h.curBatch = nil     // TODO, remove after issue 579
		h.curBatchErrs = nil // TODO, remove after issue 579
	}()

	block, err := ledger.GetBlockByNumber(size - 1)
	if err != nil {
		return nil, fmt.Errorf("Failed to get the block at the head of the chain: %v", err)
	}

	logger.Debugf("Committed block with %d transactions, intended to include %d", len(block.Transactions), len(h.curBatch))

	return block, nil
	*/
	return nil,nil
}

// RollbackTxBatch discards all the state changes that may have taken
// place during the execution of current transaction-batch
func (h *Helper) RollbackTaskBatch(id interface{}) error {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return fmt.Errorf("Failed to get the ledger: %v", err)
	}
	if err := ledger.RollbackTxBatch(id); err != nil {
		return fmt.Errorf("Failed to rollback transaction with the ledger: %v", err)
	}
	h.curBatch = nil     // TODO, remove after issue 579
	h.curBatchErrs = nil // TODO, remove after issue 579
	return nil
	*/
	return nil
}

// PreviewCommitTxBatch retrieves a preview of the block info blob (as
// returned by GetBlockchainInfoBlob) that would describe the
// blockchain if CommitTxBatch were invoked.  The blockinfo will
// change if additional ExecTXs calls are invoked.
func (h *Helper) PreviewCommitTaskBatch(id interface{}, metadata []byte) ([]byte, error) {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the ledger: %v", err)
	}
	// TODO fix this once the underlying API is fixed
	blockInfo, err := ledger.GetTXBatchPreviewBlockInfo(id, h.curBatch, metadata)
	if err != nil {
		return nil, fmt.Errorf("Failed to preview commit: %v", err)
	}
	rawInfo, _ := proto.Marshal(blockInfo)
	return rawInfo, nil
	*/
	return nil,nil
}

// GetBlock returns a block from the chain
func (h *Helper) GetState(blockNumber uint64) (state *message.StateInfo, err error) {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the ledger :%v", err)
	}
	return ledger.GetBlockByNumber(blockNumber)
	*/
	return nil,nil
}

// GetCurrentStateHash returns the current/temporary state hash
func (h *Helper) GetCurrentStateHash() (stateHash []byte, err error) {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the ledger :%v", err)
	}
	return ledger.GetTempStateHash()
	*/
	return nil,nil
}

// GetBlockchainInfoBlob marshals a ledger's BlockchainInfo into a protobuf
func (h *Helper) GetBlockchainInfoBlob() []byte {
	/*
	ledger, _ := ledger.GetLedger()
	info, _ := ledger.GetBlockchainInfo()
	rawInfo, _ := proto.Marshal(info)
	return rawInfo*/
	return nil
}

// GetBlockHeadMetadata returns metadata from block at the head of the blockchain
func (h *Helper) GetBlockHeadMetadata() ([]byte, error) {
	/*
	ledger, err := ledger.GetLedger()
	if err != nil {
		return nil, err
	}
	head := ledger.GetBlockchainSize()
	block, err := ledger.GetBlockByNumber(head - 1)
	if err != nil {
		return nil, err
	}
	return block.ConsensusMetadata, nil*/
	return nil,nil
}

// InvalidateState is invoked to tell us that consensus realizes the ledger is out of sync
func (h *Helper) InvalidateState() {
	log.Debug("Invalidating the current state")
	h.valid = false
}

// ValidateState is invoked to tell us that consensus has the ledger back in sync
func (h *Helper) ValidateState() {
	log.Debug("Validating the current state")
	h.valid = true
}

// Execute will execute a set of transactions, this may be called in succession
func (h *Helper) Execute(tag interface{}, txs []*message.Task) {
	h.executor.Execute(tag, txs)
}

// Commit will commit whatever transactions have been executed
func (h *Helper) Commit(tag interface{}, metadata []byte) {
	h.executor.Commit(tag, metadata)
}

// Rollback will roll back whatever transactions have been executed
func (h *Helper) Rollback(tag interface{}) {
	h.executor.Rollback(tag)
}

// UpdateState attempts to synchronize state to a particular target, implicitly calls rollback if needed
func (h *Helper) UpdateState(tag interface{}, target *message.StateInfo, peers []*pbftTypes.PeerID) {
	if h.valid {
		log.Warn("State transfer is being called for, but the state has not been invalidated")
	}

	h.executor.UpdateState(tag, target, peers)
}

// Executed is called whenever Execute completes
func (h *Helper) Executed(tag interface{}) {
	if h.consenter != nil {
		h.consenter.Executed(tag)
	}
}

// Committed is called whenever Commit completes
func (h *Helper) Committed(tag interface{}, target* message.StateInfo) {
	if h.consenter != nil {
		h.consenter.Committed(tag, target)
	}
}

// RolledBack is called whenever a Rollback completes
func (h *Helper) RolledBack(tag interface{}) {
	if h.consenter != nil {
		h.consenter.RolledBack(tag)
	}
}

// StateUpdated is called when state transfer completes, if target is nil, this indicates a failure and a new target should be supplied
func (h *Helper) StateUpdated(tag interface{}, target* message.StateInfo) {
	if h.consenter != nil {
		h.consenter.StateUpdated(tag, target)
	}
}

// Start his is a byproduct of the consensus API needing some cleaning, for now it's a no-op
func (h *Helper) Start() {}

// Halt is a byproduct of the consensus API needing some cleaning, for now it's a no-op
func (h *Helper) Halt() {}
