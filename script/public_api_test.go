package script_test

import (
	"context"
	"testing"

	"go-drive/script"
)

type publicBox struct {
	script.ClassHost
	value int64
}

func TestValuePassThrough(t *testing.T) {
	vm, err := script.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = vm.Dispose() }()

	if err := vm.DefineGlobal("value", vm.ToJSValue("opaque")); err != nil {
		t.Fatal(err)
	}
	if err := vm.DefineGlobal("zero", (*script.Value)(nil)); err != nil {
		t.Fatal(err)
	}
	if err := vm.DefineGlobal("nullValue", vm.ToJSValue(nil)); err != nil {
		t.Fatal(err)
	}
	result, err := vm.Run(context.Background(), `[value, typeof zero, nullValue === null]`, "")
	if err != nil {
		t.Fatal(err)
	}
	items := result.Array()
	if len(items) != 3 || items[0].String() != "opaque" || items[1].String() != "undefined" || !items[2].Bool() {
		t.Fatalf("value round trip = %v", result.Raw())
	}
}

func TestJSClassConstructorUsesValues(t *testing.T) {
	vm, err := script.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = vm.Dispose() }()

	classes := script.NewClassSet(&script.JSClass{
		Name:   "PublicBox",
		Handle: publicBox{},
		Construct: func(vm *script.VM, args script.Values) any {
			return publicBox{ClassHost: script.NewClassHost(vm), value: args.Get(0).Integer()}
		},
		Getters: map[string]script.ClassMethod{
			"value": func(vm *script.VM, this *script.Value, _ script.Values) any {
				return script.This[publicBox](vm, this, "PublicBox.value").value
			},
		},
	})
	if err := vm.AddClassSet(classes); err != nil {
		t.Fatal(err)
	}
	result, err := vm.Run(context.Background(), `new PublicBox(42).value`, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Integer() != 42 {
		t.Fatalf("constructor value = %d, want 42", result.Integer())
	}
}

func TestValueCannotCrossVMs(t *testing.T) {
	first, err := script.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Dispose() }()
	second, err := script.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Dispose() }()

	value := first.ToJSValue("foreign")
	if _, err := second.Call(context.Background(), "String", value); err == nil {
		t.Fatal("expected a cross-VM Value error")
	}
}
