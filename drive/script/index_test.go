package script

import (
	"context"
	"errors"
	"testing"
	"time"

	s "go-drive/script"
)

func TestScriptDriveCallUsesCallerContext(t *testing.T) {
	vm, e := s.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })

	if _, e = vm.Run(context.Background(), `function __drive_get() { while (true) {} }`); e != nil {
		t.Fatal(e)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, e = (&ScriptDrive{}).call(ctx, vm, "get")
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", e)
	}
}
