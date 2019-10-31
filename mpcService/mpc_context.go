package mpcService

import (
	"sync"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/common"
)

type MemStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
	Self uint64 `json:"self"`
}



type MpcContext struct {
	ContextID   common.Hash //Unique id for every content
	quitMu      sync.Mutex
	bQuit       bool
	NodeInfo     protocol.MpcNodeInterface
	mpcResult   protocol.MpcResultInterface
	MpcSteps    []protocol.MpcStepFunc
	MapStepChan map[uint64]chan *protocol.StepMessage
	result      chan<- *protocol.MpcResult
}

func (mpcCtx *MpcContext) SubscribeResult(result chan<- *protocol.MpcResult){
	mpcCtx.result = result
}
func (mpcCtx *MpcContext) getMessage(PeerID *discover.NodeID, msg *protocol.MpcMessage) error {
	var peers []protocol.PeerInfo
	if len(msg.Peers)>0 {
		rlp.DecodeBytes(msg.Peers,&peers)
	}
	mpcCtx.MapStepChan[msg.StepID] <- &protocol.StepMessage{Msgcode:0, PeerID:PeerID, Peers:peers, Data:msg.Data}
	return nil
}

func createMpcContext(contextID common.Hash,peers []protocol.PeerInfo,leader *discover.NodeID, mpcResult protocol.MpcResultInterface) *MpcContext {
	mpc := &MpcContext{
		ContextID:   contextID,
		NodeInfo: 	 NewNodeQuorum(peers,leader),
		bQuit:       false,
		quitMu:      sync.Mutex{},
		mpcResult:   mpcResult,
		MapStepChan: make(map[uint64]chan *protocol.StepMessage),
	}
	return mpc
}
func (mpcCtx *MpcContext) setMpcStep(mpcSteps ...protocol.MpcStepFunc) {
	mpcCtx.MpcSteps = mpcSteps
	for i, step := range mpcSteps {
		mpcCtx.MapStepChan[uint64(i)] = step.GetMessageChan()
	}
}

func (mpcCtx *MpcContext) quit(err error) {
	if err == nil{
		log.Info("MpcContext.quit")
	} else {
//		log.Error("MpcContext.quit,","err", err.Error())
	}

	mpcCtx.quitMu.Lock()
	defer mpcCtx.quitMu.Unlock()
	if mpcCtx.bQuit {
		return
	}
	mpcCtx.bQuit = true
	for i := 0; i < len(mpcCtx.MpcSteps); i++ {
		mpcCtx.MpcSteps[i].Quit(err)
	}
}

func (mpcCtx *MpcContext) mainMPCProcess(MpcManager protocol.MpcManager) error {
//	log.Error("mainMPCProcess begin,","PeerId",MpcManager.SelfNodeId(),"ctxid", mpcCtx.ContextID)
	mpcErr := error(nil)
	for _, mpcCt := range mpcCtx.MpcSteps {
		err := mpcCt.InitMessageLoop(mpcCt)
		if err != nil {
			mpcErr = err
			break
		}
	}

	peerIDs := mpcCtx.NodeInfo.GetPeerIDs()

	if mpcErr == nil {
		mpcCtx.mpcResult.Initialize()
		for i := 0; i < len(mpcCtx.MpcSteps); i++ {
			err := mpcCtx.MpcSteps[i].InitStep()
			if err != nil {
				mpcErr = err
				break
			}

			log.Info("step init finished","ctxid", mpcCtx.ContextID, "stepId", i)
			msg := mpcCtx.MpcSteps[i].CreateMessage()
			if msg != nil {
				for _, item := range msg {
					mpcMsg := &protocol.MpcMessage{ContextID: mpcCtx.ContextID,
						StepID:    uint64(i),
						Data:      item.Data}
						if len(item.Peers)>0{
							mpcMsg.Peers,_ = rlp.EncodeToBytes(item.Peers)
						}
					if item.PeerID != nil {
						go MpcManager.P2pMessage(item.PeerID, item.Msgcode, mpcMsg)
						log.Info("step send a p2p msg.", "ctxid", mpcCtx.ContextID, "stepIdd", i)
					} else {
						if item.Msgcode == protocol.MSG_RequestMPC{
							go MpcManager.BoardcastMessage(nil, item.Msgcode, mpcMsg)
						}else{
							go MpcManager.BoardcastMessage(peerIDs, item.Msgcode, mpcMsg)
						}
						log.Info("step boardcast a p2p msg.", "PeerId",MpcManager.SelfNodeId(), "peers", len(peerIDs))
					}
				}
			}

			log.Info("step send p2p msg finished.", "PeerId",MpcManager.SelfNodeId(),"ctxid", mpcCtx.ContextID, "stepIdd", i)
			err = mpcCtx.MpcSteps[i].FinishStep(MpcManager)
			if err != nil {
				mpcErr = err
				break
			}

			log.Info("step message finished.", "PeerId",MpcManager.SelfNodeId(),"ctxid", mpcCtx.ContextID, "stepId", i)
		}
	}

	if mpcErr != nil {
		log.Error("mainMPCProcess fail.","Error", mpcErr.Error())
		mpcMsg := &protocol.MpcMessage{ContextID: mpcCtx.ContextID,
			StepID: 0,
			Error: mpcErr.Error()}
		go MpcManager.BoardcastMessage(peerIDs, protocol.MSG_MPCError, mpcMsg)
		if mpcCtx.result!= nil {
			mpcCtx.result <- nil
		}
	}

	mpcCtx.quit(nil)
	log.Info("MpcContext finished.", "ctx ID", mpcCtx.ContextID)
	if mpcCtx.result!= nil {
		hash,_ := mpcCtx.mpcResult.GetValue(protocol.MpcTxHash)
		value,err := mpcCtx.mpcResult.GetValue(protocol.MpcContextResult)
		txHash := common.Hash{}
		if hash != nil {
			txHash = common.BytesToHash(hash.([]byte))
		}
		if err == nil{
			mpcCtx.result <- &protocol.MpcResult{txHash,value.([]byte)}
		}else{
			mpcCtx.result <- nil
		}
	}
	return mpcErr
}
