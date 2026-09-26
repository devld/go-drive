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
func (e *testThumbnailEntry) GetReader(context.Context, types.ReaderRange) (io.ReadCloser, error) {
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

func TestMakerSpecUsesSortedCommaSeparatedExtensions(t *testing.T) {
	maker := &Maker{
		handlers: map[string]TypeHandler{
			"png": nil,
			"/":   nil,
			"jpg": nil,
		},
	}
	config := maker.Spec().Config
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

func TestMakerMarksGenerationFailureCacheable(t *testing.T) {
	calls := &atomic.Int32{}
	entry := &testThumbnailEntry{path: "images/demo.png", real: "drive/images/demo.png", size: 10, modTime: 1}
	maker := &Maker{
		handlers: map[string]TypeHandler{
			"png": &failingThumbnailHandler{calls: calls},
		},
	}
	request := artifact.Request{Source: entry}
	resolved, err := maker.Resolve(request)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Key != "" || resolved.Fingerprint != "v1" {
		t.Fatalf("Resolve() = %#v, want empty key and v1 fingerprint", resolved)
	}
	if err := maker.Produce(task.DummyContext(), request, discardWriter{}); err == nil || !artifact.IsCacheable(err) {
		t.Fatalf("Produce() = %v, want cacheable error", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
	entry.size++
	after, err := maker.Resolve(request)
	if err != nil {
		t.Fatal(err)
	}
	if after.Fingerprint != resolved.Fingerprint {
		t.Fatalf("handler fingerprint changed after source update: %q -> %q", resolved.Fingerprint, after.Fingerprint)
	}
}

func TestThumbnailArtifactNameUsesOutputExtension(t *testing.T) {
	if got := thumbnailArtifactName("demo.png", "image/jpeg"); got != "demo.jpg" {
		t.Fatalf("jpeg thumbnail name = %q, want demo.jpg", got)
	}
	if got := thumbnailArtifactName("notes.txt", "image/svg+xml"); got != "notes.svg" {
		t.Fatalf("svg thumbnail name = %q, want notes.svg", got)
	}
	if got := thumbnailArtifactName("video.mp4", ""); got != "video.thumb" {
		t.Fatalf("unknown thumbnail name = %q, want video.thumb", got)
	}
}

type discardWriter struct{}

func (discardWriter) WriteMeta(artifact.Meta) error { return nil }

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
