package driveutil

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactCacheAtomicWriteAndOpen(t *testing.T) {
	cache, err := NewArtifactCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := cache.ItemPath("archive-index")
	if err := cache.Write(path, func(w io.Writer) error {
		_, err := io.WriteString(w, "metadata")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	f, exists, err := cache.OpenIfExists(path)
	if err != nil || !exists {
		t.Fatalf("OpenIfExists() = exists=%v err=%v", exists, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "metadata" {
		t.Fatalf("data = %q", data)
	}
}

func TestArtifactCacheStartupCleanupAndLock(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewArtifactCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"stale.lock", "stale.tmp", "empty"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	lock, acquired, err := cache.TryLock(filepath.Join(dir, "active.lock"))
	if err != nil || !acquired {
		t.Fatal(err)
	}
	_, acquired, err = cache.TryLock(filepath.Join(dir, "active.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if acquired {
		t.Fatal("second TryLock acquired an existing lock")
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	removed, err := cache.CleanStartup()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 4 {
		t.Fatalf("removed = %d, want 4", removed)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep")); err != nil {
		t.Fatal(err)
	}
}
