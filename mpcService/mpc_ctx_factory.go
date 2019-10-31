package mpcService

import (
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type MpcCtxFactory struct {
}

func (*MpcCtxFactory) CreateContext(ctxType uint, hash common.Hash, peers []protocol.PeerInfo,leader *discover.NodeID, preSetValue ...protocol.MpcValue) (MpcInterface, error) {
	switch ctxType {
	case protocol.MpcCreateLockAccountLeader:
		return requestCreateLockAccountMpc(hash, peers,leader, preSetValue...)
	case protocol.MpcCreateLockAccountPeer:
		return acknowledgeCreateLockAccountMpc(hash, peers,leader, preSetValue...)

	case protocol.MpcTXSignLeader:
		return requestTxSignMpc(hash, peers,leader, preSetValue...)
	case protocol.MpcTXSignPeer:
		return acknowledgeTxSignMpc(hash, peers,leader, preSetValue...)
	}

	return nil, protocol.ErrContextType
}
