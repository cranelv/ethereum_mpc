package mpcService

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/step"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

//send create LockAccount from leader
func requestTxSignMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash, peers,leader, result)
	requestMpc := step.CreateRequestMpcStep(result,mpc.NodeInfo, protocol.MpcTXSignLeader)
	mpcReady := step.CreateMpcReadyStep(result,mpc.NodeInfo)
	return generateTxSignMpc(mpc,result, requestMpc, mpcReady)
}

//get message from leader and create Context
func acknowledgeTxSignMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash, peers,leader, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(result,mpc.NodeInfo, protocol.MpcTXSignPeer)
	mpcReady := step.CreateGetMpcReadyStep(result,mpc.NodeInfo)
	return generateTxSignMpc(mpc,result, AcknowledgeMpc, mpcReady)
}

func generateTxSignMpc(mpc *MpcContext,result protocol.MpcResultInterface, firstStep protocol.MpcStepFunc, readyStep protocol.MpcStepFunc) (*MpcContext, error) {
	log.Info("generateTxSignMpc begin")

	JRJZ := step.CreateTXSignJR_JZ_Step(result,protocol.MPCDegree, mpc.NodeInfo)

	preKeys := []string{protocol.MpcSignA0}
	ResultKeys := []string{protocol.MpcSignAPoint}
	AGPoint := step.CreateMpcPoint_Step(result,mpc.NodeInfo, preKeys, ResultKeys)
	
	lagStepPreValueKeys := []string{protocol.MpcSignARSeed}
	lagStepResultKeys := []string{protocol.MpcSignARResult}
	ARLag := step.CreateTXSign_Lagrange_Step(result,mpc.NodeInfo, lagStepPreValueKeys, lagStepResultKeys)

	TXSignLag := step.CreateTxSign_CalSignStep(result,mpc.NodeInfo, protocol.MpcTxSignResult)
	mpc.setMpcStep(firstStep, readyStep, JRJZ, AGPoint, ARLag, TXSignLag)
	return mpc, nil
}
