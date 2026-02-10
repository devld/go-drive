package script

import (
	"context"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/robertkrimen/otto"
)

func newValue(vm *VM, v otto.Value) *Value {
	return &Value{vm, v, nil}
}

func newValues(vm *VM, vs []otto.Value) Values {
	return Values{vm, utils.ArrayMap(vs, func(t *otto.Value) *Value { return newValue(vm, *t) })}
}

type Values struct {
	vm *VM
	vs []*Value
}

func (vs Values) Get(index int) *Value {
	if index >= len(vs.vs) {
		return newValue(vs.vm, otto.UndefinedValue())
	}
	return vs.vs[index]
}

func (vs Values) Len() int {
	return len(vs.vs)
}

type Value struct {
	vm  *VM
	v   otto.Value
	obj *otto.Object
}

func (v *Value) IsNil() bool {
	return v.v.IsUndefined() || v.v.IsNull()
}

func (v *Value) IsNumber() bool {
	return v.v.IsNumber()
}

func (v *Value) String() string {
	if v.IsNil() {
		return ""
	}
	return v.v.String()
}

func (v *Value) Bool() bool {
	b, e := v.v.ToBoolean()
	if e != nil {
		v.vm.ThrowTypeError(e.Error())
	}
	return b
}

func (v *Value) Integer() int64 {
	i, e := v.v.ToInteger()
	if e != nil {
		v.vm.ThrowTypeError(e.Error())
	}
	return i
}

func (v *Value) Float() float64 {
	f, e := v.v.ToFloat()
	if e != nil {
		v.vm.ThrowTypeError(e.Error())
	}
	return f
}

func (v *Value) object() *otto.Object {
	if v.IsNil() {
		return nil
	}
	if v.obj == nil {
		v.obj = v.v.Object()
	}
	return v.obj
}

func (v *Value) Get(prop string) *Value {
	obj := v.object()
	if obj == nil {
		return nil
	}
	ov, e := obj.Get(prop)
	if e != nil {
		return nil
	}
	return newValue(v.vm, ov)
}

func (v *Value) Has(prop string) bool {
	return !v.Get(prop).IsNil()
}

func (v *Value) Keys() []string {
	obj := v.object()
	if obj == nil {
		return nil
	}
	return obj.Keys()
}

func (v *Value) SM() types.SM {
	obj := v.object()
	if obj == nil {
		return nil
	}
	r := make(types.SM)

	for _, k := range obj.Keys() {
		p, e := obj.Get(k)
		if e != nil {
			v.vm.ThrowTypeError(e.Error())
		}
		r[k] = p.String()
	}
	return r
}

func (v *Value) Array() []*Value {
	v.object()
	if v.obj == nil {
		return nil
	}
	len := int(v.Get("length").Integer())
	if len < 0 {
		v.vm.ThrowTypeError("not a valid array")
	}
	r := make([]*Value, len)
	for i := 0; i < len; i++ {
		r[i] = v.Get(strconv.FormatInt(int64(i), 10))
	}
	return r
}

func (v *Value) M() types.M {
	if v.IsNil() {
		return nil
	}
	obj := v.object()
	if obj == nil {
		return nil
	}
	r := make(types.M)

	for _, k := range obj.Keys() {
		p, e := obj.Get(k)
		if e != nil {
			v.vm.ThrowTypeError(e.Error())
		}
		r[k], _ = p.Export()
	}
	return r
}

func (v *Value) ParseInto(dest any) {
	defer func() {
		if er := recover(); er != nil {
			if ee, ok := er.(error); ok {
				v.vm.ThrowTypeError(ee.Error())
			} else {
				v.vm.ThrowTypeError(fmt.Sprintf("%v", er))
			}
		}
	}()
	parseValue(v, reflect.ValueOf(dest))
}

func (v *Value) Raw() any {
	r, _ := v.v.Export()
	return r
}

func (v *Value) InternalValue() any {
	return v.v
}

func (v *Value) Call(thisValue any, args ...any) *Value {
	thisV, e := v.vm.o.ToValue(thisValue)
	if e != nil {
		v.vm.ThrowTypeError(e.Error())
	}
	rv, e := v.v.Call(thisV, args...)
	if e != nil {
		v.vm.ThrowTypeError(e.Error())
	}
	return newValue(v.vm, rv)
}

func parseValue(ov *Value, v reflect.Value) {
	if ov == nil || !v.IsValid() {
		return
	}
	vt := v.Type()
	switch v.Kind() {
	case reflect.Ptr:
		if ov.IsNil() {
			return
		}
		if v.IsNil() {
			value := reflect.New(vt.Elem())
			v.Set(value)
		}
		parseValue(ov, v.Elem())
	case reflect.Map:
		keys := ov.Keys()
		if keys == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		valueType := vt.Elem()
		r := reflect.MakeMapWithSize(v.Type(), len(keys))
		for _, key := range keys {
			value := reflect.New(valueType)
			parseValue(ov.Get(key), value.Elem())
			r.SetMapIndex(reflect.ValueOf(key), value.Elem())
		}
		v.Set(r)
	case reflect.Slice:
		arr := ov.Array()
		if arr == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		n := len(arr)
		r := reflect.MakeSlice(vt, 0, n)

		for i := 0; i < n; i++ {
			value := reflect.New(vt.Elem())
			parseValue(arr[i], value.Elem())
			r = reflect.Append(r, value.Elem())
		}
		v.Set(r)
	case reflect.Array:
		arr := ov.Array()
		if arr == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		n := len(arr)
		var arrLen = vt.Len()
		r := reflect.New(reflect.ArrayOf(arrLen, vt.Elem()))
		for i := 0; i < n && i < arrLen; i++ {
			value := reflect.New(vt.Elem())
			parseValue(arr[i], value.Elem())
			r.Elem().Index(i).Set(value.Elem())
		}
		v.Set(r.Elem())
	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			fv := v.Field(i)
			if !fv.CanSet() {
				continue
			}
			fName := vt.Field(i).Name
			value := ov.Get(fName)
			parseValue(value, fv)
		}
	case reflect.Bool:
		v.Set(reflect.ValueOf(ov.Bool()))
	case reflect.Int:
		v.Set(reflect.ValueOf(int(ov.Integer())))
	case reflect.Uint:
		v.Set(reflect.ValueOf(uint(ov.Integer())))
	case reflect.Int8:
		v.Set(reflect.ValueOf(int8(ov.Integer())))
	case reflect.Uint8:
		v.Set(reflect.ValueOf(uint8(ov.Integer())))
	case reflect.Int16:
		v.Set(reflect.ValueOf(int16(ov.Integer())))
	case reflect.Uint16:
		v.Set(reflect.ValueOf(uint16(ov.Integer())))
	case reflect.Int32:
		v.Set(reflect.ValueOf(int32(ov.Integer())))
	case reflect.Uint32:
		v.Set(reflect.ValueOf(uint32(ov.Integer())))
	case reflect.Int64:
		v.Set(reflect.ValueOf(int64(ov.Integer())))
	case reflect.Uint64:
		v.Set(reflect.ValueOf(uint64(ov.Integer())))
	case reflect.Float32:
		v.Set(reflect.ValueOf(float32(ov.Float())))
	case reflect.Float64:
		v.Set(reflect.ValueOf(ov.Float()))
	case reflect.String:
		v.Set(reflect.ValueOf(ov.String()))
	}
}

func WrapVmCall(vm *VM, fn func(vm *VM, args Values) any) any {
	return func(c otto.FunctionCall) otto.Value {
		vm = vm.vms[c.Otto]
		if vm == nil {
			panic("detached vm")
		}

		defer func() {
			if e := recover(); e != nil {
				vm.ThrowError(e)
			}
		}()

		ret := fn(vm, newValues(vm, c.ArgumentList))
		ov, e := vm.o.ToValue(ret)
		if e != nil {
			vm.ThrowTypeError(e.Error())
		}
		return ov
	}
}

func wrapVmRun(ctx context.Context, vm *VM, fn func() (otto.Value, error)) (value *Value, e error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(error); ok {
				e = ee
				return
			}
			e = fmt.Errorf("%s", r)
		}

		e = mapError(e)
	}()

	finished := make(chan struct{}, 1)
	go func() {
		select {
		case <-ctx.Done():
			vm.o.Interrupt <- func() { panic(ctx.Err()) }
		case <-finished:
		}
	}()
	var ottoValue otto.Value
	defer func() { finished <- struct{}{} }()
	ottoValue, e = fn()
	value = newValue(vm, ottoValue)
	return
}

func mapError(e error) error {
	if e == nil {
		return nil
	}

	if oe, ok := e.(*otto.Error); ok {
		log.Println(oe.String())
	}

	if re, ok := e.(err.Error); ok {
		return re
	}
	data := e.Error()
	if !strings.HasPrefix(data, "Error: E:") {
		return e
	}

	parts := strings.SplitN(data, ":", 5)
	if len(parts) != 5 {
		return e
	}
	typ := parts[2]
	status := utils.ToInt(parts[3], 0)
	msg := parts[4]

	switch typ {
	case "BAD_REQUEST":
		return err.NewBadRequestError(msg)
	case "NOT_FOUND":
		if msg == "" {
			return err.NewNotFoundError()
		} else {
			return err.NewNotFoundMessageError(msg)
		}
	case "NOT_ALLOWED":
		if msg == "" {
			return err.NewNotAllowedError()
		} else {
			return err.NewNotAllowedMessageError(msg)
		}
	case "UNSUPPORTED":
		if msg == "" {
			return err.NewUnsupportedError()
		} else {
			return err.NewUnsupportedMessageError(msg)
		}
	case "REMOTE_API":
		return err.NewRemoteApiError(status, msg)
	}
	return e
}
