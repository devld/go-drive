package script

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestTypedNilHostValuesBecomeNull(t *testing.T) {
	vm := newPoolTestVM(t)
	var reader *bytes.Reader
	var file *os.File
	if AcceptsNonNil[io.Reader]()(reader) || AcceptsNonNil[io.ReadCloser]()(file) {
		t.Fatal("typed nil matched a non-nil host capability")
	}
	if !AcceptsNonNil[io.Reader]()(bytes.NewReader(nil)) {
		t.Fatal("non-nil empty reader rejected")
	}
	for name, value := range map[string]any{"reader": reader, "file": file} {
		t.Run(name, func(t *testing.T) {
			if e := vm.DefineGlobal(name, value); e != nil {
				t.Fatal(e)
			}
			result, e := vm.Run(context.Background(), name+` === null`, "")
			if e != nil || !result.Bool() {
				t.Fatalf("typed nil did not become null: %v", e)
			}
			if _, e := vm.Run(context.Background(), name+`.readAsString()`, ""); e == nil {
				t.Fatal("null reader access should return a JS error")
			}
		})
	}
	if e := vm.DefineGlobal("nested", map[string]any{"reader": reader}); e != nil {
		t.Fatal(e)
	}
	result, e := vm.Run(context.Background(), `nested.reader === null`, "")
	if e != nil || !result.Bool() {
		t.Fatalf("nested typed nil did not become null: %v", e)
	}
}

func TestHostClassesInstanceOf(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		var bytes = Bytes.fromString("hello");
		var tmp = new TempFile();
		var limited = tmp.limitReader(1);
		var hmac = new Hmac("sha256", Bytes.fromString("key"));
		[
			bytes instanceof Bytes,
			tmp instanceof TempFile,
			tmp instanceof ReadCloser,
			tmp instanceof Reader,
			limited instanceof Reader,
			!(limited instanceof TempFile),
			hmac instanceof Hmac,
			hmac instanceof Hash,
			Object.getPrototypeOf(TempFile.prototype) === ReadCloser.prototype,
			Object.getPrototypeOf(Hmac.prototype) === Hash.prototype
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|true|true|true|true|true|true|true|true" {
		t.Fatalf("instanceof = %q", got.String())
	}
}

func TestHostClassMethodsLiveOnPrototype(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		var a = Bytes.fromString("ab");
		var b = Bytes.fromString("cd");
		var tmp = new TempFile();
		var hash = new Hash("md5");
		var hmac = new Hmac("sha256", Bytes.fromString("key"));
		var own = function (obj, key) {
			return Object.prototype.hasOwnProperty.call(obj, key);
		};
		[
			a.slice === Bytes.prototype.slice,
			a.slice === b.slice,
			!own(a, "slice"),
			!own(a, "length"),
			a.length === 2,
			typeof Object.getOwnPropertyDescriptor(Bytes.prototype, "length").get === "function",
			Object.getOwnPropertyDescriptor(Bytes.prototype, "length").set === undefined,
			typeof Bytes.prototype.slice === "function",
			!Object.prototype.propertyIsEnumerable.call(Bytes.prototype, "slice"),
			tmp.read === Reader.prototype.read,
			tmp.close === TempFile.prototype.close,
			tmp.close !== ReadCloser.prototype.close,
			!own(tmp, "read"),
			!own(tmp, "close"),
			hmac.write === Hash.prototype.write,
			hash.write(Bytes.fromString("a")) === hash,
			hmac.write(Bytes.fromString("a")) === hmac
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(got.String(), "|")
	for i, part := range parts {
		if part != "true" {
			t.Fatalf("prototype[%d] = %q (all=%q)", i, part, got.String())
		}
	}
	if len(parts) != 17 {
		t.Fatalf("prototype check count = %d, want 17 (%q)", len(parts), got.String())
	}
}

func TestHostClassCatalogIsRegistered(t *testing.T) {
	vm := newPoolTestVM(t)
	if vm.classes != BuiltinClasses {
		t.Fatal("NewVM should share BuiltinClasses")
	}
	if len(vm.ctors) != len(BuiltinClasses.classes) {
		t.Fatalf("registered %d ctors, catalog %d", len(vm.ctors), len(BuiltinClasses.classes))
	}
	if len(BuiltinClasses.names) != len(BuiltinClasses.classes) {
		t.Fatalf("names %d, catalog %d", len(BuiltinClasses.names), len(BuiltinClasses.classes))
	}
	for i := 1; i < len(BuiltinClasses.names); i++ {
		if BuiltinClasses.names[i-1] >= BuiltinClasses.names[i] {
			t.Fatalf("names not sorted: %v", BuiltinClasses.names)
		}
	}
	for _, class := range BuiltinClasses.classes {
		if vm.ctors[class.Name] == nil {
			t.Fatalf("class %q ctor is nil", class.Name)
		}
		if class.Construct == nil {
			t.Fatalf("class %q is missing Construct", class.Name)
		}
		if class.Parent != jsParentError && class.Handle == nil {
			t.Fatalf("class %q is missing Handle", class.Name)
		}
		if class.Handle != nil && !isJSClassHandleType(reflect.TypeOf(class.Handle)) {
			t.Fatalf("class %q Handle does not embed ClassHost", class.Name)
		}
		if class.Parent != "" && class.Parent != jsParentError && class.parent == nil {
			t.Fatalf("class %q parent %q was not resolved", class.Name, class.Parent)
		}
		if class.parent != nil && class.parent.Name != class.Parent {
			t.Fatalf("class %q parent pointer is %q, want %q", class.Name, class.parent.Name, class.Parent)
		}
		got, e := vm.Run(context.Background(), `typeof globalThis[`+strconv.Quote(class.Name)+`]`, "")
		if e != nil {
			t.Fatal(e)
		}
		if got.String() != "function" {
			t.Fatalf("global %s typeof = %q", class.Name, got.String())
		}
		tag, e := vm.Run(context.Background(), `
			Object.getOwnPropertyDescriptor(`+class.Name+`.prototype, Symbol.toStringTag).value
		`, "")
		if e != nil {
			t.Fatalf("%s toStringTag: %v", class.Name, e)
		}
		if tag.String() != class.Name {
			t.Fatalf("%s @@toStringTag = %q", class.Name, tag.String())
		}
	}
}

func TestReaderAndReadCloserAreHostOnly(t *testing.T) {
	vm := newPoolTestVM(t)
	if _, e := vm.Run(context.Background(), `new Reader()`, ""); e == nil || !strings.Contains(e.Error(), "Reader cannot be constructed from JavaScript") {
		t.Fatalf("new Reader = %v", e)
	}
	if _, e := vm.Run(context.Background(), `new ReadCloser()`, ""); e == nil || !strings.Contains(e.Error(), "ReadCloser cannot be constructed from JavaScript") {
		t.Fatalf("new ReadCloser = %v", e)
	}
}

func TestToJSValueWrapsNativeHostValues(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "buf", []byte("abc"))
	mustDefineGlobal(t, vm, "reader", strings.NewReader("hi"))
	mustDefineGlobal(t, vm, "closer", io.NopCloser(strings.NewReader("hi")))
	mustDefineGlobal(t, vm, "str", "not-bytes")

	got, e := vm.Run(context.Background(), `
		[
			buf instanceof Bytes,
			buf.toString(),
			reader instanceof Reader,
			!(reader instanceof ReadCloser),
			closer instanceof ReadCloser,
			closer instanceof Reader,
			typeof str
		].join("|")
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|abc|true|true|true|true|string" {
		t.Fatalf("native host wrapping = %q", got.String())
	}
	ret, e := vm.Run(context.Background(), `buf`, "")
	if e != nil {
		t.Fatal(e)
	}
	b, ok := HostAs[jsObjBytes](ret.Raw())
	if !ok || string(b.b) != "abc" {
		t.Fatalf("Raw() handle = %T %#v", ret.Raw(), ret.Raw())
	}
}

type closeProbe struct {
	io.Reader
	closed bool
}

func (c *closeProbe) Close() error {
	c.closed = true
	return nil
}

func TestReadCloserRemovedFromDisposablesSurvivesVMDispose(t *testing.T) {
	vm := newPoolTestVM(t)
	probe := &closeProbe{Reader: strings.NewReader("hello")}
	obj := vm.NewInstance("ReadCloser", probe)
	handle := unwrapHost(obj)
	rc := GetReadCloser(vm, handle, "")
	if rc == nil {
		t.Fatal("GetReadCloser = nil")
	}
	vm.RemoveDisposable(handle)
	if e := vm.disposeDisposables(); e != nil {
		t.Fatal(e)
	}
	if probe.closed {
		t.Fatal("VM dispose closed a reader removed from disposables")
	}
	got, e := io.ReadAll(rc)
	if e != nil || string(got) != "hello" {
		t.Fatalf("read after RemoveDisposable = %q %v", got, e)
	}
	if e := rc.Close(); e != nil {
		t.Fatal(e)
	}
	if !probe.closed {
		t.Fatal("caller Close did not close")
	}
}

func TestReadCloserDisposedWithVM(t *testing.T) {
	vm := newPoolTestVM(t)
	probe := &closeProbe{Reader: strings.NewReader("hello")}
	_ = vm.NewInstance("ReadCloser", probe)
	if e := vm.disposeDisposables(); e != nil {
		t.Fatal(e)
	}
	if !probe.closed {
		t.Fatal("VM dispose should close a ReadCloser still in disposables")
	}
}

type readerTestEntry struct {
	inspectTestEntry
}

func (e readerTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("hi")), nil
}

func TestHostClassReturnedReadersInstanceOf(t *testing.T) {
	vm := newPoolTestVM(t)
	entry := jsObjEntry{ClassHost: NewClassHost(vm), e: readerTestEntry{inspectTestEntry{path: "f", name: "f", typ: "file"}}}
	mustDefineGlobal(t, vm, "entry", entry)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)
	mustDefineGlobal(t, vm, "url", srv.URL)

	got, e := vm.Run(context.Background(), `
		var body = http(url).body;
		var reader = entry.getReader(-1, -1);
		var limited = reader.limitReader(1);
		[
			body instanceof ReadCloser,
			body instanceof Reader,
			reader instanceof ReadCloser,
			reader instanceof Reader,
			limited instanceof Reader,
			!(limited instanceof ReadCloser)
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|true|true|true|true" {
		t.Fatalf("returned readers instanceof = %q", got.String())
	}
}

type definedBox struct {
	ClassHost
	n int
}

func (b definedBox) N() int { return b.n }

type definedSubBox struct {
	definedBox
}

func boxClass() *JSClass {
	return &JSClass{
		Name:   "Box",
		Handle: definedBox{},
		Construct: func(vm *VM, args Values) any {
			return definedBox{ClassHost: NewClassHost(vm), n: int(args.Get(0).Integer())}
		},
		Methods: map[string]ClassMethod{
			"value": func(vm *VM, this *Value, _ Values) any {
				return This[interface{ N() int }](vm, this, "Box.value").N()
			},
		},
	}
}

func newTestVMWithClasses(t *testing.T, extra ...*JSClass) *VM {
	t.Helper()
	vm := newPoolTestVM(t)
	if e := vm.AddClassSet(NewClassSet(extra...)); e != nil {
		t.Fatal(e)
	}
	return vm
}

type nativeBoxInput struct{ n int }

func TestClassSetAcceptsPrefersDescendant(t *testing.T) {
	base := boxClass()
	base.Name = "ABase"
	base.Accepts = AcceptsNonNil[*nativeBoxInput]()
	base.Wrap = func(vm *VM, value any) any {
		return definedBox{ClassHost: NewClassHost(vm), n: value.(*nativeBoxInput).n}
	}
	middle := boxClass()
	middle.Name = "Middle"
	middle.Accepts = func(any) bool { return false }
	child := &JSClass{
		Name: "ZChild", Parent: "ABase", Handle: definedSubBox{},
		Construct: ConstructorHostOnly("ZChild"),
		Accepts:   base.Accepts,
		Wrap: func(vm *VM, value any) any {
			return definedSubBox{definedBox{ClassHost: NewClassHost(vm), n: value.(*nativeBoxInput).n}}
		},
		Methods: map[string]ClassMethod{
			"childValue": func(*VM, *Value, Values) any { return "child" },
		},
	}
	for _, order := range [][]*JSClass{
		{base, middle, child},
		{child, middle, base},
	} {
		t.Run(order[0].Name+"/"+order[1].Name, func(t *testing.T) {
			vm := newTestVMWithClasses(t, order...)
			mustDefineGlobal(t, vm, "wrapped", &nativeBoxInput{n: 7})
			got, e := vm.Run(context.Background(), `
				[wrapped instanceof ZChild, wrapped instanceof ABase,
				 wrapped.value(), wrapped.childValue()].join("|")
			`, "")
			if e != nil {
				t.Fatal(e)
			}
			if got.String() != "true|true|7|child" {
				t.Fatalf("native class match = %q", got.String())
			}
		})
	}
}

func TestClassSetCopiesDefinitionMaps(t *testing.T) {
	for _, compose := range []struct {
		name string
		make func(*JSClass) *ClassSet
	}{
		{"NewClassSet", func(c *JSClass) *ClassSet { return NewClassSet(c) }},
		{"With", func(c *JSClass) *ClassSet { return BuiltinClasses.With(c) }},
	} {
		t.Run(compose.name, func(t *testing.T) {
			class := boxClass()
			class.Getters = map[string]ClassMethod{
				"next": func(vm *VM, this *Value, _ Values) any {
					return This[definedBox](vm, this, "Box.next").n + 1
				},
			}
			class.Statics = map[string]any{"label": "original"}
			set := compose.make(class)
			class.Methods["value"] = func(*VM, *Value, Values) any { return 999 }
			class.Getters["next"] = func(*VM, *Value, Values) any { return 999 }
			class.Statics["label"] = "changed"
			vm := newPoolTestVM(t)
			if e := vm.AddClassSet(set); e != nil {
				t.Fatal(e)
			}
			got, e := vm.Run(context.Background(), `
				const box = new Box(7);
				[box.value(), box.next, Box.label].join("|")
			`, "")
			if e != nil {
				t.Fatal(e)
			}
			if got.String() != "7|8|original" {
				t.Fatalf("class changed after creation: %q", got.String())
			}
		})
	}
}

func TestClassSetInjectedClass(t *testing.T) {
	vm := newTestVMWithClasses(t, boxClass())

	got, e := vm.Run(context.Background(), `
		var b = new Box(7);
		[b instanceof Box, b.value()].join("|")
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|7" {
		t.Fatalf("injected class = %q", got.String())
	}

	mustDefineGlobal(t, vm, "fromGo", definedBox{n: 3})
	wrapped, e := vm.Run(context.Background(), `fromGo instanceof Box && fromGo.value()`, "")
	if e != nil {
		t.Fatal(e)
	}
	if wrapped.Integer() != 3 {
		t.Fatalf("ToJSValue wrap = %q", wrapped.String())
	}

	obj := vm.NewInstance("Box", definedBox{n: 9})
	mustDefineGlobal(t, vm, "inst", obj)
	n, e := vm.Run(context.Background(), `inst instanceof Box && inst.value()`, "")
	if e != nil {
		t.Fatal(e)
	}
	if n.Integer() != 9 {
		t.Fatalf("NewInstance = %q", n.String())
	}
}

func TestClassSetParent(t *testing.T) {
	vm := newTestVMWithClasses(t, boxClass(), &JSClass{
		Name:   "SubBox",
		Parent: "Box",
		Handle: definedSubBox{},
		Construct: func(_ *VM, args Values) any {
			return definedSubBox{definedBox{n: int(args.Get(0).Integer())}}
		},
	})

	got, e := vm.Run(context.Background(), `
		var s = new SubBox(4);
		[
			s instanceof SubBox,
			s instanceof Box,
			s.value(),
			Object.getPrototypeOf(SubBox.prototype) === Box.prototype
		].join("|")
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|4|true" {
		t.Fatalf("ClassSet parent = %q", got.String())
	}
	tag, e := vm.Run(context.Background(), `Object.prototype.toString.call(new SubBox(1))`, "")
	if e != nil {
		t.Fatal(e)
	}
	if tag.String() != "[object SubBox]" {
		t.Fatalf("subclass toStringTag = %q", tag.String())
	}

	mustDefineGlobal(t, vm, "subGo", definedSubBox{definedBox{n: 5}})
	wrapped, e := vm.Run(context.Background(), `subGo instanceof SubBox && subGo instanceof Box && subGo.value()`, "")
	if e != nil {
		t.Fatal(e)
	}
	if wrapped.Integer() != 5 {
		t.Fatalf("subclass wrap = %q", wrapped.String())
	}
}

func TestClassSetInPool(t *testing.T) {
	pool, e := NewVMPool(context.Background(), nil, &VMPoolConfig{
		MaxTotal: 2,
		MaxIdle:  2,
		MinIdle:  1,
		Classes:  BuiltinClasses.With(boxClass()),
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })

	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = pool.Return(context.Background(), vm) }()

	got, e := vm.Run(context.Background(), `new Box(2).value()`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.Integer() != 2 {
		t.Fatalf("pooled ClassSet = %q", got.String())
	}

	vm2, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = pool.Return(context.Background(), vm2) }()
	if vm.classes != vm2.classes || vm.classes != pool.config.Classes {
		t.Fatal("pool VMs should share VMPoolConfig.Classes")
	}
}

func TestClassSetSharedAcrossVMs(t *testing.T) {
	set := BuiltinClasses.With(boxClass())
	vm1 := newPoolTestVM(t)
	vm2 := newPoolTestVM(t)
	if e := vm1.AddClassSet(set); e != nil {
		t.Fatal(e)
	}
	if e := vm2.AddClassSet(set); e != nil {
		t.Fatal(e)
	}
	if vm1.classes != set || vm2.classes != set {
		t.Fatal("AddClassSet should keep a complete extra set shared")
	}

	partial := NewClassSet(boxClass())
	vm3 := newPoolTestVM(t)
	if e := vm3.AddClassSet(partial); e != nil {
		t.Fatal(e)
	}
	if vm3.classes == partial || vm3.classes == BuiltinClasses {
		t.Fatal("partial set should be merged onto builtins")
	}
	if vm3.classSet().classByName("Box") == nil || vm3.classSet().classByName("Bytes") == nil {
		t.Fatal("merged set should have Box and Bytes")
	}
}

func TestAddClassSetAfterFreezeGlobal(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = pool.Return(context.Background(), vm) }()
	if e := vm.AddClassSet(NewClassSet(boxClass())); e == nil {
		t.Fatal("expected AddClassSet after FreezeGlobal to fail")
	}
}

func TestNewClassSetUnknownParentPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewClassSet(&JSClass{
		Name:      "Broken",
		Parent:    "NoSuchClass",
		Construct: ConstructorHostOnly("Broken"),
	})
}

func TestNewClassSetHandleMustEmbedClassHost(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewClassSet(&JSClass{
		Name:      "NoHost",
		Handle:    struct{ N int }{},
		Construct: ConstructorHostOnly("NoHost"),
	})
}

func TestRejectExtendBuiltin(t *testing.T) {
	hostOnly := func(name, parent string) *JSClass {
		return &JSClass{Name: name, Parent: parent, Construct: ConstructorHostOnly(name)}
	}
	cases := []struct {
		name string
		fn   func()
	}{
		{"NewClassSet Error", func() { NewClassSet(hostOnly("MyError", "Error")) }},
		{"With Error", func() { BuiltinClasses.With(hostOnly("MyError", "Error")) }},
		{"With NotFoundError", func() { BuiltinClasses.With(hostOnly("MyNotFound", "NotFoundError")) }},
		{"With Reader", func() { BuiltinClasses.With(hostOnly("MyReader", "Reader")) }},
		{"NewClassSet Reader", func() { NewClassSet(hostOnly("MyReader", "Reader")) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				e, ok := r.(error)
				if !ok || !strings.Contains(e.Error(), "cannot extend builtin class") {
					t.Fatalf("panic = %v", r)
				}
			}()
			tc.fn()
		})
	}
}

func TestDetachLimitedReaderClosesOwner(t *testing.T) {
	for _, detach := range []bool{false, true} {
		t.Run(strconv.FormatBool(detach), func(t *testing.T) {
			vm := newPoolTestVM(t)
			probe := &closeProbe{Reader: strings.NewReader("abcdef")}
			mustDefineGlobal(t, vm, "response", newHttpResponse(vm, &http.Response{Body: probe, Header: http.Header{}}))
			var reader io.ReadCloser
			e := vm.Do(context.Background(), func() error {
				v, e := vm.Run(context.Background(), `response.body.limitReader(4).limitReader(2)`, "")
				if e != nil {
					return e
				}
				if detach {
					reader = DetachReader(vm, v.Raw(), "")
				}
				return nil
			})
			if e != nil {
				t.Fatal(e)
			}
			if e := vm.Dispose(); e != nil {
				t.Fatal(e)
			}
			if !detach {
				if !probe.closed {
					t.Fatal("VM did not close owner")
				}
				return
			}
			if probe.closed {
				t.Fatal("owner closed before caller finished")
			}
			b, e := io.ReadAll(reader)
			if e != nil || string(b) != "ab" {
				t.Fatalf("body=%q, %v", b, e)
			}
			if e := reader.Close(); e != nil {
				t.Fatal(e)
			}
			if !probe.closed {
				t.Fatal("caller Close did not close owner")
			}
		})
	}
}

func TestDetachLimitedTempFileRemovesFileOnClose(t *testing.T) {
	vm := newPoolTestVM(t)
	var reader io.ReadCloser
	var name string
	e := vm.Do(context.Background(), func() error {
		v, e := vm.Run(context.Background(), `var file=new TempFile();file.write(Bytes.fromString("abc"));file.seekTo(0,SEEK_START);file.limitReader(2)`, "")
		if e != nil {
			return e
		}
		f, _ := vm.GetValue("file")
		name = f.Raw().(*jsObjTempFile).f.Name()
		reader = DetachReader(vm, v.Raw(), "")
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	defer reader.Close()
	if e := vm.Dispose(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(name); e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(reader)
	if e != nil || string(b) != "ab" {
		t.Fatalf("body=%q, %v", b, e)
	}
	if e := reader.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(name); !os.IsNotExist(e) {
		t.Fatalf("temporary file remains: %v", e)
	}
}
