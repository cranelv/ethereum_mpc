package mpcService

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/mpcService/step"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type MpcTestCtxFactory struct {
	test bool
	step int
	result *BaseMpcResult
}

func (mf *MpcTestCtxFactory) CreateContext(ctxType uint, hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (MpcInterface, error) {
	switch ctxType {
	case protocol.MpcCreateLockAccountLeader:
		return mf.CreateRequestAccountMpc(hash, peers,leader, preSetValue...)

	case protocol.MpcCreateLockAccountPeer:
		return mf.acknowledgeCreateAccountMpc(hash, peers,leader, preSetValue...)

	case protocol.MpcTXSignLeader:
		return mf.requestTxSignMpc(hash, peers,leader, preSetValue...)

	case protocol.MpcTXSignPeer:
		return mf.acknowledgeTxSignMpc(hash, peers,leader, preSetValue...)
	}

	return nil, protocol.ErrContextType
}

//send create LockAccount from leader
func (mf *MpcTestCtxFactory)CreateRequestAccountMpc(mpcID common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	mf.result = result
	result.Initialize(preSetValue...)
	mpc := createMpcContext(mpcID, peers,leader, result)
	requestMpc := step.CreateRequestMpcStep(result,mpc.NodeInfo, protocol.MpcCreateLockAccountLeader)
	mpcReady := step.CreateMpcReadyStep(result,mpc.NodeInfo)
	return mf.generateCreateTestMpc(mpc,result, requestMpc, mpcReady)
}

//get message from leader and create Context
func (mf *MpcTestCtxFactory)acknowledgeCreateAccountMpc(mpcID common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	findMap := make(map[uint64]bool)
	for _, item := range peers {
		_, exist := findMap[item.Seed]
		if exist {
			return nil, protocol.ErrMpcSeedDuplicate
		}
		findMap[item.Seed] = true
	}

	result := createMpcBaseMpcResult()
	mf.result = result
	result.Initialize(preSetValue...)
	mpc := createMpcContext(mpcID, peers,leader, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(result,mpc.NodeInfo, protocol.MpcCreateLockAccountPeer)
	mpcReady := step.CreateGetMpcReadyStep(result,mpc.NodeInfo)
	return mf.generateCreateTestMpc(mpc, result,AcknowledgeMpc, mpcReady)
}

func (mf *MpcTestCtxFactory)generateCreateTestMpc(mpc *MpcContext,result protocol.MpcResultInterface, firstStep protocol.MpcStepFunc, readyStep protocol.MpcStepFunc) (*MpcContext, error) {
	if mf.test{
		test := 1000
		mpcTest := make([]protocol.MpcStepFunc, test+2)
		mpcTest[0] = firstStep
		mpcTest[1] = readyStep
		for i := 0; i < test; i++ {
			mpcTest[i+2] = step.CreateMpcJRSS_Step(result,protocol.MPCDegree, mpc.NodeInfo)
		}

		mpc.setMpcStep(mpcTest[:mf.step]...)
		return mpc, nil
	}else {
		JRSS := step.CreateMpcJRSS_Step(result,protocol.MPCDegree, mpc.NodeInfo)
		PublicKey := step.CreateMpcAddressStep(result,mpc.NodeInfo)
		ackAddress := step.CreateAckMpcAccountStep(result,mpc.NodeInfo)
		mpcTest := []protocol.MpcStepFunc{firstStep, readyStep, JRSS, PublicKey, ackAddress}
		mpc.setMpcStep(mpcTest[:mf.step]...)
		return mpc, nil
	}
}
func (mf *MpcTestCtxFactory)requestTxSignMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	mf.result = result
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash, peers,leader, result)
	requestMpc := step.CreateRequestMpcStep(result,mpc.NodeInfo, protocol.MpcTXSignLeader)
	mpcReady := step.CreateMpcReadyStep(result,mpc.NodeInfo)
	return mf.generateTxSignMpc(mpc,result, requestMpc, mpcReady)
}

//get message from leader and create Context
func (mf *MpcTestCtxFactory)acknowledgeTxSignMpc(hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	mf.result = result
	result.Initialize(preSetValue...)
	mpc := createMpcContext(hash, peers,leader, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(result,mpc.NodeInfo, protocol.MpcTXSignPeer)
	mpcReady := step.CreateGetMpcReadyStep(result,mpc.NodeInfo)
	return mf.generateTxSignMpc(mpc,result, AcknowledgeMpc, mpcReady)
}

func (mf *MpcTestCtxFactory)generateTxSignMpc(mpc *MpcContext,result protocol.MpcResultInterface, firstStep protocol.MpcStepFunc, readyStep protocol.MpcStepFunc) (*MpcContext, error) {

	JRJZ := step.CreateTXSignJR_JZ_Step(result,protocol.MPCDegree, mpc.NodeInfo)

	preKeys := []string{protocol.MpcSignA0}
	ResultKeys := []string{protocol.MpcSignAPoint}
	AGPoint := step.CreateMpcPoint_Step(result,mpc.NodeInfo, preKeys, ResultKeys)

	lagStepPreValueKeys := []string{protocol.MpcSignARSeed}
	lagStepResultKeys := []string{protocol.MpcSignARResult}
	ARLag := step.CreateTXSign_Lagrange_Step(result,mpc.NodeInfo, lagStepPreValueKeys, lagStepResultKeys)

	TXSignLag := step.CreateTxSign_CalSignStep(result,mpc.NodeInfo, protocol.MpcTxSignResult)
	mpcTest := []protocol.MpcStepFunc{firstStep, readyStep, JRJZ, AGPoint, ARLag, TXSignLag}
	mpc.setMpcStep(mpcTest[:mf.step]...)
	return mpc, nil
}
