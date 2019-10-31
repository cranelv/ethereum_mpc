package mpcService

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	"sync"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common"
)
const (
	MpcName = "eth"
	MpcVersion = 1
	MpcVersionStr = "1.0"
)
type MpcAddress struct {
	Address string `json:"address"`
	PassWord string     `json:"password"`
}
type MpcConfig struct {
	Address []MpcAddress `json:"mpcAddress"`
	PassWord string     `json:"password"`
	DataDir  string     `json:"datadir"`
}
type MpcService struct {
	protocol       p2p.Protocol
	peerMu         sync.RWMutex  // Mutex to sync the active peer set
	peers          map[discover.NodeID] *Peer
	self 		   *discover.Node
	MpcPeerGroup   []*discover.NodeID
	quit           chan struct{} // Channel used for graceful exit
	message 	   chan protocol.PeerMessage
	mpc 		   *MpcDistributor
}

// MaxMessageSize returns the maximum accepted message size.
func (ms *MpcService) MaxMessageSize() uint32 {
	// TODO what is the max size
	return uint32(1024 * 1024)
}

// APIs returns the RPC descriptors the Whisper implementation offers
func (ms *MpcService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: MpcName,
			Version:   MpcVersionStr,
			Service:   &StoremanAPI{ms},
			Public:    true,
		},
	}
}
type StoremanAPI struct {
	sm *MpcService
}

func (sa *StoremanAPI) CreateMpcAccount(ctx context.Context) (hexutil.Bytes, error) {
	if len(sa.sm.peers) < len(sa.sm.MpcPeerGroup)-1 {
		return nil, protocol.ErrTooLessStoreman
	}

	addr, err := sa.sm.mpc.MpcAccountRequest()
	if err == nil {
		log.Info("CreateMpcAccount end","address", addr.String())
	} else {
		log.Error("CreateMpcAccount end","error", err.Error())
	}

	return addr, err
}

// Protocols returns the whisper sub-protocols ran by this particular client.
func (ms *MpcService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{ms.protocol}
}
// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (ms *MpcService) Start(server *p2p.Server) error {
	ms.MpcPeerGroup = make([]*discover.NodeID,0,len(server.BootstrapNodes))
	ms.self = server.Self()
	for i:=0;i<len(server.BootstrapNodes);i++{
		ms.MpcPeerGroup = append(ms.MpcPeerGroup,&server.BootstrapNodes[i].ID)
	}
	ms.mpc.Start()
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (ms *MpcService) Stop() error {
	ms.mpc.Stop()
	select {
	case ms.quit <- struct{}{}:
	default:
	}
	return nil
}
func (ms *MpcService) hasNodeID(nodeID discover.NodeID) bool {
	for i:=0;i<len(ms.MpcPeerGroup);i++{
		if *ms.MpcPeerGroup[i] == nodeID {
			return true
		}
	}
	return false
}
// HandlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (ms *MpcService) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {

	if !ms.hasNodeID(peer.ID()) {
		log.Error("mpcService PeerID not found.", "peer",peer.ID())
		return nil
	}
	mpcPeer := newPeer(peer, rw)

	ms.peerMu.Lock()
	ms.peers[mpcPeer.ID()] = mpcPeer
	ms.peerMu.Unlock()

	defer func() {
		ms.peerMu.Lock()
		delete(ms.peers, mpcPeer.ID())
		ms.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := mpcPeer.handshake(); err != nil {
		log.Error("mpcService.handshake failed.", "peer",peer.ID(), "error" ,err)
		return err
	}

	return ms.runMessageLoop(mpcPeer, rw)
}
func (ms *MpcService) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			//mpcsyslog.Err("runMessageLoop, peer:%s, err:%s", p.Peer.ID().String(), err.Error())
			return err
		}

		//		mpcsyslog.Info("runMessageLoop, received a msg, peer:%s, packet size:%d", p.Peer.ID().String(), packet.Size)
		if packet.Size > ms.MaxMessageSize() {
			packet.Discard()
			log.Warn("runMessageLoop, oversized message received")
		} else {
			ms.message<-protocol.PeerMessage{p.ID(),&packet}
		}

	}

}

func (mp *MpcService) MessageChannel() <-chan protocol.PeerMessage{
	return mp.message
}
func (mp *MpcService) GetPeers()[]*discover.NodeID{
	return mp.MpcPeerGroup
}
func (mp *MpcService) Self() *discover.Node{
	return mp.self
}
func (ms *MpcService) IsActivePeer(peerID *discover.NodeID) bool {
	ms.peerMu.RLock()
	defer ms.peerMu.RUnlock()
	_, exist := ms.peers[*peerID]
	return exist
}

func (ms *MpcService) SendToPeer(peerID *discover.NodeID, msgcode uint64, data interface{}) error {
	ms.peerMu.RLock()
	defer ms.peerMu.RUnlock()
	peer, exist := ms.peers[*peerID]
	if exist {
		return p2p.Send(peer.ws, msgcode, data)
	} else {
		log.Error("peer not find. peer","PeerID",peerID)
	}
	return nil
}
// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *MpcConfig) *MpcService {
	service := &MpcService{
		peers: make(map[discover.NodeID]*Peer),
		message:make(chan protocol.PeerMessage,128),
		quit:  make(chan struct{}),
	}
	service.mpc = CreateMpcDistributor(cfg.DataDir, service,cfg.PassWord)
	for i:=0;i<len(cfg.Address);i++{
		service.mpc.UnlockKeystore(common.HexToMpcAddress(cfg.Address[i].Address),cfg.Address[i].PassWord)
	}
	service.protocol = p2p.Protocol{
		Name:    "MpcService",
		Version: MpcVersion,
		Length:  protocol.NumberOfMessageCodes,
		Run:     service.HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"name":"MpcService",
				"version": "v1.0",
			}
		},
	}

	return service
}
