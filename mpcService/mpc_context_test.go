package mpcService

import (
	"testing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"time"
	"github.com/ethereum/go-ethereum/log"
	"math/rand"
	"math/big"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
	manCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	strings "strings"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)
const (
	password = "1111111111"
	)
func TestContextQuit(t *testing.T) {
	testHash := []byte{130, 250, 171, 196, 164, 207, 30, 58, 73, 250, 188, 18, 53, 172, 48, 39, 187, 92, 60, 59, 208, 169, 165, 21, 184, 139, 28, 130, 176, 176, 66}
	t.Log(common.BytesToHash(testHash).String())
	nThread := 21
	nodeId := common.Hex2Bytes("8f8581b96c387d80c64b8924ec466b17d3994db98ea5601c4ccb4b0a5acead74f43b7ffde50cfcf96efd19005e3986186268d0c0865b4217d28eff61d693bc16")
	peers := make([]protocol.PeerInfo, nThread)
	for i := 0; i < nThread; i++ {
		peers[i].PeerID =  &discover.NodeID{}
		copy(peers[i].PeerID[:], nodeId)
		peers[i].PeerID[0] = byte(i + 1)
		peers[i].PeerID[1] = byte(i + 1)
		peers[i].PeerID[2] = byte(i + 1)
		peers[i].Seed = uint64(i + 1)
	}

	mpc, err := requestTxSignMpc(common.Hash{}, peers,nil)
	if err != nil {
		t.Error("mpc create error")
	}

	go func() {
		for _, mpcCt := range mpc.MpcSteps {
			err := mpcCt.InitMessageLoop(mpcCt)
			if err != nil {
				t.Error("mpc loop error error")
				break
			}
		}
	}()

	for i := 0; i < 10; i++ {
		mpc.quit(nil)
	}
}
func TestAccountContextStep(t *testing.T){
	log.InitLog(2)
	mpcs := newDistributor(21)
	for i:=0;i<21;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,5,nil}
	}
	rand.Seed(int64(time.Now().Second()))
	peers := mpcs[0].P2pMessager.GetPeers()
	peerInfo := make([]protocol.PeerInfo,len(peers))
	for i:=0;i<len(peers);i++{
		peerInfo[i].PeerID = peers[i]
		peerInfo[i].Seed = rand.Uint64()
		if peerInfo[i].Seed == 0{
			peerInfo[i].Seed = 1
		}
	}
	mpcs[0].createAccountRequestMpcContext(peerInfo)
	/*
	for i := 0;i<len(mpcs);i++{
		mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
			StepID:    uint64(1),
			Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(i+10)}},}
		mpcs[i].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
	}*/
	time.Sleep(60*time.Second)

}
func TestSignContextStep(t *testing.T){
	log.InitLog(2)
	mpcs := newDistributor(21)
	address := common.HexToMpcAddress("0x1A88d47c1E0F9082Ae26670A5091b194b1377327")
	hash := common.BytesToHash([]byte{12,13,14,15,16,17})
	peers := mpcs[0].P2pMessager.GetPeers()
	leader := mpcs[0].SelfNodeId()
	for i:=0;i<17;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,6,nil}
		mpcs[i].mpcKeyStore.UnlockAddress(address,password)
	}
	for i:=0;i<17;i++ {
		key := mpcs[i].mpcKeyStore.GetMpcKey(address)
		seed := key.MPCSeed
		for j:=0;j<17;j++{
			mpcs[j].nodeInfoMap.addNodeInfo(hash,seed,mpcs[i].SelfNodeId(),peers,leader,mpcs[j].mpcKeyStore.GetMpcKey(address))
		}
	}

	_,_,err := mpcs[0].createRequestMpcContext(hash,address)
	if err != nil{
		log.Error("createRequestMpcContext","Error",err)
	}
	/*
	for i := 0;i<len(mpcs);i++{
		mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
			StepID:    uint64(1),
			Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(i+10)}},}
		mpcs[i].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
	}*/
	time.Sleep(30*time.Second)

}

func TestAccountContextCheck(t *testing.T){
	log.InitLog(2)
	mpcs := newDistributor(21)
	for i:=0;i<21;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,5,nil}
	}
	seedMap := make(map[uint64]discover.NodeID)
	rand.Seed(int64(time.Now().Second()))
	peers := mpcs[0].P2pMessager.GetPeers()
	peerInfo := make([]protocol.PeerInfo,len(peers))
	for i:=0;i<len(peers);i++{
		peerInfo[i].PeerID = peers[i]
		peerInfo[i].Seed = rand.Uint64()
		if peerInfo[i].Seed == 0{
			peerInfo[i].Seed = 1
		}
		seedMap[peerInfo[i].Seed] = *peerInfo[i].PeerID
	}
	addr,_ := mpcs[0].createAccountRequestMpcContext(peerInfo)
	log.Error("createAccountRequestMpcContext","address",addr)
	address := common.BytesToMpcAddress(addr)
	time.Sleep(10*time.Second)
	privateKey := big.NewInt(0)
	for i := 0;i<21;i++{
		f := make([]*big.Int,protocol.MPCDegree+1)
		seed := make([]*big.Int,protocol.MPCDegree+1)
		for j:=0;j<=protocol.MPCDegree;j++{
			index := (i+j)%21
			mpcs[index].UnlockKeystore(address,password)
			key := mpcs[index].mpcKeyStore.GetMpcKey(address)
			seed[j] = new(big.Int).SetUint64(key.MPCSeed)
			f[j] = key.PrivateKey
		}
		result := crypto.Lagrange(f, seed)
		key := manCrypto.ToECDSAUnsafe(result.Bytes())
		getAddr := keystore.PubkeyToMpcAddress(key.PublicKey)
		if getAddr != address {
			t.Error("Address Error","want",address,"get",getAddr)
		}
		privateKey = result
	}
	log.Info("privateKey","privateKey",privateKey)
	leader := mpcs[0].SelfNodeId()
	mpcLen := protocol.MPCDegree*2+1
	for i:=0;i<mpcLen;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,6,nil}
		mpcs[i].mpcKeyStore.UnlockAddress(address,password)
	}
	seeds := make([]*big.Int,mpcLen)
	for count:=0;count<100;count++{
		hash := common.BytesToHash([]byte{byte(count),12,13,14,15,16,17})
		for i:=0;i<mpcLen;i++ {
			key := mpcs[i].mpcKeyStore.GetMpcKey(address)
			seed := key.MPCSeed
			seeds[i] = new(big.Int).SetUint64(seed)
			if nodeId,exist := seedMap[seed];exist{
				if nodeId != *mpcs[i].SelfNodeId(){
					t.Error("NodeId and Seed Error")
				}
			}else{
				t.Error("Error Seed")
			}
			for j:=0;j<mpcLen;j++{
				mpcs[j].nodeInfoMap.addNodeInfo(hash,seed,mpcs[i].SelfNodeId(),peers,leader,mpcs[j].mpcKeyStore.GetMpcKey(address))
			}
		}

		peerInfo1 := mpcs[0].nodeInfoMap.getPeerInfo(hash,address)
		for i:=0;i<len(peerInfo1);i++{
			if nodeId,exist := seedMap[peerInfo1[i].Seed];exist{
				if nodeId != *peerInfo1[i].PeerID{
					t.Error("NodeId and Seed Error")
				}
			}else{
				t.Error("Error Seed")
			}
		}
		leader := mpcs[0].GetPeerLeader(true)
		if leader == nil{
			time.Sleep(time.Second)
			continue
		}
		for i:=0;i<mpcLen;i++ {
			if *mpcs[i].SelfNodeId() == *leader {
				_,_,err := mpcs[i].createRequestMpcContext(hash,address)
				if err != nil{
					log.Error("createRequestMpcContext","Error",err)
				}
			}
			break
		}

		time.Sleep(5*time.Second)
		a := big.NewInt(0)
		r := big.NewInt(0)

		af := make([]*big.Int,mpcLen)
		rf := make([]*big.Int,mpcLen)
		arf := make([]*big.Int,mpcLen)
		for i := 0;i<mpcLen;i++{
			fac := mpcs[i].mpcCreater.(*MpcTestCtxFactory)
			a0,_ := fac.result.GetValue(protocol.MpcSignA0)
			a.Add(a,a0.(*big.Int))
			r0,_ := fac.result.GetValue(protocol.MpcSignR0)
			r.Add(r,r0.(*big.Int))
			//		ar,_ := fac.result.GetValue(protocol.MpcSignARResult)
			//		log.Error("ARVALUE0","ar",ar.(*big.Int))
			al,_ := fac.result.GetValue(protocol.MpcSignA)
			af[i] = al.(*big.Int)
			rl,_ := fac.result.GetValue(protocol.MpcSignR)
			rf[i] = rl.(*big.Int)
			bl,_ := fac.result.GetValue(protocol.MpcSignB)
			arf[i] = new(big.Int).Mul(af[i],rf[i])
			arf[i].Add(arf[i],bl.(*big.Int))
			arf[i].Mod(arf[i],crypto.Secp256k1N)
			log.Error("ARSeed","Seed",seeds[i],"ARdata",arf[i])
			Sign,_ := fac.result.GetValue(protocol.MpcTxSignResult)
			log.Error("Sign","Sign",Sign.(*big.Int))
		}
		alag := crypto.Lagrange(af,seeds)
		alag.Mod(alag,crypto.Secp256k1N)
		a.Mod(a,crypto.Secp256k1N)
		log.Error("AValue--","lagrange",alag,"sum",a)
		rlag := crypto.Lagrange(rf,seeds)
		rlag.Mod(rlag,crypto.Secp256k1N)
		r.Mod(r,crypto.Secp256k1N)
		log.Error("RValue--","lagrange",rlag,"sum",r)
		a.Mul(a,r)
		a.Mod(a,crypto.Secp256k1N)
		arlag := crypto.Lagrange(arf,seeds)
		arlag.Mod(arlag,crypto.Secp256k1N)
		log.Error("ARValue--","lagrange",arlag,"sum",a)
		r = r.ModInverse(r,crypto.Secp256k1N)
		x,_ := secp256k1.S256().ScalarBaseMult(r.Bytes())
		x.Mod(x, crypto.Secp256k1N)
		//	log.Error("TxSign_CalSignStep.InitStep","InvR-X",x,"InvR-Y", y)
		/*
		for i := 0;i<len(mpcs);i++{
			mpcMsg := &protocol.MpcMessage{ContextID: emptyHash,
				StepID:    uint64(1),
				Data:      []*protocol.MpcData{&protocol.MpcData{protocol.MpcSeed,uint64(i+10)}},}
			mpcs[i].P2pMessage(mpcs[0].SelfNodeId(),protocol.MSG_RequestPrepare,mpcMsg)
		}*/
		time.Sleep(200*time.Millisecond)
	}

}
func TestAccountContextSendHash(t *testing.T){
	log.InitLog(2)
	mpcs := newDistributor(21)
	for i:=0;i<21;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,5,nil}
	}
	seedMap := make(map[uint64]discover.NodeID)
	rand.Seed(int64(time.Now().Second()))
	peers := mpcs[0].P2pMessager.GetPeers()
	peerInfo := make([]protocol.PeerInfo,len(peers))
	for i:=0;i<len(peers);i++{
		peerInfo[i].PeerID = peers[i]
		peerInfo[i].Seed = rand.Uint64()
		if peerInfo[i].Seed == 0{
			peerInfo[i].Seed = 1
		}
		seedMap[peerInfo[i].Seed] = *peerInfo[i].PeerID
	}
	mpcs[0].createAccountRequestMpcContext(peerInfo)
	address := common.HexToMpcAddress("0x029222e982b54c23208409bb066a60d784e94a1e1c312e94db6d9d9abbf6354ecd")
	time.Sleep(3*time.Second)
	privateKey := big.NewInt(0)
	for i := 0;i<21;i++{
		f := make([]*big.Int,protocol.MPCDegree+1)
		seed := make([]*big.Int,protocol.MPCDegree+1)
		for j:=0;j<=protocol.MPCDegree;j++{
			index := (i+j)%21
			mpcs[index].UnlockKeystore(address,password)
			key := mpcs[index].mpcKeyStore.GetMpcKey(address)
			seed[j] = new(big.Int).SetUint64(key.MPCSeed)
			f[j] = key.PrivateKey
		}
		result := crypto.Lagrange(f, seed)
		key := manCrypto.ToECDSAUnsafe(result.Bytes())
		getAddr := keystore.PubkeyToMpcAddress(key.PublicKey)
		if getAddr != address {
			t.Error("Address Error","want",address,"get",getAddr)
		}
		privateKey = result
	}
	log.Info("privateKey","privateKey",privateKey)
	hash := common.BytesToHash([]byte{12,13,14,15,16,17})
//	leader := mpcs[0].SelfNodeId()
	mpcLen := protocol.MPCDegree*2+1
	for i:=0;i<21;i++ {
		mpcs[i].mpcCreater = &MpcTestCtxFactory{false,6,nil}
		mpcs[i].mpcKeyStore.UnlockAddress(address,password)
	}


	peerInfo1 := mpcs[0].nodeInfoMap.getPeerInfo(hash,address)
	for i:=0;i<len(peerInfo1);i++{
		if nodeId,exist := seedMap[peerInfo1[i].Seed];exist{
			if nodeId != *peerInfo1[i].PeerID{
				t.Error("NodeId and Seed Error")
			}
		}else{
			t.Error("Error Seed")
		}
	}
	nLen := 500
	nCell := mpcLen+4
	for j := 0;j<nLen ;j++{
		binary.BigEndian.PutUint16(hash[:],uint16(j+10))
		for i:=0;i<nCell;i++ {
			index := (i+j)%21
			//index := i
//			21/17
			mpcs[index].SendSignRequestPrepare(hash,address)
		}
//		time.Sleep(1*time.Second)
	}
	time.Sleep(200*time.Second)
	for key,value := range protocol.TLog.LogMap{
		if strings.Contains(key,"prepqare-from") || strings.Contains(key,"prepqare-seed"){
			if value%20 != 0{
				log.Error("protocol.TLog.LogMap","key",key,"value",value)
			}
		}else if strings.Contains(key,"prepqare-to"){
			if value != (nCell-1)*nLen{
				log.Error("protocol.TLog.LogMap","key",key,"value",value)
			}
		}
	}
	log.Error("createRequestMpcContext Successful","count",protocol.TLog.LogMap["createRequestMpcContext Successful"])
//	panic("aaaaaa")
	time.Sleep(3*time.Second)

}