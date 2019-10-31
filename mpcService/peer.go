package mpcService

import (
	"fmt"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
)

// peer represents a whisper protocol peer connection.
type Peer struct {
//	host    *Storeman
	*p2p.Peer
	ws      p2p.MsgReadWriter
	trusted bool

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer( remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		Peer:    remote,
		ws:      rw,
		trusted: false,
		known:   set.New(),
	}
}


// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.ws, protocol.StatusCode, protocol.ProtocolVersion)
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		log.Error("storeman peer read msg fail.", "peer",p.ID(), "error" ,err)
		return err
	}
	defer packet.Discard()

	log.Info("storeman received handshake. ", "peer",p.ID(), "packet code" ,packet.Code)
	if packet.Code != protocol.StatusCode {
		log.Error("storeman peer sent packet before status packet", "peer",p.ID(), "packet code" ,packet.Code)
		return fmt.Errorf("storman peer [%s] sent packet %x before status packet", p.ID().String(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	peerVersion, err := s.Uint()
	if err != nil {
		log.Error("storman peer sent bad status message:","peer",p.ID(), "error" ,err)
		return fmt.Errorf("storman peer [%s] sent bad status message: %v", p.ID().String(), err)
	}
	if peerVersion != protocol.ProtocolVersion {
		log.Error("storman peer : protocol version mismatch ", "peer",p.ID(),  peerVersion, protocol.ProtocolVersion)
		return fmt.Errorf("storman peer [%s]: protocol version mismatch %d != %d", p.ID().String(), peerVersion, protocol.ProtocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		log.Error("storman peer failed to send status packet:", "peer",p.ID(), "error" ,err)
		return fmt.Errorf("storman peer [%s] failed to send status packet: %v", p.ID().String(), err)
	}
	return nil
}

func (p *Peer) ID() discover.NodeID {
	id := p.Peer.ID()
	return id
}
