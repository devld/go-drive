package script

import (
	"context"
	"sync"
	"time"
)

// vm_newContext: () Context
func vm_newContext(vm *VM, args Values) interface{} {
	return NewContext(vm, context.Background())
}

// vm_newContextWithTimeout: (parent Context, timeout time.Duration) contextWithTimeout
func vm_newContextWithTimeout(vm *VM, args Values) interface{} {
	parent := GetContext(args.Get(0).Raw())
	timeout := time.Duration(args.Get(1).Integer())
	ctx, cancel := context.WithTimeout(GetContext(parent), timeout)
	return contextWithTimeout{NewContext(vm, ctx), cancel}
}

func vm_newTaskCtx(vm *VM, args Values) interface{} {
	ctx := GetContext(args.Get(0).Raw())
	onUpdate := args.Get(1)
	if onUpdate.IsNil() {
		onUpdate = nil
	}
	return NewTaskCtx(vm, &scriptTaskCtx{ctx, onUpdate, 0, 0, sync.Mutex{}})
}

type scriptTaskCtx struct {
	context.Context
	onUpdate *Value

	loaded int64
	total  int64
	mu     sync.Mutex
}

func (s *scriptTaskCtx) Progress(loaded int64, abs bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if abs {
		s.loaded = loaded
	} else {
		s.loaded += loaded
	}
	if s.onUpdate != nil {
		s.onUpdate.Call(s, s.loaded, s.total)
	}
}

func (s *scriptTaskCtx) Total(total int64, abs bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if abs {
		s.total = total
	} else {
		s.total += total
	}
	if s.onUpdate != nil {
		s.onUpdate.Call(s, s.loaded, s.total)
	}
}

type contextWithTimeout struct {
	Context Context
	Cancel  func()
}
