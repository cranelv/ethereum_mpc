package mpcService

import (
	"testing"
	"math/rand"
	"time"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"encoding/binary"
)

func TestFind(t *testing.T) {
	rand.Seed(int64(time.Now().Second()))
	peerAry := make([]protocol.PeerInfo,0)
	for i:=0;i<10000;i++{
		peerInfo := protocol.PeerInfo{nil,rand.Uint64()}
		if peerInfo.Seed == 0{
			peerInfo.Seed = 1
		}
		index,exist := find(peerInfo.Seed,peerAry)
		if exist {
			if peerAry[index].Seed != peerInfo.Seed{
				t.Error("find result false")
			}
			continue
		}else if index < len(peerAry){
			if peerAry[index].Seed <= peerInfo.Seed{
				t.Error("find result false")
			}
		}
		insert(&peerAry,index,peerInfo)
		if peerAry[index].Seed != peerInfo.Seed {
			t.Error("Insert Error")
		}
	}
	for i:=0;i<99;i++{
		if peerAry[i].Seed>peerAry[i+1].Seed{
			t.Error("Insert Error")
		}
	}
}
func TestNodeInfo(t *testing.T) {
	rand.Seed(int64(time.Now().Second()))
	len := 10000
	key := keystore.MpcKey{MPCGroup:make([]uint64,len)}
	peers:=make([]*discover.NodeID,len)
	for i:=0;i<len;i++{
		key.MPCGroup[i] = rand.Uint64()
		peers[i] = &discover.NodeID{}
		binary.BigEndian.PutUint64(peers[i][:],key.MPCGroup[i])
	}
	hash := common.Hash{}
	nodeInfo := NewNodeCollection(hash,peers,peers[0],&key)
	for i:=0;i<len;i++{
		go nodeInfo.AddNode(key.MPCGroup[i],peers[i])
	}
	time.Sleep(time.Second)
	for i:=0;i<len;i++{
		if nodeInfo.Seeds[i].PeerID == nil{
			t.Error("Insert Error")
		}
	}
}
