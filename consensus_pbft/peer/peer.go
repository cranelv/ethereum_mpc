package peer

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)
const (
	Response_UNDEFINED = 0
	Response_SUCCESS   = 200
	Response_FAILURE   = 500
)

type Response struct {
	Status uint32				 `protobuf:"varint,1,opt,name=status,enum=protos.Response_StatusCode" json:"status,omitempty"`
	Msg    []byte              `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}

// MessageHandler standard interface for handling Openchain messages.
type MessageHandler interface {
	HandleMessage(msg *message.Message) error
	SendMessage(msg *message.Message) error
	To() (pbftTypes.Peer, error)
	Stop() error
}
// HandlerFactory for creating new MessageHandlers
type HandlerFactory func(MessageHandlerCoordinator, ChatStream, bool) (MessageHandler, error)
// MessageHandlerCoordinator responsible for coordinating between the registered MessageHandler's
type MessageHandlerCoordinator interface {
	pbftTypes.Peer
	consensusInterface.NodeExecutor
	GetStateInfo() *message.StateInfo
	GetChainNode()consensusInterface.NodeInterface
	RegisterHandler(messageHandler MessageHandler) error
	DeregisterHandler(messageHandler MessageHandler) error
	Broadcast(*message.Message, consensusInterface.PeerEndpoint_Type) []error
	Unicast(*message.Message, *pbftTypes.PeerID) error
	GetPeers() ([]pbftTypes.Peer, error)
	PeersDiscovered([]pbftTypes.Peer) error
	ExecuteTasks(*message.Task) *Response

}

// TransactionProccesor responsible for processing of Transactions
type TransactionProccesor interface {
	ProcessTaskMsg(*message.Message, *message.Task) *Response
}

// Engine Responsible for managing Peer network communications (Handlers) and processing of Transactions
type Engine interface {
	TransactionProccesor
	// GetHandlerFactory return a handler for an accepted Chat stream
	GetHandlerFactory() HandlerFactory
	//GetInputChannel() (chan<- *pb.Transaction, error)
}
// ChatStream interface supported by stream between Peers
type ChatStream interface {
	Send(*message.Message) error
	Recv() (*message.Message, error)
}
