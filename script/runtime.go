package script

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/dop251/goja"
)

//go:embed js/*.js
var jsDir embed.FS
var commonPrograms = loadCommonPrograms()

func loadCommonPrograms() []*Program {
	entries, e := jsDir.ReadDir("js")
	if e != nil {
		panic(e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	programs := make([]*Program, 0, len(entries))
	for _, entry := range entries {
		source, e := jsDir.ReadFile("js/" + entry.Name())
		if e != nil {
			panic(e)
		}
		programs = append(programs, MustCompile(entry.Name(), source))
	}
	return programs
}

// VM owns one Goja Runtime. Do not share a VM or its values with another VM
// or goroutine.
type VM struct {
	j *goja.Runtime

	ctors       map[string]*goja.Object
	classes     *ClassSet
	disposables map[any]struct{}
	disposed    bool
	reusable    bool
	runCtx      context.Context
	runDepth    int

	jsVars struct {
		objectFreeze goja.Callable
		arrayIsArray goja.Callable
		errorCtor    *goja.Object
	}
}

func NewVM() (*VM, error) {
	runtime := goja.New()
	runtime.SetFieldNameMapper(goFieldNameMapper{})
	vm := &VM{
		j:           runtime,
		classes:     BuiltinClasses,
		ctors:       make(map[string]*goja.Object, len(BuiltinClasses.classes)),
		disposables: make(map[any]struct{}),
		reusable:    true,
	}
	// Capture the builtins before any user code can replace them.
	vm.jsVars.errorCtor = runtime.Get("Error").ToObject(runtime)
	vm.jsVars.objectFreeze, _ = goja.AssertFunction(runtime.Get("Object").ToObject(runtime).Get("freeze"))
	vm.jsVars.arrayIsArray, _ = goja.AssertFunction(runtime.Get("Array").ToObject(runtime).Get("isArray"))
	if e := vm.AddClassSet(BuiltinClasses); e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	if e := vm.installHostGlobals(); e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	for _, program := range commonPrograms {
		if _, e := vm.runProgram(context.Background(), program); e != nil {
			_ = vm.Dispose()
			return nil, e
		}
	}
	return vm, nil
}

func (vm *VM) freeze(value goja.Value) error {
	if vm.jsVars.objectFreeze == nil {
		return errors.New("Object.freeze is unavailable")
	}
	_, e := vm.jsVars.objectFreeze(goja.Undefined(), value)
	return e
}

// freezeGlobal freezes the global object after pool initialization.
// The caller must run it inside Do.
func (vm *VM) freezeGlobal() error {
	return vm.freeze(vm.j.GlobalObject())
}

// DefineGlobal installs a non-writable, non-configurable binding. nil is
// JavaScript undefined.
func (vm *VM) DefineGlobal(name string, value any) error {
	var jsValue goja.Value
	if value == nil {
		jsValue = goja.Undefined()
	} else {
		jsValue = vm.toJSValue(value, name)
	}
	return vm.j.GlobalObject().DefineDataProperty(
		name,
		jsValue,
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
		goja.FLAG_TRUE,
	)
}

const bridgeGlobal = "__goDrive_bridge__"

// WithBridge installs a frozen `__goDrive_bridge__` for fn, then deletes it.
func (vm *VM) WithBridge(values map[string]any, fn func() error) (err error) {
	bridge := vm.j.NewObject()
	for name, value := range values {
		if e := bridge.Set(name, vm.toJSValue(value, name)); e != nil {
			return e
		}
	}
	if e := vm.j.Set(bridgeGlobal, bridge); e != nil {
		return e
	}
	if e := vm.freeze(bridge); e != nil {
		_ = vm.j.GlobalObject().Delete(bridgeGlobal)
		return e
	}
	defer func() {
		if e := vm.j.GlobalObject().Delete(bridgeGlobal); e != nil && err == nil {
			err = e
		}
	}()
	return fn()
}

// ExecutionContext is the context of the current Run/Call/Do.
func (vm *VM) ExecutionContext() context.Context {
	return vm.runCtx
}

func (vm *VM) Run(ctx context.Context, code any, filename string) (*Value, error) {
	if program, ok := code.(*Program); ok {
		return vm.runProgram(ctx, program)
	}
	program, e := Compile(filename, code)
	if e != nil {
		return nil, e
	}
	return vm.runProgram(ctx, program)
}

func (vm *VM) runProgram(ctx context.Context, program *Program) (*Value, error) {
	var result *Value
	e := vm.Do(ctx, func() error {
		jsValue, e := vm.j.RunProgram(program.compiled)
		if e != nil {
			return e
		}
		result, e = vm.rejectPromiseResult(newValue(vm, jsValue), nil)
		return e
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

// Call invokes a synchronous JavaScript function. A Promise return is an
// error: there is no event loop, and pooling the Runtime could retain
// request-owned Go values.
func (vm *VM) Call(ctx context.Context, fn string, args ...any) (*Value, error) {
	var result *Value
	e := vm.Do(ctx, func() error {
		var err error
		result, err = vm.invoke(newValue(vm, vm.j.Get(fn)), goja.Undefined(), args...)
		return err
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

// Do runs fn with context interrupt and JavaScript exception conversion.
// An already-canceled context prevents a new execution from starting.
// Nested Do/Call/Run join the current execution instead of starting another.
func (vm *VM) Do(ctx context.Context, fn func() error) error {
	if vm.runDepth > 0 {
		return captureScriptResult(vm, fn)
	}
	if e := ctx.Err(); e != nil {
		return e
	}

	vm.j.ClearInterrupt()
	vm.runCtx = ctx
	vm.runDepth = 1
	defer func() {
		vm.j.ClearInterrupt()
		vm.runCtx = nil
		vm.runDepth = 0
	}()
	if ctx.Done() != nil {
		// stop() can race a just-started Interrupt; wait before ClearInterrupt
		// so it cannot land on the next Call.
		var interruptWG sync.WaitGroup
		interruptWG.Add(1)
		stop := context.AfterFunc(ctx, func() {
			defer interruptWG.Done()
			vm.j.Interrupt(ctx.Err())
		})
		defer func() {
			if stop() {
				interruptWG.Done()
			}
			interruptWG.Wait()
		}()
	}
	return captureScriptResult(vm, fn)
}

// ErrFunctionUndefined is returned when Call targets a missing global.
var ErrFunctionUndefined = errors.New("JavaScript function is undefined")

func (vm *VM) callValue(ctx context.Context, function *Value, this goja.Value, args ...any) (*Value, error) {
	var result *Value
	e := vm.Do(ctx, func() error {
		var err error
		result, err = vm.invoke(function, this, args...)
		return err
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

func (vm *VM) invoke(function *Value, this goja.Value, args ...any) (*Value, error) {
	if function == nil || function.IsNil() {
		return nil, ErrFunctionUndefined
	}
	callable, ok := goja.AssertFunction(function.v)
	if !ok {
		return nil, fmt.Errorf("JavaScript value is not callable")
	}
	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = vm.ToJSValue(arg).v
	}
	jsValue, e := callable(this, jsArgs...)
	if e != nil {
		return nil, e
	}
	return vm.rejectPromiseResult(newValue(vm, jsValue), nil)
}

func (vm *VM) rejectPromiseResult(result *Value, e error) (*Value, error) {
	if e != nil || result == nil {
		return result, e
	}
	if result.isPromise() {
		vm.reusable = false
		return nil, errors.New("Promise return values are not supported")
	}
	return result, nil
}

func (vm *VM) GetValue(prop string) (*Value, error) {
	return newValue(vm, vm.j.Get(prop)), nil
}

// ThrowError panics a JavaScript exception. Only call it from host functions
// on a JavaScript stack; Run/Call convert it to an error.
func (vm *VM) ThrowError(e any) {
	if value, ok := e.(goja.Value); ok {
		panic(value)
	}
	if ee, ok := e.(error); ok {
		panic(vm.jsErrorFromGo(ee))
	}
	panic(vm.j.NewGoError(fmt.Errorf("%v", e)))
}

func (vm *VM) ThrowTypeError(message string) {
	panic(vm.j.NewTypeError("%s", message))
}

// throwTypeErrorRequired panics when required is non-empty; empty means optional.
func (vm *VM) throwTypeErrorRequired(required string) {
	if required != "" {
		vm.ThrowTypeError(required)
	}
}

func (vm *VM) PutDisposable(o any) {
	rv := reflect.ValueOf(o)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		panic("disposable must be a non-nil pointer")
	}
	vm.disposables[o] = struct{}{}
}

func (vm *VM) RemoveDisposable(o any) {
	delete(vm.disposables, o)
}

func (vm *VM) disposeDisposables() (disposeErr error) {
	for o := range vm.disposables {
		e := func() (e error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recoveredErr, ok := recovered.(error); ok {
						e = recoveredErr
					} else {
						e = fmt.Errorf("resource cleanup panic: %v", recovered)
					}
				}
			}()
			if disposable, ok := o.(ObjectDisposable); ok {
				disposable.Dispose()
				return nil
			}
			if closable, ok := o.(ObjectClosable); ok {
				closable.Close()
			}
			return nil
		}()
		disposeErr = errors.Join(disposeErr, e)
	}
	vm.disposables = make(map[any]struct{})
	if disposeErr != nil {
		vm.reusable = false
	}
	return disposeErr
}

func (vm *VM) Reusable() bool { return vm != nil && vm.reusable && !vm.disposed }

func (vm *VM) Dispose() error {
	if vm == nil || vm.disposed {
		return nil
	}
	disposeErr := vm.disposeDisposables()
	vm.disposed = true
	vm.reusable = false
	vm.j = nil
	vm.ctors = nil
	vm.jsVars.errorCtor = nil
	vm.jsVars.objectFreeze = nil
	vm.jsVars.arrayIsArray = nil
	return disposeErr
}

type ObjectDisposable interface {
	Dispose()
}

type ObjectClosable interface {
	Close()
}
