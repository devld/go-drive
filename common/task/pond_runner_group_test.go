package task

import (
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/types"
	"sync/atomic"
	"testing"
	"time"
)

func TestPondRunnerUsesExactGroupSubpool(t *testing.T) {
	runner := newTestRunnerWithConfig(t, common.Config{
		MaxConcurrentTask: 3,
	})
	if err := runner.RegisterGroup("drive/thumbnail", 1); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	var running atomic.Int32
	var maxRunning atomic.Int32
	for range 2 {
		_, err := runner.Execute(func(ctx types.TaskCtx) (any, error) {
			current := running.Add(1)
			for {
				old := maxRunning.Load()
				if current <= old || maxRunning.CompareAndSwap(old, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			running.Add(-1)
			return nil, nil
		}, WithGroup("drive/thumbnail"))
		if err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("thumbnail task did not start")
	}
	select {
	case <-started:
		t.Fatal("thumbnail subpool exceeded its concurrency limit")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && running.Load() != 0 {
		time.Sleep(time.Millisecond)
	}
	if got := maxRunning.Load(); got != 1 {
		t.Fatalf("maximum thumbnail concurrency = %d, want 1", got)
	}
}

func newTestRunnerWithConfig(t *testing.T, config common.Config) *PondRunner {
	t.Helper()
	runner := NewPondRunner(config, registry.NewComponentHolder())
	t.Cleanup(func() { _ = runner.Dispose() })
	return runner
}
