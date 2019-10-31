package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
)

type GetMpcReadyStep struct {
	BaseStep
}

func (ready *GetMpcReadyStep) InitStep() error {
	return nil
}

func CreateGetMpcReadyStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface) *GetMpcReadyStep {
	return &GetMpcReadyStep{*CreateBaseStep(result,nodeInfo, 1,false)}
}

func (ready *GetMpcReadyStep) CreateMessage() []protocol.StepMessage {
	return nil
}

func (ready *GetMpcReadyStep) FinishStep( mpc protocol.MpcManager) error {
	return ready.BaseStep.FinishStep()
}

func (ready *GetMpcReadyStep) HandleMessage(msg *protocol.StepMessage) bool {
	return true
}
