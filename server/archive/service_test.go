package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"go-drive/common"
	driveErr "go-drive/common/errors"
	"go-drive/common/types"
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
	file, err := os.Open(e.filename)
	if err != nil {
		return nil, err
	}
	if !e.seekable && e.data != nil {
		_ = file.Close()
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

func TestServiceListAndOpenZip(t *testing.T) {
	filename, size := makeZip(t)
	reads := &atomic.Int32{}
	entry := &testEntry{
		path:     "archives/demo.zip",
		filename: filename,
		size:     size,
		modTime:  time.Now().UnixMilli(),
		reads:    reads,
	}
	tempDir := t.TempDir()
	service, err := NewService(archiveTestConfig(), tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	root, err := service.List(context.Background(), entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(root) != 2 || root[0].Name != "docs" || root[1].Name != "README.txt" {
		t.Fatalf("root entries = %+v", root)
	}
	nested, err := service.List(context.Background(), entry, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) != 1 || nested[0].Path != "docs/info.md" {
		t.Fatalf("nested entries = %+v", nested)
	}

	opened, err := service.Open(context.Background(), entry, "docs/info.md")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(opened.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Reader.Close(); err != nil {
		t.Fatal(err)
	}
	if string(data) != "nested content" {
		t.Fatalf("content = %q", data)
	}
	if reads.Load() == 0 {
		t.Fatal("expected the entry reader to be used")
	}
	if files, err := os.ReadDir(filepath.Join(tempDir, "archive-sources")); err != nil {
		t.Fatal(err)
	} else if len(files) != 0 {
		t.Fatalf("local source was copied into the range cache: %v", files)
	}
}

func TestServicePersistsIndexWithoutReopeningSource(t *testing.T) {
	filename, size := makeZip(t)
	reads := &atomic.Int32{}
	entry := &testEntry{path: "demo.zip", filename: filename, size: size, reads: reads}
	tempDir := t.TempDir()
	service, err := NewService(archiveTestConfig(), tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), entry, ""); err != nil {
		t.Fatal(err)
	}
	firstReads := reads.Load()
	if firstReads == 0 {
		t.Fatal("expected the first list to read the source")
	}
	if err := service.Dispose(); err != nil {
		t.Fatal(err)
	}

	second, err := NewService(archiveTestConfig(), tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.List(context.Background(), entry, ""); err != nil {
		t.Fatal(err)
	}
	if got := reads.Load(); got != firstReads {
		t.Fatalf("persisted index reopened source: reads %d -> %d", firstReads, got)
	}
}

func TestServiceUsesCacheForNonSeekableSource(t *testing.T) {
	filename, size := makeZip(t)
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	entry := &testEntry{
		path:     "remote/demo.zip",
		filename: filename,
		data:     data,
		size:     size,
		reads:    &atomic.Int32{},
	}
	service, err := NewService(archiveTestConfig(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), entry, ""); err != nil {
		t.Fatal(err)
	}
	readsAfterList := entry.reads.Load()
	opened, err := service.Open(context.Background(), entry, "README.txt")
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(opened.Reader)
	_ = opened.Reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "archive preview" {
		t.Fatalf("content = %q", content)
	}
	if got := entry.reads.Load(); got != readsAfterList {
		t.Fatalf("non-seekable source opened %d times after cache hit, want %d", got, readsAfterList)
	}
}

func TestServiceListAndOpenSevenZip(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("N3q8ryccAASgR6WICAAAAAAAAABmAAAAAAAAAN2R8/FiYXIKZm9vCgEEBgACCQQEAAcLAgABAQABAQAMBAQACAoB6bOiBKhlMn4AAAUCGQUAAAAAABERAGIAYQByAAAAZgBvAG8AAAAZAgAAFBIBAACFM3PyY9YBAFgCcvJj1gEVCgEAIICkgSCApIEAAA==")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "demo.7z")
	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	entry := &testEntry{path: "demo.7z", filename: archivePath, size: int64(len(data))}
	service, err := NewService(archiveTestConfig(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.List(context.Background(), entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Name != "bar" || items[1].Name != "foo" {
		t.Fatalf("entries = %+v", items)
	}
	opened, err := service.Open(context.Background(), entry, "foo")
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(opened.Reader)
	closeErr := opened.Reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if string(content) != "foo\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestServicePreservesFormatAndSizeErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-an-archive.zip")
	if err := os.WriteFile(path, []byte("not an archive"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := &testEntry{path: "not-an-archive.zip", filename: path, size: 14}
	service, err := NewService(archiveTestConfig(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.List(context.Background(), entry, "")
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("unsupported archive error = %v, want ErrUnsupportedFormat", err)
	}

	limitedConfig := archiveTestConfig()
	limitedConfig.MaxSize = "1b"
	limited, err := NewService(limitedConfig, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = limited.List(context.Background(), entry, "")
	if !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("large archive error = %v, want ErrArchiveTooLarge", err)
	}
}

func TestServiceDoesNotTrustPersistedEntryLimit(t *testing.T) {
	filename, size := makeZip(t)
	entry := &testEntry{path: "demo.zip", filename: filename, size: size}
	tempDir := t.TempDir()
	first, err := NewService(archiveTestConfig(), tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.List(context.Background(), entry, ""); err != nil {
		t.Fatal(err)
	}

	limitedConfig := archiveTestConfig()
	limitedConfig.MaxEntries = 1
	second, err := NewService(limitedConfig, tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = second.List(context.Background(), entry, "")
	if !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("limited persisted index error = %v, want ErrTooManyEntries", err)
	}
}

func TestServiceRejectsUnsafeMember(t *testing.T) {
	filename, size := makeZip(t)
	entry := &testEntry{path: "demo.zip", filename: filename, size: size}
	service, err := NewService(archiveTestConfig(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, openErr := service.Open(context.Background(), entry, "../README.txt")
	if openErr == nil {
		t.Fatal("Open() error = nil, want invalid path")
	}
	if _, ok := openErr.(driveErr.BadRequestError); !ok {
		t.Fatalf("Open() error type = %T, want BadRequestError", openErr)
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
	service := &Service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			got, err := service.detectFormat(tt.name, reader, int64(len(tt.data)))
			if err != nil {
				t.Fatal(err)
			}
			if name := fmt.Sprintf("%T", got); name != tt.want {
				t.Fatalf("format = %s, want %s", name, tt.want)
			}
		})
	}
}
