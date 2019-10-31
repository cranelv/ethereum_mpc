package message

import "github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
type Block struct {
	ConsensusMetadata []byte
	PreviousBlockHash pbftTypes.MessageDigest
	StateResults	 []*Result
	Tasks		 []*Task
}
