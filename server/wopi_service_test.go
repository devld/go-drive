package server

import (
	"context"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"strings"
	"testing"
	"time"
)

func newTestWOPIService() *wopiService {
	return &wopiService{
		config:       common.Config{APIPath: "/api"},
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(4),
	}
}

func TestWOPISessionTokenScopeAndExpiry(t *testing.T) {
	service := newTestWOPIService()
	session, token, e := service.createSession("alice", "docs/file.docx", "https://a.example", "drive/docs/file.docx", true, time.Time{})
	if e != nil {
		t.Fatal(e)
	}
	if token == "" || strings.Contains(service.fileURL(session.origin, session.id), token) {
		t.Fatal("raw access token must not be part of the WOPISrc")
	}
	if got, ok := service.validateSession(session.id, token); !ok || got.username != "alice" || !got.writable {
		t.Fatalf("session validation failed: %#v, ok=%v", got, ok)
	}
	if _, ok := service.validateSession(session.id, token+"x"); ok {
		t.Fatal("wrong token validated")
	}

	service.sessionsMu.Lock()
	expired := service.sessions[session.id]
	expired.expiresAt = time.Now().Add(-time.Second)
	service.sessions[session.id] = expired
	service.sessionsMu.Unlock()
	if _, ok := service.validateSession(session.id, token); ok {
		t.Fatal("expired token validated")
	}
}

func TestWOPIMultipleOriginsAndAPIPath(t *testing.T) {
	service := newTestWOPIService()
	for _, origin := range []string{"https://a.example", "https://b.example"} {
		validated, e := validateWOPIOrigin(origin, strings.TrimPrefix(origin, "https://"))
		if e != nil {
			t.Fatalf("validate origin %q: %v", origin, e)
		}
		got := service.fileURL(validated, "id")
		want := origin + "/api/wopi/files/id"
		if got != want {
			t.Fatalf("file URL=%q, want %q", got, want)
		}
	}
	if _, e := validateWOPIOrigin("https://evil.example", "a.example"); e == nil {
		t.Fatal("mismatched origin was accepted")
	}
}

func TestWOPILockValidationAndExpiry(t *testing.T) {
	service := newTestWOPIService()
	if !validWOPILock("lock-id") || validWOPILock("") || validWOPILock("锁") || validWOPILock(strings.Repeat("x", 1025)) {
		t.Fatal("unexpected lock validation result")
	}
	service.setLock("resource", wopiLock{value: "lock-id", expiresAt: time.Now().Add(-time.Second)})
	if lock, ok := service.currentLock("resource", time.Now()); ok || lock.value != "" {
		t.Fatalf("expired lock remained active: %#v", lock)
	}
}

func TestRedactWOPIAccessToken(t *testing.T) {
	got := redactWOPIAccessToken("/wopi/files/id?access_token=secret&x=1")
	if strings.Contains(got, "secret") || !strings.Contains(got, "access_token=") || !strings.Contains(got, "x=1") {
		t.Fatalf("unexpected redacted path: %s", got)
	}
	plain := "/entries/path?x=1"
	if got := redactWOPIAccessToken(plain); got != plain {
		t.Fatalf("unrelated query changed: %s", got)
	}
}

func TestWOPIVersionUsesCanonicalDispatchedPath(t *testing.T) {
	drive := &wopiTestDrive{}
	entry := &wopiTestEntry{drive: drive, path: "inner/file.docx", size: 12, modTime: 34}
	wrapper := &wopiTestDispatcherEntry{IEntry: entry, realPath: "drive/inner/file.docx"}
	if got := canonicalWOPIResourceKey(wrapper); got != "drive/inner/file.docx" {
		t.Fatalf("resource key=%q", got)
	}
	if wopiVersion(wrapper) == "" {
		t.Fatal("empty WOPI version")
	}
}

type wopiTestDispatcherEntry struct {
	types.IEntry
	realPath string
}

func (e *wopiTestDispatcherEntry) GetIEntry() types.IEntry { return e.IEntry }
func (e *wopiTestDispatcherEntry) GetDispatchedDrive() (string, types.IDrive) {
	return "drive", e.IEntry.Drive()
}
func (e *wopiTestDispatcherEntry) GetRealPath() string { return e.realPath }

type wopiTestEntry struct {
	drive   types.IDrive
	path    string
	size    int64
	modTime int64
}

func (e *wopiTestEntry) Path() string          { return e.path }
func (e *wopiTestEntry) Name() string          { return "file.docx" }
func (e *wopiTestEntry) Type() types.EntryType { return types.TypeFile }
func (e *wopiTestEntry) Size() int64           { return e.size }
func (e *wopiTestEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}
func (e *wopiTestEntry) ModTime() int64      { return e.modTime }
func (e *wopiTestEntry) Drive() types.IDrive { return e.drive }
func (e *wopiTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, nil
}
func (e *wopiTestEntry) GetURL(context.Context) (*types.ContentURL, error) { return nil, nil }

type wopiTestDrive struct{}

func (*wopiTestDrive) Meta(context.Context) (types.DriveMeta, error) { panic("not used") }
func (*wopiTestDrive) Get(context.Context, string) (types.IEntry, error) {
	panic("not used")
}
func (*wopiTestDrive) Save(types.TaskCtx, string, int64, bool, io.Reader) (types.IEntry, error) {
	panic("not used")
}
func (*wopiTestDrive) MakeDir(context.Context, string) (types.IEntry, error) { panic("not used") }
func (*wopiTestDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (*wopiTestDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (*wopiTestDrive) List(context.Context, string) ([]types.IEntry, error) { panic("not used") }
func (*wopiTestDrive) Delete(types.TaskCtx, string) error                   { panic("not used") }
func (*wopiTestDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	panic("not used")
}
