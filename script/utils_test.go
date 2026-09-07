package script

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-drive/common/types"
)

func TestFormatConsoleArgSerializesObjects(t *testing.T) {
	vm := newPoolTestVM(t)

	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "string", code: `"hello"`, want: "hello"},
		{name: "number", code: `42`, want: "42"},
		{name: "boolean", code: `true`, want: "true"},
		{name: "null", code: `null`, want: "null"},
		{name: "undefined", code: `undefined`, want: "undefined"},
		{name: "object", code: `({a: 1, b: "x"})`, want: `{"a":1,"b":"x"}`},
		{name: "array", code: `[1, {a: 2}, "x"]`, want: `[ 1, {"a":2}, x ]`},
		{name: "nested", code: `({a: {b: [1, 2]}})`, want: `{"a":{"b":[1,2]}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalConsoleArg(t, vm, tt.code)
			if got != tt.want {
				t.Fatalf("formatConsoleArg(%s) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestFormatConsoleArgFallsBackForCircular(t *testing.T) {
	vm := newPoolTestVM(t)
	got := evalConsoleArg(t, vm, `(function() { var o = {}; o.self = o; return o; })()`)
	if got != "[object Object]" {
		t.Fatalf("circular object = %q, want [object Object]", got)
	}
}

func TestFormatConsoleArgTruncatesCircularArray(t *testing.T) {
	vm := newPoolTestVM(t)
	got := evalConsoleArg(t, vm, `(function() { var a = []; a.push(a); return a; })()`)
	if got != "[ [ ... ] ]" {
		t.Fatalf("circular array = %q, want [ [ ... ] ]", got)
	}
}

func TestFormatConsoleArgTruncatesLongArray(t *testing.T) {
	vm := newPoolTestVM(t)

	got100 := evalConsoleArg(t, vm, `(function() {
		var a = [];
		for (var i = 0; i < 100; i++) a.push(i);
		return a;
	})()`)
	if strings.Contains(got100, "more items") {
		t.Fatalf("100-item array should not truncate: %s", got100)
	}
	if !strings.HasPrefix(got100, "[ 0, ") || !strings.HasSuffix(got100, ", 99 ]") {
		t.Fatalf("100-item array = %s", got100)
	}

	got101 := evalConsoleArg(t, vm, `(function() {
		var a = [];
		for (var i = 0; i < 101; i++) a.push(i);
		return a;
	})()`)
	if !strings.HasSuffix(got101, ", 99, ... 1 more items ]") {
		t.Fatalf("101-item array = %s, want 100 items then remainder", got101)
	}
}

func TestFormatConsoleArgKeepsDateReadable(t *testing.T) {
	vm := newPoolTestVM(t)

	date := evalConsoleArg(t, vm, `new Date(0)`)
	if strings.HasPrefix(date, `"`) {
		t.Fatalf("Date should not be JSON-quoted, got %q", date)
	}
}

func TestFormatConsoleArgIncludesErrorStack(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `
function boom() {
	return new Error("boom");
}
`, "err.js")
	if e != nil {
		t.Fatal(e)
	}

	errMsg := evalConsoleArg(t, vm, `boom()`)
	if !strings.Contains(errMsg, "boom") {
		t.Fatalf("Error = %q, want message", errMsg)
	}
	if !strings.Contains(errMsg, "at boom (err.js:") {
		t.Fatalf("Error = %q, want stack with script name", errMsg)
	}
}

func TestFormatConsoleArgsJoinsValues(t *testing.T) {
	root := newPoolTestVM(t)
	vm := root
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	mustDefineGlobal(t, vm, "capture", NativeFunction(func(_ *VM, args Values) any {
		got = FormatConsoleArgs(args)
		return nil
	}))
	if _, e := vm.Run(context.Background(), `capture("hello", {a: 1}, null)`, ""); e != nil {
		t.Fatal(e)
	}
	if got != `hello {"a":1} null` {
		t.Fatalf("FormatConsoleArgs = %q, want joined console output", got)
	}
}

func evalConsoleArg(t *testing.T, vm *VM, code string) string {
	t.Helper()
	value, e := vm.Run(context.Background(), code, "")
	if e != nil {
		t.Fatal(e)
	}
	return formatConsoleArg(value)
}

func TestGetDurationStringAndMs(t *testing.T) {
	vm := newScriptTestVM(t)

	must := func(src string, want time.Duration) {
		t.Helper()
		v, e := vm.Run(context.Background(), src, "")
		if e != nil {
			t.Fatal(e)
		}
		d := GetDuration(vm, v, "test requires a Duration or duration string")
		if d != want {
			t.Fatalf("%s = %v want %v", src, d, want)
		}
	}
	must(`"2s"`, 2*time.Second)
	must(`ms(1500)`, 1500*time.Millisecond)
	must(`"1h30m"`, 90*time.Minute)
	must(`"2d3h4m5s"`, 2*24*time.Hour+3*time.Hour+4*time.Minute+5*time.Second)
	must(`""`, 0)

	v, e := vm.Run(context.Background(), `parseDuration("2s")`, "")
	if e != nil {
		t.Fatal(e)
	}
	if time.Duration(v.Integer()) != 2*time.Second {
		t.Fatalf("parseDuration(2s) = %d", v.Integer())
	}

	v, e = vm.Run(context.Background(), `parseDuration("")`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.Integer() != 0 {
		t.Fatalf("parseDuration empty = %d", v.Integer())
	}

	if _, e := vm.Run(context.Background(), `parseDuration("nope")`, ""); e == nil {
		t.Fatal("expected parseDuration(\"nope\") to fail")
	}

	v, e = vm.Run(context.Background(), `"nope"`, "")
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected TypeError for invalid duration")
		}
	}()
	GetDuration(vm, v, "test requires a Duration or duration string")
}

func TestValueArrayRequiresArrayClass(t *testing.T) {
	vm := newPoolTestVM(t)

	obj, e := vm.Run(context.Background(), `({length: 2, 0: "a", 1: "b"})`, "")
	if e != nil {
		t.Fatal(e)
	}
	if obj.Array() != nil {
		t.Fatal("plain object must not be treated as an array")
	}

	arr, e := vm.Run(context.Background(), `["a", "b"]`, "")
	if e != nil {
		t.Fatal(e)
	}
	got := arr.Array()
	if len(got) != 2 || got[0].String() != "a" || got[1].String() != "b" {
		t.Fatalf("array = %v", got)
	}
}

func TestParseIntoJSONTagAndInterfaceMap(t *testing.T) {
	vm := newPoolTestVM(t)
	v, e := vm.Run(context.Background(), `({
		oauth: { url: "https://example.com", principal: "user" },
		props: { kind: "doc", n: 1 }
	})`, "")
	if e != nil {
		t.Fatal(e)
	}

	var parsed struct {
		OAuth *struct {
			URL       string `json:"url"`
			Principal string
		} `json:"oauth"`
		Props types.M
	}
	if e := v.ParseInto(&parsed); e != nil {
		t.Fatal(e)
	}
	if parsed.OAuth == nil || parsed.OAuth.URL != "https://example.com" || parsed.OAuth.Principal != "user" {
		t.Fatalf("oauth = %#v", parsed.OAuth)
	}
	if parsed.Props["kind"] != "doc" {
		t.Fatalf("props.kind = %#v", parsed.Props["kind"])
	}
	switch n := parsed.Props["n"].(type) {
	case int64:
		if n != 1 {
			t.Fatalf("props.n = %d", n)
		}
	case float64:
		if n != 1 {
			t.Fatalf("props.n = %v", n)
		}
	default:
		t.Fatalf("props.n type = %T", parsed.Props["n"])
	}
}

func TestParseThrowsInHostFunction(t *testing.T) {
	vm := newPoolTestVM(t)
	type payload struct {
		Style int
	}
	mustDefineGlobal(t, vm, "read", NativeFunction(func(_ *VM, args Values) any {
		return Parse[payload](args.Get(0)).Style
	}))
	got, e := vm.Run(context.Background(), `String(read({ style: 2 }))`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "2" {
		t.Fatalf("Parse = %q", got.String())
	}
}

func TestParseIntoDetachesNestedProxy(t *testing.T) {
	vm := newPoolTestVM(t)
	v, e := vm.Run(context.Background(), `({
		props: { nested: new Proxy({ kind: "doc" }, {}) }
	})`, "")
	if e != nil {
		t.Fatal(e)
	}

	var parsed struct {
		Props types.M
	}
	if e := v.ParseInto(&parsed); e != nil {
		t.Fatal(e)
	}
	nested, ok := parsed.Props["nested"].(types.M)
	if !ok {
		t.Fatalf("nested type = %T", parsed.Props["nested"])
	}
	if nested["kind"] != "doc" {
		t.Fatalf("nested.kind = %#v", nested["kind"])
	}
}

func TestParseIntoDoesNotRerunNestedGetters(t *testing.T) {
	vm := newScriptTestVM(t)
	v, e := vm.Run(context.Background(), `
		var f = new TempFile();
		f.write(Bytes.fromString("payload"));
		f.seekTo(0, SEEK_START);
		var n = 0;
		({
			props: {
				nested: {
					get payload() { n++; return f.readAsString(); }
				}
			}
		})
	`, "")
	if e != nil {
		t.Fatal(e)
	}

	var parsed struct {
		Props types.M
	}
	if e := v.ParseInto(&parsed); e != nil {
		t.Fatal(e)
	}
	nested, ok := parsed.Props["nested"].(types.M)
	if !ok {
		t.Fatalf("nested type = %T", parsed.Props["nested"])
	}
	if nested["payload"] != "payload" {
		t.Fatalf("payload = %#v", nested["payload"])
	}
	got, e := vm.Run(context.Background(), `n`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.Integer() != 1 {
		t.Fatalf("getter ran %d times", got.Integer())
	}
}

func TestParseIntoDetachesErrorCustomProps(t *testing.T) {
	vm := newPoolTestVM(t)
	v, e := vm.Run(context.Background(), `({
		props: {
			err: Object.assign(new Error("oops"), {
				nested: new Proxy({ kind: "doc" }, {})
			})
		}
	})`, "")
	if e != nil {
		t.Fatal(e)
	}

	var parsed struct {
		Props types.M
	}
	if e := v.ParseInto(&parsed); e != nil {
		t.Fatal(e)
	}
	errObj, ok := parsed.Props["err"].(types.M)
	if !ok {
		t.Fatalf("err type = %T", parsed.Props["err"])
	}
	if errObj["message"] != "oops" {
		t.Fatalf("err.message = %#v", errObj["message"])
	}
	nested, ok := errObj["nested"].(types.M)
	if !ok {
		t.Fatalf("nested type = %T", errObj["nested"])
	}
	if nested["kind"] != "doc" {
		t.Fatalf("nested.kind = %#v", nested["kind"])
	}

	vm2 := newPoolTestVM(t)
	if e := vm2.DefineGlobal("err", parsed.Props["err"]); e != nil {
		t.Fatal(e)
	}
	got, e := vm2.Run(context.Background(), `err.nested.kind`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "doc" {
		t.Fatalf("cross-VM kind = %q", got.String())
	}
}

func TestFinalizeScriptErrorDoesNotUnwrapGenericErrors(t *testing.T) {
	inner := errors.New("inner")
	wrapped := fmt.Errorf("outer: %w", inner)
	if got := finalizeScriptError(nil, wrapped, nil); got != wrapped {
		t.Fatalf("finalizeScriptError unwrapped generic error: %v", got)
	}
}

func TestValueMissingPropertyAccessors(t *testing.T) {
	vm := newPoolTestVM(t)
	obj, e := vm.Run(context.Background(), `({path: "file"})`, "")
	if e != nil {
		t.Fatal(e)
	}
	if obj.Get("isDir").Bool() {
		t.Fatal("missing isDir should be false")
	}
	if obj.Get("size").Integer() != 0 {
		t.Fatalf("missing size = %d", obj.Get("size").Integer())
	}
	if obj.Get("missing").Float() != 0 {
		t.Fatalf("missing float = %v", obj.Get("missing").Float())
	}
	if !obj.Get("nope").IsUndefined() {
		t.Fatal("missing property should be undefined")
	}
}

func TestStringMapHandlesDisappearingKeys(t *testing.T) {
	for _, source := range []string{
		`({get A(){delete this.B;return "a"},B:"b"})`,
		`new Proxy({}, {ownKeys(){return ["B"]},getOwnPropertyDescriptor(){return {enumerable:true,configurable:true}}})`,
	} {
		t.Run(source, func(t *testing.T) {
			vm := newScriptTestVM(t)
			var got types.SM
			e := vm.Do(context.Background(), func() error {
				v, e := vm.Run(context.Background(), source, "")
				if e != nil {
					return e
				}
				got = v.SM()
				return nil
			})
			if e != nil {
				t.Fatal(e)
			}
			if _, ok := got["B"]; ok {
				t.Fatalf("deleted key retained: %#v", got)
			}
		})
	}
}

func TestProxyArraysPreserveShape(t *testing.T) {
	vm := newPoolTestVM(t)
	e := vm.Do(context.Background(), func() error {
		v, e := vm.Run(context.Background(), `new Proxy(["x", "y"], {})`, "")
		if e != nil {
			return e
		}
		if !v.IsArray() || len(v.Array()) != 2 {
			t.Fatal("proxy array rejected")
		}
		var array []string
		if e := v.ParseInto(&array); e != nil {
			return e
		}
		if len(array) != 2 || array[1] != "y" {
			t.Fatalf("array=%v", array)
		}
		var detached any
		if e := v.ParseInto(&detached); e != nil {
			return e
		}
		if _, ok := detached.([]any); !ok {
			t.Fatalf("detached array is %T", detached)
		}
		_, e = vm.Run(context.Background(), `
     if (urlUtils.buildSearchParams({a: new Proxy(["x", "y"], {})}) !== "?a=x&a=y")
       throw new Error("proxy query values lost");
     for (const f of [urlUtils.build, urlUtils.buildSearchParams]) {
       let rejected = false;
       try { f(new Proxy([], {})); } catch (e) { rejected = e instanceof TypeError; }
       if (!rejected) throw new Error("array accepted as object");
     }
     Array.isArray = () => false;
   `, "")
		if e != nil {
			return e
		}
		if !v.IsArray() {
			t.Fatal("array intrinsic was overwritten")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
