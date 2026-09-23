package task

import (
	"context"
	"encoding/json"
	"errors"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"strings"
	"testing"
	"time"
)

func newTestRunner(t *testing.T, concurrency int) Runner {
	t.Helper()
	runner := NewTaskRunner(common.Config{MaxConcurrentTask: concurrency}, registry.NewComponentHolder())
	t.Cleanup(func() {
		if e := runner.Dispose(); e != nil {
			t.Errorf("dispose runner: %v", e)
		}
	})
	return runner
}

func waitForTask(t *testing.T, runner Runner, id, status string) Task {
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

func TestExecuteAndWaitContinuesAfterContextCancel(t *testing.T) {
	runner := newTestRunner(t, 1)
	release := make(chan struct{})
	finished := make(chan struct{})
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	var snapshot Task
	go func() {
		var err error
		snapshot, err = runner.ExecuteAndWait(ctx, func(types.TaskCtx) (any, error) {
			close(started)
			<-release
			close(finished)
			return "done", nil
		}, time.Minute)
		errCh <- err
	}()
	<-started
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("canceled wait returned error: %v", err)
	}
	if snapshot.ID == "" {
		t.Fatal("canceled wait returned an empty snapshot")
	}

	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("task did not continue in the background")
	}
	completed := waitForTask(t, runner, snapshot.ID, Done)
	if completed.Result != "done" {
		t.Fatalf("unexpected result: %#v", completed.Result)
	}
}

func TestExecuteAndWaitNonPositiveTimeoutWaitsForTask(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			runner := newTestRunner(t, 1)
			release := make(chan struct{})
			started := make(chan struct{})
			go func() {
				<-started
				time.Sleep(30 * time.Millisecond)
				close(release)
			}()

			task, e := runner.ExecuteAndWait(context.Background(), func(ctx types.TaskCtx) (any, error) {
				close(started)
				<-release
				return "done", nil
			}, timeout)
			if e != nil {
				t.Fatal(e)
			}
			if task.Status != Done || task.Result != "done" {
				t.Fatalf("task = %#v, want completed result", task)
			}
		})
	}
}

func TestTaskContextCanceledAfterCompletion(t *testing.T) {
	tests := []struct {
		name    string
		run     func() (any, error)
		status  string
		wantErr string
	}{
		{name: "success", run: func() (any, error) { return "ok", nil }, status: Done},
		{name: "error", run: func() (any, error) { return nil, errors.New("boom") }, status: Error, wantErr: "boom"},
		{name: "panic", run: func() (any, error) { panic("boom") }, status: Error, wantErr: "task panicked: boom"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := newTestRunner(t, 1)
			var got types.TaskCtx
			task, e := runner.ExecuteAndWait(context.Background(), func(ctx types.TaskCtx) (any, error) {
				got = ctx
				return test.run()
			}, time.Second)
			if e != nil {
				t.Fatal(e)
			}
			if task.Status != test.status {
				t.Fatalf("status = %q, want %s", task.Status, test.status)
			}
			if test.wantErr != "" && (task.Error == nil || !strings.Contains(task.Error.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want containing %q", task.Error, test.wantErr)
			}
			select {
			case <-got.Done():
			case <-time.After(time.Second):
				t.Fatal("task context was not canceled after completion")
			}
			if !errors.Is(got.Err(), context.Canceled) {
				t.Fatalf("context error = %v, want canceled", got.Err())
			}
		})
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

func TestExecuteRejectsTaskWhenQueueIsFull(t *testing.T) {
	runner := newTestRunner(t, 1)
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
	queued, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	rejected, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, nil
	})
	close(release)
	unavailable, ok := errors.AsType[err.UnavailableError](e)
	if !ok {
		t.Fatalf("submit error = %v, want unavailable", e)
	}
	if unavailable.Code() != 503 {
		t.Fatalf("status = %d, want 503", unavailable.Code())
	}
	if _, getErr := runner.GetTask(rejected.ID); !errors.Is(getErr, ErrorNotFound) {
		t.Fatalf("rejected task lookup = %v", getErr)
	}
	waitForTask(t, runner, queued.ID, Done)
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
