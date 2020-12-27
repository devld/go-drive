package task

import (
	"context"
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

func ContextWrapper(ctx context.Context) types.TaskCtx {
	c := ctx.Done()
	t := &taskContextWrapper{}
	if c != nil {
		go func() {
			select {
			case <-c:
				t.canceled = true
			}
		}()
	}
	return t
}

var dummyCtx = &taskContextWrapper{}

type taskContextWrapper struct {
	ctx      context.Context
	canceled bool
}

func (d *taskContextWrapper) Progress(int64, bool) {
}

func (d *taskContextWrapper) Total(int64, bool) {
}

func (d *taskContextWrapper) Canceled() bool {
	return d.canceled
}

func (d *taskContextWrapper) Deadline() (deadline time.Time, ok bool) {
	if d.ctx != nil {
		deadline, ok = d.ctx.Deadline()
	}
	return
}

func (d *taskContextWrapper) Done() <-chan struct{} {
	if d.ctx != nil {
		return d.ctx.Done()
	}
	return nil
}

func (d *taskContextWrapper) Err() error {
	if d.ctx != nil {
		return d.ctx.Err()
	}
	return nil
}

func (d *taskContextWrapper) Value(v interface{}) interface{} {
	if d.ctx != nil {
		return d.ctx.Value(v)
	}
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
