package script

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/dop251/goja"
)

// ToJSValue converts a Go value into a JavaScript value for this Runtime.
// Maps, slices, and structs become a read-only view: writes do not reach Go.
// Nested values are interned by identity within one call so cycles and
// repeated reads share objects; array indexes cache wrapped elements so value
// structs keep a stable identity. Host classes wrap matching native values.
// time.Time becomes a JavaScript Date. A Go function must be NativeFunction
// or ToJSValue panics. Exported struct methods must have the NativeFunction
// signature; other Go methods panic so goja reflection is never used.
func (vm *VM) ToJSValue(value any) *Value {
	return newValue(vm, vm.toJSValue(value))
}

func (vm *VM) toJSValue(value any) goja.Value {
	if value == nil {
		return goja.Null()
	}
	if fn, ok := value.(NativeFunction); ok {
		return vm.nativeFunction(fn)
	}
	switch value := value.(type) {
	case *Value:
		if value == nil {
			return goja.Undefined()
		}
		if value.vm != vm {
			panic(vm.j.NewTypeError("JavaScript values cannot cross Runtime boundaries"))
		}
		return value.v
	}
	if d, ok := value.(time.Duration); ok {
		return vm.j.ToValue(int64(d))
	}
	if t, ok := value.(time.Time); ok {
		return vm.wrapGoTime(t)
	}
	return vm.wrapGoData(value, nil)
}

func (vm *VM) wrapGoTime(t time.Time) goja.Value {
	date, e := vm.j.New(vm.j.Get("Date"), vm.j.ToValue(t.UnixMilli()))
	if e != nil {
		panic(e)
	}
	return date
}

func panicNonNativeFunction(value any) {
	panic(fmt.Sprintf("ToJSValue: function must be NativeFunction, got %T", value))
}

func (vm *VM) nativeFunction(fn NativeFunction) goja.Value {
	value := vm.j.ToValue(func(call goja.FunctionCall) goja.Value {
		result := fn(vm, newValues(vm, call.Arguments))
		if result == nil {
			return goja.Undefined()
		}
		return vm.ToJSValue(result).v
	})
	_ = vm.freeze(value)
	return value
}

func sortedMapKeys[V any](value map[string]V) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// FromJSValue deep-copies a JSON-compatible JavaScript value into Go.
// Functions and other non-JSON values return an error.
func (vm *VM) FromJSValue(value *Value) (any, error) {
	encoded, e := vm.encodeJSONValue(value)
	if e != nil {
		return nil, e
	}
	var cloned any
	if e := json.Unmarshal(encoded, &cloned); e != nil {
		return nil, e
	}
	return cloned, nil
}

func (vm *VM) encodeJSONValue(value *Value) ([]byte, error) {
	if value == nil || value.IsUndefined() {
		return nil, errors.New("value is not JSON serializable")
	}
	if goja.IsNull(value.v) {
		return []byte("null"), nil
	}
	obj, ok := value.v.(*goja.Object)
	if !ok {
		return json.Marshal(unwrapExported(value.v.Export()))
	}
	// Reject functions: Object.MarshalJSON would encode them as null.
	switch obj.ClassName() {
	case "Function", "AsyncFunction", "GeneratorFunction":
		return nil, errors.New("value is not JSON serializable")
	}
	return obj.MarshalJSON()
}
