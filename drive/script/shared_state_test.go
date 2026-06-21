package script

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	s "go-drive/script"
)

func newSharedStateTestVM(t *testing.T, d *ScriptDrive) *s.VM {
	t.Helper()
	root, e := s.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	vm := root.Fork()
	vm.Set("setData", s.WrapVmCall(vm, d.setData))
	vm.Set("getData", s.WrapVmCall(vm, d.getData))
	t.Cleanup(func() {
		_ = vm.Dispose()
		_ = root.Dispose()
	})
	return vm
}

func TestSharedStateIsCopiedBetweenVMs(t *testing.T) {
	d := &ScriptDrive{data: make(map[string]json.RawMessage)}
	first := newSharedStateTestVM(t, d)
	second := first.Fork()
	t.Cleanup(func() { _ = second.Dispose() })

	if _, e := first.Run(context.Background(), `setData({state: {count: 1, values: ["a"]}})`); e != nil {
		t.Fatal(e)
	}
	if _, e := second.Run(context.Background(), `var local = getData("state"); local.count = 2; local.values.push("b")`); e != nil {
		t.Fatal(e)
	}
	value, e := first.Run(context.Background(), `JSON.stringify(getData("state"))`)
	if e != nil {
		t.Fatal(e)
	}
	if got := value.String(); got != `{"count":1,"values":["a"]}` {
		t.Fatalf("nested mutation changed shared state: %s", got)
	}

	if _, e := second.Run(context.Background(), `setData({state: local})`); e != nil {
		t.Fatal(e)
	}
	value, e = first.Run(context.Background(), `JSON.stringify(getData("state"))`)
	if e != nil {
		t.Fatal(e)
	}
	if got := value.String(); got != `{"count":2,"values":["a","b"]}` {
		t.Fatalf("reassigned shared state was not persisted: %s", got)
	}
}

func TestSharedStateRejectsNonJSONValues(t *testing.T) {
	for name, code := range map[string]string{
		"function": `setData({state: function () {}})`,
		"cycle":    `var state = {}; state.self = state; setData({state: state})`,
	} {
		t.Run(name, func(t *testing.T) {
			d := &ScriptDrive{data: make(map[string]json.RawMessage)}
			vm := newSharedStateTestVM(t, d)

			_, e := vm.Run(context.Background(), code)
			if e == nil || !strings.Contains(e.Error(), "shared state must be JSON serializable") {
				t.Fatalf("expected JSON serialization error, got %v", e)
			}
		})
	}
}
