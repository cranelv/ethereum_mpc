package mpcService

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/step"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

//send create LockAccount from leader
func requestCreateLockAccountMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash,peers,leader, result)
	requestMpc := step.CreateRequestMpcStep(result,mpc.NodeInfo, protocol.MpcCreateLockAccountLeader)
	mpcReady := step.CreateMpcReadyStep(result,mpc.NodeInfo)
	return generateCreateLockAccountMpc(mpc,result, requestMpc, mpcReady)

}

//get message from leader and create Context
func acknowledgeCreateLockAccountMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	log.Info("acknowledgeCreateLockAccountMpc begin.")

	result := createMpcBaseMpcResult()
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash, peers, leader,result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(result,mpc.NodeInfo, protocol.MpcCreateLockAccountPeer)
	mpcReady := step.CreateGetMpcReadyStep(result,mpc.NodeInfo)
	return generateCreateLockAccountMpc(mpc, result,AcknowledgeMpc, mpcReady)
}

func generateCreateLockAccountMpc(mpc *MpcContext,result protocol.MpcResultInterface, firstStep protocol.MpcStepFunc, readyStep protocol.MpcStepFunc) (*MpcContext, error) {
	JRSS := step.CreateMpcJRSS_Step(result,protocol.MPCDegree, mpc.NodeInfo)
	PublicKey := step.CreateMpcAddressStep(result,mpc.NodeInfo)
	ackAddress := step.CreateAckMpcAccountStep(result,mpc.NodeInfo)
	mpc.setMpcStep(firstStep, readyStep, JRSS, PublicKey, ackAddress)
	return mpc, nil
}
