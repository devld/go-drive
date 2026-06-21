package script

import (
	"context"
	"testing"

	err "go-drive/common/errors"
)

func TestDetachedErrorSurvivesVMFork(t *testing.T) {
	root := newPoolTestVM(t)
	root.Set("fail", func() { ThrowDetachedError(err.NewNotFoundMessageError("missing")) })
	vm := root.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	if _, e := vm.Run(context.Background(), `
		try {
			fail();
		} catch (e) {
			if (!isNotFoundErr(e)) throw new Error("wrong error type: " + e);
		}
	`); e != nil {
		t.Fatal(e)
	}

	if _, e := vm.Run(context.Background(), `fail()`); !err.IsNotFoundError(e) {
		t.Fatalf("expected not found error, got %v", e)
	}
}
