package message

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)

type Message struct {
	Type      uint32               `protobuf:"varint,1,opt,name=type,enum=protos.Message_Type" json:"type,omitempty"`
	Timestamp uint64					 `protobuf:"bytes,2,opt,name=timestamp" json:"timestamp,omitempty"`
	Sender	  pbftTypes.PeerID
	Payload   []byte                     `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	Signature []byte                     `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (msg* Message)Unmarshal() (MessageInterface,error) {
	var inner MessageInterface
	err := UnmarshalMsg(msg.Payload,&inner)
	if err != nil{
		return nil,err
	}
	return inner,err
}
const (
	Message_UNDEFINED = iota + 0
	Message_DISC_HELLO
	Message_DISC_DISCONNECT
	Message_DISC_GET_PEERS
	Message_DISC_PEERS
	Message_DISC_NEWMSG
	Message_CHAIN_TASKS
	Message_RESPONSE
	Message_CONSENSUS = 20
)

