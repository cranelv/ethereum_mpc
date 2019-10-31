package step

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"time"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
)

type BaseStep struct {
	nodeInfo protocol.MpcNodeInterface
	mpcResult protocol.MpcResultInterface
	msgChan chan *protocol.StepMessage
	msgFilter     map[common.Hash]bool
	finish  chan error
	waiting int

}

func CreateBaseStep(result protocol.MpcResultInterface, nodeInfo protocol.MpcNodeInterface, wait int, bFilter bool) *BaseStep {
	step := &BaseStep{mpcResult:result,nodeInfo: nodeInfo, msgChan: make(chan *protocol.StepMessage, wait+3), finish: make(chan error, 3)}
	step.waiting = wait
	if bFilter {
		step.msgFilter = make(map[common.Hash]bool)
	}
	return step
}

func (step *BaseStep) InitMessageLoop(msger protocol.GetMessageInterface) error {
	log.Info("BaseStep.InitMessageLoop begin")
	if step.waiting <= 0 {
		step.finishMessage(nil)
	} else {
		go func() {
			log.Info("InitMessageLoop begin")

			for {
//				err := step.HandleMessage(msger)
				if step.HandleMessage(msger) {
//					if err != protocol.ErrQuit {
//						log.Error("InitMessageLoop fail, get message err, err:%s", err.Error())
//					}

					break
				}
			}
		}()
	}

	return nil
}
func (step *BaseStep) quitMessage(){
	select {
	case step.msgChan <- nil:
	default:
	}
}
func (step *BaseStep) finishMessage(err error){
	select {
	case step.finish <- err:
	default:
	}
}
func (step *BaseStep) Quit(err error) {
	step.quitMessage()
	step.finishMessage(err)
}

func (step *BaseStep) FinishStep() error {
	select {
	case err := <-step.finish:
		if err != nil {
//			log.Error("BaseStep.FinishStep, get a step finish error.","error", err.Error())
		}

		step.quitMessage()
		return err
	case <-time.After(protocol.MPCTimeOut):
//		log.Error("BaseStep.FinishStep, wait step finish timeout")
		step.quitMessage()
		return protocol.ErrTimeOut
	}
}

func (step *BaseStep) GetMessageChan() chan *protocol.StepMessage {
	return step.msgChan
}

func (step *BaseStep) HandleMessage(msger protocol.GetMessageInterface) bool {
	var msg *protocol.StepMessage
	select {
	case msg = <-step.msgChan:
		if msg == nil {
			log.Info("BaseStep get a quit msg")
			return true
		}
		if step.msgFilter != nil {
			_, exist := step.msgFilter[common.BytesToHash(msg.PeerID[:32])]
			if exist {
				log.Error("BaseStep.HandleMessage, get message from peerID fail", "peer", msg.PeerID)
				return false
			}
			step.msgFilter[common.BytesToHash(msg.PeerID[:32])] = true
		}
		if step.waiting > 0 && msger.HandleMessage(msg) {
			step.waiting--
			if step.waiting <= 0 {
				step.finishMessage(nil)
				return true
			}
		}
	}

	return false
}
