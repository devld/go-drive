package storage

import (
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/testutil"
	"os"
	"testing"
)

func newTestDB(t *testing.T) (*DB, *registry.ComponentsHolder, func()) {
	t.Helper()
	config := testutil.DefaultTestConfig() // uses shared config from GetSharedTestConfig
	ch := registry.NewComponentHolder()
	db, err := NewDB(config, ch)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	return db, ch, func() {
		_ = db.Dispose()
		_ = ch.Dispose()
	}
}

func TestMain(m *testing.M) {
	_, cleanup := testutil.GetSharedTestConfig()
	defer cleanup()
	os.Exit(m.Run())
}

func TestNewDB(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("DB is nil")
	}
	if db.C() == nil {
		t.Fatal("DB.C() is nil")
	}
	if ch == nil {
		t.Fatal("ComponentsHolder is nil")
	}
}

func TestNewDB_MigrateAndReadWrite(t *testing.T) {
	db, _, cleanup := newTestDB(t)
	defer cleanup()

	// tryInitDbData inserts default admin; verify we can read users
	var n int64
	if err := db.C().Model(&types.User{}).Count(&n).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if n < 1 {
		t.Errorf("expected at least 1 user (default admin), got %d", n)
	}

	// simple write + read on a table that doesn't require init
	opt := types.Option{Key: "test_key", Value: "test_value"}
	if err := db.C().Create(&opt).Error; err != nil {
		t.Fatalf("create option: %v", err)
	}
	var got types.Option
	if err := db.C().Where("key = ?", "test_key").First(&got).Error; err != nil {
		t.Fatalf("read option: %v", err)
	}
	if got.Value != "test_value" {
		t.Errorf("option value: got %q, want %q", got.Value, "test_value")
	}
}
