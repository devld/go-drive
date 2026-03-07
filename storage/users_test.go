package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"testing"
)

func TestUserDAO_AddUser_passwordNotReturned(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewUserDAO(db, ch)
	u, e := dao.AddUser(types.User{Username: "u1", Password: "plain"})
	if e != nil {
		t.Fatalf("AddUser: %v", e)
	}
	if u.Password == "plain" {
		t.Error("AddUser must not return plain password")
	}
}

func TestUserDAO_AddUser_duplicateReturnsNotAllowed(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewUserDAO(db, ch)
	_, _ = dao.AddUser(types.User{Username: "dup", Password: "p"})
	_, e := dao.AddUser(types.User{Username: "dup", Password: "p2"})
	if e == nil {
		t.Fatal("expected error when adding duplicate user")
	}
	var notAllowed err.NotAllowedError
	if !errors.As(e, &notAllowed) {
		t.Errorf("expected NotAllowedError, got %T: %v", e, e)
	}
}

func TestUserDAO_GetUser_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewUserDAO(db, ch)
	_, e := dao.GetUser("nonexistent")
	if e == nil {
		t.Fatal("expected error for nonexistent user")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}

func TestUserDAO_UpdateUser_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewUserDAO(db, ch)
	e := dao.UpdateUser("nonexistent", types.User{RootPath: "/"})
	if e == nil {
		t.Fatal("expected error when updating nonexistent user")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}

func TestUserDAO_DeleteUser_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewUserDAO(db, ch)
	e := dao.DeleteUser("nonexistent")
	if e == nil {
		t.Fatal("expected error when deleting nonexistent user")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}
