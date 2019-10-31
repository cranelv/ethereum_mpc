package message

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
	"github.com/ethereum/go-ethereum/consensus_pbft/singletons"
	"github.com/pkg/errors"
)

type MessageInterface interface {
	Marshal()([]byte,error)
	Sender()pbftTypes.ReplicaID
}
type RawMessage struct {
	Type uint32
	Payload []byte
}
func MarshalMsgList(msgType uint32,msgList interface{})([]byte,error){
	payload,err := singletons.Marshaler.Marshal(msgList)
	if err != nil {
		return nil,err
	}
	rmsg := RawMessage{
		Type: msgType,
		Payload:payload,
	}
	return singletons.Marshaler.Marshal(rmsg)
}
func MarshalMsg(msgType uint32,msg MessageInterface)([]byte,error){
	payload,err := singletons.Marshaler.Marshal(msg)
	if err != nil {
		return nil,err
	}
	rmsg := RawMessage{
		Type: msgType,
		Payload:payload,
	}
	return singletons.Marshaler.Marshal(rmsg)
}
func UnmarshalMsgList(buff []byte) (interface{},error){
	var rmsg RawMessage
	err := singletons.Marshaler.Unmarshal(buff,&rmsg)
	if err != nil{
		return nil,err
	}
	var msg interface{}
	switch rmsg.Type {
	case Message_Request:
		msg = &[]*Request{}
	case Message_RequestBatch:
		msg = &[]RequestBatch{}
	case Message_PrePrepare:
		msg = &[]PrePrepare{}
	case Message_Prepare:
		msg = &[]Prepare{}
	case Message_Commit:
		msg = &[]Commit{}
	case Message_Checkpoint:
		msg = &[]Checkpoint{}
	case Message_ViewChange:
		msg = &[]ViewChange{}
	case Message_NewView:
		msg = &[]NewView{}
	case Message_FetchRequestBatch:
		msg = &[]FetchRequestBatch{}
	case Message_ReturnRequestBatch:
		msg = &[]ReturnRequestBatch{}
	default:
		return nil,errors.Errorf("Missing Message Type %d !",rmsg.Type)
	}
	err = singletons.Marshaler.Unmarshal(rmsg.Payload,msg)
	return msg,err
}
func UnmarshalMsg(buff []byte,msg *MessageInterface)error{
	var rmsg RawMessage
	err := singletons.Marshaler.Unmarshal(buff,&rmsg)
	if err != nil{
		return err
	}
	switch rmsg.Type {
	case Message_Request:
		*msg = &Request{}
	case Message_RequestBatch:
		*msg = &RequestBatch{}
	case Message_PrePrepare:
		*msg = &PrePrepare{}
	case Message_Prepare:
		*msg = &Prepare{}
	case Message_Commit:
		*msg = &Commit{}
	case Message_Checkpoint:
		*msg = &Checkpoint{}
	case Message_ViewChange:
		*msg = &ViewChange{}
	case Message_NewView:
		*msg = &NewView{}
	case Message_FetchRequestBatch:
		*msg = &FetchRequestBatch{}
	case Message_ReturnRequestBatch:
		*msg = &ReturnRequestBatch{}
	default:
		return errors.Errorf("Missing Message Type %d !",rmsg.Type)
	}
	return singletons.Marshaler.Unmarshal(rmsg.Payload,*msg)
}
func Digest(msg MessageInterface)pbftTypes.MessageDigest{
	payload,_ := msg.Marshal()
	return singletons.Hasher.Hash(payload)
}
type Request struct {
	Timestamp uint64
	ReplicaId      pbftTypes.ReplicaID
	Tasks []*Task
	Signature []byte
}
func (req*Request)Marshal()([]byte,error){
	return MarshalMsg(Message_Request,req)
}
func (req*Request)Sender()pbftTypes.ReplicaID{
	return req.ReplicaId
}
type RequestBatch []*Request
func (req *RequestBatch)Marshal()([]byte,error){
	return MarshalMsg(Message_RequestBatch,req)
}
func (req*RequestBatch)Sender()pbftTypes.ReplicaID{
	return pbftTypes.ReplicaID(0)
}
type PrePrepare struct {
	View           uint64        `protobuf:"varint,1,opt,name=view" json:"view,omitempty"`
	SequenceNumber uint64        `protobuf:"varint,2,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	BatchDigest    pbftTypes.MessageDigest        `protobuf:"bytes,3,opt,name=batch_digest,json=batchDigest" json:"batch_digest,omitempty"`
	RequestBatch   *RequestBatch		 `protobuf:"bytes,4,opt,name=request_batch,json=requestBatch" json:"request_batch,omitempty"`
	ReplicaId      pbftTypes.ReplicaID        `protobuf:"varint,5,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
}
func (pp *PrePrepare)Marshal()([]byte,error){
	return MarshalMsg(Message_PrePrepare,pp)
}
func (pp *PrePrepare)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
type Prepare struct {
	View           uint64 `protobuf:"varint,1,opt,name=view" json:"view,omitempty"`
	SequenceNumber uint64 `protobuf:"varint,2,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	BatchDigest    pbftTypes.MessageDigest `protobuf:"bytes,3,opt,name=batch_digest,json=batchDigest" json:"batch_digest,omitempty"`
	ReplicaId      pbftTypes.ReplicaID `protobuf:"varint,4,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
}
func (pre *Prepare)Marshal()([]byte,error){
	return MarshalMsg(Message_Prepare,pre)
}
func (pp *Prepare)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
type Commit struct {
	View           uint64 `protobuf:"varint,1,opt,name=view" json:"view,omitempty"`
	SequenceNumber uint64 `protobuf:"varint,2,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	BatchDigest    pbftTypes.MessageDigest `protobuf:"bytes,3,opt,name=batch_digest,json=batchDigest" json:"batch_digest,omitempty"`
	ReplicaId      pbftTypes.ReplicaID `protobuf:"varint,4,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
}
func (cmt *Commit)Marshal()([]byte,error){
	return MarshalMsg(Message_Commit,cmt)
}
func (pp *Commit)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
type Checkpoint struct {
	SequenceNumber uint64 `protobuf:"varint,1,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	ReplicaId      pbftTypes.ReplicaID `protobuf:"varint,2,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
	Id             string `protobuf:"bytes,3,opt,name=id" json:"id,omitempty"`
}
func (cpt *Checkpoint)Marshal()([]byte,error){
	return MarshalMsg(Message_Checkpoint,cpt)
}
func (pp *Checkpoint)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
// This message should go away and become a checkpoint once replica_id is removed
type ViewChange_C struct {
	SequenceNumber uint64 `protobuf:"varint,1,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	Id             string `protobuf:"bytes,3,opt,name=id" json:"id,omitempty"`
}

type ViewChange_PQ struct {
	SequenceNumber uint64 `protobuf:"varint,1,opt,name=sequence_number,json=sequenceNumber" json:"sequence_number,omitempty"`
	BatchDigest    pbftTypes.MessageDigest `protobuf:"bytes,2,opt,name=batch_digest,json=batchDigest" json:"batch_digest,omitempty"`
	View           uint64 `protobuf:"varint,3,opt,name=view" json:"view,omitempty"`
}

type ViewChange struct {
	View      uint64           `protobuf:"varint,1,opt,name=view" json:"view,omitempty"`
	H         uint64           `protobuf:"varint,2,opt,name=h" json:"h,omitempty"`
	Cset      []*ViewChange_C  `protobuf:"bytes,3,rep,name=cset" json:"cset,omitempty"`
	Pset      []*ViewChange_PQ `protobuf:"bytes,4,rep,name=pset" json:"pset,omitempty"`
	Qset      []*ViewChange_PQ `protobuf:"bytes,5,rep,name=qset" json:"qset,omitempty"`
	ReplicaId pbftTypes.ReplicaID           `protobuf:"varint,6,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
	Signature []byte           `protobuf:"bytes,7,opt,name=signature,proto3" json:"signature,omitempty"`
}
func (pp *ViewChange)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
func (vc *ViewChange)Marshal()([]byte,error){
	return MarshalMsg(Message_ViewChange,vc)
}
func (vc *ViewChange) GetSignature() []byte {
	return vc.Signature
}

func (vc *ViewChange) SetSignature(sig []byte) {
	vc.Signature = sig
}

func (vc *ViewChange) GetID() pbftTypes.ReplicaID  {
	return vc.ReplicaId
}

func (vc *ViewChange) SetID(id pbftTypes.ReplicaID ) {
	vc.ReplicaId = id
}

func (vc *ViewChange) Serialize() ([]byte, error) {
	return vc.Marshal()
}

type NewView struct {
	View      uint64            `protobuf:"varint,1,opt,name=view" json:"view,omitempty"`
	Vset      []*ViewChange     `protobuf:"bytes,2,rep,name=vset" json:"vset,omitempty"`
	Xset      map[uint64]pbftTypes.MessageDigest `protobuf:"bytes,3,rep,name=xset" json:"xset,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	ReplicaId pbftTypes.ReplicaID            `protobuf:"varint,4,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
}
func (pp *NewView)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
func (nv *NewView)Marshal()([]byte,error){
	return MarshalMsg(Message_NewView,nv)
}
type FetchRequestBatch struct {
	BatchDigest pbftTypes.MessageDigest `protobuf:"bytes,1,opt,name=batch_digest,json=batchDigest" json:"batch_digest,omitempty"`
	ReplicaId   pbftTypes.ReplicaID `protobuf:"varint,2,opt,name=replica_id,json=replicaId" json:"replica_id,omitempty"`
}
func (pp *FetchRequestBatch)Sender()pbftTypes.ReplicaID{
	return pp.ReplicaId
}
func (fr *FetchRequestBatch)Marshal()([]byte,error){
	return MarshalMsg(Message_FetchRequestBatch,fr)
}
type ReturnRequestBatch []*Request
func (req *ReturnRequestBatch)Marshal()([]byte,error){
	return MarshalMsg(Message_ReturnRequestBatch,req)
}
func (pp *ReturnRequestBatch)Sender()pbftTypes.ReplicaID{
	return pbftTypes.ReplicaID(0)
}