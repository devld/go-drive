package task

import (
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/types"
	"testing"
	"time"
)

func newTestRunner(t *testing.T, concurrency int) *PondRunner {
	t.Helper()
	runner := NewPondRunner(common.Config{MaxConcurrentTask: concurrency}, registry.NewComponentHolder())
	t.Cleanup(func() {
		if e := runner.Dispose(); e != nil {
			t.Errorf("dispose runner: %v", e)
		}
	})
	return runner
}

func waitForTask(t *testing.T, runner *PondRunner, id, status string) Task {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, e := runner.GetTask(id)
		if e != nil {
			t.Fatal(e)
		}
		if task.Status == status {
			return task
		}
		time.Sleep(time.Millisecond)
	}
	task, _ := runner.GetTask(id)
	t.Fatalf("task did not reach %q; current status is %q", status, task.Status)
	return Task{}
}

func TestExecuteAndWaitContinuesAfterTimeout(t *testing.T) {
	runner := newTestRunner(t, 1)
	release := make(chan struct{})
	finished := make(chan struct{})

	started := time.Now()
	task, e := runner.ExecuteAndWait(func(ctx types.TaskCtx) (any, error) {
		<-release
		close(finished)
		return "done", nil
	}, 10*time.Millisecond)
	if e != nil {
		t.Fatal(e)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("ExecuteAndWait did not return on timeout: %v", elapsed)
	}

	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("task did not continue in the background")
	}
	completed := waitForTask(t, runner, task.Id, Done)
	if completed.Result != "done" {
		t.Fatalf("unexpected result: %#v", completed.Result)
	}
}

func TestRunnerSnapshotsDuringProgressUpdates(t *testing.T) {
	runner := newTestRunner(t, 1)
	task, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		for i := range 1000 {
			ctx.Progress(int64(i), true)
			ctx.Total(1000, true)
		}
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	for {
		snapshot, e := runner.GetTask(task.Id)
		if e != nil {
			t.Fatal(e)
		}
		if snapshot.Finished() {
			break
		}
	}
	waitForTask(t, runner, task.Id, Done)
}

func TestStopTaskCancelsRunningTask(t *testing.T) {
	runner := newTestRunner(t, 1)
	task, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if e != nil {
		t.Fatal(e)
	}
	waitForTask(t, runner, task.Id, Running)
	stopped, e := runner.StopTask(task.Id)
	if e != nil {
		t.Fatal(e)
	}
	if stopped.Status != Canceled {
		t.Fatalf("unexpected stopped status: %q", stopped.Status)
	}
	waitForTask(t, runner, task.Id, Canceled)
}
