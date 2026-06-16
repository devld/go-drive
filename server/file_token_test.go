package server

import (
	"go-drive/common/types"
	"testing"
	"time"
)

func newTestFileTokenStore(t *testing.T) *FileTokenStore {
	t.Helper()
	return &FileTokenStore{
		root:     t.TempDir(),
		validity: time.Hour,
	}
}

func TestFileTokenStore_CreateAndValidate(t *testing.T) {
	ft := newTestFileTokenStore(t)

	s := types.NewSession()
	s.User = types.User{Username: "alice"}

	tok, e := ft.Create(s)
	if e != nil {
		t.Fatal(e)
	}

	got, e := ft.Validate(tok.Token)
	if e != nil {
		t.Fatalf("validate failed: %v", e)
	}
	if got.Token != tok.Token {
		t.Fatalf("token mismatch: %q vs %q", got.Token, tok.Token)
	}
	if got.Value.User.Username != "alice" {
		t.Fatalf("unexpected session user: %q", got.Value.User.Username)
	}
}

// TestFileTokenStore_ValidateMissingToken ensures validating an unknown token
// returns an error without panicking (regression for the nil-stat deref).
func TestFileTokenStore_ValidateMissingToken(t *testing.T) {
	ft := newTestFileTokenStore(t)

	if _, e := ft.Validate("0000abcd-0000-0000-0000-000000000000"); e == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestFileTokenStore_ValidateInvalidTokenChars(t *testing.T) {
	ft := newTestFileTokenStore(t)

	// contains a path separator; getSessionFile maps it to an "invalid" file
	if _, e := ft.Validate("../../etc/passwd"); e == nil {
		t.Fatal("expected error for invalid token")
	}
}
