package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type AcknowledgeMpcStep struct {
	BaseStep
	messageType uint32
}

func CreateAcknowledgeMpcStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface, messageType uint32) *AcknowledgeMpcStep {
	log.Info("CreateAcknowledgeMpcStep begin")

	return &AcknowledgeMpcStep{*CreateBaseStep(result,nodeInfo, 0,false), messageType}
}

func (ack *AcknowledgeMpcStep) InitStep() error {
	return nil
}

func (ack *AcknowledgeMpcStep) CreateMessage() []protocol.StepMessage {
	log.Info("AcknowledgeMpcStep.CreateMessage begin")

	return []protocol.StepMessage{protocol.StepMessage{
		Msgcode:protocol.MSG_MPCMessage,
		PeerID:ack.nodeInfo.Leader(),
		}}
}

func (ack *AcknowledgeMpcStep) FinishStep(mpc protocol.MpcManager) error {
	log.Info("AcknowledgeMpcStep.FinishStep begin")

	err := ack.BaseStep.FinishStep()
	if err != nil {
		return err
	}
	ack.mpcResult.SetValue(protocol.MPCAction, ack.messageType)

	log.Info("AcknowledgeMpcStep.FinishStep succeed")
	return nil
}

func (ack *AcknowledgeMpcStep) HandleMessage(msg *protocol.StepMessage) bool {
	return true
}
