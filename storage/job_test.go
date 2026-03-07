package storage

import (
	"errors"
	err "go-drive/common/errors"
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
