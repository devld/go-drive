package script

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	err "go-drive/common/errors"
	"go-drive/common/types"

	"github.com/dop251/goja"
)

func TestRunUsesScriptNameInStack(t *testing.T) {
	vm := newPoolTestVM(t)

	_, e := vm.Run(context.Background(), `
function helperBoom() {
	throw new Error("from helper");
}
`, "helper.js")
	if e != nil {
		t.Fatal(e)
	}

	_, e = vm.Run(context.Background(), `
function driveBoom() {
	helperBoom();
}
`, "github.js")
	if e != nil {
		t.Fatal(e)
	}

	_, e = vm.Call(context.Background(), "driveBoom")
	if e == nil {
		t.Fatal("expected error")
	}

	stack := runtimeStack(e)
	if !strings.Contains(stack, "github.js") {
		t.Fatalf("stack missing drive script name:\n%s", stack)
	}
	if !strings.Contains(stack, "helper.js") {
		t.Fatalf("stack missing helper name:\n%s", stack)
	}
	if strings.Contains(stack, "helperBoom (<anonymous>") || strings.Contains(stack, "driveBoom (<anonymous>") {
		t.Fatalf("named functions still attributed to anonymous:\n%s", stack)
	}
}

func TestRunSyntaxErrorUsesScriptName(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `function {`, "broken.js")
	if e == nil {
		t.Fatal("expected syntax error")
	}
	msg := e.Error()
	if !strings.Contains(msg, "broken.js") {
		t.Fatalf("syntax error missing script name: %v", e)
	}
}

func TestPromiseValuesAreRejectedAndMakeVMUnreusable(t *testing.T) {
	vm := newPoolTestVM(t)
	if _, e := vm.Run(context.Background(), `
		var resolved = Promise.resolve(1);
		var asyncValue = async function () { return await resolved; };
		typeof resolved + ":" + typeof asyncValue;
	`, ""); e != nil {
		t.Fatal(e)
	}

	if _, e := vm.Run(context.Background(), `Promise.resolve(1)`, ""); e == nil || !strings.Contains(e.Error(), "Promise return values are not supported") {
		t.Fatalf("Promise.resolve() error = %v", e)
	}
	if vm.Reusable() {
		t.Fatal("VM remained reusable after returning a Promise")
	}

	vm, e := NewVM()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e = vm.Run(context.Background(), `function pending() { return new Promise(function () {}); }`, ""); e != nil {
		t.Fatal(e)
	}
	if _, e = vm.Call(context.Background(), "pending"); e == nil || !strings.Contains(e.Error(), "Promise return values are not supported") {
		t.Fatalf("pending() error = %v", e)
	}
	if vm.Reusable() {
		t.Fatal("VM remained reusable after Call returned a Promise")
	}

	vm, e = NewVM()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e = vm.Run(context.Background(), `async function asyncResult() { return 1; }`, ""); e != nil {
		t.Fatal(e)
	}
	if _, e = vm.Call(context.Background(), "asyncResult"); e == nil || !strings.Contains(e.Error(), "Promise return values are not supported") {
		t.Fatalf("asyncResult() error = %v", e)
	}
}

func TestJavaScriptValuesCannotCrossRuntimeBoundaries(t *testing.T) {
	first := newPoolTestVM(t)
	second := newPoolTestVM(t)
	value, e := first.Run(context.Background(), `({ owner: "first" })`, "")
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "cannot cross Runtime boundaries") {
			t.Fatalf("cross-Runtime DefineGlobal panic = %v", recovered)
		}
	}()
	_ = second.DefineGlobal("foreignValue", value)
}

func TestGojaCoreBridgeGlobalsAreScoped(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		[
			typeof __goDrive_bridge__,
			typeof __urlParse__,
			typeof __encToHex__,
			typeof __newHash__,
			typeof consoleWrite,
			typeof __consoleWrite__,
			typeof __newGoError__,
			typeof __isTypeOfErr__,
			typeof urlUtils.parse,
			typeof Bytes.fromHex,
			typeof console.log,
			Bytes.fromString("x").toString("hex")
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	const want = "undefined|undefined|undefined|undefined|undefined|undefined|undefined|undefined|function|function|function|78"
	if got.String() != want {
		t.Fatalf("bridge globals = %q, want %q", got.String(), want)
	}
}

func TestPublicEnvironmentIsImmutable(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		var originalHTTP = http;
		var originalParse = urlUtils.parse;
		http = function () {};
		urlUtils.parse = function () {};
		urlUtils.extra = true;
		Hash.MD5 = 99;
		delete console.log;
		[
			http === originalHTTP,
			urlUtils.parse === originalParse,
			typeof urlUtils.extra,
			typeof Hash.MD5,
			typeof console.log,
			typeof newContext,
			typeof newBytes,
			typeof newFormData,
			typeof encUtils,
			Object.isFrozen(http),
			Object.isFrozen(console),
			Object.isFrozen(pathUtils),
			Object.isFrozen(urlUtils),
			Object.isFrozen(Hash),
			Object.isFrozen(Bytes),
			Object.isFrozen(Bytes.prototype),
			Object.isFrozen(HttpFormData),
			Object.isFrozen(dayjs),
			Object.isFrozen(NotFoundError),
			typeof NotFoundError
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	const want = "true|true|undefined|undefined|function|undefined|undefined|undefined|undefined|true|false|true|false|true|true|true|true|true|true|function"
	if got.String() != want {
		t.Fatalf("immutable environment = %q, want %q", got.String(), want)
	}
}

func TestDefineGlobalHostObjectIsReadOnly(t *testing.T) {
	vm := newPoolTestVM(t)
	if e := vm.DefineGlobal("drive", notFoundDrive{}); e != nil {
		t.Fatal(e)
	}
	got, e := vm.Run(context.Background(), `
		[
			typeof drive.get,
			drive instanceof Drive,
			drive.get === Drive.prototype.get,
			Object.prototype.toString.call(drive)
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "function|true|true|[object Drive]" {
		t.Fatalf("Drive host = %q", got.String())
	}
}

func TestDayjsAcceptsTime(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "goTime", time.UnixMilli(1_700_000_000_000).UTC())
	got, e := vm.Run(context.Background(), `String(dayjs(goTime).valueOf())`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "1700000000000" {
		t.Fatalf("dayjs(Date) = %q", got.String())
	}
}

func TestToJSValueWrapsTimeAsDate(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "goTime", time.UnixMilli(1_700_000_000_000).UTC())
	got, e := vm.Run(context.Background(), `
		goTime.extra = 1;
		[goTime instanceof Date, String(goTime.getTime()), String(goTime.extra)].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|1700000000000|1" {
		t.Fatalf("time.Time as Date = %q", got.String())
	}
}

func TestPathUtilsNormalizeScriptPaths(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		[
			pathUtils.clean("/a/./b/../c"),
			pathUtils.join("a", "", "b", "c"),
			pathUtils.parent("a/b/c"),
			pathUtils.base("a/b/c.txt"),
			pathUtils.ext("a/b/C.TXT"),
			String(pathUtils.isRoot("")),
			String(pathUtils.isRoot("a"))
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "a/c|a/b/c|a/b|c.txt|txt|true|false" {
		t.Fatalf("pathUtils = %q", got.String())
	}
}

func TestToJSValueReadonlyMaps(t *testing.T) {
	t.Run("types.M", func(t *testing.T) {
		vm := newPoolTestVM(t)
		original := types.M{"a": 1, "nested": types.M{"b": "x"}}
		mustDefineGlobal(t, vm, "obj", original)
		got, e := vm.Run(context.Background(), `
			const isPlain = obj instanceof Object && obj.constructor === Object;
			obj.a = 2;
			obj.nested.b = "y";
			obj.extra = true;
			[isPlain, typeof obj.a, typeof obj.extra, obj.nested === obj.nested].join("|") + "\n" + JSON.stringify(obj);
		`, "")
		if e != nil {
			t.Fatal(e)
		}
		s := got.String()
		head, body, ok := strings.Cut(s, "\n")
		if !ok || head != "true|number|undefined|true" || !jsonEqual(body, `{"a":1,"nested":{"b":"x"}}`) {
			t.Fatalf("ToJSValue = %q", s)
		}
		if original["a"] != 1 {
			t.Fatalf("Go map mutated: %#v", original["a"])
		}
		if original["nested"].(types.M)["b"] != "x" {
			t.Fatalf("nested Go map mutated: %#v", original)
		}
	})
	t.Run("map[string]any", func(t *testing.T) {
		vm := newPoolTestVM(t)
		original := map[string]any{"a": 1, "nested": map[string]any{"b": "x"}}
		mustDefineGlobal(t, vm, "obj", original)
		got, e := vm.Run(context.Background(), `
			obj.a = 2;
			obj.nested.b = "y";
			JSON.stringify(obj);
		`, "")
		if e != nil {
			t.Fatal(e)
		}
		if !jsonEqual(got.String(), `{"a":1,"nested":{"b":"x"}}`) {
			t.Fatalf("ToJSValue = %q", got.String())
		}
		if original["a"] != 1 || original["nested"].(map[string]any)["b"] != "x" {
			t.Fatalf("Go map mutated: %#v", original)
		}
	})
	t.Run("types.SM", func(t *testing.T) {
		vm := newPoolTestVM(t)
		original := types.SM{"id": "orig"}
		mustDefineGlobal(t, vm, "obj", original)
		got, e := vm.Run(context.Background(), `
			obj.id = "mutated";
			obj.extra = true;
			[obj.id, String(obj.extra)].join("|");
		`, "")
		if e != nil {
			t.Fatal(e)
		}
		if got.String() != "orig|undefined" {
			t.Fatalf("readonly writes = %q", got.String())
		}
		if original["id"] != "orig" {
			t.Fatalf("Go map mutated: %#v", original)
		}
		if _, ok := original["extra"]; ok {
			t.Fatalf("Go map gained extra: %#v", original)
		}
	})
	t.Run("assigned property", func(t *testing.T) {
		vm := newPoolTestVM(t)
		original := types.M{}
		mustDefineGlobal(t, vm, "obj", original)
		got, e := vm.Run(context.Background(), `
			obj.nested = { x: 1 };
			typeof obj.nested;
		`, "")
		if e != nil {
			t.Fatal(e)
		}
		if got.String() != "undefined" {
			t.Fatalf("assigned object = %q", got.String())
		}
		if _, ok := original["nested"]; ok {
			t.Fatalf("Go map mutated: %#v", original)
		}
	})
}

func TestToJSValueInternsCycles(t *testing.T) {
	vm := newPoolTestVM(t)
	original := types.M{}
	original["self"] = original
	mustDefineGlobal(t, vm, "obj", original)
	got, e := vm.Run(context.Background(), `String(obj.self === obj)`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true" {
		t.Fatalf("cycle intern = %q", got.String())
	}
}

func TestToJSValueInternsSharedNestedMapsInOneView(t *testing.T) {
	vm := newPoolTestVM(t)
	child := types.M{"x": 1}
	original := types.M{"a": child, "b": child}
	mustDefineGlobal(t, vm, "obj", original)
	got, e := vm.Run(context.Background(), `String(obj.a === obj.b && obj.a === obj.a)`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true" {
		t.Fatalf("nested intern = %q", got.String())
	}
}

func TestToJSValueDoesNotInternAcrossViews(t *testing.T) {
	vm := newPoolTestVM(t)
	original := types.M{"a": 1}
	mustDefineGlobal(t, vm, "a", original)
	mustDefineGlobal(t, vm, "b", original)
	got, e := vm.Run(context.Background(), `String(a === b)`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "false" {
		t.Fatalf("cross-view intern = %q", got.String())
	}
}

func TestToJSValueInternsSlicesByPointerAndLength(t *testing.T) {
	vm := newPoolTestVM(t)
	a := []int{1, 2, 3}
	mustDefineGlobal(t, vm, "arrs", [][]int{a[:1], a[:3]})
	got, e := vm.Run(context.Background(), `[arrs[0].length, arrs[1].length].join(",")`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "1,3" {
		t.Fatalf("slice intern lengths = %q", got.String())
	}

	same := []int{4, 5, 6}
	mustDefineGlobal(t, vm, "shared", struct{ A, B []int }{same, same})
	got, e = vm.Run(context.Background(), `String(shared.a === shared.b)`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true" {
		t.Fatalf("same-slice intern = %q", got.String())
	}
}

func TestToJSValuePanicsOnNonNativeFunction(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*testing.T, *VM)
	}{
		{"func", func(t *testing.T, vm *VM) {
			_ = vm.ToJSValue(func() int { return 1 })
		}},
		{"nested", func(t *testing.T, vm *VM) {
			mustDefineGlobal(t, vm, "obj", map[string]any{"f": func() {}})
			_, _ = vm.Run(context.Background(), `obj.f`, "")
		}},
		{"methods", func(t *testing.T, vm *VM) {
			_ = vm.ToJSValue(&methodfulFixture{Name: "x", OAuth: "tok"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm := newPoolTestVM(t)
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic")
				}
				if got := fmt.Sprint(r); !strings.Contains(got, "NativeFunction") {
					t.Fatalf("panic = %v", r)
				}
			}()
			tc.run(t, vm)
		})
	}
}

func TestToJSValueAcceptsUnnamedNativeFunctionSignature(t *testing.T) {
	vm := newPoolTestVM(t)
	fn := func(_ *VM, _ Values) any { return "ok" }
	mustDefineGlobal(t, vm, "fn", fn)
	got, e := vm.Run(context.Background(), `fn()`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "ok" {
		t.Fatalf("unnamed NativeFunction = %q", got.String())
	}
}

type methodfulFixture struct {
	Name  string
	OAuth string
}

func (m *methodfulFixture) Hello() string { return "hi " + m.Name }

func TestToJSValueBindsNativeFunctionMethods(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "api", &nativeMethodAPI{N: 2})
	got, e := vm.Run(context.Background(), `String(api.add(3))`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "5" {
		t.Fatalf("native method = %q", got.String())
	}
}

type nativeMethodAPI struct {
	N int
}

func (a *nativeMethodAPI) Add(_ *VM, args Values) any {
	return a.N + int(args.Get(0).Integer())
}

func TestToJSValueWrapsSMInsideStruct(t *testing.T) {
	vm := newPoolTestVM(t)
	cfg := &struct {
		Configured bool
		OAuth      *struct {
			URL       string `json:"url"`
			Principal string
		} `json:"oauth"`
		Value types.SM
	}{
		Configured: true,
		OAuth: &struct {
			URL       string `json:"url"`
			Principal string
		}{URL: "https://example.com", Principal: ""},
		Value: types.SM{"drive_id": "abc"},
	}
	mustDefineGlobal(t, vm, "cfg", cfg)
	got, e := vm.Run(context.Background(), `
		cfg.value.drive_id = cfg.value.drive_id + "-x";
		cfg.configured = false;
		cfg.oauth.principal = "user";
		[cfg.value.drive_id, String(cfg.configured), cfg.oauth.principal, cfg.oauth.url, String(cfg.oauth === cfg.oauth)].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "abc|true||https://example.com|true" {
		t.Fatalf("struct wrap = %q", got.String())
	}
	if cfg.Value["drive_id"] != "abc" {
		t.Fatalf("SM write-through = %#v", cfg.Value)
	}
	if !cfg.Configured {
		t.Fatal("struct field write-through")
	}
	if cfg.OAuth.Principal != "" {
		t.Fatalf("nested field write-through = %#v", cfg.OAuth)
	}
}

func TestFromJSValueDeepCopiesJSONValue(t *testing.T) {
	vm := newPoolTestVM(t)
	if _, e := vm.Run(context.Background(), `var src = {count: 1, values: ["a"]}`, ""); e != nil {
		t.Fatal(e)
	}
	src, e := vm.GetValue("src")
	if e != nil {
		t.Fatal(e)
	}
	cloned, e := vm.FromJSValue(src)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), `src.count = 2; src.values.push("b")`, ""); e != nil {
		t.Fatal(e)
	}
	m, ok := cloned.(map[string]any)
	if !ok {
		t.Fatalf("FromJSValue type = %T", cloned)
	}
	if m["count"] != float64(1) {
		t.Fatalf("count = %#v", m["count"])
	}
	values, ok := m["values"].([]any)
	if !ok || len(values) != 1 || values[0] != "a" {
		t.Fatalf("values = %#v", m["values"])
	}
	if _, e := vm.FromJSValue(mustGetValue(t, vm, `(function () {})`)); e == nil {
		t.Fatal("expected function to be rejected")
	}
}

func mustGetValue(t *testing.T, vm *VM, expr string) *Value {
	t.Helper()
	got, e := vm.Run(context.Background(), expr, "")
	if e != nil {
		t.Fatal(e)
	}
	return got
}

func TestErrorStringProtocolIsNotRecognized(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `throw new Error("E:NOT_FOUND:0:missing")`, "")
	if e == nil {
		t.Fatal("expected JavaScript exception")
	}
	if err.IsNotFoundError(e) {
		t.Fatalf("string protocol unexpectedly mapped to Go error: %v", e)
	}
}

func TestThrownGetterIsMappedWithoutEscaping(t *testing.T) {
	vm := newPoolTestVM(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("VM.Run panicked: %#v", r)
		}
	}()
	_, e := vm.Run(context.Background(), `throw { get value() { throw new Error("getter failed") } }`, "")
	if e == nil {
		t.Fatal("expected JavaScript exception")
	}
	var exception *goja.Exception
	if errors.As(e, &exception) {
		t.Fatalf("returned *goja.Exception: %T %v", e, e)
	}
	_ = e.Error()
}

func TestThrownProxyDoesNotEscapeRun(t *testing.T) {
	vm := newPoolTestVM(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("VM.Run panicked: %#v", r)
		}
	}()
	_, e := vm.Run(context.Background(), `
		var p = new Proxy({}, {
			get: function() { throw p; },
			getPrototypeOf: function() { throw new Error("prototype failed"); }
		});
		throw p;
	`, "")
	if e == nil {
		t.Fatal("expected JavaScript exception")
	}
	var exception *goja.Exception
	if errors.As(e, &exception) {
		t.Fatalf("returned *goja.Exception: %T %v", e, e)
	}
	_ = e.Error()
}

func TestReturnedExceptionErrorDoesNotUseVM(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `
		var n = 0;
		throw { toString: function() { n++; return "counted"; } };
	`, "")
	if e == nil {
		t.Fatal("expected JavaScript exception")
	}
	n, getErr := vm.GetValue("n")
	if getErr != nil {
		t.Fatal(getErr)
	}
	afterConvert := n.Integer()
	if afterConvert < 1 {
		t.Fatalf("toString was not used during conversion, n=%d", afterConvert)
	}
	_ = e.Error()
	_ = e.Error()
	n, getErr = vm.GetValue("n")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if n.Integer() != afterConvert {
		t.Fatalf("Error() re-ran toString: n=%d after conversion %d", n.Integer(), afterConvert)
	}
	if e := vm.Dispose(); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(e.Error(), "counted") {
		t.Fatalf("Error after Dispose = %v", e)
	}
}

func TestToJSValueArrayElementsHaveStableIdentity(t *testing.T) {
	vm := newPoolTestVM(t)
	type entry struct {
		Name string
		Size int64
	}
	mustDefineGlobal(t, vm, "entries", []entry{{Name: "a", Size: 1}, {Name: "b", Size: 2}})
	got, e := vm.Run(context.Background(), `[
		String(entries[0] === entries[0]),
		String(entries.indexOf(entries[0])),
		String(entries[0] === entries[1]),
		entries[0].name
	].join("|")`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|0|false|a" {
		t.Fatalf("array identity = %q", got.String())
	}
}

func TestDoRejectsCanceledContextBeforeCallback(t *testing.T) {
	vm := newPoolTestVM(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	for _, ctx := range []context.Context{canceled, expired} {
		called := false
		e := vm.Do(ctx, func() error {
			called = true
			return nil
		})
		if !errors.Is(e, ctx.Err()) || called {
			t.Fatalf("Do: error=%v, called=%v; want %v without callback", e, called, ctx.Err())
		}
	}
	if _, e := vm.Run(context.Background(), `1 + 1`, ""); e != nil {
		t.Fatalf("VM reuse after rejected context: %v", e)
	}
}

func TestNestedDoKeepsOuterContext(t *testing.T) {
	vm := newPoolTestVM(t)
	outer := context.Background()
	inner, cancel := context.WithCancel(outer)
	cancel()
	called := false
	e := vm.Do(outer, func() error {
		return vm.Do(inner, func() error {
			called = true
			if vm.ExecutionContext() != outer {
				t.Fatal("nested Do replaced the outer context")
			}
			return nil
		})
	})
	if e != nil || !called {
		t.Fatalf("nested Do: error=%v, called=%v", e, called)
	}
}

func TestCallIsInterruptedByCallerContext(t *testing.T) {
	vm := newPoolTestVM(t)
	if _, e := vm.Run(context.Background(), `
		function spin() { while (true) {} }
		function ok() { return 1; }
		Object.defineProperty(globalThis, "boom", {
			get: function() { throw new Error("getter failed"); }
		});
	`, ""); e != nil {
		t.Fatal(e)
	}

	if _, e := vm.Call(context.Background(), "boom"); e == nil {
		t.Fatal("expected getter exception")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, e := vm.Call(ctx, "spin")
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("spin() error = %v, want deadline exceeded", e)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("spin() took %s, want interrupt promptly", elapsed)
	}

	canceled, cancelAlready := context.WithCancel(context.Background())
	cancelAlready()
	_, e = vm.Call(canceled, "spin")
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("already-canceled Call error = %v, want canceled", e)
	}

	got, e := vm.Call(context.Background(), "ok")
	if e != nil {
		t.Fatal(e)
	}
	if got.Integer() != 1 {
		t.Fatalf("ok() = %v, want 1", got.Integer())
	}
}

func TestUnrelatedPanicIsNotMappedToScriptError(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "boom", NativeFunction(func(_ *VM, _ Values) any {
		panic("native bug")
	}))
	defer func() {
		r := recover()
		if r != "native bug" {
			t.Fatalf("recover = %#v", r)
		}
	}()
	if _, e := vm.Run(context.Background(), `boom()`, ""); e != nil {
		t.Fatalf("got error %v, want panic", e)
	}
	t.Fatal("expected panic")
}

func runtimeStack(e error) string {
	if s, ok := e.(interface{ String() string }); ok {
		return s.String()
	}
	return e.Error()
}

func jsonEqual(got, want string) bool {
	var g, w any
	if json.Unmarshal([]byte(got), &g) != nil || json.Unmarshal([]byte(want), &w) != nil {
		return got == want
	}
	gb, e1 := json.Marshal(g)
	wb, e2 := json.Marshal(w)
	return e1 == nil && e2 == nil && string(gb) == string(wb)
}
