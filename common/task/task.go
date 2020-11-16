package task

import (
	"errors"
	"go-drive/common/types"
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

type Runnable = func(ctx types.TaskCtx) (interface{}, error)

type Runner interface {
	Execute(runnable Runnable) (Task, error)
	ExecuteAndWait(runnable Runnable, timeout time.Duration) (Task, error)
	GetTask(id string) (Task, error)
	StopTask(id string) (Task, error)
	RemoveTask(id string) error
	Dispose() error
}

func DummyContext() types.TaskCtx {
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

func (d *dummyContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (d *dummyContext) Done() <-chan struct{} {
	return nil
}

func (d *dummyContext) Err() error {
	return nil
}

func (d *dummyContext) Value(interface{}) interface{} {
	return nil
}

func NewCtxWrapper(ctx types.TaskCtx, mutableLoaded, mutableTotal bool) types.TaskCtx {
	return &ctxWrapper{
		mutableLoaded: mutableLoaded,
		mutableTotal:  mutableTotal,
		cancelable:    true,
		ctx:           ctx,
	}
}

func NewProgressCtxWrapper(ctx types.TaskCtx) types.TaskCtx {
	return &ctxWrapper{
		mutableLoaded: true,
		mutableTotal:  true,
		cancelable:    false,
		ctx:           ctx,
	}
}

type ctxWrapper struct {
	mutableLoaded bool
	mutableTotal  bool
	cancelable    bool
	ctx           types.TaskCtx
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
	if !c.cancelable {
		return false
	}
	return c.ctx.Canceled()
}

func (c *ctxWrapper) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *ctxWrapper) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *ctxWrapper) Err() error {
	return c.ctx.Err()
}

func (c *ctxWrapper) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}
