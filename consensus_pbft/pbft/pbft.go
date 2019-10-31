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

package pbft

import (
	"github.com/ethereum/go-ethereum/consensus_pbft"
	"github.com/ethereum/go-ethereum/consensus_pbft/params"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	"github.com/ethereum/go-ethereum/consensus_pbft/message"
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/consensusInterface"
)

const configPrefix = "CORE_PBFT"

var pluginInstance consensus_pbft.Consenter // singleton service
var config *params.Config

func init() {
	config = loadConfig()
}

// GetPlugin returns the handle to the Consenter singleton
func GetPlugin(c consensus_pbft.Stack) consensus_pbft.Consenter {
	if pluginInstance == nil {
		pluginInstance = New(c)
	}
	return pluginInstance
}
func NewValidatorIdentify(stack consensus_pbft.Stack) consensusInterface.ValidatorIdentifyInterface{
	return nil
}
//调用controller获取一个plugin，当选择是pbft算法时，它会调用pbft.go 里的 GetPlugin(c consensus.Stack)方法，
// 在pbft.go里面把所有的外部参数读进算法内部。
// New creates a new Obc* instance that provides the Consenter interface.
// Internally, it uses an opaque pbft-core instance.
func New(stack consensus_pbft.Stack) consensus_pbft.Consenter {
	handle, _, _ := stack.GetNetworkNodeIDs()
	identify := NewValidatorIdentify(stack)
	id,err := identify.GetValidatorID(handle)
	if err != nil {
		singletons.Log.Error(err)
	}
	return newObcBatch(id, config, stack)
//	id, _ := getValidatorID(handle)

//	switch strings.ToLower(config.GetString("general.mode")) {
//	case "batch":
//	default:
//		panic(fmt.Errorf("Invalid PBFT mode: %s", config.GetString("general.mode")))
//	}
}

func loadConfig() (config *params.Config) {
	return params.DefaultConfig()
	/*
	config = viper.New()

	// for environment variables
	config.SetEnvPrefix(configPrefix)
	config.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)

	config.SetConfigName("config")
	config.AddConfigPath("./")
	config.AddConfigPath("../consensus/pbft/")
	config.AddConfigPath("../../consensus/pbft")
	// Path to look for the config file in based on GOPATH
	gopath := os.Getenv("GOPATH")
	for _, p := range filepath.SplitList(gopath) {
		pbftpath := filepath.Join(p, "src/github.com/hyperledger/fabric/consensus/pbft")
		config.AddConfigPath(pbftpath)
	}

	err := config.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error reading %s plugin config: %s", configPrefix, err))
	}
	return
	*/
}
/*
// Returns the uint64 ID corresponding to a peer handle
func getValidatorID(handle *discover.NodeID) (id uint64, err error) {
	/ *
	// as requested here: https://github.com/hyperledger/fabric/issues/462#issuecomment-170785410
	if startsWith := strings.HasPrefix(handle.Name, "vp"); startsWith {
		id, err = strconv.ParseUint(handle.Name[2:], 10, 64)
		if err != nil {
			return id, fmt.Errorf("Error extracting ID from \"%s\" handle: %v", handle.Name, err)
		}
		return
	}

	err = fmt.Errorf(`For MVP, set the VP's peer.id to vpX,
		where X is a unique integer between 0 and N-1
		(N being the maximum number of VPs in the network`)
	* /
	return
}

// Returns the peer handle that corresponds to a validator ID (uint64 assigned to it for PBFT)
func getValidatorHandle(id uint64) (handle *discover.Node, err error) {
	// as requested here: https://github.com/hyperledger/fabric/issues/462#issuecomment-170785410
//	name := "vp" + strconv.FormatUint(id, 10)
//	return &pb.PeerID{Name: name}, nil
	return nil,nil
}

// Returns the peer handles corresponding to a list of replica ids
func getValidatorHandles(ids []uint64) (handles []*discover.Node) {
	handles = make([]*discover.Node, len(ids))
	for i, id := range ids {
		handles[i], _ = getValidatorHandle(id)
	}
	return
}
*/
type obcGeneric struct {
	NetValidator
	stack consensus_pbft.Stack
	pbft  *pbftCore
}
func (op *obcGeneric)findPeer(handle *pbftTypes.PeerID)(int,pbftTypes.Peer) {
	_,peers,_ := op.stack.GetNetworkNodes()
	for i,peer := range peers{
		if *peer.GetPeerId() == *handle{
			return i,peer
		}
	}
	return -1,nil
}
func (op *obcGeneric)ChangeToReplicaID(index int) pbftTypes.ReplicaID {
	return pbftTypes.ReplicaID(index)
}
func (op *obcGeneric)ChangeToIndex(id pbftTypes.ReplicaID) int {
	return int(id)
}

func (op *obcGeneric) skipTo(seqNo uint64, resultBuffer []byte, ids []pbftTypes.ReplicaID) {
	state := &message.StateInfo{}
	err := singletons.Marshaler.Unmarshal(resultBuffer,state)
//	err := proto.Unmarshal(id, info)
	if err != nil {
		singletons.Log.Errorf("Error unmarshaling: %s", err)
		return
	}
	peers,err := op.GetValidatorNodeIds(ids)
	if err != nil {
		singletons.Log.Errorf("Error getValidatorNodeIds: %s", err)
		return
	}
	op.stack.UpdateState(&checkpointMessage{seqNo, resultBuffer}, state, peers)
}

func (op *obcGeneric) invalidateState() {
	op.stack.InvalidateState()
}

func (op *obcGeneric) validateState() {
	op.stack.ValidateState()
}

func (op *obcGeneric) getState() []byte {
	return op.stack.GetBlockchainInfoBlob()
}

func (op *obcGeneric) getLastSeqNo() (uint64, error) {
	/*
	raw, err := op.stack.GetBlockHeadMetadata()
	if err != nil {
		return 0, err
	}
	meta := &Metadata{}
	proto.Unmarshal(raw, meta)
	return meta.SeqNo, nil
	*/
	return uint64(1),nil
}
