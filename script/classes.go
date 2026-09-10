package script

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"

	"github.com/dop251/goja"
)

// JSClass maps a Go handle type to a JavaScript class.
type JSClass struct {
	Name string
	// Parent is another JSClass name in the same ClassSet.
	// Callers cannot extend BuiltinClasses or JavaScript Error.
	Parent    string
	Construct NativeFunction
	// Handle is a zero value of the instance type. It must embed ClassHost.
	Handle any
	// Accepts reports whether a native Go value should wrap as this class.
	// It must not allocate or register disposables.
	Accepts func(any) bool
	Wrap    func(*VM, any) any
	Statics map[string]any
	Methods map[string]ClassMethod
	Getters map[string]ClassMethod

	parent      *JSClass
	methodNames []string
	getterNames []string
	staticNames []string
}

// ClassMethod is a host-class prototype method.
type ClassMethod func(vm *VM, this *Value, args Values) any

// ClassSet is an immutable catalog of host classes and may be shared by VMs.
// Compose extras with With; install with AddClassSet (before pool initialization completes)
// or VMPoolConfig.Classes. Lookups (by name, handle type, Accepts) live here.
type ClassSet struct {
	classes   []*JSClass
	byName    map[string]*JSClass
	accepting []*JSClass
	byType    map[reflect.Type][]*JSClass
	names     []string
}

// ClassHost must be embedded in JSClass handle types so goja hides their Go methods.
// Do not embed it on types that still export methods through goja (Drive, Entry).
type ClassHost struct {
	vm *VM
}

func (ClassHost) jsClassHandle() {}

func (h ClassHost) VM() *VM { return h.vm }

// NewClassHost binds a handle to the Runtime that constructed it.
func NewClassHost(vm *VM) ClassHost { return ClassHost{vm: vm} }

type jsClassHandle interface {
	jsClassHandle()
}

var (
	jsClassHandleType = reflect.TypeFor[jsClassHandle]()
	hostObjectType    = reflect.TypeFor[*hostObject]()
)

var (
	// BuiltinClasses is installed by NewVM.
	BuiltinClasses *ClassSet
)

func init() {
	BuiltinClasses = newClassSet(
		&jsClassDrive,
		&jsClassEntry,
		&jsClassBytes,
		&jsClassReader,
		&jsClassReadCloser,
		&jsClassTempFile,
		&jsClassProgressReporter,
		&jsClassHash,
		&jsClassHmac,
		&jsClassHttpFormData,
		&jsClassBadRequestError,
		&jsClassNotFoundError,
		&jsClassNotAllowedError,
		&jsClassUnsupportedError,
		&jsClassRemoteApiError,
	)
}

// NewClassSet clones and finalizes classes. Invalid definitions panic.
// Parent cannot be Error or a class from BuiltinClasses.
func NewClassSet(classes ...*JSClass) *ClassSet {
	mustNotExtendBuiltin(classes)
	return newClassSet(classes...)
}

func newClassSet(classes ...*JSClass) *ClassSet {
	s := &ClassSet{
		byName: make(map[string]*JSClass, len(classes)),
		byType: make(map[reflect.Type][]*JSClass),
	}
	for _, c := range classes {
		if e := s.add(c.clone()); e != nil {
			panic(e)
		}
	}
	if e := s.finalize(); e != nil {
		panic(e)
	}
	return s
}

// With returns a new finalized set containing s plus classes. s may be nil.
func (s *ClassSet) With(classes ...*JSClass) *ClassSet {
	mustNotExtendBuiltin(classes)
	n := len(classes)
	if s != nil {
		n += len(s.classes)
	}
	all := make([]*JSClass, 0, n)
	if s != nil {
		all = append(all, s.classes...)
	}
	all = append(all, classes...)
	return newClassSet(all...)
}

func (c *JSClass) clone() *JSClass {
	if c == nil {
		return nil
	}
	cp := *c
	cp.parent = nil
	cp.Methods = maps.Clone(c.Methods)
	cp.Getters = maps.Clone(c.Getters)
	cp.Statics = maps.Clone(c.Statics)
	return &cp
}

func mustNotExtendBuiltin(classes []*JSClass) {
	for _, c := range classes {
		if e := rejectExtendBuiltin(c); e != nil {
			panic(e)
		}
	}
}

func rejectExtendBuiltin(c *JSClass) error {
	if c == nil || c.Parent == "" {
		return nil
	}
	if c.Parent == jsParentError || (BuiltinClasses != nil && BuiltinClasses.byName[c.Parent] != nil) {
		return fmt.Errorf("host class %s cannot extend builtin class %s", c.Name, c.Parent)
	}
	return nil
}

func (s *ClassSet) add(c *JSClass) error {
	if c == nil || c.Name == "" {
		return errors.New("host class is missing Name")
	}
	if c.Construct == nil {
		return fmt.Errorf("host class %s is missing Construct", c.Name)
	}
	if s.byName[c.Name] != nil {
		return fmt.Errorf("duplicate host class %s", c.Name)
	}
	c.methodNames = sortedMapKeys(c.Methods)
	c.getterNames = sortedMapKeys(c.Getters)
	c.staticNames = sortedMapKeys(c.Statics)
	for name := range c.Getters {
		if _, ok := c.Methods[name]; ok {
			return fmt.Errorf("host class %s getter/method name collision: %s", c.Name, name)
		}
	}
	s.classes = append(s.classes, c)
	s.byName[c.Name] = c
	if c.Accepts != nil {
		s.accepting = append(s.accepting, c)
	}
	if e := s.indexHandle(c); e != nil {
		return e
	}
	return nil
}

func (s *ClassSet) indexHandle(c *JSClass) error {
	if c.Handle == nil {
		return nil
	}
	t := reflect.TypeOf(c.Handle)
	if !isJSClassHandleType(t) {
		return fmt.Errorf("host class %s Handle must embed ClassHost", c.Name)
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for _, key := range []reflect.Type{t, reflect.PointerTo(t)} {
		if slices.Contains(s.byType[key], c) {
			continue
		}
		s.byType[key] = append(s.byType[key], c)
	}
	return nil
}

func (s *ClassSet) finalize() error {
	for _, c := range s.classes {
		if e := c.bindParent(s.byName); e != nil {
			return e
		}
	}
	for _, c := range s.classes {
		if e := c.checkParentCycle(); e != nil {
			return e
		}
	}
	names := make([]string, 0, len(s.classes))
	for _, c := range s.classes {
		names = append(names, c.Name)
	}
	sort.Strings(names)
	s.names = names
	// A depth-first order keeps descendants ahead of ancestors without
	// mixing a partial inheritance order with alphabetical comparisons.
	depths := make(map[*JSClass]int, len(s.accepting))
	for _, c := range s.accepting {
		for p := c.parent; p != nil; p = p.parent {
			depths[c]++
		}
	}
	sort.SliceStable(s.accepting, func(i, j int) bool {
		a, b := s.accepting[i], s.accepting[j]
		if depths[a] != depths[b] {
			return depths[a] > depths[b]
		}
		return a.Name < b.Name
	})
	return nil
}

func (c *JSClass) bindParent(byName map[string]*JSClass) error {
	if c.Parent == "" || c.Parent == jsParentError {
		return nil
	}
	p := byName[c.Parent]
	if p == nil {
		return fmt.Errorf("host class %s parent %s is not registered", c.Name, c.Parent)
	}
	c.parent = p
	return nil
}

func (c *JSClass) checkParentCycle() error {
	seen := map[*JSClass]struct{}{c: {}}
	for p := c.parent; p != nil; p = p.parent {
		if _, ok := seen[p]; ok {
			return fmt.Errorf("host class cycle involving %s", c.Name)
		}
		seen[p] = struct{}{}
	}
	return nil
}

// AcceptsNonNil is an Accepts func for a non-nil T.
func AcceptsNonNil[T any]() func(any) bool {
	return func(v any) bool {
		t, ok := v.(T)
		if !ok {
			return false
		}
		rv := reflect.ValueOf(t)
		if !rv.IsValid() {
			return false
		}
		switch rv.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return !rv.IsNil()
		default:
			return true
		}
	}
}

// HostAs extracts a host capability or concrete handle from a JS value or Go
// handle. Interface T matches subclass handles; *T is also accepted.
func HostAs[T any](v any) (T, bool) {
	switch x := unwrapHost(v).(type) {
	case T:
		return x, true
	case *T:
		if x != nil {
			return *x, true
		}
	}
	var zero T
	return zero, false
}

func This[T any](vm *VM, this *Value, name string) T {
	v, ok := HostAs[T](this)
	if !ok {
		vm.ThrowTypeError(name + " called on incompatible receiver")
	}
	return v
}

func isJSClassHandleType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t.Implements(jsClassHandleType) {
		return true
	}
	if t.Kind() == reflect.Pointer {
		return t.Elem().Implements(jsClassHandleType)
	}
	return reflect.PointerTo(t).Implements(jsClassHandleType)
}

// hostObject is the JS instance itself. The Go handle lives on this
// DynamicObject so it is not a VM map root (which would pin handles and
// any JS they retain). Instances have no
// own data properties; methods and Symbol.toStringTag live on the prototype.
type hostObject struct {
	handle any
}

func (*hostObject) Get(string) goja.Value { return nil }

func (*hostObject) Set(string, goja.Value) bool { return false }

func (*hostObject) Has(string) bool { return false }

func (*hostObject) Delete(string) bool { return true }

func (*hostObject) Keys() []string { return nil }

func (vm *VM) newHostObject(proto *goja.Object, handle any) *goja.Object {
	if vm == nil || vm.j == nil {
		if vm != nil {
			vm.ThrowTypeError("Runtime has been disposed")
		}
		panic("Runtime has been disposed")
	}
	obj := vm.j.NewDynamicObject(&hostObject{handle: handle})
	if proto != nil {
		if e := obj.SetPrototype(proto); e != nil {
			vm.ThrowError(e)
		}
	}
	return obj
}

func unwrapHost(v any) any {
	if v == nil {
		return nil
	}
	if value, ok := v.(*Value); ok {
		if value == nil {
			return nil
		}
		return unwrapHost(value.v)
	}
	if gv, ok := v.(goja.Value); ok {
		if goja.IsUndefined(gv) || goja.IsNull(gv) {
			return nil
		}
		return unwrapExported(gv.Export())
	}
	return v
}

// ConstructorHostOnly is Construct for classes that JavaScript cannot `new`.
func ConstructorHostOnly(name string) NativeFunction {
	return func(vm *VM, _ Values) any {
		vm.ThrowTypeError(name + " cannot be constructed from JavaScript")
		return nil
	}
}

func (vm *VM) classSet() *ClassSet {
	if vm != nil && vm.classes != nil {
		return vm.classes
	}
	return BuiltinClasses
}

func (s *ClassSet) classByName(name string) *JSClass {
	if s == nil {
		return nil
	}
	return s.byName[name]
}

func (s *ClassSet) classOf(v any) *JSClass {
	if s == nil || v == nil {
		return nil
	}
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	classes := s.byType[t]
	if len(classes) == 0 {
		return nil
	}
	return classes[0]
}

func (s *ClassSet) classAccepting(v any) *JSClass {
	if s == nil {
		return nil
	}
	for _, c := range s.accepting {
		if c.Accepts(v) {
			return c
		}
	}
	return nil
}

func (s *ClassSet) containsNamesOf(other *ClassSet) bool {
	if other == nil {
		return true
	}
	if s == nil {
		return false
	}
	for _, c := range other.classes {
		if s.byName[c.Name] == nil {
			return false
		}
	}
	return true
}

// AddClassSet installs classes whose names are not already on this Runtime.
// Call before pool initialization completes. NewVM uses this for BuiltinClasses as well.
func (vm *VM) AddClassSet(set *ClassSet) error {
	if vm == nil || vm.disposed || vm.j == nil {
		return errors.New("Runtime has been disposed")
	}
	if set == nil {
		return nil
	}
	if vm.ctors == nil {
		vm.ctors = make(map[string]*goja.Object, len(set.classes))
	}
	added, e := vm.installClasses(set.classes)
	if e != nil {
		return e
	}
	if len(added) == 0 {
		return nil
	}
	cur := vm.classSet()
	if set.containsNamesOf(cur) {
		vm.classes = set
	} else {
		vm.classes = cur.With(added...)
	}
	return nil
}

func (vm *VM) installClasses(classes []*JSClass) ([]*JSClass, error) {
	added := make([]*JSClass, 0)
	for _, c := range classes {
		if c == nil || vm.ctors[c.Name] != nil {
			continue
		}
		added = append(added, c)
	}
	if len(added) == 0 {
		return nil, nil
	}
	for _, class := range added {
		if e := vm.installClassConstructor(class); e != nil {
			return nil, e
		}
	}
	for _, class := range added {
		if e := vm.linkInstalledClassParent(class); e != nil {
			return nil, e
		}
	}
	for _, class := range added {
		if e := vm.installClassPrototype(class); e != nil {
			return nil, e
		}
	}
	for _, class := range added {
		if e := vm.freezeClass(class); e != nil {
			return nil, e
		}
	}
	for _, class := range added {
		if e := vm.bindClassGlobal(class.Name); e != nil {
			return nil, e
		}
	}
	return added, nil
}

func (vm *VM) installClassConstructor(class *JSClass) error {
	ctor := vm.j.ToValue(vm.hostConstructor(class)).ToObject(vm.j)
	if e := ctor.DefineDataProperty(
		"name",
		vm.j.ToValue(class.Name),
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
		goja.FLAG_TRUE,
	); e != nil {
		return e
	}
	for _, key := range class.staticNames {
		if e := ctor.Set(key, vm.ToJSValue(class.Statics[key]).v); e != nil {
			return e
		}
	}
	vm.ctors[class.Name] = ctor
	return nil
}

func (vm *VM) linkInstalledClassParent(class *JSClass) error {
	ctor := vm.ctors[class.Name]
	switch {
	case class.Parent == jsParentError:
		return vm.linkClassParent(ctor, vm.jsVars.errorCtor)
	case class.parent != nil:
		return vm.linkClassParent(ctor, vm.ctors[class.parent.Name])
	}
	return nil
}

func (vm *VM) freezeClass(class *JSClass) error {
	ctor := vm.ctors[class.Name]
	if e := vm.freeze(ctor.Get("prototype")); e != nil {
		return e
	}
	return vm.freeze(ctor)
}

func (vm *VM) bindClassGlobal(name string) error {
	return vm.j.GlobalObject().DefineDataProperty(
		name,
		vm.ctors[name],
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
		goja.FLAG_TRUE,
	)
}

func (vm *VM) linkClassParent(ctor, parent *goja.Object) error {
	if e := ctor.Get("prototype").ToObject(vm.j).SetPrototype(parent.Get("prototype").ToObject(vm.j)); e != nil {
		return e
	}
	return ctor.SetPrototype(parent)
}

func (vm *VM) installClassPrototype(class *JSClass) error {
	proto := vm.ctors[class.Name].Get("prototype").ToObject(vm.j)
	// Same attributes as Map/Promise in goja: writable false, enumerable false, configurable true.
	if e := proto.DefineDataPropertySymbol(
		goja.SymToStringTag,
		vm.j.ToValue(class.Name),
		goja.FLAG_FALSE,
		goja.FLAG_TRUE,
		goja.FLAG_FALSE,
	); e != nil {
		return e
	}
	if len(class.methodNames) == 0 && len(class.getterNames) == 0 {
		return nil
	}
	for _, key := range class.methodNames {
		if e := proto.DefineDataProperty(
			key,
			vm.classMethodValue(class.Methods[key]),
			goja.FLAG_TRUE,  // writable
			goja.FLAG_TRUE,  // configurable
			goja.FLAG_FALSE, // enumerable (ES class methods)
		); e != nil {
			return e
		}
	}
	for _, key := range class.getterNames {
		if e := proto.DefineAccessorProperty(
			key,
			vm.classMethodValue(class.Getters[key]),
			nil,
			goja.FLAG_TRUE,  // configurable
			goja.FLAG_FALSE, // enumerable (ES class getters)
		); e != nil {
			return e
		}
	}
	return nil
}

func (vm *VM) classMethodValue(fn ClassMethod) goja.Value {
	return vm.j.ToValue(func(call goja.FunctionCall) goja.Value {
		result := fn(vm, newValue(vm, call.This), newValues(vm, call.Arguments))
		if result == nil {
			return goja.Undefined()
		}
		return vm.ToJSValue(result).v
	})
}

func (vm *VM) hostConstructor(class *JSClass) func(goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		value := class.Construct(vm, newValues(vm, call.Arguments))
		if class.Parent == jsParentError {
			obj, ok := value.(*goja.Object)
			if !ok {
				panic(vm.j.NewTypeError(class.Name + " constructor did not produce an object"))
			}
			if proto := call.This.Prototype(); proto != nil {
				if e := obj.SetPrototype(proto); e != nil {
					panic(e)
				}
			}
			return obj
		}
		return vm.newHostObject(call.This.Prototype(), value)
	}
}

// NewInstance builds a host instance from Go without calling the JavaScript
// constructor. value is the native Go value for classes with Wrap, or the
// handle itself otherwise.
func (vm *VM) NewInstance(name string, value any) *Value {
	class := vm.classSet().classByName(name)
	if class == nil {
		panic("unknown host class: " + name)
	}
	handle := value
	if class.Wrap != nil {
		if class.Accepts == nil || !class.Accepts(value) {
			panic("cannot construct " + name + " from native value")
		}
		handle = class.Wrap(vm, value)
	}
	return newValue(vm, vm.instantiate(class, handle))
}

func (vm *VM) instantiate(class *JSClass, handle any) *goja.Object {
	proto := vm.ctors[class.Name].Get("prototype").ToObject(vm.j)
	return vm.newHostObject(proto, handle)
}

func (vm *VM) wrapHostClass(value any) goja.Value {
	if !mightBeHostValue(value) {
		return nil
	}
	set := vm.classSet()
	if c := set.classOf(value); c != nil {
		return vm.instantiate(c, value)
	}
	if c := set.classAccepting(value); c != nil {
		return vm.NewInstance(c.Name, value).v
	}
	return nil
}

func mightBeHostValue(v any) bool {
	switch v.(type) {
	case bool, string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return false
	default:
		return true
	}
}
