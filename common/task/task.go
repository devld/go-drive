package task

import (
	"context"
	"encoding/json"
	"errors"
	err "go-drive/common/errors"
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
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	Progress  Progress  `json:"progress"`
	Result    any       `json:"result"`
	Error     error     `json:"error"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// meta data
	Name  string `json:"name"`
	Group string `json:"group"`
}

type jsonError struct{ error }

func (e jsonError) MarshalJSON() ([]byte, error) {
	payload := types.M{"message": e.Error()}
	if public, ok := errors.AsType[err.Error](e.error); ok {
		payload["code"] = public.Code()
	}
	return json.Marshal(payload)
}

func (e jsonError) Unwrap() error { return e.error }

func (t Task) Finished() bool {
	return t.Status == Done || t.Status == Error || t.Status == Canceled
}

type Runnable = func(ctx types.TaskCtx) (any, error)

type TaskIDProvider interface {
	TaskID() string
}

type Runner interface {
	Execute(runnable Runnable, options ...Option) (Task, error)
	// ExecuteAndWait limits only how long the caller waits. The task is created
	// with its own detached context and continues in the background after the
	// waiter context or timeout is done. Ending the wait returns the current
	// snapshot and a nil error. The error is only for failing to submit the
	// task.
	ExecuteAndWait(ctx context.Context, runnable Runnable, timeout time.Duration, options ...Option) (Task, error)
	// RegisterGroup bounds concurrency for one exact task group. Non-positive
	// concurrency is a no-op. Duplicate names return an error. Unregistered
	// groups use the runner's global limit.
	RegisterGroup(group string, concurrency int) error
	GetTask(id string) (Task, error)
	GetTasks(group string) ([]Task, error)
	StopTask(id string) (Task, error)
	RemoveTask(id string) error
	Dispose() error
}
