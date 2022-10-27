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

func GetContext(v interface{}) context.Context {
	switch v := v.(type) {
	case Context:
		return v.v
	case TaskCtx:
		return v.v
	}
	return nil
}

func GetTaskCtx(v interface{}) types.TaskCtx {
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
