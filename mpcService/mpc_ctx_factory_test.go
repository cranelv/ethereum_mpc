package mpcService

import (
	"testing"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
)

func TestMpcCtxFactory(t *testing.T) {
	mpcFactory := MpcCtxFactory{}
	_, err := mpcFactory.CreateContext(protocol.MpcCreateLockAccountLeader, common.Hash{}, nil,nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(protocol.MpcCreateLockAccountPeer,  common.Hash{}, nil,nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(protocol.MpcTXSignLeader,  common.Hash{}, nil,nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(protocol.MpcTXSignPeer,  common.Hash{}, nil,nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(5, common.Hash{}, nil,nil)
	if err != nil {
		t.Log("mpcFactory create err:", err)
	}
}
