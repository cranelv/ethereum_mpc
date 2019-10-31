package pbftTypes

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
)
// Peer provides interface for a peer
type Peer_Type int32

const (
	Peer_UNDEFINED     Peer_Type = 0
	Peer_VALIDATOR     Peer_Type = 1
	Peer_NON_VALIDATOR Peer_Type = 2
)
type Peer interface {
	GetPeerId() *PeerID
	GetType()  Peer_Type
}
type PeerID discover.NodeID
type ReplicaID uint32
type MessageDigest string
