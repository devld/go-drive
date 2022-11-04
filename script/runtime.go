package script

import (
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
	vmsMu *sync.Mutex
}

func NewVM() (*VM, error) {
	ovm := otto.New()
	vm := &VM{
		o:     ovm,
		vms:   make(map[*otto.Otto]*VM),
		vmsMu: &sync.Mutex{},
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
		if _, e := v.Run(script); e != nil {
			return e
		}
	}

	return nil
}

func (v *VM) Set(name string, value interface{}) {
	v.o.Set(name, value)
}

func (v *VM) Fork() *VM {
	v.vmsMu.Lock()
	defer v.vmsMu.Unlock()
	newVM := &VM{
		o:     v.o.Copy(),
		vms:   v.vms,
		vmsMu: v.vmsMu,
	}
	v.vms[newVM.o] = newVM
	return newVM
}

// Run runs code with this VM. Run can NOT be executed concurrency
func (v *VM) Run(code interface{}) (*Value, error) {
	return wrapVmRun(v, func() (otto.Value, error) {
		return v.o.Run(code)
	})
}

// Call calls function with this VM. Call can NOT be executed concurrency
func (v *VM) Call(fn string, args ...interface{}) (value *Value, e error) {
	return wrapVmRun(v, func() (otto.Value, error) {
		a, b := v.o.Call(fn, nil, args...)
		return a, b
	})
}

func (v *VM) GetValue(prop string) (value *Value, e error) {
	vv, e := v.o.Get(prop)
	return newValue(v, vv), e
}

// ForkRun runs code with a cloned VM. ForkRun can be executed concurrency
func (v *VM) ForkRun(code interface{}) (*Value, error) {
	rt := v.Fork()
	defer func() {
		_ = rt.Dispose()
	}()
	return rt.Run(code)
}

// ForkCall calls function with a cloned VM. ForkCall can be executed concurrency
func (v *VM) ForkCall(fn string, args ...interface{}) (*Value, error) {
	rt := v.Fork()
	defer func() {
		_ = rt.Dispose()
	}()
	return rt.Call(fn, args...)
}

func (v *VM) ThrowError(e interface{}) {
	if oe, ok := e.(otto.Value); ok {
		panic(oe)
	}
	if re, ok := e.(err.Error); ok {
		rev, e := v.o.ToValue(re)
		if e != nil {
			panic(e)
		}
		panic(rev)
	}
	if ee, ok := e.(error); ok {
		panic(v.o.MakeCustomError("Error", ee.Error()))
	}
	panic(v.o.MakeCustomError("Error", fmt.Sprintf("%v", e)))
}

func (v *VM) ThrowTypeError(message string) {
	panic(v.o.MakeTypeError(message))
}

func (v *VM) Dispose() error {
	v.vmsMu.Lock()
	defer v.vmsMu.Unlock()
	delete(v.vms, v.o)
	// nothing
	return nil
}
