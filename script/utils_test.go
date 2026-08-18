package script

import (
	"context"
	"strings"
	"testing"
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
		{name: "array", code: `[1, {a: 2}, "x"]`, want: `[1,{"a":2},"x"]`},
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

func TestFormatConsoleArgKeepsDateReadable(t *testing.T) {
	vm := newPoolTestVM(t)

	date := evalConsoleArg(t, vm, `new Date(0)`)
	if strings.HasPrefix(date, `"`) {
		t.Fatalf("Date should not be JSON-quoted, got %q", date)
	}
}

func TestFormatConsoleArgIncludesErrorStack(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.RunNamed(context.Background(), "err.js", `
function boom() {
	return new Error("boom");
}
`)
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
	vm := root.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	vm.Set("capture", WrapVmCall(vm, func(_ *VM, args Values) any {
		got = FormatConsoleArgs(args)
		return nil
	}))
	if _, e := vm.Run(context.Background(), `capture("hello", {a: 1}, null)`); e != nil {
		t.Fatal(e)
	}
	if got != `hello {"a":1} null` {
		t.Fatalf("FormatConsoleArgs = %q, want joined console output", got)
	}
}

func evalConsoleArg(t *testing.T, vm *VM, code string) string {
	t.Helper()
	value, e := vm.Run(context.Background(), code)
	if e != nil {
		t.Fatal(e)
	}
	return formatConsoleArg(value)
}
