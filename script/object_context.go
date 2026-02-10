package script

import (
	"context"
	"go-drive/common/types"
)

func NewContext(vm *VM, c context.Context) Context {
	return Context{vm, c}
}

func NewTaskCtx(vm *VM, c types.TaskCtx) TaskCtx {
	return TaskCtx{NewContext(vm, c), c}
}

func GetContext(v any) context.Context {
	switch v := v.(type) {
	case Context:
		return v.v
	case TaskCtx:
		return v.v
	}
	return nil
}

func GetTaskCtx(v any) types.TaskCtx {
	switch v := v.(type) {
	case TaskCtx:
		return v.v
	}
	return nil
}

type Context struct {
	vm *VM
	v  context.Context
}

func (c Context) Err() {
	e := c.v.Err()
	if e != nil {
		c.vm.ThrowError(e)
	}
}

type TaskCtx struct {
	Context
	v types.TaskCtx
}

func (t TaskCtx) Progress(loaded int64, abs bool) {
	t.v.Progress(loaded, abs)
}

func (t TaskCtx) Total(total int64, abs bool) {
	t.v.Total(total, abs)
}
