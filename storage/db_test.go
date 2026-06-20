package storage

import (
	"context"
	"database/sql"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/testutil"
	"os"
	"strings"
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

func TestNewDB_SQLitePragmasApplyToEveryConnection(t *testing.T) {
	db, _, cleanup := newTestDB(t)
	defer cleanup()

	sqlDB, e := db.C().DB()
	if e != nil {
		t.Fatalf("DB: %v", e)
	}
	sqlDB.SetMaxOpenConns(2)
	ctx := context.Background()
	conn1, e := sqlDB.Conn(ctx)
	if e != nil {
		t.Fatalf("first connection: %v", e)
	}
	defer conn1.Close()
	conn2, e := sqlDB.Conn(ctx)
	if e != nil {
		t.Fatalf("second connection: %v", e)
	}
	defer conn2.Close()

	for i, conn := range []*sql.Conn{conn1, conn2} {
		var busyTimeout int
		if e := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); e != nil {
			t.Fatalf("connection %d busy_timeout: %v", i+1, e)
		}
		if busyTimeout != 5000 {
			t.Errorf("connection %d busy_timeout = %d, want 5000", i+1, busyTimeout)
		}

		var journalMode string
		if e := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); e != nil {
			t.Fatalf("connection %d journal_mode: %v", i+1, e)
		}
		if strings.ToLower(journalMode) != "wal" {
			t.Errorf("connection %d journal_mode = %q, want wal", i+1, journalMode)
		}
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
