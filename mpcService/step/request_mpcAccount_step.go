package step

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type RequestMpcAccountStep struct {
	BaseStep
	address     []byte
	message     map[common.Hash]bool
}
func (req *RequestMpcAccountStep) InitStep()error{
	return nil
}
func (req *RequestMpcAccountStep) CreateMessage() []protocol.StepMessage {
	msg := protocol.StepMessage{
		Msgcode:protocol.MSG_RequestMPC,
		PeerID:nil,
		Peers:req.nodeInfo.GetPeers()}
	return []protocol.StepMessage{msg}
}

func (req *RequestMpcAccountStep) FinishStep(mpc protocol.MpcManager) error {
	err := req.BaseStep.FinishStep()
	if err != nil {
		return err
	}
	return nil
}

func (req *RequestMpcAccountStep) HandleMessage(msg *protocol.StepMessage) bool {
	log.Info("RequestMpcStep.HandleMessage begin," ,"peerID", msg.PeerID)
	return true
}


func CreateRequestMpcStep(result protocol.MpcResultInterface,nodeinfo protocol.MpcNodeInterface, messageType uint32) protocol.MpcStepFunc {
	if messageType == protocol.MpcCreateLockAccountLeader{
		return &RequestMpcAccountStep{BaseStep: *CreateBaseStep(result,nodeinfo, nodeinfo.NeedQuorum()-1,true), message: make(map[common.Hash]bool)}
	}else if messageType == protocol.MpcTXSignLeader{
		return &RequestMpcSignStep{BaseStep: *CreateBaseStep(result,nodeinfo, nodeinfo.NeedQuorum()-1,true), message: make(map[common.Hash]bool)}
	}
	return nil
}


