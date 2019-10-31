package singletons

import "github.com/ethereum/go-ethereum/consensus_pbft/pbftTypes"

type Logger interface {
	Debug(...interface{})
	Debugln(...interface{})
	Debugf(string, ...interface{})

	Info(...interface{})
	Infoln(...interface{})
	Infof(string, ...interface{})

	Warn(...interface{})
	Warnln(...interface{})
	Warnf(string, ...interface{})

	Error(...interface{})
	Errorln(...interface{})
	Errorf(string, ...interface{})

	Fatal(...interface{})
	Fatalln(...interface{})
	Fatalf(string, ...interface{})

	SetFormat(string) error
	SetLevel(string) error
}
type MarshalInterface interface {
	Marshal(data interface{}) ([]byte, error)
	Unmarshal(buf []byte, data interface{}) error
}
type HashInterface interface {
	Hash(interface{}) pbftTypes.MessageDigest
}