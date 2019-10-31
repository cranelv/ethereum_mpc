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

package pbft

import (
	"github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"
)

type signable interface {
	GetSignature() []byte
	SetSignature(s []byte)
	GetID() pbftTypes.ReplicaID
	SetID(id pbftTypes.ReplicaID)
	Serialize() ([]byte, error)
}

func (instance *pbftCore) sign(s signable) error {
	s.SetSignature(nil)
	raw, err := s.Serialize()
	if err != nil {
		return err
	}
	signedRaw, err := instance.consumer.sign(raw)
	if err != nil {
		return err
	}
	s.SetSignature(signedRaw)

	return nil
}

func (instance *pbftCore) verify(s signable) error {
	origSig := s.GetSignature()
	s.SetSignature(nil)
	raw, err := s.Serialize()
	s.SetSignature(origSig)
	if err != nil {
		return err
	}
	return instance.consumer.verify(s.GetID(), origSig, raw)
}

