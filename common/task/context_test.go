package task

import (
	"context"
	"testing"
)

func TestWithContextPreservesProgressAndUsesReplacementCancellation(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}
	progress := NewTaskContext(context.WithValue(context.Background(), key, "original"))
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), key, "replacement"))
	w := WithContext(ctx, progress)

	w.Progress(3, false)
	w.Total(10, true)
	if progress.GetProgress() != 3 || progress.GetTotal() != 10 {
		t.Fatalf("progress = %d/%d, want 3/10", progress.GetProgress(), progress.GetTotal())
	}
	if value := w.Value(key); value != "replacement" {
		t.Fatalf("Value() = %v, want replacement context value", value)
	}

	cancel()
	if e := w.Err(); e != context.Canceled {
		t.Fatalf("Err() = %v, want context.Canceled", e)
	}
}
