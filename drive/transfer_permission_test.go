package drive

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/event"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"

	err "go-drive/common/errors"
	localfs "go-drive/drive/fs"
	"go-drive/server/webdav"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type transferTestFS struct{ *driveutil.BoundDriveFS }

func (f transferTestFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return f.BoundDriveFS.OpenFile(ctx, name, flag, perm)
}

func newTransferPermissionFixture(t *testing.T, deny types.Permission) (string, *PermissionWrapperDrive, *webdav.Handler) {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"source/locked/secret.txt", "source/public/deeper/ok.txt", "target/keep.txt"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte("original"), 0644); e != nil {
			t.Fatal(e)
		}
	}
	backend, e := localfs.NewDrive(context.Background(), types.SM{"path": root}, driveutil.DriveEnv{Config: common.Config{FreeFs: true}})
	if e != nil {
		t.Fatal(e)
	}
	dispatcher := NewDispatcherDrive(common.Config{TempDir: t.TempDir()})
	dispatcher.setDrives(map[string]types.IDrive{"disk": backend})
	r, locked, sibling := "", "disk/source/locked", "disk/source/public/deeper"
	pm := utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &locked, Subject: types.AnySubject, Permission: deny, Policy: types.PolicyReject},
		{Path: &sibling, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
	})
	d := NewPermissionWrapperDrive(dispatcher, pm)
	files, e := driveutil.NewDriveFS(common.Config{TempDir: t.TempDir()})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = files.Dispose() })
	h := &webdav.Handler{FileSystem: transferTestFS{files.Bind(context.Background(), d)}, LockSystem: webdav.NewMemLS(), Logger: func(_ *http.Request, e error) { t.Logf("handler error: %v", e) }}
	return root, d, h
}

func TestTransferDirectMovePreflightPreservesDestination(t *testing.T) {
	root, d, _ := newTransferPermissionFixture(t, types.PermissionWrite)
	from, e := d.Get(context.Background(), "disk/source")
	if e != nil {
		t.Fatal(e)
	}
	_, e = d.Move(task.DummyContext(), from, "disk/target", true)
	if e == nil {
		t.Fatal("restricted move unexpectedly succeeded")
	}
	if body, e := os.ReadFile(filepath.Join(root, "target", "keep.txt")); e != nil || string(body) != "original" {
		t.Fatalf("destination changed: %q, %v", body, e)
	}
	t.Log("direct Move rejects restricted source before deleting destination")
}

func TestTransferWebDAVMovePreservesDestinationOnPermissionRejection(t *testing.T) {
	root, _, h := newTransferPermissionFixture(t, types.PermissionWrite)
	req := httptest.NewRequest("MOVE", "http://example.com/disk/source", nil)
	req.Header.Set("Destination", "http://example.com/disk/target")
	req.Header.Set("Overwrite", "T")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	t.Logf("MOVE status=%d body=%q", w.Code, w.Body.String())
	if w.Code < 400 {
		t.Fatal("restricted move unexpectedly succeeded")
	}
	if body, e := os.ReadFile(filepath.Join(root, "target", "keep.txt")); e != nil || string(body) != "original" {
		t.Fatalf("rejected MOVE destroyed existing destination: body=%q error=%v", body, e)
	}
}

func TestTransferWebDAVCopyRejectsUnreadableSubtreeBeforeWrite(t *testing.T) {
	root, _, h := newTransferPermissionFixture(t, types.PermissionRead)
	req := httptest.NewRequest("COPY", "http://example.com/disk/source", nil)
	req.Header.Set("Destination", "http://example.com/disk/new-target")
	req.Header.Set("Depth", "infinity")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	_, publicErr := os.Stat(filepath.Join(root, "new-target", "public", "deeper", "ok.txt"))
	_, secretErr := os.Stat(filepath.Join(root, "new-target", "locked", "secret.txt"))
	t.Logf("COPY status=%d copiedPublic=%t copiedSecret=%t", w.Code, publicErr == nil, secretErr == nil)
	if w.Code < 400 {
		t.Fatal("COPY reported success instead of rejecting an unreadable subtree")
	}
	if _, e := os.Stat(filepath.Join(root, "new-target")); !os.IsNotExist(e) {
		t.Fatalf("denied COPY changed destination: %v", e)
	}
}

func TestTransferRenamedMoveChecksActualDestination(t *testing.T) {
	root, d, _ := newTransferPermissionFixture(t, types.PermissionEmpty)
	r, protected := "", "disk/target_1/locked"
	d.pm = utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &protected, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
	})
	from, e := d.Get(context.Background(), "disk/source")
	if e != nil {
		t.Fatal(e)
	}
	result, e := d.Move(task.DummyContext(), from, "disk/target", false)
	if result != nil {
		t.Logf("actual destination=%s", result.Path())
	}
	_, statErr := os.Stat(filepath.Join(root, "target_1", "locked", "secret.txt"))
	t.Logf("Move error=%v protected file created=%t", e, statErr == nil)
	if e == nil || !os.IsNotExist(statErr) {
		t.Fatal("Move wrote to a forbidden subtree after automatic collision renaming")
	}
}

func TestPrepareWriteChecksRenamedDestination(t *testing.T) {
	for _, operation := range []string{"save", "upload", "copy", "move"} {
		t.Run(operation, func(t *testing.T) {
			root, d, _ := newTransferPermissionFixture(t, types.PermissionEmpty)
			r, forbidden := "", "disk/target_1"
			if operation == "save" || operation == "upload" {
				if e := os.WriteFile(filepath.Join(root, "target.txt"), []byte("original"), 0644); e != nil {
					t.Fatal(e)
				}
				forbidden = "disk/target_1.txt"
			}
			d.pm = utils.NewPermMap([]types.PathPermission{
				{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
				{Path: &forbidden, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
			})
			requested := "disk/target"
			if operation == "save" || operation == "upload" {
				requested += ".txt"
			}
			if _, e := d.PrepareWrite(context.Background(), requested, types.WriteRename); e == nil {
				t.Fatal("prepared forbidden destination")
			}
			// Raw execution must reject the original conflict rather than rename around authorization.
			var e error
			switch operation {
			case "save":
				_, e = d.Save(task.DummyContext(), requested, 4, false, strings.NewReader("evil"))
			case "upload":
				_, e = d.Upload(context.Background(), requested, 4, false, nil)
			default:
				source, getErr := d.Get(context.Background(), "disk/source")
				if getErr != nil {
					t.Fatal(getErr)
				}
				if operation == "copy" {
					_, e = d.Copy(task.DummyContext(), source, requested, false)
				} else {
					_, e = d.Move(task.DummyContext(), source, requested, false)
				}
			}
			if e == nil {
				t.Fatal("conflicting execution succeeded")
			}
			if _, e := os.Stat(filepath.Join(root, strings.TrimPrefix(forbidden, "disk/"))); !os.IsNotExist(e) {
				t.Fatalf("forbidden destination changed: %v", e)
			}
			original := filepath.Join(root, "target", "keep.txt")
			if operation == "save" || operation == "upload" {
				original = filepath.Join(root, "target.txt")
			}
			if body, e := os.ReadFile(original); e != nil || string(body) != "original" {
				t.Fatalf("original target changed: %q, %v", body, e)
			}
		})
	}
}

func TestPrepareWriteReadOnlyAndExecutionConflict(t *testing.T) {
	root, d, _ := newTransferPermissionFixture(t, types.PermissionEmpty)
	plan, e := d.PrepareWrite(context.Background(), "disk/new/child/file.txt", types.WriteRename)
	if e != nil {
		t.Fatal(e)
	}
	if len(plan.Parents) != 2 || plan.Parents[0] != "disk/new" || plan.Parents[1] != "disk/new/child" {
		t.Fatalf("parents=%v", plan.Parents)
	}
	if _, e := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(e) {
		t.Fatalf("preparation created directories: %v", e)
	}
	plan, e = d.PrepareWrite(context.Background(), "disk/target/keep.txt", types.WriteRename)
	if e != nil {
		t.Fatal(e)
	}
	if plan.Path != "disk/target/keep_1.txt" || plan.Override {
		t.Fatalf("plan=%+v", plan)
	}
	target := filepath.Join(root, "target", "keep_1.txt")
	if e := os.WriteFile(target, []byte("concurrent"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := d.Save(task.DummyContext(), plan.Path, 4, plan.Override, strings.NewReader("evil")); e == nil {
		t.Fatal("late conflict did not reject")
	}
	if body, e := os.ReadFile(target); e != nil || string(body) != "concurrent" {
		t.Fatalf("late conflict overwritten: %q, %v", body, e)
	}
	if _, e := os.Stat(filepath.Join(root, "target", "keep_1_1.txt")); !os.IsNotExist(e) {
		t.Fatalf("execution allocated another name: %v", e)
	}
}

func TestPrepareWriteChecksMissingParentPermissions(t *testing.T) {
	root, d, _ := newTransferPermissionFixture(t, types.PermissionEmpty)
	r, parent, file := "", "disk/new", "disk/new/child/file.txt"
	d.pm = utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &parent, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
		{Path: &file, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
	})
	if _, e := d.PrepareWrite(context.Background(), file, types.WriteReject); !err.IsPermissionDeniedNotFoundError(e) {
		t.Fatalf("missing parent permission: %v", e)
	}
	if _, e := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(e) {
		t.Fatalf("unauthorized directory created: %v", e)
	}
}

func TestPrepareWriteChrootNamespace(t *testing.T) {
	_, d, _ := newTransferPermissionFixture(t, types.PermissionEmpty)
	bus := event.NewBus()
	events := 0
	bus.SubscribeEntryAccessed(func(types.DriveEvent, string) { events++ })
	bus.SubscribeEntryUpdated(func(types.DriveEvent, string, bool) { events++ })
	c := NewChrootWrapper(NewListenerWrapper(d, types.DriveEvent{}, bus), NewChroot("disk", nil))
	// Preparing through the listener must not publish read or mutation events.
	plan, e := c.PrepareWrite(context.Background(), "target/keep.txt", types.WriteRename)
	if e != nil {
		t.Fatal(e)
	}
	if plan.Path != "target/keep_1.txt" {
		t.Fatalf("chroot plan leaked backend namespace: %+v", plan)
	}
	if events != 0 {
		t.Fatalf("preparation published %d events", events)
	}
}

func TestTransferWebDAVDepthAndOverwrite(t *testing.T) {
	for _, tc := range []struct {
		name, method, depth, overwrite, target string
		deny                                   types.Permission
		status                                 int
	}{
		{"shallow directory", "COPY", "0", "T", "disk/shallow", types.PermissionRead, http.StatusCreated},
		{"full readable copy", "COPY", "infinity", "T", "disk/copied", types.PermissionEmpty, http.StatusCreated},
		{"replace directory copy", "COPY", "infinity", "T", "disk/target", types.PermissionEmpty, http.StatusNoContent},
		{"replace directory move", "MOVE", "", "T", "disk/target", types.PermissionEmpty, http.StatusNoContent},
		{"reject copy conflict", "COPY", "infinity", "F", "disk/target", types.PermissionEmpty, http.StatusPreconditionFailed},
		{"reject move conflict", "MOVE", "", "F", "disk/target", types.PermissionEmpty, http.StatusPreconditionFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, h := newTransferPermissionFixture(t, tc.deny)
			req := httptest.NewRequest(tc.method, "http://example.com/disk/source", nil)
			req.Header.Set("Destination", "http://example.com/"+tc.target)
			req.Header.Set("Overwrite", tc.overwrite)
			if tc.depth != "" {
				req.Header.Set("Depth", tc.depth)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
			}
			target := filepath.Join(root, strings.TrimPrefix(tc.target, "disk/"))
			if tc.status == http.StatusPreconditionFailed {
				if body, e := os.ReadFile(filepath.Join(root, "target", "keep.txt")); e != nil || string(body) != "original" {
					t.Fatalf("conflict changed target: %q, %v", body, e)
				}
				if _, e := os.Stat(filepath.Join(root, "source", "locked", "secret.txt")); e != nil {
					t.Fatal(e)
				}
			} else if tc.depth == "0" {
				entries, e := os.ReadDir(target)
				if e != nil || len(entries) != 0 {
					t.Fatalf("shallow copy contains children: %v, %v", entries, e)
				}
			} else {
				if _, e := os.Stat(filepath.Join(target, "locked", "secret.txt")); e != nil {
					t.Fatalf("missing copied child: %v", e)
				}
				if _, e := os.Stat(filepath.Join(target, "keep.txt")); !os.IsNotExist(e) {
					t.Fatalf("stale target retained: %v", e)
				}
			}
		})
	}
}

func TestTransferWebDAVShallowCopyChecksReplacedSubtree(t *testing.T) {
	root, d, h := newTransferPermissionFixture(t, types.PermissionRead)
	r, locked := "", "disk/target/keep.txt"
	d.pm = utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &locked, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
	})
	req := httptest.NewRequest("COPY", "http://example.com/disk/source", nil)
	req.Header.Set("Destination", "http://example.com/disk/target")
	req.Header.Set("Depth", "0")
	req.Header.Set("Overwrite", "T")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code < 400 {
		t.Fatalf("shallow copy ignored protected target: %d", w.Code)
	}
	if body, e := os.ReadFile(filepath.Join(root, "target", "keep.txt")); e != nil || string(body) != "original" {
		t.Fatalf("shallow copy changed protected target: %q, %v", body, e)
	}
}

func TestPrepareWriteMountVirtualNamespace(t *testing.T) {
	root, mountDAO, _, cleanup := newTestDispatcher(t, []string{"prepareA", "prepareB"})
	defer cleanup()
	ctx := context.Background()
	if _, e := root.MakeDir(ctx, "prepareB/sub"); e != nil {
		t.Fatal(e)
	}
	if _, e := root.Save(task.DummyContext(), "prepareB/sub/file.txt", 3, false, strings.NewReader("old")); e != nil {
		t.Fatal(e)
	}
	mountParent := "prepareA"
	setupMount(t, root, mountDAO, []types.PathMount{{Path: &mountParent, Name: "alias", MountAt: "prepareB/sub"}})
	r, denied := "", "prepareA/alias/file_1.txt"
	d := NewPermissionWrapperDrive(root, utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &denied, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
	}))
	if _, e := d.PrepareWrite(ctx, "prepareA/alias/file.txt", types.WriteRename); e == nil {
		t.Fatal("mount preparation ignored virtual path restriction")
	}
	// A backing-path restriction does not follow a mount into another virtual path.
	denied = "prepareB/sub/file_1.txt"
	d.pm = utils.NewPermMap([]types.PathPermission{
		{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &denied, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
	})
	plan, e := d.PrepareWrite(ctx, "prepareA/alias/file.txt", types.WriteRename)
	if e != nil {
		t.Fatal(e)
	}
	if plan.Path != "prepareA/alias/file_1.txt" {
		t.Fatalf("plan namespace=%q", plan.Path)
	}
	if _, e := d.Save(task.DummyContext(), plan.Path, 3, plan.Override, strings.NewReader("new")); e != nil {
		t.Fatal(e)
	}
	if _, e := root.Get(ctx, "prepareB/sub/file_1.txt"); e != nil {
		t.Fatal(e)
	}
}
