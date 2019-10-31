package mpcService

import (
	"testing"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"time"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
)
var testPath = "/home/cranelv/work/test/"
var emptyHash = common.Hash{}
func newDistributor(peers int)[]*MpcDistributor{
	var mpcs []*MpcDistributor
	ts := newTestNetWork(peers)
	i := 0
	for _,peer := range ts.network{
		dir := fmt.Sprintf("%speer%d/",testPath,i)
		mpcs = append(mpcs,CreateMpcDistributor(dir,peer,"1111111111"))
		mpcs[i].Start()
		i++
	}
	return mpcs
}
func TestNewDistributor(t *testing.T){
	log.InitLog(2)
	mpcs := newDistributor(21)
	mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
		StepID:    uint64(1),
		Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(10)}},}
//	mpcs[0].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
//	mpcs[1].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
//	mpcs[2].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
	go mpcs[2].BoardcastMessage(nil,protocol.MSG_RequestPrepare,mpcMsg)
	time.Sleep(20*time.Second)
}
func TestMessageDistributor(t *testing.T){
	log.InitLog(5)
	mpcs := newDistributor(21)
	for i := 0;i<len(mpcs);i++{
		mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
			StepID:    uint64(1),
			Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(i+10)}},}
		go mpcs[i].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
	}
	time.Sleep(20*time.Second)
}
func TestDistributorMsg(t *testing.T)  {
	log.InitLog(5)
	mpcs := newDistributor(21)
	mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
		StepID:    uint64(1),
		Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(10)}},}
	for _,mpc := range mpcs {
		go mpc.BoardcastMessage(nil,protocol.MSG_RequestPrepare,mpcMsg)
	}
	time.Sleep(20*time.Second)
}
func TestDistributorEnough(t *testing.T)  {
	log.InitLog(5)
	mpcs := newDistributor(21)
	for i := 0;i<len(mpcs);i++{
		mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
			StepID:    uint64(1),
			Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(i+10)}},}
		go mpcs[i].BoardcastMessage(nil,protocol.MSG_RequestPrepare,mpcMsg)
	}
	time.Sleep(30*time.Second)
}
