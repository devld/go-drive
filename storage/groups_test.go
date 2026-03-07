package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"testing"
)

func TestGroupDAO_AddGroup_duplicateReturnsNotAllowed(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewGroupDAO(db, ch)
	_, _ = dao.AddGroup(GroupWithUsers{Group: types.Group{Name: "g1"}})
	_, e := dao.AddGroup(GroupWithUsers{Group: types.Group{Name: "g1"}})
	if e == nil {
		t.Fatal("expected error when adding duplicate group")
	}
	var notAllowed err.NotAllowedError
	if !errors.As(e, &notAllowed) {
		t.Errorf("expected NotAllowedError, got %T: %v", e, e)
	}
}

func TestGroupDAO_GetGroup_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewGroupDAO(db, ch)
	_, e := dao.GetGroup("nonexistent")
	if e == nil {
		t.Fatal("expected error for nonexistent group")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}
