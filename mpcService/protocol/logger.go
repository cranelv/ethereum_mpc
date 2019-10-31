package protocol

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"sync"
	"strings"
)
type TestLogger struct {
	mu sync.Mutex
	LogMap map[string]int
}
func (lg *TestLogger)Error(format string, ctx ...interface{}){
	txt := fmt.Sprintf(format,ctx...)
//	log.Error(txt)
	lg.mu.Lock()
	txtAry := strings.Split(txt,",")
	for _,tx := range txtAry{
		item := strings.Trim(tx, " ")
		lg.LogMap[item]++
	}
	defer lg.mu.Unlock()
}
var TLog = &TestLogger{
	LogMap:make(map[string]int),
}
type MpcLogger struct {
	Log bool
}
func (pl *MpcLogger)Debug(args... interface{}){
	if pl.Log{
		log.Debug(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Debugln(args ...interface{}){
	if pl.Log{
	log.Debug(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Debugf(msg string, args... interface{}){
	if pl.Log{
		log.Debug(fmt.Sprintf(msg,args...))
	}
}

func (pl *MpcLogger)Info(args ...interface{}){
	if pl.Log{
		log.Info(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Infoln(args ...interface{}){
	if pl.Log{
		log.Info(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Infof(msg string, args ...interface{}){
	if pl.Log{
		log.Info(fmt.Sprintf(msg,args...))
	}
}

func (pl *MpcLogger)Warn(args ...interface{}){
	if pl.Log{
		log.Warn(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Warnln(args ...interface{}){
	if pl.Log{
		log.Warn(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Warnf(msg string, args ...interface{}){
	if pl.Log{
		log.Warn(fmt.Sprintf(msg,args...))
	}
}

func (pl *MpcLogger)Error(args ...interface{}){
	if pl.Log{
		log.Error(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Errorln(args ...interface{}){
	if pl.Log{
		log.Error(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Errorf(msg string, args ...interface{}){
	if pl.Log{
		log.Error(fmt.Sprintf(msg,args...))
	}
}

func (pl *MpcLogger)Fatal(args ...interface{}){
	if pl.Log{
		log.Crit(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Fatalln(args ...interface{}){
	if pl.Log{
		log.Crit(args[0].(string),args[1:]...)
	}
}
func (pl *MpcLogger)Fatalf(msg string, args ...interface{}){
	if pl.Log{
		log.Crit(fmt.Sprintf(msg,args...))
	}
}

func (pl *MpcLogger)SetFormat(string) error{
	return nil
}
func (pl *MpcLogger)SetLevel(string) error{
	return nil
}

