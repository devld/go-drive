package script

import (
	"context"
	"strconv"

	"go-drive/common/task"
	"go-drive/common/types"
)

func NewContext(vm *VM, c context.Context) Context {
	return Context{vm, c}
}

func NewTaskCtx(vm *VM, c types.TaskCtx) TaskCtx {
	return TaskCtx{NewContext(vm, c), c}
}

func scriptContext(v any) (Context, bool) {
	switch v := v.(type) {
	case Context:
		return v, true
	case *Context:
		if v != nil {
			return *v, true
		}
	case TaskCtx:
		return v.Context, true
	case *TaskCtx:
		if v != nil {
			return v.Context, true
		}
	case contextWithTimeout:
		return v.Context, true
	case *contextWithTimeout:
		if v != nil {
			return v.Context, true
		}
	}
	return Context{}, false
}

func GetContext(v any) context.Context {
	if c, ok := scriptContext(v); ok {
		return c.v
	}
	if c, ok := v.(context.Context); ok {
		return c
	}
	return nil
}

func GetVM(v any) *VM {
	if c, ok := scriptContext(v); ok {
		return c.vm
	}
	return nil
}

func GetTaskCtx(v any) types.TaskCtx {
	switch v := v.(type) {
	case TaskCtx:
		return v.v
	case *TaskCtx:
		if v != nil {
			return v.v
		}
	}
	c := GetContext(v)
	if c == nil {
		return nil
	}
	if tc, ok := c.(types.TaskCtx); ok {
		return tc
	}
	return task.NewContextWrapper(c)
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

func (c Context) ConsoleString() string {
	if c.v == nil {
		return "Context {}"
	}
	if e := c.v.Err(); e != nil {
		return formatGoInspect("Context", []string{"Err: " + strconv.Quote(e.Error())}, true)
	}
	return formatGoInspect("Context", nil, true)
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

func (t TaskCtx) ConsoleString() string {
	return formatGoInspect("TaskCtx", nil, true)
}
