package pbft

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/pkg/errors"
	"github.com/ethereum/go-ethereum/consensus_pbft"
)
type PbftPeer struct {
	Node *discover.Node
	Type pbftTypes.Peer_Type
}
func(pp* PbftPeer)GetPeerId() *pbftTypes.PeerID{
	peerID := pbftTypes.PeerID(pp.Node.ID)
	return &peerID
}
func(pp* PbftPeer)GetType() pbftTypes.Peer_Type{
	return pp.Type
}

type PbftInfo struct{
	ReplicaId pbftTypes.ReplicaID
	peer      pbftTypes.Peer
}
type PbftIdentify struct {
	validatorInfo []PbftInfo
}

func (pi* PbftIdentify)GetValidatorID(handle *pbftTypes.PeerID) (pbftTypes.ReplicaID, error) {
	for _,item := range pi.validatorInfo{
		if *item.peer.GetPeerId() == *handle{
			return item.ReplicaId,nil
		}
	}
	return 0,errors.New("Error peer id")
}
func (pi* PbftIdentify)GetValidatorNodeId(id pbftTypes.ReplicaID) (*pbftTypes.PeerID,error){
	for _,item := range pi.validatorInfo{
		if item.ReplicaId == id {
			return item.peer.GetPeerId(),nil
		}
	}
	return nil,errors.New("Error replica id")

}
func (pi* PbftIdentify)GetValidatorNodeIds(ids []pbftTypes.ReplicaID) ([]*pbftTypes.PeerID,error){
	var peerIds []*pbftTypes.PeerID
	for _,id := range ids{
		peerId,err := pi.GetValidatorNodeId(id)
		if err != nil{
			return nil,err
		}
		peerIds = append(peerIds,peerId)
	}
	return peerIds,nil
}
func (pi* PbftIdentify)GetValidatorNode(id pbftTypes.ReplicaID) (pbftTypes.Peer,error){
	for _,item := range pi.validatorInfo{
		if item.ReplicaId == id {
			return item.peer,nil
		}
	}
	return nil,errors.New("Error replica id")
}
func (pi* PbftIdentify)GetValidatorNodes(ids []pbftTypes.ReplicaID) ([]pbftTypes.Peer,error){
	var peers []pbftTypes.Peer
	for _,id := range ids{
		peer,err := pi.GetValidatorNode(id)
		if err != nil{
			return nil,err
		}
		peers = append(peers,peer)
	}
	return peers,nil

}
type ValidatorIndexer interface {
	ChangeToReplicaID(index int) pbftTypes.ReplicaID
	ChangeToIndex(id pbftTypes.ReplicaID) int
}
type NetValidator struct {
	net consensus_pbft.Inquirer
	indexer ValidatorIndexer
}
func (nv *NetValidator)findPeer(handle *pbftTypes.PeerID)(int,pbftTypes.Peer) {
	_,peers,_ := nv.net.GetNetworkNodes()
	for i,peer := range peers{
		if *peer.GetPeerId() == *handle{
			return i,peer
		}
	}
	return -1,nil
}
func (nv *NetValidator)GetValidatorID(handle *pbftTypes.PeerID) (pbftTypes.ReplicaID, error) {
	id,peer := nv.findPeer(handle)
	if peer != nil{
		return nv.indexer.ChangeToReplicaID(id),nil
	}
	return 0,errors.New("Error peer id")
}

func (nv *NetValidator)GetValidatorNodeIds(ids []pbftTypes.ReplicaID) ([]*pbftTypes.PeerID,error){
	var peerIds []*pbftTypes.PeerID
	for _,id := range ids{
		peerId,err := nv.GetValidatorNodeId(id)
		if err != nil{
			return nil,err
		}
		peerIds = append(peerIds,peerId)
	}
	return peerIds,nil
}
func (nv *NetValidator)GetValidatorNode(id pbftTypes.ReplicaID) (pbftTypes.Peer,error){
	index := nv.indexer.ChangeToIndex(id)
	_,peers,_ := nv.net.GetNetworkNodes()
	if index>=0 && index < len(peers){
		return peers[index],nil
	}
	return nil,errors.New("Error replica id")
}
func (nv *NetValidator)GetValidatorNodes(ids []pbftTypes.ReplicaID) ([]pbftTypes.Peer,error){
	var peers []pbftTypes.Peer
	for _,id := range ids{
		peer,err := nv.GetValidatorNode(id)
		if err != nil{
			return nil,err
		}
		peers = append(peers,peer)
	}
	return peers,nil

}
func (nv *NetValidator)GetValidatorNodeId(id pbftTypes.ReplicaID) (*pbftTypes.PeerID,error){
	peer,err := nv.GetValidatorNode(id)
	if err == nil{
		return peer.GetPeerId(),nil
	}
	return nil,errors.New("Error replica id")

}
