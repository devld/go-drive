package job

import (
	"context"
	"testing"

	"go-drive/common/types"
)

func TestJobLogFormatsLikeConsole(t *testing.T) {
	vm, e := newJobVM(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	bindJobLog(vm, func(s string) { got = s })

	if _, e := vm.Run(context.Background(), `log("hello", {a: 1}, null)`, ""); e != nil {
		t.Fatal(e)
	}
	if got != `hello {"a":1} null` {
		t.Fatalf("log = %q, want console-style output", got)
	}
}

func TestJobEvalDefinesEvent(t *testing.T) {
	vm, e := newJobVM(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })

	var got string
	bindJobLog(vm, func(s string) { got = s })
	setJobGlobals(vm, nil)

	if _, e := vm.Run(context.Background(), `log("triggered by event:", $event)`, ""); e != nil {
		t.Fatal(e)
	}
	if got != "triggered by event: undefined" {
		t.Fatalf("log = %q, want $event to be undefined", got)
	}
}

func TestJobHostGlobalsAreFrozen(t *testing.T) {
	vm, e := newJobVM(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })

	if e := bindJobLog(vm, func(string) {}); e != nil {
		t.Fatal(e)
	}
	if e := setJobGlobals(vm, types.M{jobEventName: types.M{"type": "cron"}}); e != nil {
		t.Fatal(e)
	}
	got, e := vm.Run(context.Background(), `
		try { log.extra = true; } catch (e) {}
		$event.extra = true;
		[
			Object.isFrozen(log),
			Object.isFrozen($event),
			typeof log.extra,
			typeof $event.extra,
			$event.type
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|false|undefined|undefined|cron" {
		t.Fatalf("job host globals = %q", got.String())
	}
}
