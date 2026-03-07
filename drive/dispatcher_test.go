package drive

import (
	"bytes"
	"context"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/storage"
	"go-drive/testutil"
	"os"
	"path/filepath"
	"testing"
)

// Import fs so its init() registers the "fs" drive type.
import _ "go-drive/drive/fs"

func TestMain(m *testing.M) {
	_, cleanup := testutil.GetSharedTestConfig()
	defer cleanup()
	os.Exit(m.Run())
}

// newTestDispatcher creates a DispatcherDrive with real fs drives under DataDir/local.
// driveNames are used as both dispatcher drive names and fs path segments (e.g. "driveA" -> data/local/driveA).
// Returns dispatcher, PathMountDAO, config, and cleanup.
func newTestDispatcher(t *testing.T, driveNames []string) (*DispatcherDrive, *storage.PathMountDAO, common.Config, func()) {
	t.Helper()
	config := testutil.DefaultTestConfig()
	ch := registry.NewComponentHolder()
	db, e := storage.NewDB(config, ch)
	if e != nil {
		t.Fatalf("NewDB: %v", e)
	}
	mountDAO := storage.NewPathMountDAO(db, ch)
	dispatcher := NewDispatcherDrive(mountDAO, config)

	localRoot, e := config.GetDir("local", true)
	if e != nil {
		t.Fatalf("GetDir local: %v", e)
	}
	for _, name := range driveNames {
		if e := os.MkdirAll(filepath.Join(localRoot, name), 0755); e != nil {
			t.Fatalf("MkdirAll %s: %v", name, e)
		}
	}

	cfg := drive_util.GetDrive("fs", config)
	if cfg == nil {
		t.Fatal("fs drive not registered")
	}
	driveUtils := drive_util.DriveUtils{Config: config}
	drives := make(map[string]types.IDrive, len(driveNames))
	ctx := context.Background()
	for _, name := range driveNames {
		d, e := cfg.Factory.Create(ctx, types.SM{"path": name}, driveUtils)
		if e != nil {
			t.Fatalf("Create fs drive %s: %v", name, e)
		}
		drives[name] = d
	}
	dispatcher.setDrives(drives)

	cleanupFn := func() {
		_ = dispatcher.Dispose()
		_ = db.Dispose()
		_ = ch.Dispose()
	}
	return dispatcher, mountDAO, config, cleanupFn
}

// --- 3.1 resolve: path to drive ---

func TestDispatcher_Get_ResolvePathToDrive(t *testing.T) {
	d, _, _, cleanup := newTestDispatcher(t, []string{"driveA"})
	defer cleanup()
	ctx := context.Background()

	// Root path returns root entry
	ent, e := d.Get(ctx, "")
	if e != nil {
		t.Fatalf("Get root: %v", e)
	}
	if ent.Path() != "" || ent.Name() != "" {
		t.Errorf("root entry path=%q name=%q", ent.Path(), ent.Name())
	}

	// Resolve to drive root
	ent, e = d.Get(ctx, "driveA")
	if e != nil {
		t.Fatalf("Get driveA: %v", e)
	}
	if ent.Path() != "driveA" {
		t.Errorf("Get driveA path=%q", ent.Path())
	}

	// Resolve to path under drive (create a dir first)
	_, _ = d.MakeDir(ctx, "driveA/sub")
	ent, e = d.Get(ctx, "driveA/sub")
	if e != nil {
		t.Fatalf("Get driveA/sub: %v", e)
	}
	if ent.Path() != "driveA/sub" {
		t.Errorf("Get driveA/sub path=%q", ent.Path())
	}

	// Non-existent drive -> NotFound
	_, e = d.Get(ctx, "noSuchDrive/x")
	if e == nil || !err.IsNotFoundError(e) {
		t.Errorf("Get noSuchDrive/x: want NotFound, got %v", e)
	}
}

// --- 3.2 resolveMount / custom mount points ---

func TestDispatcher_ResolveMount_OneLevel(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	// Create driveB/sub so we can mount it
	_, _ = d.MakeDir(ctx, "driveB/sub")
	// Mount: at path "driveA", name "m1", mountAt "driveB/sub"
	p := "driveA"
	mounts := []types.PathMount{{
		Path:    &p,
		Name:    "m1",
		MountAt: "driveB/sub",
	}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}

	// Request driveA/m1/x should resolve to driveB, path sub/x
	ent, e := d.Get(ctx, "driveA/m1")
	if e != nil {
		t.Fatalf("Get driveA/m1: %v", e)
	}
	if ent.Path() != "driveA/m1" {
		t.Errorf("path=%q", ent.Path())
	}

	// Create a file under mount and get it
	_, _ = d.MakeDir(ctx, "driveB/sub/foo")
	ent, e = d.Get(ctx, "driveA/m1/foo")
	if e != nil {
		t.Fatalf("Get driveA/m1/foo: %v", e)
	}
	if ent.Path() != "driveA/m1/foo" {
		t.Errorf("path=%q", ent.Path())
	}
}

func TestDispatcher_ResolveMount_MaxDepthExceeded(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"d1", "d2"})
	defer cleanup()
	ctx := context.Background()

	_, _ = d.MakeDir(ctx, "d1/a")
	_, _ = d.MakeDir(ctx, "d2/b")
	p1 := "d1"
	m1 := types.PathMount{Path: &p1, Name: "a", MountAt: "d2/b"}
	p2 := "d2"
	m2 := types.PathMount{Path: &p2, Name: "b", MountAt: "d1/a"}
	if e := mountDAO.SaveMounts([]types.PathMount{m1, m2}, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}
	// Chain: d1/a -> d2/b -> d1/a -> ... ; requesting d1/a/x will eventually exceed max depth
	_, e := d.Get(ctx, "d1/a/x")
	if e == nil {
		t.Fatal("expected error for max mount depth")
	}
	if !bytes.Contains([]byte(e.Error()), []byte("maximum mounting depth")) {
		t.Errorf("expected max depth error, got: %v", e)
	}
}

// --- 3.3 Copy/Move/Delete with mount points ---

func TestDispatcher_Copy_NoMount_TargetExists_Renames(t *testing.T) {
	d, _, config, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	// Create source file on driveA
	srcContent := []byte("hello")
	_, e := d.Save(taskCtx, "driveA/src.txt", int64(len(srcContent)), true, bytes.NewReader(srcContent))
	if e != nil {
		t.Fatalf("Save src: %v", e)
	}
	// Create existing file at target on driveB (DataDir/local/driveB/to.txt)
	localB := filepath.Join(config.DataDir, "local", "driveB")
	if e := os.MkdirAll(localB, 0755); e != nil {
		t.Fatalf("Mkdir driveB: %v", e)
	}
	if e := os.WriteFile(filepath.Join(localB, "to.txt"), []byte("old"), 0644); e != nil {
		t.Fatalf("Write to.txt: %v", e)
	}

	from, _ := d.Get(ctx, "driveA/src.txt")
	ent, e := d.Copy(taskCtx, from, "driveB/to.txt", false)
	if e != nil {
		t.Fatalf("Copy: %v", e)
	}
	// Should be renamed to to_1.txt
	if ent.Path() != "driveB/to_1.txt" {
		t.Errorf("Copy result path=%q, want driveB/to_1.txt", ent.Path())
	}
}

func TestDispatcher_Delete_PathWithMount_RemovesMountOnly(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"delA", "delB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}
	_, _ = d.MakeDir(ctx, "delB/sub")
	p := "delA"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "delB/sub"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}
	if e := d.Delete(taskCtx, "delA/m1"); e != nil {
		t.Fatalf("Delete delA/m1: %v", e)
	}
	all, _ := mountDAO.GetMounts()
	if len(all) != 0 {
		t.Errorf("expected no mounts after delete, got %d", len(all))
	}
	_, errGet := d.Get(ctx, "delB/sub")
	if errGet != nil {
		t.Errorf("delB/sub should still exist: %v", errGet)
	}
}

func TestDispatcher_Move_OrdinaryFile(t *testing.T) {
	d, _, _, cleanup := newTestDispatcher(t, []string{"driveA"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.Save(taskCtx, "driveA/a.txt", 5, true, bytes.NewReader([]byte("hello")))
	from, _ := d.Get(ctx, "driveA/a.txt")
	_, e := d.Move(taskCtx, from, "driveA/b.txt", false)
	if e != nil {
		t.Fatalf("Move: %v", e)
	}
	_, errGet := d.Get(ctx, "driveA/a.txt")
	if !err.IsNotFoundError(errGet) {
		t.Errorf("source should be gone: %v", errGet)
	}
	_, errGet = d.Get(ctx, "driveA/b.txt")
	if errGet != nil {
		t.Errorf("dest should exist: %v", errGet)
	}
}

// --- 3.4 FindNonExistsEntryName / rename when target exists ---

func TestDispatcher_Save_TargetExists_NoOverride_Renames(t *testing.T) {
	d, _, config, cleanup := newTestDispatcher(t, []string{"driveA"})
	defer cleanup()
	taskCtx := task.DummyContext()

	localA := filepath.Join(config.DataDir, "local", "driveA")
	_ = os.MkdirAll(localA, 0755)
	if e := os.WriteFile(filepath.Join(localA, "file.txt"), []byte("old"), 0644); e != nil {
		t.Fatalf("write file: %v", e)
	}

	ent, e := d.Save(taskCtx, "driveA/file.txt", 6, false, bytes.NewReader([]byte("newcon")))
	if e != nil {
		t.Fatalf("Save: %v", e)
	}
	if ent.Path() != "driveA/file_1.txt" {
		t.Errorf("path=%q, want driveA/file_1.txt", ent.Path())
	}
}

func TestDispatcher_Upload_TargetExists_NoOverride_ReturnsNewPath(t *testing.T) {
	d, _, config, cleanup := newTestDispatcher(t, []string{"driveA"})
	defer cleanup()
	ctx := context.Background()

	localA := filepath.Join(config.DataDir, "local", "driveA")
	_ = os.MkdirAll(localA, 0755)
	if e := os.WriteFile(filepath.Join(localA, "up.txt"), []byte("x"), 0644); e != nil {
		t.Fatalf("write file: %v", e)
	}

	cfg, e := d.Upload(ctx, "driveA/up.txt", 10, false, nil)
	if e != nil {
		t.Fatalf("Upload: %v", e)
	}
	if cfg != nil && cfg.Path != "" && cfg.Path != "driveA/up.txt" {
		// When renamed, Path should be the new path
		if cfg.Path != "driveA/up_1.txt" {
			t.Errorf("Upload Path=%q, expect driveA/up_1.txt when renamed", cfg.Path)
		}
	}
}

// --- 3.5 List ---

func TestDispatcher_List_Root_ReturnsDrives(t *testing.T) {
	d, _, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	entries, e := d.List(ctx, "")
	if e != nil {
		t.Fatalf("List root: %v", e)
	}
	if len(entries) != 2 {
		t.Fatalf("List root: want 2 entries, got %d", len(entries))
	}
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["driveA"] || !names["driveB"] {
		t.Errorf("entries %v", names)
	}
}

func TestDispatcher_List_DriveNoMount_ReturnsDriveContents(t *testing.T) {
	d, _, _, cleanup := newTestDispatcher(t, []string{"listOnly"})
	defer cleanup()
	ctx := context.Background()
	_, _ = d.MakeDir(ctx, "listOnly/foo")
	entries, e := d.List(ctx, "listOnly")
	if e != nil {
		t.Fatalf("List listOnly: %v", e)
	}
	var listNames []string
	for _, ent := range entries {
		listNames = append(listNames, ent.Name())
	}
	if len(entries) != 1 || entries[0].Name() != "foo" {
		t.Errorf("List listOnly: got %d entries, names %v", len(entries), listNames)
	}
}

func TestDispatcher_List_DriveWithMount_IncludesMountedEntry(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"listA", "listB"})
	defer cleanup()
	ctx := context.Background()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}
	_, _ = d.MakeDir(ctx, "listA/real")
	_, _ = d.MakeDir(ctx, "listB/sub")
	p := "listA"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "listB/sub"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}

	entries, e := d.List(ctx, "listA")
	if e != nil {
		t.Fatalf("List listA: %v", e)
	}
	names := make(map[string]bool)
	for _, ent := range entries {
		names[ent.Name()] = true
	}
	if !names["real"] {
		t.Error("expected entry real")
	}
	// Mounted entry m1 (MountAt listB/sub) should appear when path_mount path matches list path.
	if names["m1"] {
		for _, ent := range entries {
			if ent.Name() == "m1" {
				if ent.Meta().Props["mountAt"] != "listB/sub" {
					t.Errorf("m1 mountAt=%v", ent.Meta().Props["mountAt"])
				}
				break
			}
		}
	}
}

func TestDispatcher_List_MountTargetNotFound_SkipsWithoutPanic(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	// Mount m1 to driveB/nonexistent (no such path on driveB)
	p := "driveA"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "driveB/nonexistent"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}

	entries, e := d.List(ctx, "driveA")
	if e != nil {
		t.Fatalf("List driveA: %v", e)
	}
	// m1 should not appear (Get returns NotFound for driveB/nonexistent)
	for _, ent := range entries {
		if ent.Name() == "m1" {
			t.Error("m1 should not appear when MountAt target does not exist")
			break
		}
	}
}
