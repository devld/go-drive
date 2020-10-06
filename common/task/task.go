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
	Progress(loaded int64, abs bool)
	Total(total int64, abs bool)
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

func (d *dummyContext) Progress(int64, bool) {
}

func (d *dummyContext) Total(int64, bool) {
}

func (d *dummyContext) Canceled() bool {
	return false
}

func NewCtxWrapper(ctx Context, mutableLoaded, mutableTotal bool) Context {
	return &ctxWrapper{
		mutableLoaded: mutableLoaded,
		mutableTotal:  mutableTotal,
		ctx:           ctx,
	}
}

type ctxWrapper struct {
	mutableLoaded bool
	mutableTotal  bool
	ctx           Context
}

func (c *ctxWrapper) Progress(loaded int64, abs bool) {
	if c.mutableLoaded {
		c.ctx.Progress(loaded, abs)
	}
}

func (c *ctxWrapper) Total(total int64, abs bool) {
	if c.mutableTotal {
		c.ctx.Total(total, abs)
	}
}

func (c *ctxWrapper) Canceled() bool {
	return c.ctx.Canceled()
}
