package task

import (
	"errors"
	"time"
)

const (
	Pending  = "pending"
	Running  = "running"
	Done     = "done"
	Error    = "error"
	Canceled = "canceled"
)

var (
	ErrorNotFound = errors.New("task not found")
	ErrorCanceled = errors.New("canceled")
)

type Status = string

type Progress struct {
	Loaded int64 `json:"loaded"`
	Total  int64 `json:"total"`
}

type Task struct {
	Id        string      `json:"id"`
	Status    Status      `json:"status"`
	Progress  Progress    `json:"progress"`
	Result    interface{} `json:"result"`
	Error     interface{} `json:"error"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (t Task) Finished() bool {
	return t.Status == Done || t.Status == Error || t.Status == Canceled
}

type Context interface {
	Progress(loaded int64)
	Total(total int64)
	Canceled() bool
}

type Runnable = func(ctx Context) (interface{}, error)

type Runner interface {
	Execute(runnable Runnable) (Task, error)
	ExecuteAndWait(runnable Runnable, timeout time.Duration) (Task, error)
	GetTask(id string) (Task, error)
	StopTask(id string) (Task, error)
	RemoveTask(id string) error
	Dispose() error
}

func DummyContext() Context {
	return dummyCtx
}

var dummyCtx = &dummyContext{}

type dummyContext struct {
}

func (d *dummyContext) Progress(int64) {
}

func (d *dummyContext) Total(int64) {
}

func (d *dummyContext) Canceled() bool {
	return false
}
