package script

import (
	"context"
	"time"
)

// vm_newContext: () Context
func vm_newContext(vm *VM, args []*Value) interface{} {
	return NewContext(vm, context.Background())
}

// vm_newContextWithTimeout: (parent Context, timeout time.Duration) contextWithTimeout
func vm_newContextWithTimeout(vm *VM, args []*Value) interface{} {
	parent := GetContext(args[0].Raw())
	timeout := time.Duration(args[1].Integer())
	ctx, cancel := context.WithTimeout(GetContext(parent), timeout)
	return contextWithTimeout{NewContext(vm, ctx), cancel}
}

type contextWithTimeout struct {
	Context Context
	Cancel  func()
}
