package mpcService

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"time"
	"sort"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/types"
	"math/rand"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
)
const (
	MpcInterval = 30
	MpcCacity =   20
)
type MpcContextCreater interface {
	CreateContext(uint, common.Hash, []protocol.PeerInfo,*discover.NodeID, ...protocol.MpcValue) (MpcInterface, error) //createContext
}

type MpcInterface interface {
	getMessage(*discover.NodeID, *protocol.MpcMessage) error
	mainMPCProcess(manager protocol.MpcManager) error
	SubscribeResult(chan<- *protocol.MpcResult)
	quit(error)
}

type P2pMessager interface {
	SendToPeer(*discover.NodeID, uint64, interface{}) error
	MessageChannel() <-chan protocol.PeerMessage
	GetPeers()[]*discover.NodeID
	Self() *discover.Node
	IsActivePeer(*discover.NodeID) bool
}
type MpcKey struct {
	Hash common.Hash
	Address common.MpcAddress
}
func NodeInfoKey (hash common.Hash,addr common.MpcAddress)common.Hash{
	return types.RlpHash(&MpcKey{hash,addr})
}
type mpcNodeInfo struct {
	mu             sync.RWMutex
	mpcHashTimer   [][]*common.Hash
	mpcNodeInfo    map[common.Hash]protocol.MpcNodeInterface
}
func NewMpcNodeInfo()*mpcNodeInfo{
	return &mpcNodeInfo{
		mpcHashTimer: make([][]*common.Hash,1),
		mpcNodeInfo: make(map[common.Hash]protocol.MpcNodeInterface),
	}
}
func (mn* mpcNodeInfo) getEnoughNodeInfo()[]protocol.MpcNodeInterface{
	var NodeAry []protocol.MpcNodeInterface
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	for _,nodeInfo := range mn.mpcNodeInfo{
		if nodeInfo.FetchQuorum() {
			NodeAry = append(NodeAry,nodeInfo)
		}
	}
	return NodeAry
}
func (mn* mpcNodeInfo) getNodeInfo(hash common.Hash,addr common.MpcAddress)protocol.MpcNodeInterface{
	nodeHash := NodeInfoKey(hash,addr)
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	return mn.mpcNodeInfo[nodeHash]
}
func (mn* mpcNodeInfo) RunContextNodeID(hash common.Hash,addr common.MpcAddress,seed uint64,nodeId *discover.NodeID)protocol.MpcState{
	nodeHash := NodeInfoKey(hash,addr)
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	item,exist := mn.mpcNodeInfo[nodeHash]
	if exist {
		return item.RunNode(seed,nodeId)
	}
	return protocol.MpcNotFound
}
func (mn* mpcNodeInfo) ChangeMpcState(hash common.Hash,addr common.MpcAddress,state protocol.MpcState)protocol.MpcState{
	nodeHash := NodeInfoKey(hash,addr)
	mn.mu.Lock()
	defer mn.mu.Unlock()
	item,exist := mn.mpcNodeInfo[nodeHash]
	if exist {
		state = item.GetState()
		if state < protocol.MpcRunning{
			item.SetState(protocol.MpcRunning)
		}
		return state
	}
	return protocol.MpcNotFound
}
func (mn* mpcNodeInfo) setMpcState(hash common.Hash,addr common.MpcAddress,state protocol.MpcState){
	nodeHash := NodeInfoKey(hash,addr)
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	item,exist := mn.mpcNodeInfo[nodeHash]
	if exist {
		item.SetState(state)
	}
}
func (mn* mpcNodeInfo) addNodeInfo(hash common.Hash,seed uint64,nodeId *discover.NodeID,
	peers []*discover.NodeID,leader *discover.NodeID,key *keystore.MpcKey)error{
	nodeHash := NodeInfoKey(hash,key.Address)
	mn.mu.Lock()
	defer mn.mu.Unlock()
	item,exist := mn.mpcNodeInfo[nodeHash]
	if exist {
		return item.AddNode(seed,nodeId)
	}else {
		index := len(mn.mpcHashTimer)-1
		mn.mpcHashTimer[index] = append(mn.mpcHashTimer[index],&nodeHash)
		item = NewNodeCollection(hash,peers,leader,key)
		mn.mpcNodeInfo[nodeHash] = item
		return item.AddNode(seed,nodeId)
	}
}
func (mn* mpcNodeInfo) getPeerInfo(hash common.Hash,addr common.MpcAddress)[]protocol.PeerInfo{
	nodeHash := NodeInfoKey(hash,addr)
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	if nodeInfo,exist := mn.mpcNodeInfo[nodeHash];exist{
		return nodeInfo.GetPeers()
	}
	return nil
}
func (mn* mpcNodeInfo) newTimer(time uint64)[]*common.Hash{
	mn.mu.Lock()
	defer mn.mu.Unlock()
	var hashAry []*common.Hash
	count := len(mn.mpcHashTimer)
	if count > MpcCacity {
		for i:=0;i<count-MpcCacity ;i++  {
			hashAry = append(hashAry,mn.mpcHashTimer[i]...)
		}
		mn.mpcHashTimer = mn.mpcHashTimer[count-MpcCacity:]
	}
	mn.mpcHashTimer = append(mn.mpcHashTimer,[]*common.Hash{})
	/*
	count = len(mn.mpcHashTimer)
	if count > 2 {
		reSendhash := mn.mpcHashTimer[count-3]
		for i:=0;i<len(reSendhash);i++{
			if nodeInfo,exist := mn.mpcNodeInfo[*reSendhash[i]];exist{
				if nodeInfo.GetState() == protocol.MpcRunning{
					nodeInfo.SetState(protocol.MpcWaiting)
				}
			}
		}
	}
	if count > 3 {
		reSendhash := mn.mpcHashTimer[count-3]
		for i:=0;i<len(reSendhash);i++{
			if nodeInfo,exist := mn.mpcNodeInfo[*reSendhash[i]];exist{
				state := nodeInfo.GetState()
				if state == protocol.MpcRunning || state  == protocol.MpcWaiting {
					nodeInfo.SetState(protocol.MpcCollection)
				}
			}
		}
	}
	*/
	for _,hash := range hashAry {
		delete (mn.mpcNodeInfo,*hash)
	}
	return hashAry
}
type mpcCtxInfo struct {
	mu             sync.RWMutex
	mpcCtxInfo    map[common.Hash]MpcInterface
}
func (mn* mpcCtxInfo) getMpcCtx(hash common.Hash)MpcInterface{
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	return mn.mpcCtxInfo[hash]
}
func (mn* mpcCtxInfo) hasMpcCtx(hash common.Hash)bool{
	mn.mu.RLock()
	defer mn.mu.RUnlock()
	_,exist := mn.mpcCtxInfo[hash]
	return exist
}
func (mn* mpcCtxInfo) setMpcCtx(hash common.Hash,mpcCtx MpcInterface){
	mn.mu.Lock()
	defer mn.mu.Unlock()
	mn.mpcCtxInfo[hash] = mpcCtx
}
func (mn* mpcCtxInfo) removeMpcCtx(hash common.Hash){
	mn.mu.Lock()
	defer mn.mu.Unlock()
	delete(mn.mpcCtxInfo,hash)
}
type MpcDistributor struct {
	mpcCreater     MpcContextCreater
	nodeInfoMap    *mpcNodeInfo
	mpcMap         mpcCtxInfo
	mpcKeyStore    *keystore.MpcKeyStore
	P2pMessager    P2pMessager
	quit		   chan struct{}
	result		   chan *protocol.MpcResult
	password       string
}

func CreateMpcDistributor(keystoreDir string, msger P2pMessager, password string) *MpcDistributor {
	mpc := &MpcDistributor{
		mpcCreater:     &MpcCtxFactory{},
		mpcMap:         mpcCtxInfo{mpcCtxInfo:make(map[common.Hash]MpcInterface)},
		nodeInfoMap:    NewMpcNodeInfo(),
		mpcKeyStore: 	keystore.NewMpcKeyStore(keystoreDir),
		quit:			make(chan struct{},1),
		result: 		make(chan *protocol.MpcResult,64),
		password:       password,
		P2pMessager:    msger,
	}
	return mpc
}
func (mpcServer *MpcDistributor) MessageLoop(){
	ticker := time.NewTicker(MpcInterval*time.Second)
	ticker1 := time.NewTicker(3100*time.Millisecond)
	defer ticker.Stop()
	for {
		select{
		case msg:= <-mpcServer.P2pMessager.MessageChannel():
//			length := len(mpcServer.P2pMessager.MessageChannel())
//			if length == 0 {
//				log.Error("MessageLoop","MessageLoopSeed",mpcServer.SelfNodeId(),"MessageChannel Length",length)
//			}
			mpcServer.GetMessage(msg.From,msg.Message)
		case result := <- mpcServer.result :
			if result != nil{
				log.Error("Mpc Context Result:","hash",result.Hash,"Result",result.Result)
			}
		case <-ticker.C:
			go mpcServer.NewTime()
		case <-ticker1.C:
			go mpcServer.CreateRequestMpc()
		case <-mpcServer.quit:
			mpcServer.quitLoop()
			return
		}
	}
}
func (mpcServer *MpcDistributor)NewTime(){
//	log.Error("(mpcServer *MpcDistributor)NewTime()")
	sec := time.Now().Second()/MpcInterval
	hashAry := mpcServer.nodeInfoMap.newTimer(uint64(sec))
	for _,hash := range hashAry{
		mpcMsg := &protocol.MpcMessage{ContextID: *hash,
			StepID: 0,
			Error: "Mpc Time out"}
		go mpcServer.BoardcastMessage(nil, protocol.MSG_MPCError, mpcMsg)
	}
}
func (mpcServer *MpcDistributor)Start(){
	go mpcServer.MessageLoop()
}
func (mpcServer *MpcDistributor)Stop(){
	select {
	case mpcServer.quit <- struct{}{}:
	default:
	}
}
func (mpcServer *MpcDistributor)quitLoop(){
	mpcServer.mpcMap.mu.Lock()
	defer mpcServer.mpcMap.mu.Unlock()
	for _,mpc := range mpcServer.mpcMap.mpcCtxInfo{
		mpc.quit(errors.New("MpcServer Stop!"))
	}
	mpcServer.mpcMap.mpcCtxInfo = make(map[common.Hash]MpcInterface)
}

func GetAddressAndHash(message *protocol.MpcMessage)(hash common.Hash,address common.MpcAddress,have bool){
	n := 0
	for _,item := range message.Data{
		if item.Key == protocol.MpcAddress {
			address = common.BytesToMpcAddress(item.Data.([]byte))
			n++
		}else if item.Key == protocol.MpcTxHash {
			hash = common.BytesToHash(item.Data.([]byte))
			n++
		}
	}
	have = n == 2
	return
}
func (mpcServer *MpcDistributor) GetMessage(PeerID discover.NodeID, msg *p2p.Msg) error {
	log.Info("MpcDistributor GetMessage begin!", "msgCode", msg.Code)

	switch msg.Code {
	case protocol.StatusCode:
		// this should not happen, but no need to panic; just ignore this message.
		log.Info("unxepected status message received, peer:%s", PeerID.String())

	case protocol.KeepaliveCode:
		// this should not happen, but no need to panic; just ignore this message.

	case protocol.KeepaliveOkCode:
		// this should not happen, but no need to panic; just ignore this message.

	case protocol.MSG_MPCError:
		var mpcMessage protocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.Error("MpcDistributor.GetMessage, rlp decode MPCError msg fail. err:%s", err.Error())
			return err
		}

//		log.Error("MpcDistributor.GetMessage, MPCError message received.","peer",PeerID,"error", mpcMessage.Error)
		go mpcServer.QuitMpcContext(&mpcMessage)
	case protocol.MSG_RequestPrepare:
		var mpcMessage protocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.Error("MpcDistributor.GetMessage, rlp decode RequestMPC msg fail. err:%s", err.Error())
			return err
		}

		go mpcServer.handleMpcPrepqare(&PeerID,&mpcMessage)

	case protocol.MSG_RequestMPC:
//		return nil
		log.Info("MpcDistributor.GetMessage, RequestMPC message received.","peer", PeerID)
		var mpcMessage protocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.Error("MpcDistributor.GetMessage, rlp decode RequestMPC msg fail."," err", err.Error())
			return err
		}
		go func(inputMsg protocol.MpcMessage){
			err := mpcServer.createMpcContext(&inputMsg,PeerID)
			if err != nil{
				if err == protocol.ErrMpcContextExist{

				}else if err == protocol.ErrMpcFinish{
					hash,addr,have := GetAddressAndHash(&inputMsg)
					if have {
						mpcMsg := &protocol.MpcMessage{ContextID: inputMsg.ContextID,
							StepID: 0,
							Data:[]*protocol.MpcData{&protocol.MpcData{protocol.MpcTxHash,hash},
							&protocol.MpcData{protocol.MpcAddress,addr}},
						}
						go mpcServer.BoardcastMessage(nil, protocol.MSG_MPCFinish, mpcMsg)
					}
				}else{
					mpcMsg := &protocol.MpcMessage{ContextID: inputMsg.ContextID,
						StepID: 0,
						Error: err.Error()}
					var peers []protocol.PeerInfo
					if len(inputMsg.Peers)>0 {
						rlp.DecodeBytes(inputMsg.Peers,&peers)
					}
					nodeIds := make([]*discover.NodeID,len(peers))
					for i:=0;i<len(peers);i++{
						nodeIds[i] = peers[i].PeerID
					}
					//mpcServer.nodeInfoMap.setMpcState(inputMsg.ContextID,protocol.MpcCollection)
					go mpcServer.BoardcastMessage(nodeIds, protocol.MSG_MPCError, mpcMsg)
				}
			}else{
				hash,addr,have := GetAddressAndHash(&inputMsg)
				if have {
					mpcServer.nodeInfoMap.setMpcState(hash,addr,protocol.MpcFinish)
				}
			}
		}(mpcMessage)
	case protocol.MSG_MPCMessage:
//		return nil
		var mpcMessage protocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.Error("GetP2pMessage fail. err:%s", err.Error())
			return err
		}

		log.Info("MpcDistributor.GetMessage, MPCMessage message received","peer", PeerID)
		go mpcServer.getMpcMessage(&PeerID, &mpcMessage)
	case protocol.MSG_MPCFinish:
		var mpcMessage protocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.Error("GetP2pMessage fail. err:%s", err.Error())
			return err
		}
		go mpcServer.FinishMpc(&PeerID, &mpcMessage)
	default:
		// New message types might be implemented in the future versions of Whisper.
		// For forward compatibility, just ignore.
	}

	return nil
}
func (mpcServer *MpcDistributor) SendSignRequestPrepare(txHash common.Hash,address common.MpcAddress) error {
	key := mpcServer.mpcKeyStore.GetMpcKey(address)
	if key == nil {
		return errors.New("Keystore is not found")
	}
	peers := mpcServer.P2pMessager.GetPeers()
	err := mpcServer.nodeInfoMap.addNodeInfo(txHash,key.MPCSeed,mpcServer.SelfNodeId(),peers,nil,key)
	if err != nil {
		log.Error("SendSignRequestPrepare","error",err)
		return err
	}
//	log.Error("SendSignRequestPrepare","Self",mpcServer.SelfNodeId(),"seed",key.MPCSeed,"hash",txHash)
	msg := protocol.MpcMessage{
		ContextID:txHash,
		Data:[]*protocol.MpcData{&protocol.MpcData{protocol.MpcTxHash,txHash[:]},
			&protocol.MpcData{protocol.MpcAddress,address[:]},
			&protocol.MpcData{protocol.MpcSeed,key.MPCSeed},},
	}
	go mpcServer.BoardcastMessage(nil,protocol.MSG_RequestPrepare,&msg)
	return nil
}

func (mpcServer *MpcDistributor)UnlockKeystore(address common.MpcAddress, password string)error{
	return mpcServer.mpcKeyStore.UnlockAddress(address,password)
}

func (mpcServer *MpcDistributor) loadStoremanAddress(address *common.MpcAddress) (*protocol.MpcValue,  error) {
	log.Info("MpcDistributor.loadStoremanAddress begin","address",address)

	mpcKey := mpcServer.mpcKeyStore.GetMpcKey(*address)
	if mpcKey == nil {
		return nil,  errors.New("MpcDistributor loadStoremanAddress Error")
	}
	return &protocol.MpcValue{protocol.MpcPrivateShare, mpcKey.PrivateKey}, nil
}
func (mpcServer *MpcDistributor) MpcAccountRequest() (hexutil.Bytes, error) {
	rand.Seed(int64(time.Now().Second()))
	peers := mpcServer.P2pMessager.GetPeers()
	peerInfo := make([]protocol.PeerInfo,len(peers))
	seedMap := make(map[uint64]bool)
	seedMap[0] = true
	for i:=0;i<len(peers);i++{
		seed := rand.Uint64()
		for ;; {
			if _,exist := seedMap[seed];exist{
				seed = rand.Uint64()
			}else{
				break
			}
		}
		peerInfo[i].PeerID = peers[i]
		peerInfo[i].Seed = seed
		seedMap[peerInfo[i].Seed] = true
	}
	return mpcServer.createAccountRequestMpcContext(peerInfo)
}

func (mpcServer *MpcDistributor) createAccountRequestMpcContext(peers []protocol.PeerInfo) (hexutil.Bytes, error) {
	log.Info("MpcDistributor createRequestMpcContext begin")
	seedMap := make(map[uint64]bool)
	seedMap[0] = true
	rand.Seed(int64(time.Now().Second()))
	for i:=0;i<len(peers);i++{
		seed := rand.Uint64()
		if _,exist := seedMap[seed];exist{
			log.Error("---------------------------error seed")
			continue
		}
		seedMap[seed] = true
		peers[i].Seed = seed
	}

	rand,_ := crypto.RandFieldElement(secp256k1.S256())
	hash := common.BigToHash(rand)
	mpc, err := mpcServer.mpcCreater.CreateContext(protocol.MpcCreateLockAccountLeader, hash,
		peers,mpcServer.SelfNodeId())
	if err != nil {
		log.Error("MpcDistributor createRequestMpcContext, CreateContext fail. err:%s", err.Error())
		return []byte{}, err
	}

	log.Info("MpcDistributor createRequestMpcContext, ,","mpcID:",  hash)

	mpcServer.addMpcContext(hash, mpc)
	defer mpcServer.removeMpcContext(hash)
	err = mpc.mainMPCProcess(mpcServer)
	if err != nil {
		log.Error("MpcDistributor createRequestMpcContext, mainMPCProcess fail. err:%s", err.Error())
		return []byte{}, err
	}

//	result := mpc.getMpcResult().(common.MpcAddress)

//	log.Info("MpcDistributor createRequestMpcContext, succeed," ,"result", result)
//	return result[:], nil
	return []byte{}, err
}
func (mpcServer *MpcDistributor)CreateRequestMpc(){
	peer := mpcServer.GetPeerLeader(true)
	if peer == nil{
		return
	}
	log.Info("GetPeerLeader","peer",peer)
	if peer != nil && *peer == *mpcServer.SelfNodeId(){
		nodeInfoAry := mpcServer.nodeInfoMap.getEnoughNodeInfo()
		for i:=0;i<len(nodeInfoAry);i++{
			go func(node protocol.MpcNodeInterface){
				_,ctxHash,err := mpcServer.createRequestMpcContext(*node.Hash(),*node.Address())
				if err!= nil{
					//node.SetState(protocol.MpcCollection)
				}else{
//					log.Error("createRequestMpcContext Successful","result",result)
					node.SetState(protocol.MpcFinish)
					protocol.TLog.Error("createRequestMpcContext Successful")
					mpcMsg := &protocol.MpcMessage{ContextID: *ctxHash,
						StepID: 5,
						Data:[]*protocol.MpcData{&protocol.MpcData{protocol.MpcTxHash,*node.Hash()},
							&protocol.MpcData{protocol.MpcAddress,*node.Address()}},
					}
					go mpcServer.BoardcastMessage(nil, protocol.MSG_MPCFinish, mpcMsg)
				}
			}(nodeInfoAry[i])
		}
	}
}
func (mpcServer *MpcDistributor) createRequestMpcContext(hash common.Hash, from common.MpcAddress) ([]byte,*common.Hash, error) {
//	log.Error("MpcDistributor createRequestMpcContext begin","hash",hash.String())
	nodeInfo := mpcServer.nodeInfoMap.getNodeInfo(hash,from)
	if nodeInfo == nil{
		return []byte{},nil, errors.New("MpcDistributor createRequestMpcContext nodeInfo Nil")
	}
	seed := nodeInfo.GetSeed(mpcServer.SelfNodeId())
	if seed == 0{
		return []byte{},nil, errors.New("MpcDistributor createRequestMpcContext seed Nil")
	}
	peers := nodeInfo.GetPeers()
	if len(peers) > nodeInfo.NeedQuorum(){
		rand.Seed(time.Now().UnixNano())
		nNum := len(peers)
		for ;nNum>nodeInfo.NeedQuorum(); {
			index := rand.Int()%nNum
			if peers[index].Seed != seed{
				if index!=nNum-1 {
					peers[index],peers[nNum-1] = peers[nNum-1],peers[index]
				}
				nNum--
			}
		}
		peers = peers[:nodeInfo.NeedQuorum()]
	}
	preSetValue := []protocol.MpcValue{protocol.MpcValue{protocol.MpcTxHash, hash[:]},
		protocol.MpcValue{protocol.MpcAddress, from[:]}}

	value, err := mpcServer.loadStoremanAddress(&from)
	if err != nil {
		log.Error("MpcDistributor createRequestMpcContext, loadStoremanAddress fail.","address",from, "error", err)
		return []byte{},nil, err
	}

	preSetValue = append(preSetValue, *value)
	rand,_ := crypto.RandFieldElement(secp256k1.S256())
	ctxhash := common.BigToHash(rand)
	mpc, err := mpcServer.mpcCreater.CreateContext(protocol.MpcTXSignLeader, ctxhash,peers,mpcServer.SelfNodeId(),  preSetValue...)
	if err != nil {
		log.Error("MpcDistributor createRequestMpcContext, CreateContext fail. err:%s", err.Error())
		return []byte{},nil, err
	}

	log.Info("MpcDistributor createRequestMpcContext","mpcID", ctxhash)

	mpcServer.addMpcContext(ctxhash, mpc)
	defer mpcServer.removeMpcContext(ctxhash)
	err = mpc.mainMPCProcess(mpcServer)
	if err != nil {
		log.Error("MpcDistributor createRequestMpcContext, mainMPCProcess fail","error", err)
		return []byte{},&ctxhash, err
	}


	return []byte{},&ctxhash, nil
}

func (mpcServer *MpcDistributor) QuitMpcContext(msg *protocol.MpcMessage) {
//	log.Error("QuitMpcContext","contextId",msg.ContextID,"error",msg.Error)
	mpc := mpcServer.mpcMap.getMpcCtx(msg.ContextID)
	if mpc != nil {
		mpc.quit(errors.New(msg.Error))
	}
}
func (mpcServer *MpcDistributor) GetPeerLeader(bRequest bool)*discover.NodeID{
	second := time.Now().Second()
	if bRequest{
		if second%MpcInterval <=1 || second%MpcInterval+1>=MpcInterval{
			return nil
		}
	}
	epoch := second/MpcInterval
	peers := mpcServer.P2pMessager.GetPeers()
	sort.Sort(protocol.SlicePeers(peers))
	index := epoch%len(peers)
	return peers[index]

}
func (mpcServer *MpcDistributor) handleMpcPrepqare(peerId *discover.NodeID,mpcMessage *protocol.MpcMessage){
	log.Info("MpcDistributor handleMpcPrepqare begin")
	var seed uint64
	var address common.MpcAddress
	n := 0
	for _,data := range mpcMessage.Data {
		if data.Key == protocol.MpcSeed {
			seed = data.Data.(uint64)
			n++
			if(n>=2){
				break
			}
		}else if data.Key == protocol.MpcAddress{
			address = common.BytesToMpcAddress(data.Data.([]byte))
			n++
			if(n>=2){
				break
			}
		}
	}
	protocol.TLog.Error("handleMpcPrepqare,prepqare-from=%s,prepqare-to=%s,prepqare-seed=%v",peerId.TerminalString(),
		mpcServer.SelfNodeId().TerminalString(),seed)
	if (seed == 0){
		log.Error("handleMpcPrepqare need nonzero seed ")
		return
	}
	peers := mpcServer.P2pMessager.GetPeers()
	key := mpcServer.mpcKeyStore.GetMpcKey(address)

	if key == nil{
		log.Error("Signature Request need Address")
		return
	}
	err := mpcServer.nodeInfoMap.addNodeInfo(mpcMessage.ContextID,seed,peerId,peers,nil,key)
	if err != nil {
		log.Error(err.Error())
	}
	/*
	peer := mpcServer.GetPeerLeader()
	log.Info("GetPeerLeader","peer",peer)
	enough,err := mpcServer.nodeInfoMap.addNodeInfo(mpcMessage.ContextID,seed,peerId,peers,nil,key)
	if err != nil {
		log.Error(err.Error())
	}else if *peer == *mpcServer.SelfNodeId(){
		log.Info("Mpc Leader Info:","Enough",enough,"Error",err)
		if enough{
			if requestType == 0{
				peers := mpcServer.nodeInfoMap.getPeerInfo(mpcMessage.ContextID)
				go mpcServer.createAccountRequestMpcContext(peers)
			}else{
				go func() {
					_,err := mpcServer.createRequestMpcContext(mpcMessage.ContextID,address)
					if err != nil{
						log.Error("createRequestMpcContext","Error",err)
					}
				}()
			}
		}
	}
	*/
}
func (mpcServer *MpcDistributor) createMpcContext(mpcMessage *protocol.MpcMessage,leader discover.NodeID, preSetValue ...protocol.MpcValue) error {
	log.Info("MpcDistributor createMpcContext begin")
	if mpcServer.mpcMap.hasMpcCtx(mpcMessage.ContextID) {
//		log.Error("createMpcContext fail. err:%s", protocol.ErrMpcContextExist.Error())
		return protocol.ErrMpcContextExist
	}
	var peers []protocol.PeerInfo
	if len(mpcMessage.Peers)>0 {
		rlp.DecodeBytes(mpcMessage.Peers,&peers)
	}
	var ctxType uint
	if len(mpcMessage.Data) == 0{
		ctxType = protocol.MpcCreateLockAccountPeer
	} else {
		ctxType = protocol.MpcTXSignPeer
	}

	log.Info("createMpcContext","ctxType", ctxType, "ctxId", mpcMessage.ContextID)
	if ctxType == protocol.MpcTXSignPeer {
		log.Info("createMpcContext MpcTXSignPeer")
		if leader != *mpcServer.GetPeerLeader(false){
			return errors.New("Mpc Leader Checked Error")
		}
		address := common.MpcAddress{}
		txHash := common.Hash{}
		for _,item := range mpcMessage.Data{
			preSetValue = append(preSetValue, protocol.MpcValue{item.Key,item.Data})
			if item.Key == protocol.MpcAddress {
				address = common.BytesToMpcAddress(item.Data.([]byte))
			}else if item.Key == protocol.MpcTxHash {
				txHash = common.BytesToHash(item.Data.([]byte))
			}

		}
		bHave := false
		for i:=0;i<len(peers);i++{
			if *peers[i].PeerID == *mpcServer.SelfNodeId(){
				bHave = true
				break
			}
		}
		if !bHave{
			mpcServer.nodeInfoMap.ChangeMpcState(txHash,address,protocol.MpcRunning)
			return protocol.ErrMpcContextExist
		}

		// load account
		MpcPrivateShare, err := mpcServer.loadStoremanAddress(&address)
		if err != nil {
			return err
		}
		mpcKey := mpcServer.mpcKeyStore.GetMpcKey(address)
		state := mpcServer.nodeInfoMap.RunContextNodeID(txHash,address,mpcKey.MPCSeed,mpcServer.SelfNodeId())
		if state == protocol.MpcNotFound{
			return errors.New("Mpcserver is not prepared Mpc")
		}else if state == protocol.MpcFinish{
			return protocol.ErrMpcFinish
		}else if state == protocol.MpcRunning{
			return protocol.ErrMpcContextExist
		}
		preSetValue = append(preSetValue, *MpcPrivateShare)
	}
//	log.Error("CreateContext","leader",leader)
	mpc, err := mpcServer.mpcCreater.CreateContext(ctxType, mpcMessage.ContextID,peers,&leader,preSetValue...)
	if err != nil {
		log.Error("createMpcContext, createContext fail, err:%s", err.Error())
		return err
	}

	go func() {
		mpcServer.addMpcContext(mpcMessage.ContextID, mpc)
		defer mpcServer.removeMpcContext(mpcMessage.ContextID)
		err = mpc.mainMPCProcess(mpcServer)
	}()

	return nil
}

func (mpcServer *MpcDistributor) addMpcContext(hash common.Hash, mpc MpcInterface) {
	log.Info("addMpcContext. ","ctxId", hash)

	mpc.SubscribeResult(mpcServer.result)
	mpcServer.mpcMap.setMpcCtx(hash,mpc)
//	mpcServer.nodeInfoMap.setMpcState(hash,protocol.MpcRunning)
}

func (mpcServer *MpcDistributor) removeMpcContext(hash common.Hash) {
	log.Info("removeMpcContext. ","ctxId", hash)
	mpcServer.mpcMap.removeMpcCtx(hash)
}

func (mpcServer *MpcDistributor) getMpcMessage(PeerID *discover.NodeID, mpcMessage *protocol.MpcMessage) error {
	log.Info("getMpcMessage.", "peerid", PeerID,"ctxId",mpcMessage.ContextID,  "stepID", mpcMessage.StepID)

	mpc := mpcServer.mpcMap.getMpcCtx(mpcMessage.ContextID)
	if mpc != nil {
		return mpc.getMessage(PeerID, mpcMessage)
	}

	return nil
}
func (mpcServer *MpcDistributor) FinishMpc(PeerID *discover.NodeID, mpcMessage *protocol.MpcMessage) error {
//	log.Error("FinishMpc.", "peerid", PeerID,"ctxId",mpcMessage.ContextID)

	if mpcMessage.StepID == 0{
//		log.Error("FinishMpc.")
		hash,addr,have := GetAddressAndHash(mpcMessage)
		if have {
			mpcServer.nodeInfoMap.setMpcState(hash,addr,protocol.MpcFinish)
			mpc := mpcServer.mpcMap.getMpcCtx(mpcMessage.ContextID)
			if mpc != nil {
				mpc.quit(protocol.ErrMpcContextExist)
			}
		}
	}else{
		mpc := mpcServer.mpcMap.getMpcCtx(mpcMessage.ContextID)
		if mpc == nil {
			hash,addr,have := GetAddressAndHash(mpcMessage)
			if have {
				mpcServer.nodeInfoMap.setMpcState(hash,addr, protocol.MpcFinish)
			}
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) getOwnerP2pMessage(PeerID *discover.NodeID, code uint64, msg interface{}) error {
	switch code {
	case protocol.MSG_MPCMessage:
		mpcMessage := msg.(*protocol.MpcMessage)
		mpcServer.getMpcMessage(PeerID, mpcMessage)
	case protocol.MSG_RequestPrepare:
//		mpcMessage := msg.(*protocol.MpcMessage)
//		go mpcServer.handleMpcPrepqare(PeerID,mpcMessage)
	case protocol.MSG_RequestMPC:
		// do nothing
	}

	return nil
}

func (mpcServer *MpcDistributor) SelfNodeId() *discover.NodeID {
	return &mpcServer.P2pMessager.Self().ID
}

func (mpcServer *MpcDistributor) P2pMessage(peerID *discover.NodeID, code uint64, msg interface{}) error {
	if *peerID == *mpcServer.SelfNodeId() {
		mpcServer.getOwnerP2pMessage(mpcServer.SelfNodeId(), code, msg)
	} else {
		err := mpcServer.P2pMessager.SendToPeer(peerID, code, msg)
		if err != nil {
			log.Error("BoardcastMessage fail. err:%s", err.Error())
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) BoardcastMessage(peers []*discover.NodeID, code uint64, msg interface{}) error {
	if peers == nil {
		peers = mpcServer.P2pMessager.GetPeers()
//		if code != 4 && code != 7 {
//			log.Error("----------------------BoardcastMessage","code",code)
//		}
	}
	for i:=0;i<len(peers);i++ {
		if *peers[i] == *mpcServer.SelfNodeId() {
			mpcServer.getOwnerP2pMessage(mpcServer.SelfNodeId(), code, msg)
		} else {
			err := mpcServer.P2pMessager.SendToPeer(peers[i], code, msg)
			if err != nil {
				log.Error("BoardcastMessage fail."," peer,", peers[i],"error", err.Error())
			}
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) CreateKeystore(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface) error {
	log.Info("MpcDistributor.CreateKeystore begin")
	point, err := result.GetValue(protocol.PublicKeyResult)
	if err != nil {
		log.Error("CreateKeystore fail. get PublicKeyResult fail")
		return err
	}

	private, err := result.GetValue(protocol.MpcPrivateShare)
	if err != nil {
		log.Error("CreateKeystore fail. get MpcPrivateShare fail")
		return err
	}
	pub := point.([2]*big.Int)
	result1 := new(ecdsa.PublicKey)
	result1.Curve = secp256k1.S256()
	result1.X = pub[0]
	result1.Y = pub[1]
	seed := nodeInfo.GetSeed(mpcServer.SelfNodeId())
	peers := nodeInfo.GetPeers()
	mpcGroup := make([]uint64,len(peers))
	for i,peer := range peers{
		mpcGroup[i] = peer.Seed
	}
	account,err := mpcServer.mpcKeyStore.StoreMpcKey(result1,private.(*big.Int),seed,uint64(protocol.MPCDegree*2+1),mpcGroup,mpcServer.password)
	if err != nil {
		return err
	}
//	log.Error("MpcCreateKeystore","address",account)
	result.SetValue(protocol.MpcContextResult, account[:])
	return nil
}

func (mpcServer *MpcDistributor) SignTransaction(result protocol.MpcResultInterface) error {
	return nil

}
