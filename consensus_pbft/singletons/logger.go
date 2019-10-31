package singletons

import (
	"github.com/ethereum/go-ethereum/log"
	"fmt"
)

type PbftLogger struct {

}
func (pl *PbftLogger)Debug(args... interface{}){
	log.Debug(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Debugln(args ...interface{}){
	log.Debug(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Debugf(msg string, args... interface{}){
	log.Debug(fmt.Sprintf(msg,args...))
}

func (pl *PbftLogger)Info(args ...interface{}){
	log.Info(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Infoln(args ...interface{}){
	log.Info(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Infof(msg string, args ...interface{}){
	log.Info(fmt.Sprintf(msg,args...))
}

func (pl *PbftLogger)Warn(args ...interface{}){
	log.Warn(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Warnln(args ...interface{}){
	log.Warn(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Warnf(msg string, args ...interface{}){
	log.Warn(fmt.Sprintf(msg,args...))
}

func (pl *PbftLogger)Error(args ...interface{}){
	log.Error(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Errorln(args ...interface{}){
	log.Error(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Errorf(msg string, args ...interface{}){
	log.Error(fmt.Sprintf(msg,args...))
}

func (pl *PbftLogger)Fatal(args ...interface{}){
	log.Crit(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Fatalln(args ...interface{}){
	log.Crit(args[0].(string),args[1:]...)
}
func (pl *PbftLogger)Fatalf(msg string, args ...interface{}){
	log.Crit(fmt.Sprintf(msg,args...))
}

func (pl *PbftLogger)SetFormat(string) error{
	return nil
}
func (pl *PbftLogger)SetLevel(string) error{
	return nil
}

