package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"testing"
)

func TestJobDAO_GetJob_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewJobDAO(db, ch)
	_, e := dao.GetJob(999999)
	if e == nil {
		t.Fatal("expected error for nonexistent job")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}

func TestJobDAO_GetJobExecutionsPagination(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewJobDAO(db, ch)

	for i := range 5 {
		if e := dao.AddJobExecution(&types.JobExecution{
			JobId:     42,
			StartedAt: uint64(100 + i),
			Status:    types.JobExecutionSuccess,
		}); e != nil {
			t.Fatal(e)
		}
	}
	if e := dao.AddJobExecution(&types.JobExecution{
		JobId: 7, StartedAt: 999, Status: types.JobExecutionSuccess,
	}); e != nil {
		t.Fatal(e)
	}

	items, total, e := dao.GetJobExecutions(42, 2, 2)
	if e != nil {
		t.Fatal(e)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(items) != 2 || items[0].StartedAt != 102 || items[1].StartedAt != 101 {
		t.Fatalf("unexpected page items: %#v", items)
	}
}
