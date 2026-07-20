package server

import (
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"go-drive/testutil"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_, cleanup := testutil.GetSharedTestConfig()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func newTestDBTokenStore(t *testing.T) (*DBTokenStore, *storage.UserDAO, func()) {
	t.Helper()
	config, _ := testutil.GetSharedTestConfig()
	ch := registry.NewComponentHolder()
	db, e := storage.NewDB(config, ch)
	if e != nil {
		t.Fatalf("NewDB: %v", e)
	}
	userDAO := storage.NewUserDAO(db, ch)
	sessionDAO := storage.NewSessionDAO(db, ch)
	ts, e := NewDBTokenStore(sessionDAO, userDAO, config, ch)
	if e != nil {
		_ = db.Dispose()
		_ = ch.Dispose()
		t.Fatalf("NewDBTokenStore: %v", e)
	}
	return ts, userDAO, func() {
		_ = ts.Dispose()
		_ = db.Dispose()
		_ = ch.Dispose()
	}
}

func TestDBTokenStore_CreateAndValidate(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()

	if _, e := userDAO.AddUser(types.User{Username: "alice", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}

	s := types.Principal{AuthType: types.AuthTypeToken}
	s.User = types.User{Username: "alice"}
	tok, e := ts.Create(s)
	if e != nil {
		t.Fatalf("Create: %v", e)
	}
	if tok.Token == "" {
		t.Fatal("expected a non-empty token")
	}

	got, e := ts.Validate(tok.Token)
	if e != nil {
		t.Fatalf("Validate: %v", e)
	}
	if got.Value.User.Username != "alice" {
		t.Fatalf("unexpected user: %q", got.Value.User.Username)
	}
}

func TestDBTokenStore_TokenStoredAsHash(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	if _, e := userDAO.AddUser(types.User{Username: "bob", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}
	s := types.Principal{AuthType: types.AuthTypeToken}
	s.User = types.User{Username: "bob"}
	tok, e := ts.Create(s)
	if e != nil {
		t.Fatalf("Create: %v", e)
	}
	// the raw token must not be stored; only its hash is
	if _, e := ts.sessionDAO.GetByHash(tok.Token); e == nil {
		t.Fatal("raw token should not be a valid lookup key")
	}
	if _, e := ts.sessionDAO.GetByHash(hashToken(tok.Token)); e != nil {
		t.Fatalf("hashed token not found: %v", e)
	}
}

func TestDBTokenStore_Revoke(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	if _, e := userDAO.AddUser(types.User{Username: "carol", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}
	s := types.Principal{AuthType: types.AuthTypeToken}
	s.User = types.User{Username: "carol"}
	tok, _ := ts.Create(s)

	if e := ts.Revoke(tok.Token); e != nil {
		t.Fatalf("Revoke: %v", e)
	}
	if _, e := ts.Validate(tok.Token); e == nil {
		t.Fatal("expected error validating a revoked token")
	}
}

func TestDBTokenStore_ValidateInvalid(t *testing.T) {
	ts, _, cleanup := newTestDBTokenStore(t)
	defer cleanup()

	if _, e := ts.Validate(""); e == nil {
		t.Fatal("expected error for empty token")
	}
	if _, e := ts.Validate("does-not-exist"); e == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestDBTokenStore_DeletedUserInvalidatesToken(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	if _, e := userDAO.AddUser(types.User{Username: "dave", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}
	s := types.Principal{AuthType: types.AuthTypeToken}
	s.User = types.User{Username: "dave"}
	tok, _ := ts.Create(s)

	if e := userDAO.DeleteUser("dave"); e != nil {
		t.Fatalf("DeleteUser: %v", e)
	}
	if _, e := ts.Validate(tok.Token); e == nil {
		t.Fatal("expected token to be invalid after the user is deleted")
	}
}

func TestDBTokenStore_RefreshDoesNotRestoreRevokedSession(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	if _, e := userDAO.AddUser(types.User{Username: "erin", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}

	tok, e := ts.Create(types.Principal{
		User:     types.User{Username: "erin"},
		AuthType: types.AuthTypeToken,
	})
	if e != nil {
		t.Fatalf("Create: %v", e)
	}
	hash := hashToken(tok.Token)
	item, ok := ts.getCached(hash)
	if !ok {
		t.Fatal("expected session in cache")
	}
	if e := ts.Revoke(tok.Token); e != nil {
		t.Fatalf("Revoke: %v", e)
	}

	// Simulate a validation that loaded the cache immediately before another
	// request revoked the database row and is now attempting a sliding refresh.
	item.expiresAt = time.Now().Add(time.Second).Unix()
	_, exists := ts.maybeRefresh(hash, item, time.Now())
	if exists {
		t.Fatal("refresh reported that the revoked session still exists")
	}
	if _, ok := ts.getCached(hash); ok {
		t.Fatal("revoked session was restored to the cache")
	}
}

func TestDBTokenStore_RefreshUsesValidityAsSlidingLifetime(t *testing.T) {
	ts, userDAO, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	if _, e := userDAO.AddUser(types.User{Username: "frank", Password: "p"}); e != nil {
		t.Fatalf("AddUser: %v", e)
	}

	ts.validity = 10 * time.Minute
	tok, e := ts.Create(types.Principal{
		User:     types.User{Username: "frank"},
		AuthType: types.AuthTypeToken,
	})
	if e != nil {
		t.Fatalf("Create: %v", e)
	}
	hash := hashToken(tok.Token)
	item, ok := ts.getCached(hash)
	if !ok {
		t.Fatal("expected session in cache")
	}

	now := time.Now()
	item.expiresAt = now.Add(time.Minute).Unix()
	expiresAt, exists := ts.maybeRefresh(hash, item, now)
	if !exists {
		t.Fatal("session unexpectedly disappeared during refresh")
	}
	want := now.Add(ts.validity).Unix()
	if expiresAt != want {
		t.Fatalf("refreshed expiry = %d, want %d", expiresAt, want)
	}
}

func TestDBTokenStore_CacheEvictsLeastRecentlyUsedSession(t *testing.T) {
	ts, _, cleanup := newTestDBTokenStore(t)
	defer cleanup()
	ts.cache = utils.NewKVCache[dbTokenCacheItem](2, 0)

	tokens := make([]types.Token, 3)
	for i := range tokens {
		var e error
		tokens[i], e = ts.Create(types.Principal{})
		if e != nil {
			t.Fatalf("Create token %d: %v", i, e)
		}
	}

	if _, ok := ts.getCached(hashToken(tokens[0].Token)); ok {
		t.Fatal("least recently used session was not evicted")
	}
	for _, token := range tokens[1:] {
		if _, ok := ts.getCached(hashToken(token.Token)); !ok {
			t.Fatal("recent session was unexpectedly evicted")
		}
	}
	cacheLen := ts.cache.Len()
	if cacheLen != 2 {
		t.Fatalf("cache length = %d, want 2", cacheLen)
	}
}
