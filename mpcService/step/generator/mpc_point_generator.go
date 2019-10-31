package generator

import (
	"crypto/ecdsa"
	"math/big"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	mpcCrypto "github.com/ethereum/go-ethereum/mpcService/crypto"
	"github.com/pkg/errors"
)

type mpcPointGenerator struct {
	seed        [2]*big.Int
	message     map[uint64][2]*big.Int
	result      [2]*big.Int
	preValueKey string
	resultKey   string
}

func CreatePointGenerator(preValueKey string,resultKey string) *mpcPointGenerator {
	return &mpcPointGenerator{message: make(map[uint64][2]*big.Int), preValueKey: preValueKey,resultKey:resultKey}
}

func (point *mpcPointGenerator) Initialize(peers []protocol.PeerInfo, result protocol.MpcResultInterface) error {
	log.Info("mpcPointGenerator.initialize begin")

	value, err := result.GetValue(point.preValueKey)
	if err != nil {
		log.Error("mpcPointGenerator.initialize get preValueKey fail")
		return err
	}
	private := value.(*big.Int)
	curve := crypto.S256()
	x, y := curve.ScalarBaseMult(private.Bytes())
	if x == nil || y == nil {
		log.Error("mpcPointGenerator.ScalarBaseMult fail. err:%s", protocol.ErrPointZero.Error())
		return protocol.ErrPointZero
	}

	point.seed[0],point.seed[1] = x, y

	log.Info("mpcPointGenerator.initialize succeed")
	return nil
}

func (point *mpcPointGenerator) CalculateResult() error {
	log.Info("mpcPointGenerator.calculateResult begin")

	result := new(ecdsa.PublicKey)
	result.Curve = crypto.S256()
	i := 0
	for _, value := range point.message {
		if i == 0 {
			result.X = new(big.Int).Set(value[0])
			result.Y = new(big.Int).Set(value[1])
			i++
		} else {
			result.X, result.Y = crypto.S256().Add(result.X, result.Y, value[0], value[1])
		}
	}

	if !mpcCrypto.ValidatePublicKey(result) {
		log.Error("mpcPointGenerator.ValidatePublicKey fail.","error", protocol.ErrPointZero)
		for _, value := range point.message {
			log.Error("mpcPointGenerator.ValidatePublicKey fail.","X",value[0],"Y",value[1])
		}
		return protocol.ErrPointZero
	}

	point.result[0],point.result[1] = result.X, result.Y

	log.Info("mpcPointGenerator.calculateResult succeed")
	return nil
}
func (point *mpcPointGenerator) GetMessageData(int)interface{}{
	return point.seed
}
func (point *mpcPointGenerator) SetMessageData(seed uint64,value interface{})error{
	_, exist := point.message[seed]
	if exist {
		log.Error("MpcPoint_Step.HandleMessage, get msg from seed fail.")
		return errors.New("MpcPoint_Step.HandleMessage, get msg from seed fail.")
	}

	point.message[seed] = value.([2]*big.Int)
	return nil
}
func (point *mpcPointGenerator) GetResultData()*protocol.MpcData{
	return &protocol.MpcData{point.resultKey,point.result}
}