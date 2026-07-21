package drive

import (
	"bytes"
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/storage"
	"go-drive/testutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	// Import fs so its init() registers the "fs" drive type.
	_ "go-drive/drive/fs"
)

func TestMain(m *testing.M) {
	_, cleanup := testutil.GetSharedTestConfig()
	defer cleanup()
	os.Exit(m.Run())
}

// newTestDispatcher creates the mounted root namespace with real fs drives under
// DataDir/local. driveNames are used as both dispatcher drive names and
// fs path segments (e.g. "driveA" -> data/local/driveA).
func newTestDispatcher(t *testing.T, driveNames []string) (
	d *PathMountOverlayDrive,
	mountDAO *storage.PathMountDAO,
	config common.Config,
	cleanup func(),
) {
	t.Helper()
	config = testutil.DefaultTestConfig()
	ch := registry.NewComponentHolder()
	db, e := storage.NewDB(config, ch)
	if e != nil {
		t.Fatalf("NewDB: %v", e)
	}
	mountDAO = storage.NewPathMountDAO(db, ch)
	dispatcher := NewDispatcherDrive(config)
	d = NewPathMountOverlayDrive(dispatcher, mountDAO)

	localRoot, e := config.GetDir("local", true)
	if e != nil {
		t.Fatalf("GetDir local: %v", e)
	}
	for _, name := range driveNames {
		if e := os.MkdirAll(filepath.Join(localRoot, name), 0755); e != nil {
			t.Fatalf("MkdirAll %s: %v", name, e)
		}
	}

	cfg := driveutil.GetDrive("fs", config)
	if cfg == nil {
		t.Fatal("fs drive not registered")
	}
	driveUtils := driveutil.DriveUtils{Config: config}
	drives := make(map[string]types.IDrive, len(driveNames))
	ctx := context.Background()
	for _, name := range driveNames {
		drv, e := cfg.Factory.Create(ctx, types.SM{"path": name}, driveUtils)
		if e != nil {
			t.Fatalf("Create fs drive %s: %v", name, e)
		}
		drives[name] = drv
	}
	dispatcher.setDrives(drives)
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}

	cleanup = func() {
		_ = dispatcher.Dispose()
		_ = db.Dispose()
		_ = ch.Dispose()
	}
	return
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

func TestDispatcher_IsMountAgnostic(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	if _, e := d.lower.MakeDir(ctx, "driveA/mounted"); e != nil {
		t.Fatalf("MakeDir lower mount path: %v", e)
	}
	if _, e := d.lower.MakeDir(ctx, "driveB/target"); e != nil {
		t.Fatalf("MakeDir target: %v", e)
	}
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{{Path: &p, Name: "mounted", MountAt: "driveB/target"}})

	lowerEntry, e := d.lower.Get(ctx, "driveA/mounted")
	if e != nil {
		t.Fatalf("dispatcher Get: %v", e)
	}
	if lowerEntry.Path() != "driveA/mounted" {
		t.Fatalf("dispatcher path=%q, want physical path", lowerEntry.Path())
	}

	overlayEntry, e := d.Get(ctx, "driveA/mounted")
	if e != nil {
		t.Fatalf("overlay Get: %v", e)
	}
	if overlayEntry.Meta().Props["mountAt"] != "driveB/target" {
		t.Fatalf("overlay mountAt=%v", overlayEntry.Meta().Props["mountAt"])
	}
}

func TestPathMountOverlay_MissingTargetDoesNotFallBackToLower(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	if _, e := d.lower.MakeDir(ctx, "driveA/mounted"); e != nil {
		t.Fatalf("MakeDir lower mount path: %v", e)
	}
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{{Path: &p, Name: "mounted", MountAt: "driveB/missing"}})

	if _, e := d.Get(ctx, "driveA/mounted"); !err.IsNotFoundError(e) {
		t.Fatalf("Get missing mount target: want NotFound, got %v", e)
	}
	if _, e := d.lower.Get(ctx, "driveA/mounted"); e != nil {
		t.Fatalf("physical lower entry should still exist: %v", e)
	}
}

func TestPathMountOverlay_EntriesBelongToTopLevelDrive(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	if _, e := d.MakeDir(ctx, "driveA/ordinary"); e != nil {
		t.Fatalf("MakeDir ordinary: %v", e)
	}
	if _, e := d.MakeDir(ctx, "driveB/target"); e != nil {
		t.Fatalf("MakeDir target: %v", e)
	}
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{{Path: &p, Name: "mounted", MountAt: "driveB/target"}})

	for _, path := range []string{"driveA/ordinary", "driveA/mounted"} {
		entry, e := d.Get(ctx, path)
		if e != nil {
			t.Fatalf("Get %s: %v", path, e)
		}
		if entry.Drive() != d {
			t.Errorf("Get %s returned entry owned by %T, want PathMountOverlayDrive", path, entry.Drive())
		}
	}

	entries, e := d.List(ctx, "driveA")
	if e != nil {
		t.Fatalf("List driveA: %v", e)
	}
	for _, entry := range entries {
		if entry.Drive() != d {
			t.Errorf("List entry %s owned by %T", entry.Path(), entry.Drive())
		}
	}
}

func TestPathMountOverlay_NestedMountThroughPhantomTarget(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB", "driveC"})
	defer cleanup()
	ctx := context.Background()

	if _, e := d.MakeDir(ctx, "driveC/target"); e != nil {
		t.Fatalf("MakeDir final target: %v", e)
	}
	if _, e := d.Save(task.DummyContext(), "driveC/target/file.txt", 1, true, bytes.NewReader([]byte("x"))); e != nil {
		t.Fatalf("Save final target file: %v", e)
	}
	rootA := "driveA"
	phantomB := "driveB/phantom"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &rootA, Name: "mounted", MountAt: "driveB/phantom"},
		{Path: &phantomB, Name: "nested", MountAt: "driveC/target"},
	})

	entries, e := d.List(ctx, "driveA/mounted")
	if e != nil {
		t.Fatalf("List nested phantom target: %v", e)
	}
	if !entryNames(entries)["nested"] {
		t.Fatal("nested mount in phantom target is not visible")
	}
	entry, e := d.Get(ctx, "driveA/mounted/nested/file.txt")
	if e != nil {
		t.Fatalf("Get through nested phantom target: %v", e)
	}
	if entry.Path() != "driveA/mounted/nested/file.txt" {
		t.Fatalf("path=%q", entry.Path())
	}
}

func TestPathMountOverlay_ListMountCreatedInsideMountedDirectory(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB", "driveC"})
	defer cleanup()
	ctx := context.Background()

	for _, target := range []string{"driveB/root", "driveC/target"} {
		if _, e := d.MakeDir(ctx, target); e != nil {
			t.Fatalf("MakeDir %s: %v", target, e)
		}
	}
	if _, e := d.MakeDir(ctx, "driveB/root/nested"); e != nil {
		t.Fatalf("MakeDir lower name collision: %v", e)
	}

	rootA := "driveA"
	virtualMountParent := "driveA/outer"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &rootA, Name: "outer", MountAt: "driveB/root"},
		// The mount API stores the requested virtual destination path as-is.
		{Path: &virtualMountParent, Name: "nested", MountAt: "driveC/target"},
	})

	entries, e := d.List(ctx, "driveA/outer")
	if e != nil {
		t.Fatalf("List mounted directory: %v", e)
	}
	var nested types.IEntry
	for _, entry := range entries {
		if entry.Name() == "nested" {
			nested = entry
			break
		}
	}
	if nested == nil {
		t.Fatal("mount created inside mounted directory is missing from List")
	}
	if nested.Meta().Props["mountAt"] != "driveC/target" {
		t.Fatalf("listed entry is the lower collision, meta=%v", nested.Meta().Props)
	}
	if _, e := d.Get(ctx, "driveA/outer/nested"); e != nil {
		t.Errorf("listed nested mount does not resolve: %v", e)
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
		t.Fatalf("ReloadMounts: %v", e)
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
		t.Fatalf("ReloadMounts: %v", e)
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
		t.Fatalf("ReloadMounts: %v", e)
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
		t.Fatalf("ReloadMounts: %v", e)
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
		t.Fatalf("ReloadMounts: %v", e)
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

func TestOverlay_List_MountPoint_ExclusiveNoLower(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}

	// Create a real directory "m1" on driveA with a file inside.
	_, _ = d.MakeDir(ctx, "driveA/m1")
	taskCtx := task.DummyContext()
	_, _ = d.Save(taskCtx, "driveA/m1/real_file.txt", 5, true, bytes.NewReader([]byte("hello")))

	// Create driveB/sub with a file — this is the mount target.
	_, _ = d.MakeDir(ctx, "driveB/sub")
	_, _ = d.Save(taskCtx, "driveB/sub/mount_file.txt", 5, true, bytes.NewReader([]byte("world")))

	// Mount driveA/m1 → driveB/sub.
	p := "driveA"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "driveB/sub"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("ReloadMounts: %v", e)
	}

	// List driveA/m1: should ONLY show mount target content (mount_file.txt),
	// NOT the real content (real_file.txt) from the lower layer.
	entries, e := d.List(ctx, "driveA/m1")
	if e != nil {
		t.Fatalf("List driveA/m1: %v", e)
	}
	names := make(map[string]bool)
	for _, ent := range entries {
		names[ent.Name()] = true
	}
	if !names["mount_file.txt"] {
		t.Error("expected mount_file.txt from mount target")
	}
	if names["real_file.txt"] {
		t.Error("real_file.txt from lower layer should NOT appear in exclusive listing")
	}
}

func TestOverlay_List_MountSubpath_ExclusiveNoLower(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}

	// Create a real directory "m1/inner" on driveA with a file.
	_, _ = d.MakeDir(ctx, "driveA/m1")
	_, _ = d.MakeDir(ctx, "driveA/m1/inner")
	taskCtx := task.DummyContext()
	_, _ = d.Save(taskCtx, "driveA/m1/inner/lower.txt", 4, true, bytes.NewReader([]byte("data")))

	// Create driveB/sub/inner with a file — mount target subpath.
	_, _ = d.MakeDir(ctx, "driveB/sub")
	_, _ = d.MakeDir(ctx, "driveB/sub/inner")
	_, _ = d.Save(taskCtx, "driveB/sub/inner/upper.txt", 4, true, bytes.NewReader([]byte("data")))

	// Mount driveA/m1 → driveB/sub.
	p := "driveA"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "driveB/sub"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("ReloadMounts: %v", e)
	}

	// List driveA/m1/inner: subpath of a mount → exclusive, only mount target
	entries, e := d.List(ctx, "driveA/m1/inner")
	if e != nil {
		t.Fatalf("List driveA/m1/inner: %v", e)
	}
	names := make(map[string]bool)
	for _, ent := range entries {
		names[ent.Name()] = true
	}
	if !names["upper.txt"] {
		t.Error("expected upper.txt from mount target subpath")
	}
	if names["lower.txt"] {
		t.Error("lower.txt from lower layer should NOT appear in exclusive listing")
	}
}

// --- 4. Mount-priority operations ---

func setupMount(t *testing.T, d *PathMountOverlayDrive, mountDAO *storage.PathMountDAO, mounts []types.PathMount) {
	t.Helper()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
	}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("reloadMounts: %v", e)
	}
}

func entryNames(entries []types.IEntry) map[string]bool {
	m := make(map[string]bool, len(entries))
	for _, e := range entries {
		m[e.Name()] = true
	}
	return m
}

// --- 4.1 Save / MakeDir / Upload through mount ---

func TestMount_Save_ThroughMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "driveB/sub")
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "driveB/sub"},
	})

	content := []byte("mount-save")
	ent, e := d.Save(taskCtx, "driveA/m1/file.txt", int64(len(content)), true, bytes.NewReader(content))
	if e != nil {
		t.Fatalf("Save: %v", e)
	}
	if ent.Path() != "driveA/m1/file.txt" {
		t.Errorf("virtual path=%q, want driveA/m1/file.txt", ent.Path())
	}

	real, e := d.Get(ctx, "driveB/sub/file.txt")
	if e != nil {
		t.Fatalf("Get real path: %v", e)
	}
	if real.Size() != int64(len(content)) {
		t.Errorf("real file size=%d, want %d", real.Size(), len(content))
	}
}

func TestMount_MakeDir_ThroughMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	_, _ = d.MakeDir(ctx, "driveB/sub")
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "driveB/sub"},
	})

	ent, e := d.MakeDir(ctx, "driveA/m1/newdir")
	if e != nil {
		t.Fatalf("MakeDir: %v", e)
	}
	if ent.Path() != "driveA/m1/newdir" {
		t.Errorf("virtual path=%q, want driveA/m1/newdir", ent.Path())
	}
	if !ent.Type().IsDir() {
		t.Error("expected directory type")
	}

	_, e = d.Get(ctx, "driveB/sub/newdir")
	if e != nil {
		t.Fatalf("real dir driveB/sub/newdir should exist: %v", e)
	}
}

func TestMount_Upload_ThroughMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()

	_, _ = d.MakeDir(ctx, "driveB/sub")
	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "driveB/sub"},
	})

	cfg, e := d.Upload(ctx, "driveA/m1/file.txt", 100, true, nil)
	if e != nil {
		t.Fatalf("Upload through mount: %v", e)
	}
	if cfg == nil {
		t.Fatal("expected non-nil upload config")
	}
}

// --- 4.2 List: merge at parent, exclusive at/under mount (user's example) ---

func TestMount_List_UserExample_MergeAndExclusive(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"a", "c"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "a/b")
	_, _ = d.Save(taskCtx, "a/b/inside.txt", 5, true, bytes.NewReader([]byte("hello")))
	_, _ = d.Save(taskCtx, "c/real.txt", 4, true, bytes.NewReader([]byte("real")))

	p := "c"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "d", MountAt: "a/b"},
	})

	// List c → merge: real.txt (from drive c) + d (mount entry)
	entries, e := d.List(ctx, "c")
	if e != nil {
		t.Fatalf("List c: %v", e)
	}
	names := entryNames(entries)
	if !names["real.txt"] {
		t.Error("expected real.txt from drive c")
	}
	if !names["d"] {
		t.Error("expected mount entry d")
	}

	// List c/d → exclusive: only a/b contents (inside.txt), no lower merge
	entries, e = d.List(ctx, "c/d")
	if e != nil {
		t.Fatalf("List c/d: %v", e)
	}
	names = entryNames(entries)
	if !names["inside.txt"] {
		t.Error("expected inside.txt from mount target a/b")
	}
	if names["real.txt"] {
		t.Error("real.txt from drive c should NOT appear in exclusive listing of c/d")
	}
}

// --- 4.3 Delete variations ---

func TestMount_Delete_InsideMount_DeletesRealFile(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "driveB/sub")
	_, _ = d.Save(taskCtx, "driveB/sub/file.txt", 7, true, bytes.NewReader([]byte("content")))

	p := "driveA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "driveB/sub"},
	})

	if e := d.Delete(taskCtx, "driveA/m1/file.txt"); e != nil {
		t.Fatalf("Delete: %v", e)
	}

	_, e := d.Get(ctx, "driveB/sub/file.txt")
	if !err.IsNotFoundError(e) {
		t.Errorf("real file driveB/sub/file.txt should be gone: %v", e)
	}

	mounts, _ := mountDAO.GetMounts()
	if len(mounts) == 0 {
		t.Error("mount should still exist after deleting file inside mount")
	}
}

func TestMount_Delete_ParentContainingMountChildren(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "driveA/parent")
	_, _ = d.Save(taskCtx, "driveA/parent/real.txt", 4, true, bytes.NewReader([]byte("data")))
	_, _ = d.MakeDir(ctx, "driveB/target")
	_, _ = d.Save(taskCtx, "driveB/target/mounted.txt", 4, true, bytes.NewReader([]byte("file")))

	p := "driveA/parent"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "driveB/target"},
	})

	if e := d.Delete(taskCtx, "driveA/parent"); e != nil {
		t.Fatalf("Delete: %v", e)
	}

	mounts, _ := mountDAO.GetMounts()
	if len(mounts) != 0 {
		t.Errorf("expected 0 mounts after delete, got %d", len(mounts))
	}

	// Mount target should still exist — only mount record removed
	_, e := d.Get(ctx, "driveB/target")
	if e != nil {
		t.Errorf("driveB/target should still exist: %v", e)
	}
	_, e = d.Get(ctx, "driveB/target/mounted.txt")
	if e != nil {
		t.Errorf("driveB/target/mounted.txt should still exist: %v", e)
	}

	// Real directory driveA/parent should be deleted
	_, e = d.Get(ctx, "driveA/parent")
	if !err.IsNotFoundError(e) {
		t.Errorf("driveA/parent should be gone: %v", e)
	}
}

// --- 4.4 Copy with mount ---

func TestMount_Copy_MountPoint_RemountsAtDest(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"cpA", "cpB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "cpB/sub")
	_, _ = d.Save(taskCtx, "cpB/sub/file.txt", 5, true, bytes.NewReader([]byte("hello")))

	p := "cpA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "cpB/sub"},
	})

	from, e := d.Get(ctx, "cpA/m1")
	if e != nil {
		t.Fatalf("Get from: %v", e)
	}
	_, e = d.Copy(taskCtx, from, "cpA/m2", true)
	if e != nil {
		t.Fatalf("Copy: %v", e)
	}

	mounts, _ := mountDAO.GetMounts()
	newFound := false
	origFound := false
	for _, mt := range mounts {
		if *mt.Path == "cpA" && mt.Name == "m2" && mt.MountAt == "cpB/sub" {
			newFound = true
		}
		if *mt.Path == "cpA" && mt.Name == "m1" && mt.MountAt == "cpB/sub" {
			origFound = true
		}
	}
	if !newFound {
		t.Error("expected new mount cpA/m2 → cpB/sub")
	}
	if !origFound {
		t.Error("original mount cpA/m1 should still exist after copy")
	}

	// Accessible via new mount
	ent, e := d.Get(ctx, "cpA/m2/file.txt")
	if e != nil {
		t.Fatalf("Get cpA/m2/file.txt: %v", e)
	}
	if ent.Path() != "cpA/m2/file.txt" {
		t.Errorf("path=%q", ent.Path())
	}
}

// --- 4.5 Move with mount ---

func TestMount_Move_MountPoint_RelocatesMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"mvA", "mvB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "mvB/sub")
	_, _ = d.Save(taskCtx, "mvB/sub/file.txt", 5, true, bytes.NewReader([]byte("hello")))

	p := "mvA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "mvB/sub"},
	})

	from, e := d.Get(ctx, "mvA/m1")
	if e != nil {
		t.Fatalf("Get from: %v", e)
	}
	_, e = d.Move(taskCtx, from, "mvA/m2", true)
	if e != nil {
		t.Fatalf("Move: %v", e)
	}

	mounts, _ := mountDAO.GetMounts()
	for _, mt := range mounts {
		if *mt.Path == "mvA" && mt.Name == "m1" {
			t.Error("old mount mvA/m1 should be gone after move")
		}
	}

	relocated := false
	for _, mt := range mounts {
		if *mt.Path == "mvA" && mt.Name == "m2" && mt.MountAt == "mvB/sub" {
			relocated = true
			break
		}
	}
	if !relocated {
		t.Error("expected relocated mount mvA/m2 → mvB/sub")
	}

	_, e = d.Get(ctx, "mvA/m1")
	if !err.IsNotFoundError(e) {
		t.Errorf("mvA/m1 should not exist after move: %v", e)
	}

	ent, e := d.Get(ctx, "mvA/m2/file.txt")
	if e != nil {
		t.Fatalf("Get mvA/m2/file.txt: %v", e)
	}
	if ent.Path() != "mvA/m2/file.txt" {
		t.Errorf("path=%q", ent.Path())
	}
}

func TestOverlay_Get_PhantomDir(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}

	_, _ = d.MakeDir(ctx, "driveB/target")
	p := "driveA/phantom"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "driveB/target"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("ReloadMounts: %v", e)
	}

	// "driveA/phantom" doesn't physically exist, but it should be a phantom dir
	ent, e := d.Get(ctx, "driveA/phantom")
	if e != nil {
		t.Fatalf("Get driveA/phantom: %v", e)
	}
	if !ent.Type().IsDir() {
		t.Error("phantom dir should be a directory")
	}
	if ent.Path() != "driveA/phantom" {
		t.Errorf("phantom path=%q", ent.Path())
	}
}

func TestOverlay_List_PhantomDir(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"driveA", "driveB"})
	defer cleanup()
	ctx := context.Background()
	existing, _ := mountDAO.GetMounts()
	if len(existing) > 0 {
		_ = mountDAO.DeleteMounts(existing)
		_ = d.reloadMounts()
	}

	_, _ = d.MakeDir(ctx, "driveB/target")
	p := "driveA/phantom"
	mounts := []types.PathMount{{Path: &p, Name: "m1", MountAt: "driveB/target"}}
	if e := mountDAO.SaveMounts(mounts, true); e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	if e := d.reloadMounts(); e != nil {
		t.Fatalf("ReloadMounts: %v", e)
	}

	// Listing driveA should synthesize "phantom" as a virtual dir
	entries, e := d.List(ctx, "driveA")
	if e != nil {
		t.Fatalf("List driveA: %v", e)
	}
	found := false
	for _, ent := range entries {
		if ent.Name() == "phantom" {
			found = true
			if !ent.Type().IsDir() {
				t.Error("synthesized phantom should be a directory")
			}
			break
		}
	}
	if !found {
		t.Error("expected synthesized 'phantom' directory in listing")
	}

	// Listing the phantom dir should show the mount
	entries, e = d.List(ctx, "driveA/phantom")
	if e != nil {
		t.Fatalf("List driveA/phantom: %v", e)
	}
	found = false
	for _, ent := range entries {
		if ent.Name() == "m1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mount 'm1' inside phantom directory")
	}
}

// --- 5. Phantom directory write/delete ---

func TestPhantom_Save_MaterializesDir(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"phA", "phB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "phB/target")
	p := "phA/phantom"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "phB/target"},
	})

	// "phA/phantom" is a phantom dir (no real directory on disk).
	// Saving a file into it should auto-create the real directory.
	content := []byte("phantom-write")
	ent, e := d.Save(taskCtx, "phA/phantom/file.txt", int64(len(content)), true, bytes.NewReader(content))
	if e != nil {
		t.Fatalf("Save into phantom dir: %v", e)
	}
	if ent.Path() != "phA/phantom/file.txt" {
		t.Errorf("path=%q, want phA/phantom/file.txt", ent.Path())
	}

	// The file should be accessible both via the dispatcher and via the
	// now-materialized real directory.
	got, e := d.Get(ctx, "phA/phantom/file.txt")
	if e != nil {
		t.Fatalf("Get phA/phantom/file.txt: %v", e)
	}
	if got.Size() != int64(len(content)) {
		t.Errorf("size=%d, want %d", got.Size(), len(content))
	}

	// Listing should now show both the real file and the mount entry.
	entries, e := d.List(ctx, "phA/phantom")
	if e != nil {
		t.Fatalf("List phA/phantom: %v", e)
	}
	names := entryNames(entries)
	if !names["file.txt"] {
		t.Error("expected file.txt in listing")
	}
	if !names["m1"] {
		t.Error("expected mount entry m1 in listing")
	}
}

func TestPhantom_Delete_RemovesMountsOnly(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"pdA", "pdB"})
	defer cleanup()
	ctx := context.Background()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(ctx, "pdB/target")
	_, _ = d.Save(taskCtx, "pdB/target/file.txt", 5, true, bytes.NewReader([]byte("hello")))

	p := "pdA/phantom"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "pdB/target"},
	})

	// "pdA/phantom" is purely virtual — no real directory on disk.
	// Deleting it should remove mount records without error.
	if e := d.Delete(taskCtx, "pdA/phantom"); e != nil {
		t.Fatalf("Delete phantom dir: %v", e)
	}

	// Mount should be gone.
	mounts, _ := mountDAO.GetMounts()
	for _, mt := range mounts {
		if *mt.Path == "pdA/phantom" && mt.Name == "m1" {
			t.Error("mount pdA/phantom/m1 should be removed")
		}
	}

	// Mount target should be untouched.
	_, e := d.Get(ctx, "pdB/target/file.txt")
	if e != nil {
		t.Errorf("pdB/target/file.txt should still exist: %v", e)
	}

	// The phantom dir itself should no longer be visible.
	_, e = d.Get(ctx, "pdA/phantom")
	if !err.IsNotFoundError(e) {
		t.Errorf("pdA/phantom should be gone after delete, got: %v", e)
	}
}

func TestMount_Save_ThroughMount_Rename(t *testing.T) {
	d, mountDAO, config, cleanup := newTestDispatcher(t, []string{"rnA", "rnB"})
	defer cleanup()
	taskCtx := task.DummyContext()

	_, _ = d.MakeDir(context.Background(), "rnB/sub")
	p := "rnA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "m1", MountAt: "rnB/sub"},
	})

	// Pre-create a file at the mount target so that override=false triggers rename.
	localB := filepath.Join(config.DataDir, "local", "rnB", "sub")
	if e := os.WriteFile(filepath.Join(localB, "file.txt"), []byte("old"), 0644); e != nil {
		t.Fatalf("write existing file: %v", e)
	}

	content := []byte("new-content")
	ent, e := d.Save(taskCtx, "rnA/m1/file.txt", int64(len(content)), false, bytes.NewReader(content))
	if e != nil {
		t.Fatalf("Save: %v", e)
	}
	// Should be renamed with the virtual mount path prefix, not the real path.
	if ent.Path() != "rnA/m1/file_1.txt" {
		t.Errorf("path=%q, want rnA/m1/file_1.txt", ent.Path())
	}
}

func TestMount_Copy_MountPoint_RenamesWholeTreeOnConflict(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"cprA", "cprB"})
	defer cleanup()
	ctx := context.Background()

	for _, path := range []string{"cprA/dest", "cprB/root", "cprB/nested"} {
		if _, e := d.MakeDir(ctx, path); e != nil {
			t.Fatalf("MakeDir %s: %v", path, e)
		}
	}
	sourceParent := "cprA"
	nestedParent := "cprA/source"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &sourceParent, Name: "source", MountAt: "cprB/root"},
		{Path: &nestedParent, Name: "child", MountAt: "cprB/nested"},
	})

	from, e := d.Get(ctx, "cprA/source")
	if e != nil {
		t.Fatalf("Get source mount: %v", e)
	}
	copied, e := d.Copy(task.DummyContext(), from, "cprA/dest", false)
	if e != nil {
		t.Fatalf("Copy: %v", e)
	}
	if copied.Path() != "cprA/dest_1" {
		t.Fatalf("copied path=%q, want cprA/dest_1", copied.Path())
	}

	mounts, e := mountDAO.GetMounts()
	if e != nil {
		t.Fatalf("GetMounts: %v", e)
	}
	want := map[string]string{
		"cprA/dest_1":       "cprB/root",
		"cprA/dest_1/child": "cprB/nested",
	}
	for _, m := range mounts {
		mountPath := filepath.ToSlash(filepath.Join(*m.Path, m.Name))
		if target, ok := want[mountPath]; ok && target == m.MountAt {
			delete(want, mountPath)
		}
		if mountPath == "cprA/dest/child" {
			t.Error("nested mount was split from the renamed destination root")
		}
	}
	if len(want) != 0 {
		t.Errorf("missing copied mounts: %v", want)
	}
}

func TestMount_Move_MountPoint_ReturnsRenamedDestination(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"mvrA", "mvrB"})
	defer cleanup()
	ctx := context.Background()

	if _, e := d.MakeDir(ctx, "mvrA/dest"); e != nil {
		t.Fatalf("MakeDir destination: %v", e)
	}
	if _, e := d.MakeDir(ctx, "mvrB/target"); e != nil {
		t.Fatalf("MakeDir target: %v", e)
	}
	p := "mvrA"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "source", MountAt: "mvrB/target"},
	})

	from, e := d.Get(ctx, "mvrA/source")
	if e != nil {
		t.Fatalf("Get source mount: %v", e)
	}
	moved, e := d.Move(task.DummyContext(), from, "mvrA/dest", false)
	if e != nil {
		t.Fatalf("Move: %v", e)
	}
	if moved.Path() != "mvrA/dest_1" {
		t.Fatalf("moved path=%q, want mvrA/dest_1", moved.Path())
	}
	if _, e := d.Get(ctx, "mvrA/source"); !err.IsNotFoundError(e) {
		t.Errorf("old mount should be gone, got %v", e)
	}
	if _, e := d.Get(ctx, "mvrA/dest_1"); e != nil {
		t.Errorf("renamed mount should exist: %v", e)
	}
}

func TestMount_CopyAndMove_ReturnVirtualDestinationPaths(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"vcpA", "vcpB"})
		defer cleanup()
		ctx := context.Background()
		content := []byte("copy")
		if _, e := d.Save(task.DummyContext(), "vcpA/source.txt", int64(len(content)), true, bytes.NewReader(content)); e != nil {
			t.Fatalf("Save source: %v", e)
		}
		if _, e := d.MakeDir(ctx, "vcpB/target"); e != nil {
			t.Fatalf("MakeDir target: %v", e)
		}
		p := "vcpA"
		setupMount(t, d, mountDAO, []types.PathMount{
			{Path: &p, Name: "mounted", MountAt: "vcpB/target"},
		})

		from, e := d.Get(ctx, "vcpA/source.txt")
		if e != nil {
			t.Fatalf("Get source: %v", e)
		}
		copied, e := d.Copy(task.DummyContext(), from, "vcpA/mounted/copied.txt", true)
		if e != nil {
			t.Fatalf("Copy: %v", e)
		}
		if copied.Path() != "vcpA/mounted/copied.txt" {
			t.Errorf("copied path=%q", copied.Path())
		}
		if _, e := d.Get(ctx, "vcpB/target/copied.txt"); e != nil {
			t.Errorf("copied file missing at real target: %v", e)
		}
	})

	t.Run("move", func(t *testing.T) {
		d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"vmvA"})
		defer cleanup()
		ctx := context.Background()
		content := []byte("move")
		if _, e := d.Save(task.DummyContext(), "vmvA/source.txt", int64(len(content)), true, bytes.NewReader(content)); e != nil {
			t.Fatalf("Save source: %v", e)
		}
		if _, e := d.MakeDir(ctx, "vmvA/target"); e != nil {
			t.Fatalf("MakeDir target: %v", e)
		}
		p := "vmvA"
		setupMount(t, d, mountDAO, []types.PathMount{
			{Path: &p, Name: "mounted", MountAt: "vmvA/target"},
		})

		from, e := d.Get(ctx, "vmvA/source.txt")
		if e != nil {
			t.Fatalf("Get source: %v", e)
		}
		moved, e := d.Move(task.DummyContext(), from, "vmvA/mounted/moved.txt", true)
		if e != nil {
			t.Fatalf("Move: %v", e)
		}
		if moved.Path() != "vmvA/mounted/moved.txt" {
			t.Errorf("moved path=%q", moved.Path())
		}
		if _, e := d.Get(ctx, "vmvA/target/moved.txt"); e != nil {
			t.Errorf("moved file missing at real target: %v", e)
		}
	})
}

func TestMount_MoveFailure_PreservesMountRecords(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"mfA", "mfB", "mfTarget"})
	defer cleanup()
	ctx := context.Background()

	for _, path := range []string{"mfA/parent", "mfTarget/target"} {
		if _, e := d.MakeDir(ctx, path); e != nil {
			t.Fatalf("MakeDir %s: %v", path, e)
		}
	}
	p := "mfA/parent"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &p, Name: "mounted", MountAt: "mfTarget/target"},
	})

	from, e := d.Get(ctx, "mfA/parent")
	if e != nil {
		t.Fatalf("Get source: %v", e)
	}
	if _, e := d.Move(task.DummyContext(), from, "mfB/parent", false); e == nil {
		t.Fatal("expected cross-drive move to fail")
	}

	mounts, e := mountDAO.GetMounts()
	if e != nil {
		t.Fatalf("GetMounts: %v", e)
	}
	if len(mounts) != 1 || *mounts[0].Path != "mfA/parent" || mounts[0].Name != "mounted" {
		t.Fatalf("mount records changed after failed move: %+v", mounts)
	}
	if _, e := d.Get(ctx, "mfA/parent/mounted"); e != nil {
		t.Errorf("original mount should remain accessible: %v", e)
	}
}

func TestMount_CopyOverride_ReplacesDestinationMountTree(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"ovA", "ovB"})
	defer cleanup()
	ctx := context.Background()

	for _, path := range []string{"ovB/source", "ovB/old", "ovB/stale"} {
		if _, e := d.MakeDir(ctx, path); e != nil {
			t.Fatalf("MakeDir %s: %v", path, e)
		}
	}
	root := "ovA"
	destination := "ovA/destination"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &root, Name: "source", MountAt: "ovB/source"},
		{Path: &root, Name: "destination", MountAt: "ovB/old"},
		{Path: &destination, Name: "stale", MountAt: "ovB/stale"},
	})

	from, e := d.Get(ctx, "ovA/source")
	if e != nil {
		t.Fatalf("Get source: %v", e)
	}
	if _, e := d.Copy(task.DummyContext(), from, "ovA/destination", true); e != nil {
		t.Fatalf("Copy override: %v", e)
	}

	mounts, e := mountDAO.GetMounts()
	if e != nil {
		t.Fatalf("GetMounts: %v", e)
	}
	for _, m := range mounts {
		mountPath := filepath.ToSlash(filepath.Join(*m.Path, m.Name))
		if mountPath == "ovA/destination/stale" {
			t.Error("stale destination child mount was not removed")
		}
		if mountPath == "ovA/destination" && m.MountAt != "ovB/source" {
			t.Errorf("destination target=%q, want ovB/source", m.MountAt)
		}
	}
	if _, e := d.Get(ctx, "ovA/destination/stale"); !err.IsNotFoundError(e) {
		t.Errorf("stale destination mount should be inaccessible, got %v", e)
	}
}

func TestMount_Move_NestedMountPointThroughOuterMount(t *testing.T) {
	tests := []struct {
		name        string
		driveNames  []string
		outerTarget string
		innerTarget string
	}{
		{
			name:        "same drive",
			driveNames:  []string{"nestedA", "nestedB"},
			outerTarget: "nestedB/root",
			innerTarget: "nestedB/target",
		},
		{
			name:        "different drives",
			driveNames:  []string{"nestedA", "nestedB", "nestedC"},
			outerTarget: "nestedB/root",
			innerTarget: "nestedC/target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, mountDAO, _, cleanup := newTestDispatcher(t, tt.driveNames)
			defer cleanup()
			ctx := context.Background()
			if _, e := d.MakeDir(ctx, tt.outerTarget); e != nil {
				t.Fatalf("MakeDir outer target: %v", e)
			}
			if _, e := d.MakeDir(ctx, tt.innerTarget); e != nil {
				t.Fatalf("MakeDir inner target: %v", e)
			}
			content := []byte("nested")
			if _, e := d.Save(task.DummyContext(), path.Join(tt.innerTarget, "file.txt"), int64(len(content)), true, bytes.NewReader(content)); e != nil {
				t.Fatalf("Save inner file: %v", e)
			}

			outerParent := "nestedA"
			innerParent := tt.outerTarget
			setupMount(t, d, mountDAO, []types.PathMount{
				{Path: &outerParent, Name: "outer", MountAt: tt.outerTarget},
				{Path: &innerParent, Name: "nested", MountAt: tt.innerTarget},
			})

			from, e := d.Get(ctx, "nestedA/outer/nested")
			if e != nil {
				t.Fatalf("Get nested mount: %v", e)
			}
			moved, e := d.Move(task.DummyContext(), from, "nestedA/outer/renamed", false)
			if e != nil {
				t.Fatalf("Move nested mount: %v", e)
			}
			if moved.Path() != "nestedA/outer/renamed" {
				t.Fatalf("moved path=%q", moved.Path())
			}
			if _, e := d.Get(ctx, path.Join(tt.innerTarget, "file.txt")); e != nil {
				t.Errorf("mount target was changed: %v", e)
			}
			if _, e := d.Get(ctx, "nestedA/outer/nested"); !err.IsNotFoundError(e) {
				t.Errorf("old nested mount still resolves: %v", e)
			}
			if _, e := d.Get(ctx, "nestedA/outer/renamed/file.txt"); e != nil {
				t.Errorf("renamed nested mount does not resolve: %v", e)
			}

			mounts, e := mountDAO.GetMounts()
			if e != nil {
				t.Fatalf("GetMounts: %v", e)
			}
			found := false
			for _, mount := range mounts {
				if *mount.Path == tt.outerTarget && mount.Name == "renamed" && mount.MountAt == tt.innerTarget {
					found = true
				}
				if *mount.Path == tt.outerTarget && mount.Name == "nested" {
					t.Error("old nested mount record was not removed")
				}
			}
			if !found {
				t.Error("renamed nested mount record was not found")
			}
		})
	}
}

func TestMount_Move_FileThroughNestedMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"nestedFileA", "nestedFileB", "nestedFileC"})
	defer cleanup()
	ctx := context.Background()
	for _, target := range []string{"nestedFileB/root", "nestedFileC/target"} {
		if _, e := d.MakeDir(ctx, target); e != nil {
			t.Fatalf("MakeDir %s: %v", target, e)
		}
	}
	content := []byte("nested file")
	if _, e := d.Save(task.DummyContext(), "nestedFileC/target/file.txt", int64(len(content)), true, bytes.NewReader(content)); e != nil {
		t.Fatalf("Save source: %v", e)
	}
	outerParent := "nestedFileA"
	innerParent := "nestedFileB/root"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &outerParent, Name: "outer", MountAt: "nestedFileB/root"},
		{Path: &innerParent, Name: "nested", MountAt: "nestedFileC/target"},
	})

	from, e := d.Get(ctx, "nestedFileA/outer/nested/file.txt")
	if e != nil {
		t.Fatalf("Get source: %v", e)
	}
	moved, e := d.Move(task.DummyContext(), from, "nestedFileA/outer/nested/renamed.txt", false)
	if e != nil {
		t.Fatalf("Move nested file: %v", e)
	}
	if moved.Path() != "nestedFileA/outer/nested/renamed.txt" {
		t.Fatalf("moved path=%q", moved.Path())
	}
	if _, e := d.Get(ctx, "nestedFileC/target/file.txt"); !err.IsNotFoundError(e) {
		t.Errorf("old real file still exists: %v", e)
	}
	if _, e := d.Get(ctx, "nestedFileC/target/renamed.txt"); e != nil {
		t.Errorf("renamed real file does not exist: %v", e)
	}
}

func TestMount_Copy_NestedMountPointThroughOuterMount(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"nestedCopyA", "nestedCopyB", "nestedCopyC"})
	defer cleanup()
	ctx := context.Background()
	for _, target := range []string{"nestedCopyB/root", "nestedCopyC/target"} {
		if _, e := d.MakeDir(ctx, target); e != nil {
			t.Fatalf("MakeDir %s: %v", target, e)
		}
	}
	outerParent := "nestedCopyA"
	innerParent := "nestedCopyB/root"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &outerParent, Name: "outer", MountAt: "nestedCopyB/root"},
		{Path: &innerParent, Name: "nested", MountAt: "nestedCopyC/target"},
	})

	from, e := d.Get(ctx, "nestedCopyA/outer/nested")
	if e != nil {
		t.Fatalf("Get nested mount: %v", e)
	}
	if _, e := d.Copy(task.DummyContext(), from, "nestedCopyA/outer/copied", false); e != nil {
		t.Fatalf("Copy nested mount: %v", e)
	}

	mounts, e := mountDAO.GetMounts()
	if e != nil {
		t.Fatalf("GetMounts: %v", e)
	}
	found := false
	for _, mount := range mounts {
		if *mount.Path == "nestedCopyB/root" && mount.Name == "copied" && mount.MountAt == "nestedCopyC/target" {
			found = true
		}
	}
	if !found {
		t.Error("copied nested mount record was not found")
	}
}

func TestMount_Move_PhantomDirectoryMaterializesDestination(t *testing.T) {
	d, mountDAO, _, cleanup := newTestDispatcher(t, []string{"phantomA", "phantomB"})
	defer cleanup()
	ctx := context.Background()
	if _, e := d.MakeDir(ctx, "phantomB/target"); e != nil {
		t.Fatalf("MakeDir target: %v", e)
	}
	content := []byte("mounted")
	if _, e := d.Save(task.DummyContext(), "phantomB/target/file.txt", int64(len(content)), true, bytes.NewReader(content)); e != nil {
		t.Fatalf("Save target file: %v", e)
	}
	phantomParent := "phantomA/phantom"
	setupMount(t, d, mountDAO, []types.PathMount{
		{Path: &phantomParent, Name: "child", MountAt: "phantomB/target"},
	})

	from, e := d.Get(ctx, "phantomA/phantom")
	if e != nil {
		t.Fatalf("Get phantom directory: %v", e)
	}
	moved, e := d.Move(task.DummyContext(), from, "phantomA/renamed", false)
	if e != nil {
		t.Fatalf("Move phantom directory: %v", e)
	}
	if moved.Path() != "phantomA/renamed" {
		t.Fatalf("moved path=%q", moved.Path())
	}
	if _, e := d.lower.Get(ctx, "phantomA/renamed"); e != nil {
		t.Errorf("destination was not materialized: %v", e)
	}
	if _, e := d.Get(ctx, "phantomA/phantom"); !err.IsNotFoundError(e) {
		t.Errorf("old phantom directory still exists: %v", e)
	}
	if _, e := d.Get(ctx, "phantomA/renamed/child/file.txt"); e != nil {
		t.Errorf("moved child mount does not resolve: %v", e)
	}
}
