package task

import (
	"context"
	"go-drive/common/types"
)

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
