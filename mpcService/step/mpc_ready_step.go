package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
)

type MpcReadyStep struct {
	BaseStep
}

func (ready *MpcReadyStep) InitStep() error {
	return nil
}

func CreateMpcReadyStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface) *MpcReadyStep {
	return &MpcReadyStep{*CreateBaseStep(result,nodeInfo, 0,false)}
}

func (ready *MpcReadyStep) CreateMessage() []protocol.StepMessage {
	return []protocol.StepMessage{protocol.StepMessage{
		Msgcode:protocol.MSG_MPCMessage,
		PeerID:nil,
		Peers:nil}}
}

func (ready *MpcReadyStep) FinishStep(mpc protocol.MpcManager) error {
	err := ready.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	return nil
}

func (ready *MpcReadyStep) HandleMessage(msg *protocol.StepMessage) bool {
	return true
}
