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
	"fmt"


	"github.com/ethereum/go-ethereum/consensus_pbft/util"
	"github.com/ethereum/go-ethereum/consensus_pbft/peer"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
)



const (
	// DefaultConsensusQueueSize value of 1000
	DefaultConsensusQueueSize int = 1000
)

// ConsensusHandler handles consensus messages.
// It also implements the Stack.
type ConsensusHandler struct {
	peer.MessageHandler
	consenterChan chan *util.Message
	coordinator   peer.MessageHandlerCoordinator
}

// NewConsensusHandler constructs a new MessageHandler for the plugin.
// Is instance of peer.HandlerFactory
func NewConsensusHandler(coord peer.MessageHandlerCoordinator,
	stream peer.ChatStream, initiatedStream bool) (peer.MessageHandler, error) {
	/*
	peerHandler, err := peer.NewPeerHandler(coord, stream, initiatedStream)
	if err != nil {
		return nil, fmt.Errorf("Error creating PeerHandler: %s", err)
	}
	*/
	handler := &ConsensusHandler{
//		MessageHandler: peerHandler,
		coordinator:    coord,
	}

	consensusQueueSize := 100//viper.GetInt("peer.validator.consensus.buffersize")

	if consensusQueueSize <= 0 {
		log.Error("peer.validator.consensus.buffersize is set to %d, but this must be a positive integer, defaulting to %d", consensusQueueSize, DefaultConsensusQueueSize)
		consensusQueueSize = DefaultConsensusQueueSize
	}

	handler.consenterChan = make(chan *util.Message, consensusQueueSize)
	getEngineImpl().consensusFan.AddFaninChannel(handler.consenterChan)

	return handler, nil
}

// HandleMessage handles the incoming Fabric messages for the Peer
func (handler *ConsensusHandler) HandleMessage(msg *message.Message) error {
	if msg.Type == message.Message_CONSENSUS {
		senderPE, _ := handler.To()
		select {
		case handler.consenterChan <- &util.Message{
			Msg:    msg,
			Sender: senderPE.GetPeerId(),
		}:
			return nil
		default:
			err := fmt.Errorf("Message channel for %v full, rejecting", senderPE.GetPeerId())
			log.Error("Failed to queue consensus message because: %v", err)
			return err
		}
	}

//	if logger.IsEnabledFor(logging.DEBUG) {
		log.Debug("Did not handle message of type %s, passing on to next MessageHandler", msg.Type)
//	}
	return handler.MessageHandler.HandleMessage(msg)
}
