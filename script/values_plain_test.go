package script

import (
	"context"
	"testing"
)

func TestToPlainJSValueMaterializesEditableData(t *testing.T) {
	vm := newPoolTestVM(t)
	if _, e := vm.Run(context.Background(), `
		Object.defineProperty(Object.prototype, "nested", {
			set() { throw new Error("object prototype setter called"); },
			configurable: true
		});
		Object.defineProperty(Array.prototype, "0", {
			set() { throw new Error("array prototype setter called"); },
			configurable: true
		});
	`, ""); e != nil {
		t.Fatal(e)
	}
	source := map[string]any{
		"nested":    map[string]int{"value": 1},
		"items":     []string{"a"},
		"fixed":     [2]int{2, 3},
		"__proto__": map[string]bool{"plain": true},
	}
	mustDefineGlobal(t, vm, "plain", vm.ToPlainJSValue(source))

	got, e := vm.Run(context.Background(), `
		const ownProto = Object.prototype.hasOwnProperty.call(plain, "__proto__");
		const normalPrototype = Object.getPrototypeOf(plain) === Object.prototype;
		plain.nested.value = 2;
		plain.items.push("b");
		plain.extra = true;
		[
			ownProto,
			normalPrototype,
			plain.__proto__.plain,
			plain.nested.value,
			plain.items.join(","),
			plain.fixed.join(","),
			plain.extra
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|true|2|a,b|2,3|true" {
		t.Fatalf("plain value = %q", got.String())
	}
	if source["nested"].(map[string]int)["value"] != 1 || len(source["items"].([]string)) != 1 {
		t.Fatalf("source mutated: %#v", source)
	}
}

func TestToPlainJSValuePreservesValuesAndContainerIdentity(t *testing.T) {
	vm := newPoolTestVM(t)
	embedded := mustGetValue(t, vm, `({ marker: 1 })`)
	shared := map[string]any{"value": 1}
	cycle := map[string]any{}
	cycle["self"] = cycle
	sliceCycle := make([]any, 1)
	sliceCycle[0] = sliceCycle

	mustDefineGlobal(t, vm, "embedded", embedded)
	mustDefineGlobal(t, vm, "plain", vm.ToPlainJSValue(map[string]any{
		"embedded": embedded,
		"first":    shared,
		"second":   shared,
		"cycle":    cycle,
		"slice":    sliceCycle,
	}))
	got, e := vm.Run(context.Background(), `[
		plain.embedded === embedded,
		plain.first === plain.second,
		plain.cycle === plain.cycle.self,
		plain.slice === plain.slice[0]
	].join("|")`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|true|true" {
		t.Fatalf("identity = %q", got.String())
	}
}

func TestToPlainJSValueRejectsUnsupportedValues(t *testing.T) {
	vm := newPoolTestVM(t)
	tests := []struct {
		name  string
		value any
	}{
		{name: "struct", value: struct{}{}},
		{name: "map-key", value: map[int]string{1: "one"}},
		{name: "function", value: func() {}},
		{name: "pointer", value: (*int)(nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("ToPlainJSValue(%T) did not panic", test.value)
				}
			}()
			_ = vm.ToPlainJSValue(test.value)
		})
	}
}

func TestToPlainJSValueRejectsForeignValue(t *testing.T) {
	first := newPoolTestVM(t)
	second := newPoolTestVM(t)
	foreign := first.ToPlainJSValue(map[string]int{"value": 1})
	defer func() {
		if recover() == nil {
			t.Fatal("foreign Value did not panic")
		}
	}()
	_ = second.ToPlainJSValue(map[string]any{"foreign": foreign})
}
