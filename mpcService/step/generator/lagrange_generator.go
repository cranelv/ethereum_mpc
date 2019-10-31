package generator

import (
	"math/big"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mpcService/crypto"
	"errors"
)

type lagrangeGenerator struct {
	seed        *big.Int
	message     map[uint64]*big.Int
	result      *big.Int
	resultKey   string
	preValueKey string
}

func CreateLagrangeGenerator(preValueKey,resultKey string) *lagrangeGenerator {
	return &lagrangeGenerator{message: make(map[uint64]*big.Int), preValueKey: preValueKey,resultKey:resultKey}
}

func (lag *lagrangeGenerator) Initialize(peers []protocol.PeerInfo, result protocol.MpcResultInterface) error {
	log.Info("lagrangeGenerator.initialize begin")

	value, err := result.GetValue(lag.preValueKey)
	if err != nil {
		return err
	}

	lag.seed = value.(*big.Int)

	log.Info("lagrangeGenerator.initialize succeed")
	return nil
}

func (lag *lagrangeGenerator) CalculateResult() error {
	log.Info("lagrangeGenerator.calculateResult begin")

	f := []*big.Int{}
	seed := []*big.Int{}
	for key, value := range lag.message {
		f = append(f, value)
		seed = append(seed, new(big.Int).SetUint64(key))
	}

	lag.result = crypto.Lagrange(f, seed)
	return nil
}
func (lag *lagrangeGenerator)GetMessageData(int)interface{}{
	return lag.seed
}
func (lag *lagrangeGenerator) SetMessageData(seed uint64,value interface{})error{
	_, exist := lag.message[seed]
	if exist {
		log.Error("MpcPoint_Step.HandleMessage, get msg from seed fail.")
		return errors.New("MpcPoint_Step.HandleMessage, get msg from seed fail.")
	}

	lag.message[seed] = value.(*big.Int)
	return nil
}
func (lag *lagrangeGenerator)GetResultData()*protocol.MpcData{
//	if lag.resultKey == protocol.MpcSignARResult{
//		log.Error("ARVALUE1","ar",lag.result)
//	}
	return &protocol.MpcData{lag.resultKey,lag.result}
}