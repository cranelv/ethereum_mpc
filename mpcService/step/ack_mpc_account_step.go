package step

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type AckMpcAccountStep struct {
	BaseStep
	mpcAddr        common.MpcAddress
}

func CreateAckMpcAccountStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface) *AckMpcAccountStep {
	return &AckMpcAccountStep{*CreateBaseStep(result,nodeInfo, nodeInfo.NeedQuorum(),true),  common.MpcAddress{}}
}

func (ack *AckMpcAccountStep) InitStep() error {
	log.Info("AckMpcAccountStep.InitStep begin")
	mpcAddr, err := ack.mpcResult.GetValue(protocol.MpcContextResult)
	if err != nil {
		log.Error("ack mpc account step, init fail. err:%s", err.Error())
		return err
	}
	ack.mpcAddr = common.BytesToMpcAddress(mpcAddr.([]byte))
	return nil
}

func (ack *AckMpcAccountStep) CreateMessage() []protocol.StepMessage {
	return []protocol.StepMessage{protocol.StepMessage{
		Msgcode:protocol.MSG_MPCMessage,
		PeerID:nil,
		Peers:nil,
		Data:nil}}
}

func (ack *AckMpcAccountStep) FinishStep( mpc protocol.MpcManager) error {
	log.Info("AckMpcAccountStep.FinishStep begin")
	err := ack.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	return nil
}

func (ack *AckMpcAccountStep) HandleMessage(msg *protocol.StepMessage) bool {
	return true
}
