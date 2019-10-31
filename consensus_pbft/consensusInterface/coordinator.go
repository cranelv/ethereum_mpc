package consensusInterface

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)
// PartialStack is a subset of peer.MessageHandlerCoordinator functionality which is necessary to perform state transfer
// Coordinator is used to initiate state transfer.  Start must be called before use, and Stop should be called to free allocated resources
type Coordinator interface {
	Start() // Start the block transfer go routine
	Stop()  // Stop up the block transfer go routine

	// SyncToTarget attempts to move the state to the given target, returning an error, and whether this target might succeed if attempted at a later time
	SyncToTarget(blockNumber uint64, digest pbftTypes.MessageDigest, peerIDs []*pbftTypes.PeerID) (error, bool)
}
