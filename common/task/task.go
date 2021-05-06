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

// NewContextWrapper wraps Context as a TaskCtx
func NewContextWrapper(ctx context.Context) types.TaskCtx {
	return &taskContextWrapper{ctx}
}

var dummyCtx = &taskContextWrapper{context.Background()}

type taskContextWrapper struct {
	context.Context
}

func (d *taskContextWrapper) Progress(int64, bool) {
}

func (d *taskContextWrapper) Total(int64, bool) {
}

func NewCtxWrapper(ctx types.TaskCtx, mutableLoaded, mutableTotal bool) types.TaskCtx {
	return &ctxWrapper{
		TaskCtx:       ctx,
		mutableLoaded: mutableLoaded,
		mutableTotal:  mutableTotal,
	}
}

type ctxWrapper struct {
	types.TaskCtx
	mutableLoaded bool
	mutableTotal  bool
}

func (c *ctxWrapper) Progress(loaded int64, abs bool) {
	if c.mutableLoaded {
		c.TaskCtx.Progress(loaded, abs)
	}
}

func (c *ctxWrapper) Total(total int64, abs bool) {
	if c.mutableTotal {
		c.TaskCtx.Total(total, abs)
	}
}
