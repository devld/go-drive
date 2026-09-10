package script

import (
	"fmt"
	"math"
	"sync"

	"go-drive/common/task"
	"go-drive/common/types"
)

const maxJSSafeInteger = 1<<53 - 1

// ProgressReporter is an operation-scoped, attenuable capability for updating
// task progress. Derived reporters share the same lifetime and can only remove
// permissions; they can never regain a permission removed by their parent.
type ProgressReporter struct {
	state *struct {
		mu  sync.RWMutex
		ctx types.TaskCtx
	}
	allowLoaded bool
	allowTotal  bool
}

func NewProgressReporter(ctx types.TaskCtx, allowLoaded, allowTotal bool) *ProgressReporter {
	return &ProgressReporter{
		state: &struct {
			mu  sync.RWMutex
			ctx types.TaskCtx
		}{ctx: ctx},
		allowLoaded: allowLoaded,
		allowTotal:  allowTotal,
	}
}

func (p *ProgressReporter) Dispose() { p.close() }

// close invalidates this reporter and every capability derived from it, and
// releases the underlying task context if JavaScript retained a wrapper.
func (p *ProgressReporter) close() {
	if p == nil || p.state == nil {
		return
	}
	p.state.mu.Lock()
	p.state.ctx = nil
	p.state.mu.Unlock()
}

func (p *ProgressReporter) activeContext() types.TaskCtx {
	if p == nil || p.state == nil {
		return nil
	}
	p.state.mu.RLock()
	defer p.state.mu.RUnlock()
	return p.state.ctx
}

func (p *ProgressReporter) derive(allowLoaded, allowTotal bool) (*ProgressReporter, error) {
	if p == nil || p.activeContext() == nil {
		return nil, fmt.Errorf("ProgressReporter is no longer active")
	}
	if allowLoaded && !p.allowLoaded {
		return nil, fmt.Errorf("ProgressReporter cannot grant loaded permission")
	}
	if allowTotal && !p.allowTotal {
		return nil, fmt.Errorf("ProgressReporter cannot grant total permission")
	}
	return &ProgressReporter{state: p.state, allowLoaded: allowLoaded, allowTotal: allowTotal}, nil
}

func (p *ProgressReporter) addLoaded(delta int64) error {
	if p == nil || !p.allowLoaded {
		return fmt.Errorf("ProgressReporter cannot update loaded")
	}
	ctx := p.activeContext()
	if ctx == nil {
		return fmt.Errorf("ProgressReporter is no longer active")
	}
	ctx.Progress(delta, false)
	return nil
}

func (p *ProgressReporter) addTotal(delta int64) error {
	if p == nil || !p.allowTotal {
		return fmt.Errorf("ProgressReporter cannot update total")
	}
	ctx := p.activeContext()
	if ctx == nil {
		return fmt.Errorf("ProgressReporter is no longer active")
	}
	ctx.Total(delta, false)
	return nil
}

// addLoadedFromIO deliberately ignores an expired reporter: progress
// bookkeeping must not turn a successful read into an I/O failure.
func (p *ProgressReporter) addLoadedFromIO(delta int64) {
	if delta <= 0 || p == nil || !p.allowLoaded {
		return
	}
	if ctx := p.activeContext(); ctx != nil {
		ctx.Progress(delta, false)
	}
}

type jsObjProgressReporter struct {
	ClassHost
	p *ProgressReporter
}

var jsClassProgressReporter = JSClass{
	Name:      "ProgressReporter",
	Handle:    jsObjProgressReporter{},
	Construct: ConstructorHostOnly("ProgressReporter"),
	Accepts:   AcceptsNonNil[*ProgressReporter](),
	Wrap: func(vm *VM, v any) any {
		p := v.(*ProgressReporter)
		vm.PutDisposable(p)
		return jsObjProgressReporter{ClassHost: NewClassHost(vm), p: p}
	},
	Methods: map[string]ClassMethod{
		"addLoaded": func(vm *VM, this *Value, args Values) any {
			p := This[jsObjProgressReporter](vm, this, "ProgressReporter.addLoaded").p
			if e := p.addLoaded(progressDelta(vm, args, "ProgressReporter.addLoaded")); e != nil {
				vm.ThrowError(e)
			}
			return nil
		},
		"addTotal": func(vm *VM, this *Value, args Values) any {
			p := This[jsObjProgressReporter](vm, this, "ProgressReporter.addTotal").p
			if e := p.addTotal(progressDelta(vm, args, "ProgressReporter.addTotal")); e != nil {
				vm.ThrowError(e)
			}
			return nil
		},
		"derive": func(vm *VM, this *Value, args Values) any {
			p := This[jsObjProgressReporter](vm, this, "ProgressReporter.derive").p
			options := args.Get(0)
			if !options.IsObject() {
				vm.ThrowTypeError("ProgressReporter.derive requires an options object")
			}
			loaded := progressCapability(vm, options, "loaded")
			total := progressCapability(vm, options, "total")
			derived, e := p.derive(loaded, total)
			if e != nil {
				vm.ThrowError(e)
			}
			return derived
		},
	},
}

func progressCapability(vm *VM, options *Value, name string) bool {
	v := options.Get(name)
	if v == nil || v.IsUndefined() {
		return false
	}
	b, ok := v.Raw().(bool)
	if !ok {
		vm.ThrowTypeError("ProgressReporter.derive " + name + " must be boolean")
	}
	return b
}

func progressDelta(vm *VM, args Values, name string) int64 {
	if args.Len() != 1 || !args.Get(0).IsNumber() {
		vm.ThrowTypeError(name + " requires exactly one numeric delta")
	}
	n := args.Get(0).Float()
	if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > maxJSSafeInteger || math.Trunc(n) != n {
		vm.ThrowTypeError(name + " delta must be a non-negative safe integer")
	}
	return int64(n)
}

func GetProgressReporter(vm *VM, v any, required string) *ProgressReporter {
	if p, ok := HostAs[jsObjProgressReporter](v); ok && p.p != nil {
		if p.p.activeContext() == nil {
			vm.ThrowError(fmt.Errorf("ProgressReporter is no longer active"))
		}
		return p.p
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

// GetTaskCtx converts a JavaScript ProgressReporter into a capability-filtered
// TaskCtx. An optional missing reporter retains cancellation without exposing
// task progress.
func GetTaskCtx(vm *VM, v any, required string) types.TaskCtx {
	p := GetProgressReporter(vm, v, required)
	if p == nil {
		return task.NewContextWrapper(vm.ExecutionContext())
	}
	ctx := p.activeContext()
	if ctx == nil {
		vm.ThrowError(fmt.Errorf("ProgressReporter is no longer active"))
	}
	return task.NewCtxWrapper(ctx, p.allowLoaded, p.allowTotal)
}

func (p jsObjProgressReporter) ConsoleString() string {
	return formatGoInspect("ProgressReporter", nil, true)
}
