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
	vm, e := s.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	if e = vm.WithBridge(map[string]any{
		"version":  "",
		"name":     "test",
		"initData": s.NativeFunction(d.jsFunInitData),
		"setData":  s.NativeFunction(d.jsFunSetData),
		"getData":  s.NativeFunction(d.jsFunGetData),
	}, func() error {
		_, e := vm.Run(context.Background(), helperProgram, "helper.js")
		return e
	}); e != nil {
		_ = vm.Dispose()
		t.Fatal(e)
	}
	mustDefineGlobal(t, vm, "testSetData", s.NativeFunction(d.jsFunSetData))
	mustDefineGlobal(t, vm, "testGetData", s.NativeFunction(d.jsFunGetData))
	t.Cleanup(func() { _ = vm.Dispose() })
	return vm
}

func TestSharedStateIsCopiedBetweenVMs(t *testing.T) {
	d := &ScriptDrive{data: make(map[string]any)}
	first := newSharedStateTestVM(t, d)
	second := newSharedStateTestVM(t, d)

	if _, e := first.Run(context.Background(), `testSetData({state: {count: 1, values: ["a"]}})`, ""); e != nil {
		t.Fatal(e)
	}
	if _, e := second.Run(context.Background(), `
		var src = testGetData("state");
		var local = Object.assign({}, src, {count: 2, values: src.values.concat("b")});
	`, ""); e != nil {
		t.Fatal(e)
	}
	value, e := first.Run(context.Background(), `JSON.stringify(testGetData("state"))`, "")
	if e != nil {
		t.Fatal(e)
	}
	if !sharedStateJSONEqual(value.String(), `{"count":1,"values":["a"]}`) {
		t.Fatalf("nested mutation changed shared state: %s", value.String())
	}

	if _, e := second.Run(context.Background(), `testSetData({state: local})`, ""); e != nil {
		t.Fatal(e)
	}
	value, e = first.Run(context.Background(), `JSON.stringify(testGetData("state"))`, "")
	if e != nil {
		t.Fatal(e)
	}
	if !sharedStateJSONEqual(value.String(), `{"count":2,"values":["a","b"]}`) {
		t.Fatalf("reassigned shared state was not persisted: %s", value.String())
	}
}

func sharedStateJSONEqual(got, want string) bool {
	var g, w any
	if json.Unmarshal([]byte(got), &g) != nil || json.Unmarshal([]byte(want), &w) != nil {
		return got == want
	}
	gb, e1 := json.Marshal(g)
	wb, e2 := json.Marshal(w)
	return e1 == nil && e2 == nil && string(gb) == string(wb)
}

func TestSharedStateRoundTripsNull(t *testing.T) {
	d := &ScriptDrive{data: make(map[string]any)}
	vm := newSharedStateTestVM(t, d)

	got, e := vm.Run(context.Background(), `
		testSetData({$state: null});
		var value = testGetData("$state");
		testSetData({$state: value});
		String(testGetData("$state")) + "|" + typeof testGetData("$state") + "|" + String(testGetData("$state") === null)
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "null|object|true" {
		t.Fatalf("null shared state = %q", got.String())
	}
}

func TestSharedStateRejectsNonJSONValues(t *testing.T) {
	for name, code := range map[string]string{
		"function": `testSetData({state: function () {}})`,
		"cycle":    `var state = {}; state.self = state; testSetData({state: state})`,
	} {
		t.Run(name, func(t *testing.T) {
			d := &ScriptDrive{data: make(map[string]any)}
			vm := newSharedStateTestVM(t, d)

			_, e := vm.Run(context.Background(), code, "")
			if e == nil || !strings.Contains(e.Error(), "shared state must be JSON serializable") {
				t.Fatalf("expected JSON serialization error, got %v", e)
			}
		})
	}
}
