/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package persist

import (
	"sync"
	"github.com/ethereum/go-ethereum/common"
	"bytes"
)

// Helper provides an abstraction to access the Persist column family
// in the database.
type Helper struct{
	mutex sync.RWMutex
	db map[string] []byte
}

func (h *Helper)Put(key []byte,value []byte) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.db[common.Bytes2Hex(key)] = value
	return nil
}
func (h *Helper)Get(key []byte) ([]byte, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if value,exist := h.db[common.Bytes2Hex(key)];exist{
		return value,nil
	}
	return nil,nil
}
func (h *Helper)Delete(key []byte) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.db,common.Bytes2Hex(key))
	return nil
}
// StoreState stores a key,value pair
func (h *Helper) StoreState(key string, value []byte) error {
	return h.Put( []byte("consensus."+key), value)
}

// DelState removes a key,value pair
func (h *Helper) DelState(key string) {
	h.Delete([]byte("consensus."+key))
}

// ReadState retrieves a value to a key
func (h *Helper) ReadState(key string) ([]byte, error) {
	return h.Get([]byte("consensus."+key))
}

// ReadStateSet retrieves all key,value pairs where the key starts with prefix
func (h *Helper) ReadStateSet(prefix string) (map[string][]byte, error) {
	prefixRaw := []byte("consensus." + prefix)
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	ret := make(map[string][]byte)
	for key,value := range h.db	{
		strKey := common.Hex2Bytes(key)
		if bytes.HasPrefix(strKey,prefixRaw){
			subkey := string(strKey)
			subkey = subkey[len("consensus."):]
			ret[key] = append([]byte(nil), value...)
		}
	}
	return ret, nil
}
