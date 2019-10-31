package message

import "github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"

const (
	Message_Undefined = iota + 0
	Message_Request
	Message_RequestBatch
	Message_PrePrepare
	Message_Prepare
	Message_Commit
	Message_Checkpoint
	Message_ViewChange
	Message_NewView
	Message_FetchRequestBatch
	Message_ReturnRequestBatch	// == Message_RequestBatch
)
type PbftMessage struct {
	PbftType uint32
	Timestamp uint64
	Sender	  pbftTypes.PeerID
	Payload   []byte
	Signature []byte
}