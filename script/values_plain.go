package script

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/dop251/goja"
)

// ToPlainJSValue materializes value as VM-owned JavaScript data. Maps become
// ordinary mutable objects, and slices and arrays become ordinary mutable
// arrays. Supported leaves are nil, booleans, strings, numbers, and Values
// belonging to this VM. Other Go values cause a panic.
//
// The conversion preserves cycles and shared map or slice identity within one
// call. Later mutations of the source Go containers and returned JavaScript
// containers do not affect each other.
func (vm *VM) ToPlainJSValue(value any) *Value {
	intern := make(map[internKey]*goja.Object)
	return newValue(vm, vm.toPlainJSValue(value, intern))
}

func (vm *VM) toPlainJSValue(value any, intern map[internKey]*goja.Object) goja.Value {
	if value == nil {
		return goja.Null()
	}
	switch value := value.(type) {
	case *Value:
		if value == nil || value.v == nil {
			return goja.Undefined()
		}
		if value.vm != vm {
			panic(vm.j.NewTypeError("JavaScript values cannot cross Runtime boundaries"))
		}
		return value.v
	case map[string]any:
		if value == nil {
			return goja.Null()
		}
		return vm.plainJSAnyObject(value, intern)
	case []any:
		if value == nil {
			return goja.Null()
		}
		return vm.plainJSAnyArray(value, intern)
	}

	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return goja.Null()
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Bool:
		return vm.j.ToValue(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vm.j.ToValue(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return vm.j.ToValue(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return vm.j.ToValue(rv.Float())
	case reflect.String:
		return vm.j.ToValue(rv.String())
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			panic(fmt.Sprintf("ToPlainJSValue: map key must be a string, got %s", rv.Type().Key()))
		}
		if rv.IsNil() {
			return goja.Null()
		}
		return vm.plainJSObject(rv, intern)
	case reflect.Slice:
		if rv.IsNil() {
			return goja.Null()
		}
		return vm.plainJSArray(rv, intern)
	case reflect.Array:
		return vm.plainJSArray(rv, intern)
	default:
		panic(fmt.Sprintf("ToPlainJSValue: unsupported value of type %s", rv.Type()))
	}
}

func (vm *VM) plainJSObject(rv reflect.Value, intern map[internKey]*goja.Object) *goja.Object {
	id, _ := goIdentity(rv)
	if obj := intern[id]; obj != nil {
		return obj
	}
	obj := vm.j.NewObject()
	intern[id] = obj

	iter := rv.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		value := vm.toPlainJSValue(iter.Value().Interface(), intern)
		if e := obj.DefineDataProperty(key, value, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); e != nil {
			panic(e)
		}
	}
	return obj
}

func (vm *VM) plainJSAnyObject(value map[string]any, intern map[internKey]*goja.Object) *goja.Object {
	id, _ := goIdentity(reflect.ValueOf(value))
	if obj := intern[id]; obj != nil {
		return obj
	}
	obj := vm.j.NewObject()
	intern[id] = obj
	for key, item := range value {
		converted := vm.toPlainJSValue(item, intern)
		if e := obj.DefineDataProperty(key, converted, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); e != nil {
			panic(e)
		}
	}
	return obj
}

func (vm *VM) plainJSArray(rv reflect.Value, intern map[internKey]*goja.Object) *goja.Object {
	if id, ok := goIdentity(rv); ok {
		if obj := intern[id]; obj != nil {
			return obj
		}
		obj := vm.j.NewArray()
		intern[id] = obj
		vm.populatePlainJSArray(obj, rv, intern)
		return obj
	}
	obj := vm.j.NewArray()
	vm.populatePlainJSArray(obj, rv, intern)
	return obj
}

func (vm *VM) plainJSAnyArray(value []any, intern map[internKey]*goja.Object) *goja.Object {
	id, _ := goIdentity(reflect.ValueOf(value))
	if obj := intern[id]; obj != nil {
		return obj
	}
	obj := vm.j.NewArray()
	intern[id] = obj
	for i, item := range value {
		converted := vm.toPlainJSValue(item, intern)
		if e := obj.DefineDataProperty(strconv.Itoa(i), converted, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); e != nil {
			panic(e)
		}
	}
	return obj
}

func (vm *VM) populatePlainJSArray(obj *goja.Object, rv reflect.Value, intern map[internKey]*goja.Object) {
	for i := 0; i < rv.Len(); i++ {
		value := vm.toPlainJSValue(rv.Index(i).Interface(), intern)
		if e := obj.DefineDataProperty(strconv.Itoa(i), value, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); e != nil {
			panic(e)
		}
	}
}
