package task

import (
	"context"
	"go-drive/common/types"
	"sync"
)

func DummyContext() types.TaskCtx {
	return dummyCtx
}

// NewContextWrapper wraps Context as a TaskCtx
func NewContextWrapper(ctx context.Context) *TaskContextWrapper {
	return &TaskContextWrapper{ctx, 0, 0, false, nil}
}

// NewTaskContext wraps Context as a mutable TaskCtx
func NewTaskContext(ctx context.Context) *TaskContextWrapper {
	return &TaskContextWrapper{ctx, 0, 0, true, &sync.RWMutex{}}
}

var dummyCtx = &TaskContextWrapper{context.Background(), 0, 0, false, nil}

type TaskContextWrapper struct {
	context.Context
	loaded  int64
	total   int64
	mutable bool
	mu      *sync.RWMutex
}

func (d *TaskContextWrapper) Progress(loaded int64, abs bool) {
	if !d.mutable {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if abs {
		d.loaded = loaded
	} else {
		d.loaded += loaded
	}
}

func (d *TaskContextWrapper) Total(total int64, abs bool) {
	if !d.mutable {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if abs {
		d.total = total
	} else {
		d.total += total
	}
}

func (d *TaskContextWrapper) GetProgress() int64 {
	if !d.mutable {
		return 0
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.loaded
}

func (d *TaskContextWrapper) GetTotal() int64 {
	if !d.mutable {
		return 0
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.total
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
