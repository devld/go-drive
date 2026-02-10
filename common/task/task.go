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
)

type Status = string

type Progress struct {
	Loaded int64 `json:"loaded"`
	Total  int64 `json:"total"`
}

type Task struct {
	Id        string    `json:"id"`
	Status    Status    `json:"status"`
	Progress  Progress  `json:"progress"`
	Result    any       `json:"result"`
	Error     any       `json:"error"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// meta data
	Name  string `json:"name"`
	Group string `json:"group"`
}

func (t Task) Finished() bool {
	return t.Status == Done || t.Status == Error || t.Status == Canceled
}

type Runnable = func(ctx types.TaskCtx) (any, error)

type Runner interface {
	Execute(runnable Runnable, options ...Option) (Task, error)
	ExecuteAndWait(runnable Runnable, timeout time.Duration, options ...Option) (Task, error)
	GetTask(id string) (Task, error)
	GetTasks(group string) ([]Task, error)
	StopTask(id string) (Task, error)
	RemoveTask(id string) error
	Dispose() error
}
