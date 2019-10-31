package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/step/generator"
	"github.com/ethereum/go-ethereum/log"
)

type MpcPoint_Step struct {
	BaseMpcStep
	resultKeys []string
}

func CreateMpcPoint_Step(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface, preValueKeys []string, resultKeys []string) *MpcPoint_Step {
	log.Info("CreateMpcPoint_Step begin")

	signNum := len(preValueKeys)
	mpc := &MpcPoint_Step{*CreateBaseMpcStep(result,nodeInfo, signNum,true), resultKeys}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = generator.CreatePointGenerator(preValueKeys[i],resultKeys[i])
	}

	return mpc
}

func (ptStep *MpcPoint_Step) CreateMessage() []protocol.StepMessage {
	log.Info("MpcPoint_Step.CreateMessage begin")

	message := make([]protocol.StepMessage, 1)
	message[0].Msgcode = protocol.MSG_MPCMessage
	message[0].PeerID = nil

	for _,msg := range ptStep.messages {
		message[0].Data = append(message[0].Data, &protocol.MpcData{protocol.MpcPointPart,msg.GetMessageData(0)})
	}

	return message
}

func (ptStep *MpcPoint_Step) HandleMessage(msg *protocol.StepMessage) bool {
	seed := ptStep.nodeInfo.GetSeed(msg.PeerID)
	if seed == 0 {
		log.Error("MpcPoint_Step.HandleMessage, get peer seed fail. peer:%s", msg.PeerID.String())
		return false
	}

	if len(msg.Data) != len(ptStep.messages){
		log.Error("MpcPoint_Step HandleMessage, msg data len doesn't match requiremant, dataLen:%d", len(msg.Data))
		return false
	}

	for i,msger := range ptStep.messages {
		err := msger.SetMessageData(seed,msg.Data[i].Data)
		if err != nil{
			log.Error("MpcPoint_Step.HandleMessage, get msg from seed fail. peer:%s", msg.PeerID.String())
			return false
		}
	}

	return true
}

func (ptStep *MpcPoint_Step) FinishStep( mpc protocol.MpcManager) error {
	log.Info("MpcPoint_Step.FinishStep begin")
	err := ptStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}
	for _,msg := range ptStep.messages {
		result := msg.GetResultData()
		ptStep.mpcResult.SetValue(result.Key,result.Data)
	}
	log.Info("MpcPoint_Step.FinishStep succeed")
	return nil
}

