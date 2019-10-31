package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/step/generator"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	crypto "github.com/ethereum/go-ethereum/mpcService/crypto"
)

type TXSignJR_JZ_Step struct {
	BaseMpcStep
}

func CreateTXSignJR_JZ_Step(result protocol.MpcResultInterface,degree int, nodeInfo protocol.MpcNodeInterface) *TXSignJR_JZ_Step {

	mpc := &TXSignJR_JZ_Step{*CreateBaseMpcStep(result,nodeInfo, 4,true),}
	mpc.messages[0] = generator.CreateJRSSValue(degree, nodeInfo.NeedQuorum(),protocol.MpcSignA)
	mpc.messages[1] = generator.CreateJRSSValue(degree, nodeInfo.NeedQuorum(),protocol.MpcSignR)
	mpc.messages[2] = generator.CreateJZSSValue(degree*2, nodeInfo.NeedQuorum(),protocol.MpcSignB)
	mpc.messages[3] = generator.CreateJZSSValue(degree*2, nodeInfo.NeedQuorum(),protocol.MpcSignC)

	return mpc
}

func (jrjz *TXSignJR_JZ_Step) CreateMessage() []protocol.StepMessage {
	log.Info("TXSignJR_JZ_Step.CreateMessage begin")

	peerNum := jrjz.nodeInfo.NeedQuorum()
	message := make([]protocol.StepMessage, peerNum)

	peers := jrjz.nodeInfo.GetPeers()
	for i := 0; i < peerNum; i++ {
		message[i].Msgcode = protocol.MSG_MPCMessage
		message[i].PeerID = peers[i].PeerID
		message[i].Data = append(message[i].Data,
			&protocol.MpcData{protocol.MpcSignASeed,jrjz.messages[0].GetMessageData(i)},
			&protocol.MpcData{protocol.MpcSignRSeed,jrjz.messages[1].GetMessageData(i)},
			&protocol.MpcData{protocol.MpcSignBSeed,jrjz.messages[2].GetMessageData(i)},
			&protocol.MpcData{protocol.MpcSignCSeed,jrjz.messages[3].GetMessageData(i)})
	}
	return message
}


func (jrjz *TXSignJR_JZ_Step) HandleMessage(msg *protocol.StepMessage) bool {
	log.Info("TXSignJR_JZ_Step.HandleMessage, PeerID:%s, DataLen:%d", msg.PeerID.String(), len(msg.Data))
	seed := jrjz.nodeInfo.GetSeed(msg.PeerID)
	if seed == 0 {
		log.Error("TXSignJR_JZ_Step.HandleMessage, get seed fail.","peer", msg.PeerID)
	}

	if len(msg.Data) != 4 {
		log.Error("TXSignJR_JZ_Step HandleMessage, received data len doesn't match requirement, dataLen:%d", len(msg.Data))
		return false
	}
	jrjz.messages[0].SetMessageData(seed,msg.Data[0].Data)
	jrjz.messages[1].SetMessageData(seed,msg.Data[1].Data)
	jrjz.messages[2].SetMessageData(seed,msg.Data[2].Data)
	jrjz.messages[3].SetMessageData(seed,msg.Data[3].Data)
	return true
}


func (jrjz *TXSignJR_JZ_Step) FinishStep(mpc protocol.MpcManager) error {
	err := jrjz.BaseMpcStep.FinishStep()
	if err != nil {
		log.Error("TXSignJR_JZ_Step.BaseMpcStep.FinishStep fail, err:%s", err.Error())
		return err
	}

	result0 := jrjz.messages[0].GetResultData()
	jrjz.mpcResult.SetValue(result0.Key,result0.Data)
	result1 := jrjz.messages[1].GetResultData()
	jrjz.mpcResult.SetValue(result1.Key,result1.Data)
	result2 := jrjz.messages[2].GetResultData()
	jrjz.mpcResult.SetValue(result2.Key,result2.Data)
	result3 := jrjz.messages[3].GetResultData()
	jrjz.mpcResult.SetValue(result3.Key,result3.Data)
	msgA := jrjz.messages[0].(*generator.RandomPolynomialValue)
	jrjz.mpcResult.SetValue(protocol.MpcSignA0, msgA.RandCoefficient[0])
	msgR := jrjz.messages[1].(*generator.RandomPolynomialValue)
	jrjz.mpcResult.SetValue(protocol.MpcSignR0, msgR.RandCoefficient[0])
	ar := new(big.Int).Mul(result0.Data.(*big.Int), result1.Data.(*big.Int))
	ar.Mod(ar, crypto.Secp256k1N)
	ar.Add(ar, result2.Data.(*big.Int))
	ar.Mod(ar, crypto.Secp256k1N)
	err = jrjz.mpcResult.SetValue(protocol.MpcSignARSeed, ar)
//	log.Error("ARSeed","seed1",ar)
	log.Info("TXSignJR_JZ_Step.FinishStep succeed")
	return nil
}
