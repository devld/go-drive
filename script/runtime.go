package script

import (
	"context"
	"fmt"
	err "go-drive/common/errors"
	"sort"
	"strings"
	"sync"

	"embed"

	"github.com/robertkrimen/otto"
)

//go:embed js/*.js
var jsDir embed.FS

type VM struct {
	o     *otto.Otto
	vms   map[*otto.Otto]*VM
	vmsMu *sync.RWMutex

	disposables map[any]struct{}
}

func NewVM() (*VM, error) {
	ovm := otto.New()
	ovm.Interrupt = make(chan func(), 1)
	vm := &VM{
		o:           ovm,
		vms:         make(map[*otto.Otto]*VM),
		vmsMu:       &sync.RWMutex{},
		disposables: make(map[any]struct{}),
	}

	if e := vm.init(); e != nil {
		return nil, e
	}

	return vm, nil
}

func (v *VM) init() error {
	initVarsForVm(v)

	// load scripts
	entries, e := jsDir.ReadDir("js")
	if e != nil {
		return e
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.Compare(entries[i].Name(), entries[j].Name()) < 0
	})
	for _, entry := range entries {
		script, e := jsDir.ReadFile("js" + "/" + entry.Name())
		if e != nil {
			return e
		}
		if _, e := v.Run(context.Background(), script); e != nil {
			return e
		}
	}

	return nil
}

func (v *VM) Set(name string, value any) {
	v.o.Set(name, value)
}

func (v *VM) Fork() *VM {
	v.vmsMu.Lock()
	defer v.vmsMu.Unlock()
	newVM := &VM{
		o:           v.o.Copy(),
		vms:         v.vms,
		vmsMu:       v.vmsMu,
		disposables: make(map[any]struct{}),
	}
	newVM.o.Interrupt = make(chan func(), 1)
	v.vms[newVM.o] = newVM
	return newVM
}

// resolveVM returns the *VM that owns the given otto runtime. Forks share a
// single vms map, so reads must be guarded against concurrent Fork/Dispose
// writes happening on other VMs of the same family.
func (v *VM) resolveVM(o *otto.Otto) *VM {
	v.vmsMu.RLock()
	defer v.vmsMu.RUnlock()
	return v.vms[o]
}

// Run runs code with this VM. Run can NOT be executed concurrency
func (v *VM) Run(ctx context.Context, code any) (*Value, error) {
	return wrapVmRun(ctx, v, func() (otto.Value, error) {
		return v.o.Run(code)
	})
}

// Call calls function with this VM. Call can NOT be executed concurrency
func (v *VM) Call(ctx context.Context, fn string, args ...any) (value *Value, e error) {
	return wrapVmRun(ctx, v, func() (otto.Value, error) {
		a, b := v.o.Call(fn, nil, args...)
		return a, b
	})
}

func (v *VM) GetValue(prop string) (value *Value, e error) {
	vv, e := v.o.Get(prop)
	return newValue(v, vv), e
}

func (v *VM) EncodeJSONValue(value *Value) ([]byte, error) {
	encoded, e := v.o.Call("JSON.stringify", nil, value.v)
	if e != nil {
		return nil, e
	}
	if encoded.IsUndefined() {
		return nil, fmt.Errorf("value is not JSON serializable")
	}
	return []byte(encoded.String()), nil
}

func (v *VM) DecodeJSONValue(encoded []byte) (any, error) {
	return v.o.Call("JSON.parse", nil, string(encoded))
}

func (v *VM) ThrowError(e any) {
	if oe, ok := e.(otto.Value); ok {
		panic(oe)
	}
	if re, ok := e.(err.Error); ok {
		panic(v.o.MakeCustomError("Error", fmt.Sprintf("E:%s:%d:%s", re.Name(), re.Code(), re.Error())))
	}
	if ee, ok := e.(error); ok {
		panic(v.o.MakeCustomError("Error", ee.Error()))
	}
	panic(v.o.MakeCustomError("Error", fmt.Sprintf("%v", e)))
}

func ThrowDetachedError(e any) {
	message := errorMessage(e)
	value, conversionErr := otto.ToValue("Error: " + message)
	if conversionErr != nil {
		panic(conversionErr)
	}
	panic(value)
}

func throwForVM(vm *VM, e any) {
	if vm == nil {
		ThrowDetachedError(e)
	}
	vm.ThrowError(e)
}

func errorMessage(e any) string {
	if re, ok := e.(err.Error); ok {
		return fmt.Sprintf("E:%s:%d:%s", re.Name(), re.Code(), re.Error())
	}
	if ee, ok := e.(error); ok {
		return ee.Error()
	}
	return fmt.Sprintf("%v", e)
}

func (v *VM) ThrowTypeError(message string) {
	panic(v.o.MakeTypeError(message))
}

func (v *VM) PutDisposable(o any) {
	v.disposables[o] = struct{}{}
}

func (v *VM) RemoveDisposable(o any) {
	delete(v.disposables, o)
}

func (v *VM) DisposeDisposables() {
	for o := range v.disposables {
		if d, ok := o.(ObjectDisposable); ok {
			d.Dispose()
		}
		if c, ok := o.(ObjectClosable); ok {
			c.Close()
		}
	}
	v.disposables = make(map[any]struct{})
}

func (v *VM) Dispose() error {
	v.DisposeDisposables()

	v.vmsMu.Lock()
	defer v.vmsMu.Unlock()
	delete(v.vms, v.o)
	// nothing
	return nil
}

type ObjectDisposable interface {
	Dispose()
}

type ObjectClosable interface {
	Close()
}
