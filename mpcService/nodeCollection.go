package mpcService

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
	"sync"
	"github.com/pkg/errors"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)
func find (k uint64,info []protocol.PeerInfo) (int,bool) {
	left, right, mid := 0, len(info)-1, 0
	if right < 0 {
		return 0,false
	}
	for {
		mid = (left + right) / 2
		if info[mid].Seed > k{
			right = mid -1
		}else if info[mid].Seed<k{
			left = mid + 1
		}else{
			return mid,true
		}
		if left > right{
			return left, false
		}
	}
	return mid,false
}
func insert (info *[]protocol.PeerInfo,index int, peerInfo protocol.PeerInfo){
	*info = append(*info,peerInfo)
	end := len(*info) -1
	for i:=end;i>index;i--{
		(*info)[i],(*info)[i-1] = (*info)[i-1],(*info)[i]
	}
}
type NodeCollection struct {
	mu sync.RWMutex
	address common.MpcAddress
	hash common.Hash
	leader discover.NodeID
	Quorum int
	state protocol.MpcState
	haveSeed bool
	Seeds []protocol.PeerInfo
	Collection 	map[discover.NodeID]uint64
}
func NewNodeCollection(hash common.Hash,peers []*discover.NodeID,leader *discover.NodeID,key *keystore.MpcKey)*NodeCollection{
	nodecol := NodeCollection{
		Collection:make(map[discover.NodeID]uint64),
		hash:hash,
	}
	if leader != nil {
		nodecol.leader = *leader
	}
	for _,peer := range peers {
		nodecol.Collection[*peer] = 0
	}
	if key == nil {
		nodecol.Quorum = len(peers)
	}else{
		nodecol.InitfromKey(key)
	}
	return &nodecol
}
func NewNodeQuorum(peers []protocol.PeerInfo,leader *discover.NodeID)*NodeCollection{
	nodecol := NodeCollection{
		Collection:make(map[discover.NodeID]uint64),
		Quorum:len(peers),
		Seeds : make([]protocol.PeerInfo,0,len(peers)),
	}
	if leader != nil {
		nodecol.leader = *leader
	}
	for _,peer := range peers {
		nodecol.AddNode(peer.Seed,peer.PeerID)
		index,exist := find(peer.Seed,nodecol.Seeds)
		if exist{
			log.Error("node seeds duplication!")
			return nil
		}
		insert(&nodecol.Seeds,index,protocol.PeerInfo{peer.PeerID,peer.Seed})
		nodecol.Collection[*peer.PeerID]=peer.Seed
	}
	return &nodecol
}
func (nc*NodeCollection)Address()*common.MpcAddress{
	return &nc.address
}
func (nc*NodeCollection)Hash()*common.Hash{
	return &nc.hash
}
func (nc*NodeCollection)Leader()*discover.NodeID{
	return &nc.leader
}
func (nc*NodeCollection)NeedQuorum()int{
	return nc.Quorum
}
func (nc*NodeCollection)InitfromKey(key *keystore.MpcKey)  {
	nc.address = key.Address
	nc.Seeds = make([]protocol.PeerInfo,0,len(key.MPCGroup))
	for i:=0;i<len(key.MPCGroup);i++{
		index,exist := find(key.MPCGroup[i],nc.Seeds)
		if exist {
			log.Error("node seeds duplication!")
			return
		}
		insert(&nc.Seeds,index,protocol.PeerInfo{nil,key.MPCGroup[i]})
	}
	nc.Quorum = int(key.Quorum)
	nc.haveSeed = true
}
func (nc*NodeCollection) GetState()protocol.MpcState{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.state
}
func (nc*NodeCollection) SetState(state protocol.MpcState){
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.state = state
}
func (nc*NodeCollection) FetchQuorum()bool{
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if nc.isQuorum() {
		nc.state = protocol.MpcRunning
		return true
	}
	return false
}
func (nc*NodeCollection) RunNode(seed uint64,nodeId *discover.NodeID)protocol.MpcState{
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if nc.state == protocol.MpcCollection || nc.state == protocol.MpcWaiting{
		index,exist := find(seed,nc.Seeds)
		if exist && nc.Seeds[index].PeerID != nil && *nc.Seeds[index].PeerID == *nodeId{
			nc.state = protocol.MpcRunning
			return protocol.MpcCollection
		}else{
			if exist{
				if(nc.Seeds[index].PeerID == nil){
					log.Error("MpcNotFound","self",nodeId,"seed",seed,"hash",nc.hash)
				}else{
					log.Error("MpcNotFound","found",nc.Seeds[index].PeerID,"self",nodeId)
				}
			}else{
				log.Error("MpcNotFound","seed",seed,"self",nodeId)
			}
			return protocol.MpcNotFound
		}
	}
	return nc.state
}
func (nc*NodeCollection) isQuorum()bool{
	if(nc.state == protocol.MpcCollection){
		count := 0
		for i:=0;i<len(nc.Seeds);i++{
			if nc.Seeds[i].PeerID != nil {
				count++
			}
		}
//		log.Error("NodeCollection","count",count,"quorum",nc.Quorum)
		return count >= nc.Quorum
	}
	return false
}
func (nc*NodeCollection) AddNode(seed uint64,nodeId *discover.NodeID) error{
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if se,exist := nc.Collection[*nodeId];exist{
		if se == seed {
			return nil
		}else if se > 0{
			return errors.New("node send different seed!")
		}

	}else{
		return errors.New("Error node ID")
	}
	index,exist := find(seed,nc.Seeds)
	if nc.haveSeed{
		if !exist{
			return errors.New("Lost node seed")
		}
	}else if exist{
		return errors.New("node seeds duplication!")
	}
	if !exist{
		insert(&nc.Seeds,index,protocol.PeerInfo{nodeId,seed})
	}else{
		nc.Seeds[index].PeerID = nodeId
	}
	nc.Collection[*nodeId] = seed
	return nil
}
func (nc*NodeCollection) GetSeedsNum(seed uint64,nodeId *discover.NodeID)int{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	count := 0
	for _,node := range nc.Seeds{
		if node.PeerID != nil {
			count++
		}
	}
	return count
}
func (nc*NodeCollection) GetSeed(nodeId *discover.NodeID)uint64{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.Collection[*nodeId]
}
func (nc*NodeCollection) GetNodeId(seed uint64) *discover.NodeID{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	index,exist := find(seed,nc.Seeds)
	if exist {
		return nc.Seeds[index].PeerID
	}
	return nil
}
func (nc*NodeCollection) GetPeers() []protocol.PeerInfo{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	peers := make([]protocol.PeerInfo,0,len(nc.Seeds))
	for i:=0;i<len(nc.Seeds);i++{
		if nc.Seeds[i].PeerID != nil {
			peers = append(peers,nc.Seeds[i])
		}
	}
	return peers
}
func (nc*NodeCollection) GetPeerIDs() []*discover.NodeID{
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	peers := make([]*discover.NodeID,len(nc.Collection))
	i := 0
	for nodeId,_ := range nc.Collection{
		peers[i] = &discover.NodeID{}
		*peers[i] = nodeId
		i++
	}
	return peers
}