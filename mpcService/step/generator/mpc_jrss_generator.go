package generator

import (
	"math/big"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
	"github.com/ethereum/go-ethereum/log"
	"errors"
)

type RandomPolynomialValue struct {
	RandCoefficient []*big.Int          //coefficient
	message         map[uint64]big.Int //Polynomil result
	PolyValue       []*big.Int
	Result          *big.Int
	resultKey       string
	bJRSS           bool
}

func CreateJRSSValue(degree int, peerNum int,resultKey string) *RandomPolynomialValue {
	return &RandomPolynomialValue{make([]*big.Int, degree+1), make(map[uint64]big.Int),
	make([]*big.Int, peerNum), nil, resultKey,true}
}

func CreateJZSSValue(degree int, peerNum int,resultKey string) *RandomPolynomialValue {
	return &RandomPolynomialValue{make([]*big.Int, degree+1), make(map[uint64]big.Int),
	make([]*big.Int, peerNum), nil, resultKey,false}
}

func (poly *RandomPolynomialValue) Initialize(peers []protocol.PeerInfo, result protocol.MpcResultInterface) error {
	cof, err := crypto.GetRandCoefficients(len(poly.RandCoefficient))
	if err != nil {
		log.Error("RandomPolynomialValue, GetRandCoefficients fail. err:%s", err.Error())
		return err
	}

	copy(poly.RandCoefficient, cof)
	if !poly.bJRSS {
		poly.RandCoefficient[0] = big.NewInt(0)
	}

	for i := 0; i < len(poly.PolyValue); i++ {
		poly.PolyValue[i] = crypto.EvaluatePoly(poly.RandCoefficient, new(big.Int).SetUint64(uint64(peers[i].Seed)))
	}

	return nil
}

func (poly *RandomPolynomialValue) CalculateResult() error {
	poly.Result = big.NewInt(0)
	for _, value := range poly.message {
		poly.Result.Add(poly.Result, &value)
		poly.Result.Mod(poly.Result, crypto.Secp256k1N)
	}

	return nil
}
func (poly *RandomPolynomialValue)GetMessageData(i int)interface{} {
	return poly.PolyValue[i]
}
func (poly *RandomPolynomialValue) SetMessageData(seed uint64,value interface{})error{
	_, exist := poly.message[seed]
	if exist {
		log.Error("MpcPoint_Step.HandleMessage, get msg from seed fail.")
		return errors.New("MpcPoint_Step.HandleMessage, get msg from seed fail.")
	}

	poly.message[seed] = *(value.(*big.Int))
	return nil
}
func (poly *RandomPolynomialValue)GetResultData()*protocol.MpcData {
	return &protocol.MpcData{poly.resultKey,poly.Result}
}