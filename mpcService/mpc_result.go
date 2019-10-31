package mpcService

import (
	"github.com/ethereum/go-ethereum/log"
	"sync"
	"github.com/ethereum/go-ethereum/mpcService/protocol"
)

type BaseMpcResult struct {
	mu sync.RWMutex
	Result     map[string]interface{}
}
func createMpcBaseMpcResult() *BaseMpcResult {
	return &BaseMpcResult{Result:make(map[string]interface{})}
}

func (result *BaseMpcResult) Initialize(preSetValue ... protocol.MpcValue) {
	log.Info("BaseMpcResult.InitializeValue begin")
	for i := 0; i < len(preSetValue); i++ {
		result.Result[preSetValue[i].Key] = preSetValue[i].Value
	}
}


func (mpc *BaseMpcResult) SetValue(key string, value interface{}) error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()
	mpc.Result[key] = value
	return nil
}

func (mpc *BaseMpcResult) GetValue(key string) (interface{}, error) {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()
	value, exist := mpc.Result[key]
	if exist {
		return value, nil
	}

	log.Error("BaseMpcResult GetValue fail." ,"key", key)
	return value, protocol.ErrMpcResultExist
}

