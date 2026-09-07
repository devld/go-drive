package script

import (
	"context"
	"testing"

	err "go-drive/common/errors"
)

func TestNativeFunctionErrorFromVM(t *testing.T) {
	root := newPoolTestVM(t)
	mustDefineGlobal(t, root, "fail", NativeFunction(func(vm *VM, _ Values) any {
		vm.ThrowError(err.NewNotFoundMessageError("missing"))
		return nil
	}))
	vm := root
	t.Cleanup(func() { _ = vm.Dispose() })

	if _, e := vm.Run(context.Background(), `
		try {
			fail();
		} catch (e) {
			if (!(e instanceof NotFoundError)) throw new Error("wrong error type: " + e);
		}
	`, ""); e != nil {
		t.Fatal(e)
	}

	if _, e := vm.Run(context.Background(), `fail()`, ""); !err.IsNotFoundError(e) {
		t.Fatalf("expected not found error, got %v", e)
	}
}
