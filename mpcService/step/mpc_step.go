package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type MpcMessageGenerator interface {
	Initialize([]protocol.PeerInfo, protocol.MpcResultInterface) error
	GetMessageData(int)interface{}
	SetMessageData(uint64,interface{})error
	GetResultData()*protocol.MpcData
	CalculateResult() error
}

type BaseMpcStep struct {
	BaseStep
	messages []MpcMessageGenerator
}

func CreateBaseMpcStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface, messageNum int,bFilter bool) *BaseMpcStep {
	return &BaseMpcStep{*CreateBaseStep(result,nodeInfo, nodeInfo.NeedQuorum(),bFilter), make([]MpcMessageGenerator, messageNum)}
}

func (mpcStep *BaseMpcStep) InitStep() error {
	for _, message := range mpcStep.messages {
		err := message.Initialize(mpcStep.nodeInfo.GetPeers(),mpcStep.mpcResult)
		if err != nil {
			log.Error("BaseMpcStep, init msg fail.","error", err)
			return err
		}
	}

	return nil
}

func (mpcStep *BaseMpcStep) FinishStep() error {
	err := mpcStep.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	for _, message := range mpcStep.messages {
		err := message.CalculateResult()
		if err != nil {
			log.Error("BaseMpcStep, calculate msg result fail. err:%s", err.Error())
			return err
		}
	}

	return nil
}
