package message

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)

type StateInfo struct {
	Hash pbftTypes.MessageDigest
	Number uint64
	Payload interface{}
}
func (info *StateInfo)Digest() pbftTypes.MessageDigest{
//	payload,_ := singletons.Marshaler.Marshal(info)
	return info.Hash
}
func (info *StateInfo)Height() uint64{
	return info.Number
}
