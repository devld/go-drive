package task

import (
	"errors"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerLimitsGroupConcurrency(t *testing.T) {
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
	var completed atomic.Int32
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
			completed.Add(1)
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
	for time.Now().Before(deadline) && completed.Load() != 2 {
		time.Sleep(time.Millisecond)
	}
	if got := completed.Load(); got != 2 {
		t.Fatalf("completed tasks = %d, want 2", got)
	}
	if got := maxRunning.Load(); got != 1 {
		t.Fatalf("maximum thumbnail concurrency = %d, want 1", got)
	}
}

func TestUngroupedTaskRunsWhileGroupIsSaturated(t *testing.T) {
	runner := newTestRunnerWithConfig(t, common.Config{MaxConcurrentTask: 2})
	if err := runner.RegisterGroup("drive/thumbnail", 1); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	thumbStarted := make(chan struct{})
	otherStarted := make(chan struct{})
	var thumbs atomic.Int32
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		thumbs.Add(1)
		close(thumbStarted)
		<-release
		return nil, nil
	}, WithGroup("drive/thumbnail")); e != nil {
		t.Fatal(e)
	}
	<-thumbStarted
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		thumbs.Add(1)
		<-release
		return nil, nil
	}, WithGroup("drive/thumbnail")); e != nil {
		t.Fatal(e)
	}
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		close(otherStarted)
		<-release
		return nil, nil
	}); e != nil {
		t.Fatal(e)
	}
	select {
	case <-otherStarted:
	case <-time.After(time.Second):
		t.Fatal("ungrouped task did not start while a thumbnail was waiting")
	}
	if got := thumbs.Load(); got != 1 {
		t.Fatalf("running thumbnails = %d, want 1", got)
	}
	close(release)
}

func TestGroupTasksShareTheGlobalQueue(t *testing.T) {
	runner := newTestRunnerWithConfig(t, common.Config{MaxConcurrentTask: 2})
	if err := runner.RegisterGroup("drive/thumbnail", 1); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	started := make(chan struct{})
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		close(started)
		<-release
		return nil, nil
	}, WithGroup("drive/thumbnail")); e != nil {
		t.Fatal(e)
	}
	<-started
	for range 2 {
		if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
			<-release
			return nil, nil
		}, WithGroup("drive/thumbnail")); e != nil {
			t.Fatal(e)
		}
	}
	rejected, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, nil
	}, WithGroup("drive/thumbnail"))
	close(release)
	if _, ok := errors.AsType[err.UnavailableError](e); !ok {
		t.Fatalf("submit error = %v, want unavailable", e)
	}
	if _, getErr := runner.GetTask(rejected.ID); !errors.Is(getErr, ErrorNotFound) {
		t.Fatalf("rejected task lookup = %v", getErr)
	}
}

func TestGroupSubmitFailsWhenGlobalQueueIsFull(t *testing.T) {
	runner := newTestRunnerWithConfig(t, common.Config{MaxConcurrentTask: 1})
	if err := runner.RegisterGroup("artifact/archive", 1); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	started := make(chan struct{})
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		close(started)
		<-release
		return nil, nil
	}); e != nil {
		t.Fatal(e)
	}
	<-started
	if _, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		<-release
		return nil, nil
	}); e != nil {
		t.Fatal(e)
	}
	rejected, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, nil
	}, WithGroup("artifact/archive"))
	close(release)
	if _, ok := errors.AsType[err.UnavailableError](e); !ok {
		t.Fatalf("submit error = %v, want unavailable", e)
	}
	if _, getErr := runner.GetTask(rejected.ID); !errors.Is(getErr, ErrorNotFound) {
		t.Fatalf("rejected task lookup = %v", getErr)
	}
}

func newTestRunnerWithConfig(t *testing.T, config common.Config) Runner {
	t.Helper()
	runner := NewTaskRunner(config)
	t.Cleanup(func() { _ = runner.Dispose() })
	return runner
}
