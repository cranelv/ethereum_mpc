package mpcService

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p"
	"bytes"
	"github.com/ethereum/go-ethereum/rlp"
	"time"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type testPeer struct {
	self discover.Node
	message chan protocol.PeerMessage
	netWork *testP2pNetWork
}
func newTestPeer(selfId *discover.NodeID,network *testP2pNetWork)*testPeer{
	test := &testPeer{message:make(chan protocol.PeerMessage,100),netWork:network}
	test.self.ID =*selfId
	return test
}
func (ts *testPeer) SendToPeer(to *discover.NodeID, code uint64,msg interface{}) error {
	return ts.netWork.SendToPeer(&ts.self.ID,to,code,msg)
}
func (ts *testPeer) MessageChannel() <-chan protocol.PeerMessage{
	return ts.message
}
func (ts *testPeer) GetPeers()[]*discover.NodeID{
	return ts.netWork.GetPeers()
}
func (ts *testPeer) Self() *discover.Node{
	return &ts.self
}
func (ts *testPeer) IsActivePeer(*discover.NodeID) bool {
	return true
}
type testP2pNetWork struct {
	network map[discover.NodeID]*testPeer
}
func (ts *testP2pNetWork) SendToPeer(from,to *discover.NodeID, code uint64,msg interface{}) error {
	payload,err := rlp.EncodeToBytes(msg)
	if err!= nil {
		log.Error(err.Error())
		return err
	}
	ts.network[*to].message <- protocol.PeerMessage{*from,&p2p.Msg{code,uint32(len(payload)),bytes.NewReader(payload),time.Now()}}
	return nil
}
func (ts *testP2pNetWork) GetPeers()[]*discover.NodeID{
	peers := make([]*discover.NodeID,len(ts.network))
	i := 0
	for nodeid,_ := range ts.network {
		peers[i] = &discover.NodeID{}
		*peers[i] = nodeid
		i++
	}
	return peers
}
func newTestNetWork(peerNum int)*testP2pNetWork{
	ts := &testP2pNetWork{make(map[discover.NodeID]*testPeer)}
	nodeId := common.Hex2Bytes("8f8581b96c387d80c64b8924ec466b17d3994db98ea5601c4ccb4b0a5acead74f43b7ffde50cfcf96efd19005e3986186268d0c0865b4217d28eff61d693bc16")
	for i := 0; i < peerNum; i++ {
		nodeId[0] = byte(i+16)
		peerId := discover.NodeID{}
		copy(peerId[:],nodeId)
		tsPeer := newTestPeer(&peerId,ts)
		ts.network[peerId] = tsPeer
	}
	return ts
}