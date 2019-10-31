package step

import (
	"github.com/ethereum/go-ethereum/mpcService/step/generator"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

type MpcJRSS_Step struct {
	BaseMpcStep
}

func CreateMpcJRSS_Step(result protocol.MpcResultInterface,degree int, nodeInfo protocol.MpcNodeInterface) *MpcJRSS_Step {
	mpc := &MpcJRSS_Step{*CreateBaseMpcStep(result,nodeInfo, 1,true)}
	mpc.messages[0] = generator.CreateJRSSValue(degree, nodeInfo.NeedQuorum(),protocol.MpcPrivateShare)
	return mpc
}

func (jrss *MpcJRSS_Step) CreateMessage() []protocol.StepMessage {
	peerLen := jrss.nodeInfo.NeedQuorum()
	message := make([]protocol.StepMessage, peerLen)
	peerInfo := jrss.nodeInfo.GetPeers()
	for i := 0; i < peerLen; i++ {
		message[i].Msgcode = protocol.MSG_MPCMessage
		message[i].PeerID = peerInfo[i].PeerID
		message[i].Data = append(message[i].Data,&protocol.MpcData{protocol.MPCPolyvalue,jrss.messages[0].GetMessageData(i)})
	}
	return message
}

func (jrss *MpcJRSS_Step) FinishStep( mpc protocol.MpcManager) error {
	err := jrss.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	result := jrss.messages[0].GetResultData()
	err = jrss.mpcResult.SetValue(result.Key,result.Data)
	if err != nil {
		return err
	}
	JRSSvalue := jrss.messages[0].(*generator.RandomPolynomialValue)
	err = jrss.mpcResult.SetValue(protocol.MpcPublicShare, JRSSvalue.RandCoefficient[0])
	if err != nil {
		return err
	}
	return nil
}

func (jrss *MpcJRSS_Step) HandleMessage(msg *protocol.StepMessage) bool {
	seed := jrss.nodeInfo.GetSeed(msg.PeerID)
	if seed == 0 {
		log.Error("MpcJRSS_Step, can't find peer seed. peerID:%s", msg.PeerID)
	}

	jrss.messages[0].SetMessageData(seed,msg.Data[0].Data)
//	log.Error("MpcJRSS_Step.HandleMessage","peerId",msg.PeerID,"stepWaiting",jrss.waiting)
	return true
}
