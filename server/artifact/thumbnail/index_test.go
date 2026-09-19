package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type testThumbnailEntry struct {
	path    string
	real    string
	size    int64
	modTime int64
}

func (e *testThumbnailEntry) Path() string          { return e.path }
func (e *testThumbnailEntry) Name() string          { return filepath.Base(e.path) }
func (e *testThumbnailEntry) Type() types.EntryType { return types.TypeFile }
func (e *testThumbnailEntry) Size() int64           { return e.size }
func (e *testThumbnailEntry) ModTime() int64        { return e.modTime }
func (e *testThumbnailEntry) Meta() types.EntryMeta { return types.EntryMeta{Readable: true} }
func (e *testThumbnailEntry) Drive() types.IDrive   { return nil }
func (e *testThumbnailEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (e *testThumbnailEntry) GetURL(context.Context) (*types.ContentURL, error) { return nil, nil }
func (e *testThumbnailEntry) GetDispatchedDrive() (string, types.IDrive)        { return "drive", nil }
func (e *testThumbnailEntry) GetRealPath() string                               { return e.real }
func (e *testThumbnailEntry) GetExternalURL() string                            { return "/download?path=" + e.path }

type hasThumbnailEntry struct {
	testThumbnailEntry
}

func (e *hasThumbnailEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, HasThumbnail: true}
}

func (e *hasThumbnailEntry) Thumbnail(context.Context) (types.IContentReader, error) {
	return nil, errors.New("entry thumbnail not used")
}

type failingThumbnailHandler struct {
	calls *atomic.Int32
}

func (h *failingThumbnailHandler) CreateThumbnail(context.Context, ThumbnailEntry, io.Writer) error {
	h.calls.Add(1)
	return errors.New("thumbnail generation failed")
}

func (h *failingThumbnailHandler) MimeType() string       { return "image/jpeg" }
func (h *failingThumbnailHandler) Timeout() time.Duration { return -1 }

func TestMakerRegistrationUsesSortedCommaSeparatedExtensions(t *testing.T) {
	maker := &Maker{
		handlers: map[string]TypeHandler{
			"png": nil,
			"/":   nil,
			"jpg": nil,
		},
	}
	registrations := maker.Registrations()
	if len(registrations) != 1 {
		t.Fatalf("thumbnail registrations = %d, want 1", len(registrations))
	}
	config := maker.Config()
	if got := config["extensions"]; got != "/,jpg,png" {
		t.Fatalf("thumbnail extensions = %#v, want %q", got, "/,jpg,png")
	}
}

func TestMakerUsesEntryThumbnailWhenHasThumbnail(t *testing.T) {
	calls := &atomic.Int32{}
	maker := &Maker{
		handlers: map[string]TypeHandler{
			"png": &failingThumbnailHandler{calls: calls},
		},
	}
	handler, err := maker.resolveHandler(&hasThumbnailEntry{
		testThumbnailEntry: testThumbnailEntry{path: "images/demo.png", real: "drive/images/demo.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if handler != entryThumbnailTypeHandler {
		t.Fatalf("handler = %#v, want entry thumbnail", handler)
	}
}

func TestMakerMarksGenerationFailureCacheableAndChangesFingerprint(t *testing.T) {
	calls := &atomic.Int32{}
	entry := &testThumbnailEntry{path: "images/demo.png", real: "drive/images/demo.png", size: 10, modTime: 1}
	maker := &Maker{
		handlers: map[string]TypeHandler{
			"png": &failingThumbnailHandler{calls: calls},
		},
	}
	request := artifact.ArtifactRequest{Source: entry, Type: ThumbnailArtifactType}
	first, err := maker.Identity(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := maker.Produce(task.DummyContext(), request, discardWriter{}); err == nil || !artifact.IsCacheable(err) {
		t.Fatalf("Produce() = %v, want cacheable error", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
	entry.size++
	second, err := maker.Identity(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint == "" || first.Fingerprint == second.Fingerprint {
		t.Fatalf("fingerprint did not change after source update: %q -> %q", first.Fingerprint, second.Fingerprint)
	}
}

func TestThumbnailArtifactNameUsesOutputExtension(t *testing.T) {
	if got := thumbnailArtifactName("demo.png", "image/jpeg"); !strings.HasPrefix(got, "demo.") || got == "demo.png" {
		t.Fatalf("jpeg thumbnail name = %q", got)
	}
	if got := thumbnailArtifactName("notes.txt", "image/svg+xml"); !strings.HasPrefix(got, "notes.") || got == "notes.txt" {
		t.Fatalf("svg thumbnail name = %q", got)
	}
	if got := thumbnailArtifactName("video.mp4", ""); got != "video.thumb" {
		t.Fatalf("unknown thumbnail name = %q, want video.thumb", got)
	}
}

type discardWriter struct{}

func (discardWriter) WriteMeta(artifact.Meta) error { return nil }

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
