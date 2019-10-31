/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helper

import (
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/peer"

	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/consensus_pbft/controller"
	"github.com/ethereum/go-ethereum/consensus_pbft/util"
	"golang.org/x/net/context"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)

// EngineImpl implements a struct to hold consensus.Consenter, PeerEndpoint and MessageFan
type EngineImpl struct {
	consenter    consensus_pbft.Consenter
	helper       *Helper
	peer         pbftTypes.Peer
	consensusFan *util.MessageFan
	txExecute message.TaskExecuteInterface
}

// GetHandlerFactory returns new NewConsensusHandler
func (eng *EngineImpl) GetHandlerFactory() peer.HandlerFactory {
	return NewConsensusHandler
}

// ProcessTransactionMsg processes a Message in context of a Transaction
func (eng *EngineImpl) ProcessTaskMsg(msg *message.Message, task *message.Task) (response *peer.Response) {
	//TODO: Do we always verify security, or can we supply a flag on the invoke ot this functions so to bypass check for locally generated transactions?
	if true /*tx.Type == pb.Transaction_CHAINCODE_QUERY*/ {
		if !engine.helper.valid {
			singletons.Log.Warn("Rejecting query because state is currently not valid")
			return &peer.Response{Status: peer.Response_FAILURE,
				Msg: []byte("Error: state may be inconsistent, cannot query")}
		}

		// The secHelper is set during creat ChaincodeSupport, so we don't need this step
		// cxt := context.WithValue(context.Background(), "security", secHelper)
		cxt := context.Background()
		//query will ignore events as these are not stored on ledger (and query can report
		//"event" data synchronously anyway)
		result, err := eng.txExecute.Execute(cxt, task)
		if err != nil {
			response = &peer.Response{Status: peer.Response_FAILURE,
				Msg: []byte(fmt.Sprintf("Error:%s", err))}
		} else {
			response = &peer.Response{Status: peer.Response_SUCCESS, Msg: result.Paload}
		}
	} else {
		// Chaincode Transaction
		response = &peer.Response{Status: peer.Response_SUCCESS, Msg: task.Paload}

		//TODO: Do we need to verify security, or can we supply a flag on the invoke ot this functions
		// If we fail to marshal or verify the tx, don't send it to consensus plugin
		if response.Status == peer.Response_FAILURE {
			return response
		}

		// Pass the message to the consenter (eg. PBFT) NOTE: Make sure engine has been initialized
		if eng.consenter == nil {
			return &peer.Response{Status: peer.Response_FAILURE, Msg: []byte("Engine not initialized")}
		}
		// TODO, do we want to put these requests into a queue? This will block until
		// the consenter gets around to handling the message, but it also provides some
		// natural feedback to the REST API to determine how long it takes to queue messages
		err := eng.consenter.RecvMsg(msg, eng.peer.GetPeerId())
		if err != nil {
			response = &peer.Response{Status: peer.Response_FAILURE, Msg: []byte(err.Error())}
		}
	}
	return response
}

func (eng *EngineImpl) setConsenter(consenter consensus_pbft.Consenter) *EngineImpl {
	eng.consenter = consenter
	return eng
}

func (eng *EngineImpl) setNode(peer pbftTypes.Peer) *EngineImpl {
	eng.peer = peer
	return eng
}

var engineOnce sync.Once

var engine *EngineImpl

func getEngineImpl() *EngineImpl {
	return engine
}

// GetEngine returns initialized peer.Engine
func GetEngine(coord peer.MessageHandlerCoordinator) (peer.Engine, error) {
	var err error
	engineOnce.Do(func() {
		engine = new(EngineImpl)
		engine.helper = NewHelper(coord)
		engine.consenter = controller.NewConsenter(engine.helper)
		engine.helper.setConsenter(engine.consenter)
		engine.peer = coord.(pbftTypes.Peer)
		engine.consensusFan = util.NewMessageFan()

		go func() {
			log.Debug("Starting up message thread for consenter")

			// The channel never closes, so this should never break
			for msg := range engine.consensusFan.GetOutChannel() {
				engine.consenter.RecvMsg(msg.Msg, msg.Sender)
			}
		}()
	})
	return engine, err
}
