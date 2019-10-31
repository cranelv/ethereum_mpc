package protocol

import (
	"bytes"
	"time"
	"strconv"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"io"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

const (
	MpcCreateLockAccountLeader = iota + 0
	MpcCreateLockAccountPeer
	MpcTXSignLeader
	MpcTXSignPeer
	MpcReady
)
const (
	StatusCode = iota + 0 // used by storeman protocol
	KeepaliveCode
	KeepaliveOkCode
	MSG_MPCError
	MSG_RequestPrepare
	MSG_RequestMPC // ask for a new mpc Context
	MSG_MPCMessage // get a message for a Context
	MSG_MPCFinish

	KeepaliveCycle
	NumberOfMessageCodes

	MPCDegree          = 2
	//MPCDegree          = 1
	MPCTimeOut         = time.Second * 30
	ProtocolName       = "storeman"
	ProtocolVersion    = uint64(1)
	ProtocolVersionStr = "1.0"
)
const (
	MpcSeed  		 = "Seed"
	MpcPrivateShare  = "PrivateShare"
	MpcPrivateKey    = "PrivateKey"
	MpcPublicShare   = "PublicShare"
	MpcPointPart	 = "PointPart"
	MpcSignASeed     = "SignASeed"
	MpcSignA         = "SignA"
	MpcSignA0        = "SignA0"
	MpcSignRSeed     = "SignRSeed"
	MpcSignR         = "SignR"
	MpcSignR0        = "SignR0"
	MpcSignB         = "SignB"
	MpcSignBSeed     = "SignBSeed"
	MpcSignC         = "SignC"
	MpcSignCSeed     = "SignCSeed"
	MpcSignARSeed    = "SignARSeed"
	MpcSignARResult  = "SignARResult"
	MpcTxSignSeed    = "TxSignSeed"
	MpcTxSignResultR = "TxSignResultR"
	MpcTxSignResultV = "TxSignResultV"
	MpcTxSignResult  = "TxSignResult"
	MpcContextResult = "ContextResult"

	PublicKeyResult = "PublicKeyResult"
	MpcSignAPoint   = "SignAPoint"
	MpcTxHash       = "TxHash"
	MpcLagrange     = "Lagrange"
	MpcTransaction  = "Transaction"
	MpcChainType    = "ChainType"
	MpcSignType     = "SignType"
	MpcChainID      = "ChainID"
	MpcAddress      = "Address"
	MPCAction       = "Action"
	MPCPolyvalue   = "Polyvalue"
	MPCSignedFrom   = "SignedFrom"
	MpcStmAccType   = "StmAccType"
)
type PeerMessage struct {
	From discover.NodeID
	Message *p2p.Msg
}
type PeerInfo struct {
	PeerID *discover.NodeID
	Seed   uint64
}
type SlicePeers []*discover.NodeID

func (s SlicePeers) Len() int {
	return len(s)
}
func (s SlicePeers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SlicePeers) Less(i, j int) bool {
	return bytes.Compare(s[i][:], s[j][:]) < 0
}
type MpcStepFunc interface {
	GetMessageInterface
	InitMessageLoop(GetMessageInterface) error
	Quit(error)
	InitStep() error
	CreateMessage() []StepMessage
	FinishStep(MpcManager) error
	GetMessageChan() chan *StepMessage
}
type GetMessageInterface interface {
	HandleMessage(*StepMessage) bool
}
type MpcData struct{
	Key string
	Data interface{}
}
type MpcDataTemp struct{
	Key string
	Data interface{}
}
func (md *MpcData) EncodeRLP(w io.Writer) error{
	temp := MpcDataTemp{md.Key,md.Data}
	switch md.Data.(type){
	case uint64:
		buff := make([]byte,8)
		binary.BigEndian.PutUint64(buff, md.Data.(uint64))
		temp.Data = buff
	case uint32:
		buff := make([]byte,4)
		binary.BigEndian.PutUint32(buff, md.Data.(uint32))
		temp.Data = buff
	case uint16:
		buff := make([]byte,2)
		binary.BigEndian.PutUint16(buff, md.Data.(uint16))
		temp.Data = buff
	case uint:
		buff := make([]byte,32)
		binary.BigEndian.PutUint32(buff, uint32(md.Data.(uint)))
		temp.Data = buff
	}
	return rlp.Encode(w,temp)
}
func (md *MpcData)DecodeRLP(s *rlp.Stream) error{
	temp := &MpcDataTemp{}
	if err := s.Decode(temp); err != nil {
		return err
	}
	md.Key = temp.Key
	md.Data = temp.Data
	switch md.Key {
	case MpcSeed:
		md.Data = binary.BigEndian.Uint64(md.Data.([]byte))
		//big int
	case MPCPolyvalue,MpcSignA,MpcSignA0,MpcSignR,MpcSignARSeed,MpcSignB,MpcPublicShare,MpcSignC,
		MpcSignASeed,MpcSignRSeed,MpcSignBSeed,MpcSignCSeed,MpcLagrange:
		data := new(big.Int).SetBytes(md.Data.([]byte))
		md.Data = data
	case MpcPointPart:
		value := md.Data.([]interface{})
		var data [2]*big.Int
		data[0] = big.NewInt(0)
		data[0].SetBytes(value[0].([]byte))
		data[1] = big.NewInt(0)
		data[1].SetBytes(value[1].([]byte))
		md.Data = data
	}
	return nil
}
type StepMessage struct {
	Msgcode   uint64 //message code
	PeerID    *discover.NodeID
	Peers     []PeerInfo
	Data 	  []*MpcData
}

type MpcMessage struct {
	ContextID common.Hash
	StepID    uint64
	Peers     []byte
	Data 	  []*MpcData
	Error     string
}

func CheckAccountType(accType string) bool {
	if accType == "WAN" || accType == "ETH" || accType == "BTC" {
		return true
	}

	return false
}

func GetPreSetKeyArr(keySeed string, num int) []string {
	keyArr := []string{}
	for i := 0; i < num; i++ {
		keyArr = append(keyArr, keySeed + "_" + strconv.Itoa(i))
	}

	return keyArr
}