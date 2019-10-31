package message

import "context"

type Task struct {
	Type uint32
	TimeStamp uint64
	ReadOnly bool
	Payload []byte
}
type Result struct {
	Type uint32
	TimeStamp uint64
	Payload []byte
}
type TaskExecuteInterface interface {
	Execute(ctxt context.Context,task *Task)(Result, error)
}
