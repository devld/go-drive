package task

import (
	"context"
	"encoding/json"
	"errors"
	"go-drive/common"
	err "go-drive/common/errors"
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
	task, e := runner.ExecuteAndWait(context.Background(), func(ctx types.TaskCtx) (any, error) {
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
	completed := waitForTask(t, runner, task.ID, Done)
	if completed.Result != "done" {
		t.Fatalf("unexpected result: %#v", completed.Result)
	}
}

func TestExecuteAndWaitPanicsForNilContext(t *testing.T) {
	runner := newTestRunner(t, 1)
	defer func() {
		if recover() == nil {
			t.Fatal("ExecuteAndWait(nil) did not panic")
		}
	}()
	_, _ = runner.ExecuteAndWait(nil, func(ctx types.TaskCtx) (any, error) { //nolint:staticcheck // ExecuteAndWait must panic on a nil context.
		return nil, nil
	}, time.Second)
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
		snapshot, e := runner.GetTask(task.ID)
		if e != nil {
			t.Fatal(e)
		}
		if snapshot.Finished() {
			break
		}
	}
	waitForTask(t, runner, task.ID, Done)
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
	waitForTask(t, runner, task.ID, Running)
	stopped, e := runner.StopTask(task.ID)
	if e != nil {
		t.Fatal(e)
	}
	if stopped.Status != Canceled {
		t.Fatalf("unexpected stopped status: %q", stopped.Status)
	}
	waitForTask(t, runner, task.ID, Canceled)
}

func TestFailedTaskJSONUsesPublicErrorPayload(t *testing.T) {
	runner := newTestRunner(t, 1)
	want := err.NewNotFoundMessageError("missing artifact")
	created, e := runner.ExecuteAndWait(context.Background(), func(types.TaskCtx) (any, error) {
		return nil, want
	}, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if created.Status != Error {
		t.Fatalf("status = %q, want %s", created.Status, Error)
	}
	if !errors.Is(created.Error, want) {
		t.Fatalf("Error = %v, want %v", created.Error, want)
	}
	payload, e := json.Marshal(created)
	if e != nil {
		t.Fatal(e)
	}
	var body map[string]any
	if e := json.Unmarshal(payload, &body); e != nil {
		t.Fatal(e)
	}
	public, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload = %#v", body["error"])
	}
	if public["message"] != want.Error() {
		t.Fatalf("public message = %#v", public["message"])
	}
	if code, _ := public["code"].(float64); int(code) != want.Code() {
		t.Fatalf("public code = %#v", public["code"])
	}
}
