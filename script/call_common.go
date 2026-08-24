package script

import (
	"context"
	"go-drive/common/logging"
	"sync"
	"time"
)

func vm_consoleWrite(vm *VM, args Values) any {
	levelName := args.Get(0).String()
	message := formatConsoleArgs(args, 1)
	logger := logging.For("script")
	switch levelName {
	case "debug":
		logger.Debugf("%s", message)
	case "warn", "warning":
		logger.Warnf("%s", message)
	case "error":
		logger.Errorf("%s", message)
	default:
		logger.Infof("%s", message)
	}
	return nil
}

// vm_newContext: () Context
func vm_newContext(vm *VM, args Values) any {
	return NewContext(vm, context.Background())
}

// vm_newContextWithTimeout: (parent Context, timeout time.Duration) contextWithTimeout
func vm_newContextWithTimeout(vm *VM, args Values) any {
	parent := GetContext(args.Get(0).Raw())
	if parent == nil {
		vm.ThrowTypeError("newContextWithTimeout requires a Context")
	}
	timeout := RequireDuration(args.Get(1), "newContextWithTimeout")
	ctx, cancel := context.WithTimeout(parent, timeout)
	cwt := &contextWithTimeout{NewContext(vm, ctx), cancel}
	vm.PutDisposable(cwt)
	return cwt
}

// vm_newTaskCtx: (ctx Context, onUpdate func(int64, int64)) TaskCtx
func vm_newTaskCtx(vm *VM, args Values) any {
	ctx := GetContext(args.Get(0).Raw())
	onUpdate := args.Get(1)
	if onUpdate.IsNil() {
		onUpdate = nil
	}
	return NewTaskCtx(vm, &scriptTaskCtx{ctx, onUpdate, 0, 0, sync.Mutex{}})
}

// vm_sleep: (t time.Duration)
func vm_sleep(vm *VM, args Values) any {
	time.Sleep(RequireDuration(args.Get(0), "sleep"))
	return nil
}

func vm_parseDuration(vm *VM, args Values) any {
	v := args.Get(0)
	if v.IsNil() {
		return time.Duration(0)
	}
	return RequireDuration(v, "parseDuration")
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

func (cwt *contextWithTimeout) Dispose() {
	if cwt == nil {
		return
	}
	cwt.Context.vm.RemoveDisposable(cwt)
	cwt.Cancel()
}

func (cwt contextWithTimeout) ConsoleString() string {
	return formatGoInspect("ContextWithTimeout", nil, true)
}

// vm_newLocker: () *locker
func vm_newLocker(vm *VM, args Values) any {
	return &locker{&sync.Mutex{}}
}

type locker struct {
	mu *sync.Mutex
}

func (l *locker) Lock() {
	l.mu.Lock()
}

func (l *locker) Unlock() {
	l.mu.Unlock()
}

func (l *locker) ConsoleString() string {
	return formatGoInspect("Locker", nil, true)
}
