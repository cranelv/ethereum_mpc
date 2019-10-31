package consensusInterface

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
)

type NodeInterface interface {
	// Sign signs msg with this validator's signing key and outputs
	// the signature if no error occurred.
	Sign(msg []byte) ([]byte, error)
	Verify(peerID *pbftTypes.PeerID, signature []byte, message []byte) error

}
type ValidatorIdentifyInterface interface{
	GetValidatorID(handle *pbftTypes.PeerID) (pbftTypes.ReplicaID, error)
	GetValidatorNodeId(id pbftTypes.ReplicaID) (*pbftTypes.PeerID,error)
	GetValidatorNodeIds(ids []pbftTypes.ReplicaID) ([]*pbftTypes.PeerID,error)
	GetValidatorNode(id pbftTypes.ReplicaID) (pbftTypes.Peer,error)
	GetValidatorNodes(ids []pbftTypes.ReplicaID) ([]pbftTypes.Peer,error)
}

// LegacyExecutor is used to invoke transactions, potentially modifying the backing ledger
type NodeExecutor interface {
	BeginTaskBatch(id interface{}) error
	ExecTasks(id interface{}, txs []*message.Task) ([]*message.Result, error)
	CommitTaskBatch(id interface{}, metadata []byte) (*message.StateInfo, error)
	RollbackTaskBatch(id interface{}) error
	PreviewCommitTaskBatch(id interface{}, metadata []byte) ([]byte, error)
}