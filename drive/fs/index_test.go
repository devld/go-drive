package fs

import (
	"bytes"
	"context"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestDrive(t *testing.T) (d *Drive, root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "root")
	outside = filepath.Join(base, "outside")
	if e := os.MkdirAll(root, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(outside, 0755); e != nil {
		t.Fatal(e)
	}
	abs, e := filepath.Abs(root)
	if e != nil {
		t.Fatal(e)
	}
	return &Drive{path: abs}, abs, outside
}

func TestFsDriveBasicRoundTrip(t *testing.T) {
	d, _, _ := newTestDrive(t)
	ctx := context.Background()
	taskCtx := task.DummyContext()

	dir, e := d.MakeDir(ctx, "users")
	if e != nil {
		t.Fatalf("MakeDir users: %v", e)
	}
	if dir.Path() != "users" || !dir.Type().IsDir() {
		t.Fatalf("MakeDir users entry = path %q type %v", dir.Path(), dir.Type())
	}

	saved, e := d.Save(taskCtx, "users/hello.txt", 5, true, bytes.NewReader([]byte("hello")))
	if e != nil {
		t.Fatalf("Save: %v", e)
	}
	if saved.Path() != "users/hello.txt" {
		t.Fatalf("Save path = %q", saved.Path())
	}

	got, e := d.Get(ctx, "users/hello.txt")
	if e != nil {
		t.Fatalf("Get: %v", e)
	}
	reader, e := got.GetReader(ctx, -1, -1)
	if e != nil {
		t.Fatalf("GetReader: %v", e)
	}
	body, e := io.ReadAll(reader)
	_ = reader.Close()
	if e != nil {
		t.Fatal(e)
	}
	if string(body) != "hello" {
		t.Fatalf("content = %q", body)
	}

	entries, e := d.List(ctx, "users")
	if e != nil {
		t.Fatalf("List: %v", e)
	}
	if len(entries) != 1 || entries[0].Path() != "users/hello.txt" {
		t.Fatalf("List = %v", entries)
	}

	moved, e := d.Move(taskCtx, got, "users/moved.txt", false)
	if e != nil {
		t.Fatalf("Move: %v", e)
	}
	if moved.Path() != "users/moved.txt" {
		t.Fatalf("Move path = %q", moved.Path())
	}

	if e := d.Delete(taskCtx, "users/moved.txt"); e != nil {
		t.Fatalf("Delete: %v", e)
	}
	if _, e := d.Get(ctx, "users/moved.txt"); e == nil || !err.IsNotFoundError(e) {
		t.Fatalf("Get after delete: %v", e)
	}
}

func TestFsDriveCleanPathKeepsWritesInsideRoot(t *testing.T) {
	d, root, outside := newTestDrive(t)
	ctx := context.Background()
	taskCtx := task.DummyContext()

	outsideFile := filepath.Join(outside, "victim.txt")
	if e := os.WriteFile(outsideFile, []byte("keep"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := d.MakeDir(ctx, "users"); e != nil {
		t.Fatalf("MakeDir users: %v", e)
	}
	if _, e := d.MakeDir(ctx, "users/bob"); e != nil {
		t.Fatalf("MakeDir bob: %v", e)
	}

	payloads := []string{
		`../outside/evil.txt`,
		`..\\outside\\evil.txt`,
		`users/bob/..\\..\\..\\outside\\evil.txt`,
		`users/bob/../../../outside/evil.txt`,
		`C:\Windows\Temp\evil.txt`,
	}

	for _, path := range payloads {
		resolved := d.getPath(path)
		if !isInsideRoot(root, resolved) {
			t.Fatalf("getPath(%q) = %q, want inside %q", path, resolved, root)
		}

		_, _ = d.Save(taskCtx, path, 4, true, bytes.NewReader([]byte("evil")))
		if !fileStillContains(outsideFile, "keep") {
			t.Fatalf("Save(%q) mutated outside file", path)
		}
		if _, e := os.Stat(filepath.Join(outside, "evil.txt")); e == nil {
			t.Fatalf("Save(%q) created outside file", path)
		}

		_ = d.Delete(taskCtx, path)
		if !fileStillContains(outsideFile, "keep") {
			t.Fatalf("Delete(%q) removed or mutated the outside victim", path)
		}
	}

	entries, e := os.ReadDir(outside)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 1 || entries[0].Name() != "victim.txt" {
		t.Fatalf("outside dir changed: %v", names(entries))
	}
}

func isInsideRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, e := filepath.Rel(root, target)
	if e != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func fileStillContains(path, want string) bool {
	got, e := os.ReadFile(path)
	return e == nil && string(got) == want
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
}
