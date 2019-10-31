package protocol

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
)
type MpcValue struct {
	Key       string
	Value     interface{}
}
type MpcResult struct {
	Hash common.Hash
	Result []byte
}
type MpcResultInterface interface {
	Initialize(preSetValue ...MpcValue)
	SetValue(key string, value interface{}) error
	GetValue(key string) (interface{}, error)
}
type MpcState int
const (
	MpcCollection  MpcState = iota + 0
	MpcWaiting
	MpcRunning
	MpcFinish
	MpcNotFound
)
type MpcNodeInterface interface {
	NeedQuorum()int
	Address()*common.MpcAddress
	Hash()*common.Hash
	Leader() *discover.NodeID
	FetchQuorum()bool
	RunNode(uint64,*discover.NodeID)MpcState
	AddNode(seed uint64,nodeId *discover.NodeID) error
	GetSeedsNum(seed uint64,nodeId *discover.NodeID)int
	SetState(MpcState)
	GetState()MpcState
	GetSeed(nodeId *discover.NodeID)uint64
	GetNodeId(seed uint64) *discover.NodeID
	GetPeers() []PeerInfo
	GetPeerIDs() []*discover.NodeID
}