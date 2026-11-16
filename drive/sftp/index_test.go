package sftp

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	protocol "github.com/pkg/sftp"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/server/webdav"
)

// Exercise the real SFTP wire protocol without listeners or external services.
func newMoveTestDrive(t *testing.T, extensions, denyPosix bool) (*Drive, *moveCommands) {
	t.Helper()
	return newMoveTestDriveWithHandlers(t, extensions, denyPosix, protocol.InMemHandler())
}

func newMoveTestDriveWithHandlers(t *testing.T, extensions, denyPosix bool, handlers protocol.Handlers) (*Drive, *moveCommands) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	commands := &moveCommands{FileCmder: handlers.FileCmd, denyPosix: denyPosix}
	handlers.FileCmd = commands
	server := protocol.NewRequestServer(serverConn, handlers)
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve() }()
	t.Cleanup(func() { _ = server.Close(); _ = serverConn.Close(); <-done })
	var reader io.Reader = clientConn
	if !extensions {
		reader = &noExtensionsReader{Reader: reader}
	}
	client, e := protocol.NewClientPipe(reader, clientConn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = client.Close() })
	for _, item := range []struct{ path, body string }{{"/source.txt", "new-content"}, {"/target.txt", "old-content"}} {
		file, e := client.Create(item.path)
		if e != nil {
			t.Fatal(e)
		}
		_, writeErr := io.WriteString(file, item.body)
		closeErr := file.Close()
		if writeErr != nil {
			t.Fatal(writeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if _, supported := client.HasExtension("posix-rename@openssh.com"); supported != extensions {
		t.Fatalf("extension advertised=%v, want %v", supported, extensions)
	}
	return &Drive{client: client, rootPath: "/", cache: driveutil.DummyCache()}, commands
}

type moveCommands struct {
	protocol.FileCmder
	denyPosix    bool
	removes      atomic.Int32
	renames      atomic.Int32
	posixRenames atomic.Int32
	mkdirs       atomic.Int32
	failInitial  atomic.Bool
}

func (c *moveCommands) Filecmd(r *protocol.Request) error {
	switch r.Method {
	case "Remove", "Rmdir":
		c.removes.Add(1)
	case "Mkdir":
		c.mkdirs.Add(1)
	case "Rename":
		c.renames.Add(1)
	}
	if r.Method == "Rename" && r.Filepath == "/source.txt" && r.Target == "/missing.txt" && c.failInitial.Load() {
		return protocol.ErrSSHFxPermissionDenied
	}
	return c.FileCmder.Filecmd(r)
}

func (c *moveCommands) PosixRename(r *protocol.Request) error {
	c.posixRenames.Add(1)
	if c.denyPosix {
		return os.ErrPermission
	}
	return c.FileCmder.(protocol.PosixRenameFileCmder).PosixRename(r)
}

// Strip extensions only from this connection's VERSION packet. Avoid changing
// pkg/sftp's process-wide extension advertisement during concurrent tests.
type noExtensionsReader struct {
	io.Reader
	initialized bool
}

func (r *noExtensionsReader) Read(p []byte) (int, error) {
	if !r.initialized {
		var length [4]byte
		if _, e := io.ReadFull(r.Reader, length[:]); e != nil {
			return 0, e
		}
		packet := make([]byte, binary.BigEndian.Uint32(length[:]))
		if _, e := io.ReadFull(r.Reader, packet); e != nil {
			return 0, e
		}
		if len(packet) < 5 || packet[0] != 2 {
			return 0, fmt.Errorf("expected SFTP VERSION packet")
		}
		header := append([]byte{0, 0, 0, 5}, packet[:5]...)
		r.Reader = io.MultiReader(bytes.NewReader(header), r.Reader)
		r.initialized = true
	}
	return r.Reader.Read(p)
}

type moveTestDAVFS struct{ *driveutil.BoundDriveFS }

func (f moveTestDAVFS) OpenFile(ctx context.Context, path string, flags int, mode os.FileMode) (webdav.File, error) {
	return f.BoundDriveFS.OpenFile(ctx, path, flags, mode)
}

func requireRemoteContents(t *testing.T, d *Drive, path, want string) {
	t.Helper()
	file, e := d.client.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	body, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || string(body) != want {
		t.Fatalf("%s contents=%q read=%v close=%v; want %q", path, body, readErr, closeErr, want)
	}
}

func TestWebDAVOverwriteMove(t *testing.T) {
	for _, extensions := range []bool{true, false} {
		t.Run(fmt.Sprintf("posix-extension=%v", extensions), func(t *testing.T) {
			d, commands := newMoveTestDrive(t, extensions, false)
			files, e := driveutil.NewDriveFS(common.Config{TempDir: t.TempDir()})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = files.Dispose() })
			handler := webdav.Handler{FileSystem: moveTestDAVFS{files.Bind(context.Background(), d)}, LockSystem: webdav.NewMemLS()}
			req := httptest.NewRequest("MOVE", "http://example.com/source.txt", nil)
			req.Header.Set("Destination", "http://example.com/target.txt")
			req.Header.Set("Overwrite", "T")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if !extensions {
				if w.Code != http.StatusForbidden {
					t.Fatalf("MOVE status=%d body=%q", w.Code, w.Body.String())
				}
				requireRemoteContents(t, d, "/source.txt", "new-content")
				requireRemoteContents(t, d, "/target.txt", "old-content")
				requireNoMoveMutations(t, commands)
				return
			}
			if w.Code != http.StatusNoContent {
				t.Fatalf("MOVE status=%d body=%q", w.Code, w.Body.String())
			}
			requireRemoteContents(t, d, "/target.txt", "new-content")
			if _, e := d.client.Stat("/source.txt"); !os.IsNotExist(e) {
				t.Fatalf("source still exists: %v", e)
			}
			if commands.posixRenames.Load() != 1 || commands.removes.Load() != 0 || commands.renames.Load() != 0 || commands.mkdirs.Load() != 0 {
				t.Fatalf("extension path: posix=%d removes=%d renames=%d mkdirs=%d", commands.posixRenames.Load(), commands.removes.Load(), commands.renames.Load(), commands.mkdirs.Load())
			}
		})
	}
}

func TestMoveConflictPreservesFiles(t *testing.T) {
	for _, extensions := range []bool{true, false} {
		t.Run(fmt.Sprintf("posix-extension=%v", extensions), func(t *testing.T) {
			d, commands := newMoveTestDrive(t, extensions, false)
			from, e := d.Get(context.Background(), "source.txt")
			if e != nil {
				t.Fatal(e)
			}
			if _, e := d.Move(task.DummyContext(), from, "target.txt", false); e == nil {
				t.Fatal("non-overwrite move replaced destination")
			}
			requireRemoteContents(t, d, "/source.txt", "new-content")
			requireRemoteContents(t, d, "/target.txt", "old-content")
			if commands.removes.Load() != 0 || commands.renames.Load() != 0 || commands.posixRenames.Load() != 0 {
				t.Fatal("conflict issued a mutation")
			}
		})
	}
}

func TestMoveMissingSourcePreservesTarget(t *testing.T) {
	d, commands := newMoveTestDrive(t, true, false)
	from, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	if e := d.client.Remove("/source.txt"); e != nil {
		t.Fatal(e)
	}
	before := commands.removes.Load()
	if _, e := d.Move(task.DummyContext(), from, "target.txt", true); e == nil {
		t.Fatal("stale source move succeeded")
	}
	requireRemoteContents(t, d, "/target.txt", "old-content")
	if commands.removes.Load() != before {
		t.Fatal("missing source removed or renamed target")
	}
}

func TestMovePosixFailureDoesNotDeleteTarget(t *testing.T) {
	d, commands := newMoveTestDrive(t, true, true)
	from, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := d.Move(task.DummyContext(), from, "target.txt", true); e == nil {
		t.Fatal("denied extension succeeded")
	}
	requireRemoteContents(t, d, "/source.txt", "new-content")
	requireRemoteContents(t, d, "/target.txt", "old-content")
	if commands.posixRenames.Load() != 1 || commands.removes.Load() != 0 || commands.renames.Load() != 0 {
		t.Fatal("failed extension fell back to destructive replacement")
	}
}

func TestMoveToSamePathPreservesSource(t *testing.T) {
	d, commands := newMoveTestDrive(t, false, false)
	from, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := d.Move(task.DummyContext(), from, "source.txt", true); e == nil {
		t.Fatal("self move succeeded")
	}
	requireRemoteContents(t, d, "/source.txt", "new-content")
	if commands.removes.Load() != 0 || commands.renames.Load() != 0 {
		t.Fatal("self move mutated source")
	}
}

// A real protocol server with a case-insensitive namespace, independent of the
// host filesystem. Keep this behavior in the server, never in drive results.
type foldedMoveFS struct{ handlers protocol.Handlers }

func foldMoveRequest(r *protocol.Request) *protocol.Request {
	// Normalize the request in place: Request owns internal state containing a
	// mutex and must not be copied when adapting the server's namespace.
	r.Filepath = strings.ToLower(r.Filepath)
	r.Target = strings.ToLower(r.Target)
	return r
}
func (f foldedMoveFS) Fileread(r *protocol.Request) (io.ReaderAt, error) {
	return f.handlers.FileGet.Fileread(foldMoveRequest(r))
}
func (f foldedMoveFS) Filewrite(r *protocol.Request) (io.WriterAt, error) {
	return f.handlers.FilePut.Filewrite(foldMoveRequest(r))
}
func (f foldedMoveFS) Filecmd(r *protocol.Request) error {
	return f.handlers.FileCmd.Filecmd(foldMoveRequest(r))
}
func (f foldedMoveFS) Filelist(r *protocol.Request) (protocol.ListerAt, error) {
	return f.handlers.FileList.Filelist(foldMoveRequest(r))
}

func TestWebDAVCaseOnlyOverwriteWithoutExtensionPreservesSource(t *testing.T) {
	for _, folded := range []bool{true, false} {
		t.Run(fmt.Sprintf("case-insensitive=%v", folded), func(t *testing.T) {
			handlers := protocol.InMemHandler()
			if folded {
				fs := foldedMoveFS{handlers: handlers}
				handlers = protocol.Handlers{FileGet: fs, FilePut: fs, FileCmd: fs, FileList: fs}
			}
			d, commands := newMoveTestDriveWithHandlers(t, false, false, handlers)
			if !folded {
				target, e := d.client.Create("/SOURCE.txt")
				if e != nil {
					t.Fatal(e)
				}
				if _, e := io.WriteString(target, "distinct-target"); e != nil {
					t.Fatal(e)
				}
				if e := target.Close(); e != nil {
					t.Fatal(e)
				}
			}
			files, e := driveutil.NewDriveFS(common.Config{TempDir: t.TempDir()})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = files.Dispose() })
			handler := webdav.Handler{FileSystem: moveTestDAVFS{files.Bind(context.Background(), d)}, LockSystem: webdav.NewMemLS()}
			req := httptest.NewRequest("MOVE", "http://example.com/source.txt", nil)
			req.Header.Set("Destination", "http://example.com/SOURCE.txt")
			req.Header.Set("Overwrite", "T")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("case-only move status=%d body=%q", w.Code, w.Body.String())
			}
			requireRemoteContents(t, d, "/source.txt", "new-content")
			if !folded {
				requireRemoteContents(t, d, "/SOURCE.txt", "distinct-target")
			}
			requireNoMoveMutations(t, commands)
		})
	}
}

func requireNoMoveMutations(t *testing.T, commands *moveCommands) {
	t.Helper()
	if commands.removes.Load() != 0 || commands.renames.Load() != 0 || commands.posixRenames.Load() != 0 || commands.mkdirs.Load() != 0 {
		t.Fatalf("rejected move mutated server: removes=%d renames=%d posix=%d mkdirs=%d", commands.removes.Load(), commands.renames.Load(), commands.posixRenames.Load(), commands.mkdirs.Load())
	}
}

func TestMoveWithoutExtensionRejectsOverwriteToMissingTarget(t *testing.T) {
	d, commands := newMoveTestDrive(t, false, false)
	from, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := d.Move(task.DummyContext(), from, "missing.txt", true); !err.IsNotAllowedError(e) {
		t.Fatalf("overwrite without extension error=%v", e)
	}
	requireRemoteContents(t, d, "/source.txt", "new-content")
	if _, e := d.client.Stat("/missing.txt"); !os.IsNotExist(e) {
		t.Fatalf("rejected move created target: %v", e)
	}
	requireNoMoveMutations(t, commands)
}

func TestMoveWithoutOverwrite(t *testing.T) {
	for _, extensions := range []bool{true, false} {
		t.Run(fmt.Sprintf("posix-extension=%v", extensions), func(t *testing.T) {
			d, commands := newMoveTestDrive(t, extensions, false)
			from, e := d.Get(context.Background(), "source.txt")
			if e != nil {
				t.Fatal(e)
			}
			if _, e := d.Move(task.DummyContext(), from, "missing.txt", false); e != nil {
				t.Fatal(e)
			}
			requireRemoteContents(t, d, "/missing.txt", "new-content")
			if _, e := d.client.Stat("/source.txt"); !os.IsNotExist(e) {
				t.Fatalf("source still exists: %v", e)
			}
			if commands.renames.Load() != 1 || commands.posixRenames.Load() != 0 || commands.removes.Load() != 0 || commands.mkdirs.Load() != 0 {
				t.Fatal("non-overwrite move did not use ordinary rename exclusively")
			}
		})
	}
}

// Reproduce the reported behavior against a native Windows filesystem too.
func TestWebDAVCaseOnlyMoveOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires a case-insensitive Windows filesystem")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	if e := os.WriteFile(source, []byte("keep-me"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "SOURCE.txt")); e != nil {
		t.Skip("filesystem is case-sensitive")
	}
	serverConn, clientConn := net.Pipe()
	server, e := protocol.NewServer(serverConn)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve() }()
	t.Cleanup(func() { _ = server.Close(); _ = serverConn.Close(); <-done })
	client, e := protocol.NewClientPipe(&noExtensionsReader{Reader: clientConn}, clientConn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = client.Close() })
	d := &Drive{client: client, rootPath: "/" + filepath.ToSlash(root), cache: driveutil.DummyCache()}
	files, e := driveutil.NewDriveFS(common.Config{TempDir: t.TempDir()})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = files.Dispose() })
	handler := webdav.Handler{FileSystem: moveTestDAVFS{files.Bind(context.Background(), d)}, LockSystem: webdav.NewMemLS()}
	req := httptest.NewRequest("MOVE", "http://example.com/source.txt", nil)
	req.Header.Set("Destination", "http://example.com/SOURCE.txt")
	req.Header.Set("Overwrite", "T")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("native case-only move status=%d", w.Code)
	}
	body, e := os.ReadFile(filepath.Join(root, "SOURCE.txt"))
	if e != nil || string(body) != "keep-me" {
		t.Fatalf("case-only MOVE lost source: %q, %v", body, e)
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) != 1 {
		t.Fatalf("rejected MOVE changed directory contents: entries=%v error=%v", entries, e)
	}
}

func TestMoveDeniedRenameDoesNotStageOrDelete(t *testing.T) {
	d, commands := newMoveTestDrive(t, false, false)
	commands.failInitial.Store(true)
	from, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := d.Move(task.DummyContext(), from, "missing.txt", false); e == nil {
		t.Fatal("denied rename succeeded")
	}
	requireRemoteContents(t, d, "/source.txt", "new-content")
	requireRemoteContents(t, d, "/target.txt", "old-content")
	if commands.removes.Load() != 0 {
		t.Fatal("denied rename triggered removal fallback")
	}
	if commands.renames.Load() != 1 || commands.posixRenames.Load() != 0 || commands.mkdirs.Load() != 0 {
		t.Fatal("denied rename triggered fallback")
	}
}
