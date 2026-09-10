package job

import (
	"go-drive/common/task"
	"go-drive/common/types"
	"testing"
)

func TestJobExecutionLoggerPreservesMessage(t *testing.T) {
	var got string
	logger := newJobExecutionLogger(42, 960, func(message string) {
		got = message
	})

	logger.Log("executed: 1")

	if got != "executed: 1" {
		t.Fatalf("onLog message = %q, want original message", got)
	}
	if logs := logger.String(); logs != "executed: 1\n" {
		t.Fatalf("stored logs = %q, want original message", logs)
	}
}

type taskLookupRunner struct {
	task.Runner
	task task.Task
}

func (r taskLookupRunner) GetTask(id string) (task.Task, error) {
	if id != r.task.Id {
		return task.Task{}, task.ErrorNotFound
	}
	return r.task, nil
}

func TestGetExecutionProgress(t *testing.T) {
	runner := taskLookupRunner{task: task.Task{
		Id:       "task-2",
		Progress: task.Progress{Loaded: 3, Total: 5},
	}}
	executor := &JobExecutor{
		runner: runner,
		executions: map[uint]*jobExecutionItem{
			2: {JobExecution: &types.JobExecution{ID: 2}, TaskID: "task-2"},
		},
	}

	if _, ok := executor.GetExecutionProgress(1); ok {
		t.Fatal("missing execution unexpectedly has progress")
	}
	progress, ok := executor.GetExecutionProgress(2)
	if !ok || progress.Loaded != 3 || progress.Total != 5 {
		t.Fatalf("GetExecutionProgress() = %#v, %v", progress, ok)
	}
}
