package step

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type RequestMpcSignStep struct {
	BaseStep
	messageType int64
	txHash      common.Hash
	address     common.MpcAddress
	message     map[common.Hash]bool
}
func (req *RequestMpcSignStep) InitStep()error{
	addr, err := req.mpcResult.GetValue(protocol.MpcAddress)
	if err != nil {
		return err
	}
	txhash, err := req.mpcResult.GetValue(protocol.MpcTxHash)
	if err != nil {
		return err
	}
	req.address = common.BytesToMpcAddress(addr.([]byte))
	req.txHash = common.BytesToHash(txhash.([]byte))
	return nil
}

func (req *RequestMpcSignStep) CreateMessage() []protocol.StepMessage {
	msg := protocol.StepMessage{
		Msgcode:protocol.MSG_RequestMPC,
		PeerID:nil,
		Peers:req.nodeInfo.GetPeers()}
	msg.Data = append(msg.Data,&protocol.MpcData{protocol.MpcAddress,req.address})
	msg.Data = append(msg.Data,&protocol.MpcData{protocol.MpcTxHash,req.txHash[:]})
	return []protocol.StepMessage{msg}
}

func (req *RequestMpcSignStep) FinishStep(mpc protocol.MpcManager) error {
	err := req.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	return nil
}

func (req *RequestMpcSignStep) HandleMessage(msg *protocol.StepMessage) bool {
	log.Info("RequestMpcStep.HandleMessage begin, peerID:%s", msg.PeerID)
	return true
}
