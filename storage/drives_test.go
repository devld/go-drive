package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"testing"
)

func TestDriveDAO_AddDrive_duplicateReturnsNotAllowed(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewDriveDAO(db, ch)
	d := types.Drive{Name: "d1", Type: "fs", Enabled: true, Config: "{}"}
	_, _ = dao.AddDrive(d)
	_, e := dao.AddDrive(d)
	if e == nil {
		t.Fatal("expected error when adding duplicate drive")
	}
	var notAllowed err.NotAllowedError
	if !errors.As(e, &notAllowed) {
		t.Errorf("expected NotAllowedError, got %T: %v", e, e)
	}
}

func TestDriveDAO_GetDrive_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewDriveDAO(db, ch)
	_, e := dao.GetDrive("nonexistent")
	if e == nil {
		t.Fatal("expected error for nonexistent drive")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}
