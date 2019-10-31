package step

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/step/generator"
)

type TXSign_Lagrange_Step struct {
	BaseMpcStep
}

func CreateTXSign_Lagrange_Step(result protocol.MpcResultInterface,nodeinfo protocol.MpcNodeInterface, preValueKeys []string, resultKeys []string) *TXSign_Lagrange_Step {
	log.Info("CreateTXSign_Lagrange_Step begin")

	signNum := len(preValueKeys)
	mpc := &TXSign_Lagrange_Step{*CreateBaseMpcStep(result,nodeinfo, signNum,true)}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = generator.CreateLagrangeGenerator(preValueKeys[i],resultKeys[i])
	}

	log.Info("CreateTXSign_Lagrange_Step succeed")
	return mpc
}

func (lagStep *TXSign_Lagrange_Step) CreateMessage() []protocol.StepMessage {
	log.Info("TXSign_Lagrange_Step.CreateMessage begin")

	message := make([]protocol.StepMessage, 1)
	message[0].Msgcode = protocol.MSG_MPCMessage
	message[0].PeerID = nil

	for _,msg := range lagStep.messages{
		message[0].Data = append(message[0].Data, &protocol.MpcData{protocol.MpcLagrange,msg.GetMessageData(0)})
	}

	log.Info("TXSign_Lagrange_Step.CreateMessage succeed")
	return message
}

func (lagStep *TXSign_Lagrange_Step) HandleMessage(msg *protocol.StepMessage) bool {
	log.Info("TXSign_Lagrange_Step.HandleMessage begin, peerID:%s", msg.PeerID.String())

	seed := lagStep.nodeInfo.GetSeed(msg.PeerID)
	if seed == 0 {
		log.Error("TXSign_Lagrange_Step.HandleMessage, get seed fail. peer:%s", msg.PeerID.String())
		return false
	}
//	log.Error("TXSign_Lagrange_Step","Seed",seed,"Data",msg.Data[0].Data.(*big.Int))

	for i,msger := range lagStep.messages{
		msger.SetMessageData(seed,msg.Data[i].Data)
	}
	log.Info("TXSign_Lagrange_Step.HandleMessage succees")
	return true
}

func (lagStep *TXSign_Lagrange_Step) FinishStep( mpc protocol.MpcManager) error {
	log.Info("TXSign_Lagrange_Step.FinishStep begin")

	err := lagStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	for _,msger := range lagStep.messages{
		result := msger.GetResultData()
		lagStep.mpcResult.SetValue(result.Key,result.Data)
	}

	log.Info("TXSign_Lagrange_Step.FinishStep succeed")
	return nil
}


