package job

import (
	"context"
	"errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"testing"
)

func TestRunEntryOperationsReportsItemProgress(t *testing.T) {
	ctx := task.NewTaskContext(context.Background())
	groups := [][]types.IEntry{{nil, nil}, {nil}}
	calls := 0

	if e := runEntryOperations(ctx, groups, false, func(types.IEntry) error {
		calls++
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if calls != 3 || ctx.GetProgress() != 3 || ctx.GetTotal() != 3 {
		t.Fatalf("calls/progress = %d/%d/%d, want 3/3/3", calls, ctx.GetProgress(), ctx.GetTotal())
	}
}

func TestRunEntryOperationsOnlyCountsCompletedItems(t *testing.T) {
	ctx := task.NewTaskContext(context.Background())
	wantErr := errors.New("failed")
	calls := 0

	e := runEntryOperations(ctx, [][]types.IEntry{{nil, nil, nil}}, false, func(types.IEntry) error {
		calls++
		if calls == 2 {
			return wantErr
		}
		return nil
	})
	if !errors.Is(e, wantErr) {
		t.Fatalf("error = %v, want %v", e, wantErr)
	}
	if ctx.GetProgress() != 1 || ctx.GetTotal() != 3 {
		t.Fatalf("progress = %d/%d, want 1/3", ctx.GetProgress(), ctx.GetTotal())
	}
}

func TestFlowResetsProgressBeforeEveryStep(t *testing.T) {
	originalDefs := len(registeredActionDefs)
	defer func() { registeredActionDefs = registeredActionDefs[:originalDefs] }()

	ctx := task.NewTaskContext(context.Background())
	ctx.Progress(9, true)
	ctx.Total(10, true)
	snapshots := make([][2]int64, 0, 2)
	RegisterActionDef(JobActionDef{
		Name: "test-progress-reset",
		Do: func(types.TaskCtx, types.SM, *registry.ComponentsHolder, func(string)) error {
			snapshots = append(snapshots, [2]int64{ctx.GetProgress(), ctx.GetTotal()})
			ctx.Progress(4, true)
			ctx.Total(8, true)
			return nil
		},
	})

	flow := GetActionDef("flow")
	e := flow.Do(ctx, types.SM{
		"ops": `[{"$key":"test-progress-reset"},{"$key":"test-progress-reset"}]`,
	}, registry.NewComponentHolder(), func(string) {})
	if e != nil {
		t.Fatal(e)
	}
	if len(snapshots) != 2 || snapshots[0] != [2]int64{} || snapshots[1] != [2]int64{} {
		t.Fatalf("step starting progress = %#v, want two zero snapshots", snapshots)
	}
}
