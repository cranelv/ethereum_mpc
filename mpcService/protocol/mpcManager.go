package protocol

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type MpcManager interface {
	P2pMessage(*discover.NodeID, uint64, interface{}) error
	BoardcastMessage([]*discover.NodeID, uint64, interface{}) error
	SelfNodeId() *discover.NodeID
	CreateKeystore(MpcResultInterface,MpcNodeInterface) error
	SignTransaction(MpcResultInterface) error
}
