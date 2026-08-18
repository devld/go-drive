package job

import (
	"context"
	"testing"
)

func TestJobLogFormatsLikeConsole(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	bindJobLog(vm, func(s string) { got = s })

	if _, e := vm.Run(context.Background(), `log("hello", {a: 1}, null)`); e != nil {
		t.Fatal(e)
	}
	if got != `hello {"a":1} null` {
		t.Fatalf("log = %q, want console-style output", got)
	}
}

func TestJobEvalDefinesEvent(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	bindJobLog(vm, func(s string) { got = s })
	setJobGlobals(vm, nil)

	if _, e := vm.Run(context.Background(), `log("triggered by event:", $event)`); e != nil {
		t.Fatal(e)
	}
	if got != "triggered by event: undefined" {
		t.Fatalf("log = %q, want $event to be undefined", got)
	}
}
