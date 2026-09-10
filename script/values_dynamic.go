package script

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dop251/goja"
)

var (
	nativeFunctionType = reflect.TypeFor[NativeFunction]()
	vmPtrType          = reflect.TypeFor[*VM]()
	valuesType         = reflect.TypeFor[Values]()
	anyType            = reflect.TypeFor[any]()
)

type internKey struct {
	ptr uintptr
	len int
	typ reflect.Type
}

// viewIntern interns objects by Go identity for one ToJSValue tree.
type viewIntern struct {
	objs map[internKey]*goja.Object
}

func (s *viewIntern) get(id internKey) *goja.Object {
	if s == nil || s.objs == nil {
		return nil
	}
	return s.objs[id]
}

func (s *viewIntern) put(rv reflect.Value, obj *goja.Object) {
	if s == nil || obj == nil {
		return
	}
	id, ok := goIdentity(rv)
	if !ok {
		return
	}
	if s.objs == nil {
		s.objs = make(map[internKey]*goja.Object)
	}
	s.objs[id] = obj
}

type readonlyObject struct {
	vm      *VM
	intern  *viewIntern
	name    string
	v       any
	rv      reflect.Value
	methods map[string]goja.Value
	fields  map[string]goja.Value
}

type readonlyArray struct {
	vm     *VM
	intern *viewIntern
	name   string
	v      any
	rv     reflect.Value
	elems  []goja.Value
}

func (vm *VM) wrapGoDataNamed(value any, intern *viewIntern, name string) goja.Value {
	if value == nil {
		return goja.Null()
	}
	if jsValue, ok := value.(*Value); ok {
		if jsValue == nil || jsValue.v == nil {
			return goja.Undefined()
		}
		if jsValue.vm != vm {
			panic(vm.j.NewTypeError("JavaScript values cannot cross Runtime boundaries"))
		}
		return jsValue.v
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return goja.Null()
		}
		rv = rv.Elem()
		value = rv.Interface()
	}
	if !rv.IsValid() {
		return goja.Undefined()
	}
	// Check pointers before host matching: a typed nil may implement Reader,
	// Drive, or another host capability without being a usable instance.
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return goja.Null()
	}
	if gv, ok := value.(goja.Value); ok {
		return gv
	}
	if intern == nil {
		intern = &viewIntern{}
	}
	if id, ok := goIdentity(rv); ok {
		if cached := intern.get(id); cached != nil {
			return cached
		}
	}
	if rv.Kind() == reflect.Func {
		fn, ok := value.(NativeFunction)
		if !ok && rv.Type().ConvertibleTo(nativeFunctionType) {
			fn, ok = rv.Convert(nativeFunctionType).Interface().(NativeFunction), true
		}
		if !ok {
			panicNonNativeFunction(value)
		}
		return internValue(intern, rv, vm.nativeFunction(name, fn))
	}
	if wrapped := vm.wrapHostClass(value); wrapped != nil {
		return internValue(intern, rv, wrapped)
	}
	switch t := value.(type) {
	case time.Time:
		return vm.wrapGoTime(t)
	case *time.Time:
		if t == nil {
			return goja.Null()
		}
		return internValue(intern, rv, vm.wrapGoTime(*t))
	}
	switch rv.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.String:
		if rv.Kind() == reflect.String && rv.Type() != reflect.TypeOf("") && rv.Type().NumMethod() > 0 {
			return vm.j.ToValue(rv.String())
		}
		return vm.j.ToValue(value)
	case reflect.Map:
		// JS can only address string keys. Named string types are converted
		// on lookup; other key types are rejected rather than panicking.
		if rv.Type().Key().Kind() != reflect.String {
			obj := vm.j.NewObject()
			_ = vm.freeze(obj)
			return internValue(intern, rv, obj)
		}
		return internValue(intern, rv, vm.newReadonlyObject(value, rv, intern, name))
	case reflect.Slice:
		return internValue(intern, rv, vm.newReadonlyArray(value, rv, intern, name))
	case reflect.Array:
		return internValue(intern, rv, vm.newReadonlyArray(value, rv, intern, name))
	case reflect.Pointer:
		if rv.IsNil() {
			return goja.Null()
		}
		elem := rv.Elem()
		if elem.Kind() == reflect.Struct {
			_ = structPlan(rv.Type())
			return internValue(intern, rv, vm.newReadonlyObject(value, rv, intern, name))
		}
		return vm.wrapGoDataNamed(elem.Interface(), intern, name)
	case reflect.Struct:
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		_ = structPlan(ptr.Type())
		// Value structs have no stable Go identity; a copied pointer must not
		// enter the intern table or repeated reads grow it without reuse.
		return vm.newReadonlyObject(value, ptr, intern, name)
	default:
		return vm.j.ToValue(value)
	}
}

func internValue(intern *viewIntern, rv reflect.Value, val goja.Value) goja.Value {
	obj, ok := val.(*goja.Object)
	if !ok {
		return val
	}
	intern.put(rv, obj)
	return obj
}

func (vm *VM) newReadonlyObject(value any, rv reflect.Value, intern *viewIntern, name string) *goja.Object {
	return vm.j.NewDynamicObject(&readonlyObject{vm: vm, intern: intern, name: name, v: value, rv: rv})
}

func (vm *VM) newReadonlyArray(value any, rv reflect.Value, intern *viewIntern, name string) *goja.Object {
	return vm.j.NewDynamicArray(&readonlyArray{vm: vm, intern: intern, name: name, v: value, rv: rv})
}

func childNativeName(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func goIdentity(rv reflect.Value) (internKey, bool) {
	if !rv.IsValid() {
		return internKey{}, false
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map:
		if rv.IsNil() {
			return internKey{}, false
		}
		return internKey{ptr: rv.Pointer(), typ: rv.Type()}, true
	case reflect.Slice:
		if rv.IsNil() {
			return internKey{}, false
		}
		return internKey{ptr: rv.Pointer(), len: rv.Len(), typ: rv.Type()}, true
	default:
		return internKey{}, false
	}
}

func isNativeFunctionMethod(m reflect.Method) bool {
	t := m.Type
	return t.NumIn() == 3 && t.NumOut() == 1 &&
		t.In(1) == vmPtrType && t.In(2) == valuesType && t.Out(0) == anyType
}

func bindNativeFunctionMethod(recv reflect.Value, m reflect.Method) NativeFunction {
	fn := m.Func
	return func(vm *VM, args Values) any {
		out := fn.Call([]reflect.Value{recv, reflect.ValueOf(vm), reflect.ValueOf(args)})
		if !out[0].IsValid() || (out[0].Kind() == reflect.Interface && out[0].IsNil()) {
			return nil
		}
		return out[0].Interface()
	}
}

func (o *readonlyObject) Get(key string) goja.Value {
	switch o.rv.Kind() {
	case reflect.Map:
		return o.mapGet(key)
	case reflect.Pointer:
		return o.structGet(key)
	default:
		return nil
	}
}

func (o *readonlyObject) Set(string, goja.Value) bool { return false }

func (o *readonlyObject) Has(key string) bool {
	switch o.rv.Kind() {
	case reflect.Map:
		return o.mapIndex(key).IsValid()
	case reflect.Pointer:
		plan := structPlan(o.rv.Type())
		_, isField := plan.fields[key]
		_, isMethod := plan.methods[key]
		return isField || isMethod
	default:
		return false
	}
}

func (o *readonlyObject) Delete(string) bool { return false }

func (o *readonlyObject) Keys() []string {
	switch o.rv.Kind() {
	case reflect.Map:
		return o.mapKeys()
	case reflect.Pointer:
		return structPlan(o.rv.Type()).keys
	default:
		return nil
	}
}

func (o *readonlyObject) mapIndex(key string) reflect.Value {
	kt := o.rv.Type().Key()
	if kt.Kind() != reflect.String {
		return reflect.Value{}
	}
	return o.rv.MapIndex(reflect.ValueOf(key).Convert(kt))
}

func (o *readonlyObject) mapGet(key string) goja.Value {
	if cached, ok := o.fields[key]; ok {
		return cached
	}
	got := o.mapIndex(key)
	if !got.IsValid() || !got.CanInterface() {
		return nil
	}
	wrapped := o.vm.wrapGoDataNamed(got.Interface(), o.intern, childNativeName(o.name, key))
	if o.fields == nil {
		o.fields = make(map[string]goja.Value)
	}
	o.fields[key] = wrapped
	return wrapped
}

func (o *readonlyObject) mapKeys() []string {
	keys := make([]string, 0, o.rv.Len())
	iter := o.rv.MapRange()
	for iter.Next() {
		k := iter.Key()
		if k.Kind() != reflect.String {
			continue
		}
		keys = append(keys, k.String())
	}
	return keys
}

func (o *readonlyObject) structGet(key string) goja.Value {
	plan := structPlan(o.rv.Type())
	if index, ok := plan.fields[key]; ok {
		if cached, ok := o.fields[key]; ok {
			return cached
		}
		fv := o.rv.Elem().FieldByIndex(index)
		if !fv.IsValid() || !fv.CanInterface() {
			return nil
		}
		wrapped := o.vm.wrapGoDataNamed(fv.Interface(), o.intern, childNativeName(o.name, key))
		if o.fields == nil {
			o.fields = make(map[string]goja.Value)
		}
		o.fields[key] = wrapped
		return wrapped
	}
	goName, ok := plan.methods[key]
	if !ok {
		return nil
	}
	if cached, ok := o.methods[key]; ok {
		return cached
	}
	m, ok := o.rv.Type().MethodByName(goName)
	if !ok {
		return nil
	}
	wrapped := o.vm.nativeFunction(childNativeName(o.name, key), bindNativeFunctionMethod(o.rv, m))
	if o.methods == nil {
		o.methods = make(map[string]goja.Value)
	}
	o.methods[key] = wrapped
	return wrapped
}

func (a *readonlyArray) Len() int {
	if !a.rv.IsValid() {
		return 0
	}
	return a.rv.Len()
}

func (a *readonlyArray) Get(idx int) goja.Value {
	if idx < 0 || idx >= a.Len() {
		return nil
	}
	if a.elems != nil && a.elems[idx] != nil {
		return a.elems[idx]
	}
	fv := a.rv.Index(idx)
	if !fv.CanInterface() {
		return nil
	}
	name := ""
	if a.name != "" {
		name = fmt.Sprintf("%s[%d]", a.name, idx)
	}
	wrapped := a.vm.wrapGoDataNamed(fv.Interface(), a.intern, name)
	if a.elems == nil {
		a.elems = make([]goja.Value, a.Len())
	}
	a.elems[idx] = wrapped
	return wrapped
}

func (a *readonlyArray) Set(int, goja.Value) bool { return false }

func (a *readonlyArray) SetLen(int) bool { return false }

func unwrapExported(exported any) any {
	switch h := exported.(type) {
	case *readonlyObject:
		return h.v
	case *readonlyArray:
		return h.v
	case *hostObject:
		return h.handle
	default:
		return exported
	}
}

type jsStructPlan struct {
	fields  map[string][]int
	methods map[string]string
	keys    []string
}

var jsStructPlans sync.Map // reflect.Type -> *jsStructPlan

var goNameMapper = goFieldNameMapper{}

func structPlan(ptrType reflect.Type) *jsStructPlan {
	if cached, ok := jsStructPlans.Load(ptrType); ok {
		return cached.(*jsStructPlan)
	}
	elem := ptrType
	if elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	plan := &jsStructPlan{
		fields:  make(map[string][]int),
		methods: make(map[string]string),
	}
	seen := make(map[string]struct{})
	if elem.Kind() == reflect.Struct {
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := goNameMapper.FieldName(elem, field)
			plan.fields[name] = field.Index
			if _, ok := seen[name]; !ok {
				plan.keys = append(plan.keys, name)
				seen[name] = struct{}{}
			}
			for _, alias := range jsObjectFieldNames(field) {
				plan.fields[alias] = field.Index
			}
		}
	}
	n := ptrType.NumMethod()
	if ptrType.Kind() != reflect.Pointer {
		n = 0
	}
	for i := 0; i < n; i++ {
		m := ptrType.Method(i)
		name := goNameMapper.MethodName(ptrType, m)
		if name == "" {
			continue
		}
		if !isNativeFunctionMethod(m) {
			panic(fmt.Sprintf("ToJSValue: %s.%s is not NativeFunction; pass NativeFunction instead", ptrType.Elem(), m.Name))
		}
		plan.methods[name] = m.Name
		if _, ok := seen[name]; !ok {
			plan.keys = append(plan.keys, name)
			seen[name] = struct{}{}
		}
	}
	actual, _ := jsStructPlans.LoadOrStore(ptrType, plan)
	return actual.(*jsStructPlan)
}
