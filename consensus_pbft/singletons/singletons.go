package singletons

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/common"
)

var (
	Log Logger = &PbftLogger{}
	Marshaler MarshalInterface = &rlpMarshal{}
	Hasher HashInterface = &rlpHasher{}
)
type rlpMarshal struct {

}
func (rm *rlpMarshal)Marshal(data interface{}) ([]byte, error){
	return rlp.EncodeToBytes(data)
}
func (rm *rlpMarshal)Unmarshal(buf []byte, data interface{}) error{
	return rlp.DecodeBytes(buf,data)
}

type rlpHasher struct {
}

func (rh *rlpHasher)Hash(data interface{}) pbftTypes.MessageDigest {
	h := common.Hash{}
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, data)
	hw.Sum(h[:0])
	return pbftTypes.MessageDigest(h[:])
}