package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go-drive/common"
	driveErr "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type testEntry struct {
	path     string
	filename string
	data     []byte
	seekable bool
	size     int64
	modTime  int64
	reads    *atomic.Int32
}

func (e *testEntry) Path() string          { return e.path }
func (e *testEntry) Name() string          { return filepath.Base(e.path) }
func (e *testEntry) Type() types.EntryType { return types.TypeFile }
func (e *testEntry) Size() int64           { return e.size }
func (e *testEntry) ModTime() int64        { return e.modTime }
func (e *testEntry) Meta() types.EntryMeta { return types.EntryMeta{Readable: true} }
func (e *testEntry) Drive() types.IDrive   { return nil }
func (e *testEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, driveErr.NewUnsupportedError()
}
func (e *testEntry) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	if e.reads != nil {
		e.reads.Add(1)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !e.seekable && e.data != nil {
		startOffset := start
		if startOffset < 0 {
			startOffset = 0
		}
		end := int64(len(e.data))
		if size > 0 && startOffset+size < end {
			end = startOffset + size
		}
		return io.NopCloser(bytes.NewReader(e.data[startOffset:end])), nil
	}
	file, err := os.Open(e.filename)
	if err != nil {
		return nil, err
	}
	if start >= 0 {
		if _, err := file.Seek(start, io.SeekStart); err != nil {
			_ = file.Close()
			return nil, err
		}
		if size > 0 {
			return &limitedFile{File: file, Reader: io.LimitReader(file, size)}, nil
		}
	}
	return file, nil
}

type limitedFile struct {
	*os.File
	io.Reader
}

func (f *limitedFile) Read(p []byte) (int, error) { return f.Reader.Read(p) }

func archiveTestConfig() common.ArchiveConfig {
	return common.ArchiveConfig{
		MaxSize:       "1m",
		MaxMemberSize: "1m",
		MaxEntries:    100,
		CacheItems:    2,
		CacheSize:     "2m",
		IndexTTL:      time.Hour,
	}
}

func newTestPreviewer(t *testing.T, config common.ArchiveConfig, tempDir string) *Previewer {
	t.Helper()
	store, err := artifact.NewStore(filepath.Join(tempDir, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	components := registry.NewComponentHolder()
	t.Cleanup(func() { _ = components.Dispose() })
	if _, err := artifact.NewService(store, nil, artifact.HandlerContext{
		Config:     common.Config{TempDir: tempDir, Archive: config},
		Components: components,
	}); err != nil {
		t.Fatal(err)
	}
	previewer, ok := components.Get(registry.KeyArchive).(*Previewer)
	if !ok {
		t.Fatal("archive handler was not registered")
	}
	return previewer
}

func produceBody(t *testing.T, previewer *Previewer, ctx types.TaskCtx, request artifact.ArtifactRequest) []byte {
	t.Helper()
	if ctx == nil {
		ctx = task.DummyContext()
	}
	var buf bytes.Buffer
	if err := previewer.Produce(ctx, request, bufferWriter{Buffer: &buf}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type bufferWriter struct {
	*bytes.Buffer
}

func (bufferWriter) WriteMeta(artifact.Meta) error { return nil }

type discardWriter struct{}

func (discardWriter) WriteMeta(artifact.Meta) error { return nil }

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func makeZip(t *testing.T) (string, int64) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "archive-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for name, data := range map[string]string{
		"README.txt":   "archive preview",
		"docs/info.md": "nested content",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return file.Name(), info.Size()
}

func TestPreviewerProducesIndexAndContent(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	entry := &testEntry{path: "demo.zip", filename: filename, size: size, modTime: 42}

	indexBody := produceBody(t, previewer, nil, artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	})
	var entries []Entry
	if err := json.Unmarshal(indexBody, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("artifact index entries = %d, want 3", len(entries))
	}

	content := produceBody(t, previewer, nil, artifact.ArtifactRequest{
		Source: entry, Type: ContentArtifactType, Args: "docs/info.md",
	})
	if string(content) != "nested content" {
		t.Fatalf("content artifact = %q", content)
	}
}

func TestPreviewerUsesSourceCacheForNonSeekableSource(t *testing.T) {
	filename, size := makeZip(t)
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	reads := &atomic.Int32{}
	entry := &testEntry{path: "remote/demo.zip", filename: filename, data: data, size: size, reads: reads}
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	_ = produceBody(t, previewer, nil, artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	})
	readsAfterIndex := reads.Load()
	content := produceBody(t, previewer, nil, artifact.ArtifactRequest{
		Source: entry, Type: ContentArtifactType, Args: "README.txt",
	})
	if string(content) != "archive preview" {
		t.Fatalf("content = %q", content)
	}
	if got := reads.Load(); got != readsAfterIndex {
		t.Fatalf("source reopened after cached index: %d -> %d", readsAfterIndex, got)
	}
}

func TestPreviewerMarksTooManyEntriesFailureCacheable(t *testing.T) {
	filename, size := makeZip(t)
	entry := &testEntry{path: "many.zip", filename: filename, size: size}
	config := archiveTestConfig()
	config.MaxEntries = 1
	previewer := newTestPreviewer(t, config, t.TempDir())
	err := previewer.Produce(task.DummyContext(), artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	}, discardWriter{})
	if found, ok := errors.AsType[driveErr.NotFoundError](err); !ok || found.Error() != msgArchiveTooManyEntries || !artifact.IsCacheable(err) {
		t.Fatalf("Produce() = %v, want cacheable too-many-entries", err)
	}
}

func TestPreviewerMarksUnknownSizeLimitRejectionCacheable(t *testing.T) {
	filename, _ := makeZip(t)
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	entry := &testEntry{path: "remote/unknown-size.zip", filename: filename, data: data, size: -1}
	config := archiveTestConfig()
	config.MaxSize = "1b"
	previewer := newTestPreviewer(t, config, t.TempDir())
	produceErr := previewer.Produce(task.DummyContext(), artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	}, discardWriter{})
	if found, ok := errors.AsType[driveErr.NotFoundError](produceErr); !ok || found.Error() != msgArchiveTooLarge || !artifact.IsCacheable(produceErr) {
		t.Fatalf("Produce() = %v, want cacheable archive-too-large", produceErr)
	}
}

func TestPreviewerMarksUnsupportedAndInvalidArchiveCacheable(t *testing.T) {
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	unsupported := &testEntry{path: "demo.zip", data: []byte("not-an-archive"), size: 14}
	err := previewer.Produce(task.DummyContext(), artifact.ArtifactRequest{
		Source: unsupported, Type: IndexArtifactType,
	}, discardWriter{})
	if found, ok := errors.AsType[driveErr.NotFoundError](err); !ok || found.Error() != msgUnsupportedArchive || !artifact.IsCacheable(err) {
		t.Fatalf("Produce(unsupported) = %v, want cacheable unsupported format", err)
	}

	invalid := &testEntry{path: "broken.zip", data: []byte("PK\x03\x04not-a-zip"), size: 12}
	err = previewer.Produce(task.DummyContext(), artifact.ArtifactRequest{
		Source: invalid, Type: IndexArtifactType,
	}, discardWriter{})
	if found, ok := errors.AsType[driveErr.NotFoundError](err); !ok || found.Error() != msgInvalidArchive || !artifact.IsCacheable(err) {
		t.Fatalf("Produce(invalid) = %v, want cacheable invalid archive", err)
	}
}

func TestPreviewerReportsProgressAndRegistrationConfig(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	entry := &testEntry{path: "demo.zip", filename: filename, size: size}
	ctx := task.NewTaskContext(context.Background())
	_ = produceBody(t, previewer, ctx, artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	})
	if ctx.GetTotal() != size || ctx.GetProgress() != size {
		t.Fatalf("progress = %d/%d, want %d/%d", ctx.GetProgress(), ctx.GetTotal(), size, size)
	}
	registrations := previewer.Registrations()
	if len(registrations) != 2 {
		t.Fatalf("archive registrations = %d, want 2", len(registrations))
	}
	if registrations[0].Type != IndexArtifactType || registrations[1].Type != ContentArtifactType {
		t.Fatalf("archive registrations = %#v", registrations)
	}
	config := previewer.Config()
	if config["extensions"] != supportedExtensions || config["maxSize"] != int64(1024*1024) {
		t.Fatalf("archive config = %#v", config)
	}
}

func TestPreviewerSupportsSevenZipAndRejectsUnsafeMember(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("N3q8ryccAASgR6WICAAAAAAAAABmAAAAAAAAAN2R8/FiYXIKZm9vCgEEBgACCQQEAAcLAgABAQABAQAMBAQACAoB6bOiBKhlMn4AAAUCGQUAAAAAABERAGIAYQByAAAAZgBvAG8AAAAZAgAAFBIBAACFM3PyY9YBAFgCcvJj1gEVCgEAIICkgSCApIEAAA==")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "demo.7z")
	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	entry := &testEntry{path: "demo.7z", filename: archivePath, size: int64(len(data))}
	indexBody := produceBody(t, previewer, nil, artifact.ArtifactRequest{
		Source: entry, Type: IndexArtifactType,
	})
	var entries []Entry
	if err := json.Unmarshal(indexBody, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name != "bar" || entries[1].Name != "foo" {
		t.Fatalf("entries = %+v", entries)
	}
	if err := previewer.Produce(task.DummyContext(), artifact.ArtifactRequest{
		Source: entry, Type: ContentArtifactType, Args: "../foo",
	}, discardWriter{}); err == nil {
		t.Fatal("unsafe member was accepted")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"zip", []byte("PK\x03\x04"), "archives.Zip"},
		{"7z", []byte("7z\xBC\xAF\x27\x1C"), "archives.SevenZip"},
		{"rar", []byte("Rar!\x1A\x07\x01\x00"), "archives.Rar"},
	}
	previewer := &Previewer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := previewer.detectFormat(tt.name, bytes.NewReader(tt.data), int64(len(tt.data)))
			if err != nil {
				t.Fatal(err)
			}
			if name := fmt.Sprintf("%T", got); name != tt.want {
				t.Fatalf("format = %s, want %s", name, tt.want)
			}
		})
	}
}
