package driveutil

import (
	apierr "go-drive/common/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testBlobMeta struct {
	Name string `json:"name"`
}

func TestBlobCacheAtomicWriteAndOpen(t *testing.T) {
	cache, err := NewBlobCache[testBlobMeta](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "archive-index", testBlobMeta{Name: "index"}, "metadata")
	meta, size, body, err := cache.Open("archive-index")
	if err != nil {
		t.Fatalf("Open() = err=%v", err)
	}
	if meta.Name != "index" || size != int64(len("metadata")) {
		t.Fatalf("Open() = meta=%#v size=%d", meta, size)
	}
	data, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || string(data) != "metadata" {
		t.Fatalf("data = %q err=%v", data, err)
	}
	meta, size, err = cache.ReadMeta("archive-index")
	if err != nil || meta.Name != "index" || size != int64(len("metadata")) {
		t.Fatalf("ReadMeta() = %#v size=%d err=%v", meta, size, err)
	}
	items, err := cache.Items()
	if err != nil || len(items) != 1 || items[0].Key != "archive-index" || items[0].Size != int64(len("metadata")) {
		t.Fatalf("Items() = %+v err=%v", items, err)
	}
	reopened, err := NewBlobCache[testBlobMeta](cache.dir)
	if err != nil {
		t.Fatal(err)
	}
	items, err = reopened.Items()
	if err != nil || len(items) != 1 || items[0].Key != "archive-index" {
		t.Fatalf("reopened Items() = %+v err=%v", items, err)
	}
}

func TestBlobCacheWriterWriteMetaThenBody(t *testing.T) {
	cache, err := NewBlobCache[testBlobMeta](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writer, err := cache.Create("preview")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("too-early")); err == nil {
		t.Fatal("Write() succeeded before WriteMeta")
	}
	if err := writer.WriteMeta(testBlobMeta{Name: "preview"}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, "body"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	size := writer.Size()
	if size != int64(len("body")) {
		t.Fatalf("Size() = %d, want %d", size, len("body"))
	}
	meta, gotSize, body, err := cache.Open("preview")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || meta.Name != "preview" || gotSize != size || string(data) != "body" {
		t.Fatalf("Open() = meta=%#v size=%d data=%q err=%v", meta, gotSize, data, err)
	}
}

func TestBlobCacheShardsByKeyPrefix(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewBlobCache[testBlobMeta](dir)
	if err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "abcdef", testBlobMeta{}, "x")
	if _, err := os.Stat(filepath.Join(dir, "ab", "abcdef")); err != nil {
		t.Fatal(err)
	}
}

func TestBlobCacheStartupCleanup(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewBlobCache[testBlobMeta](dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"stale.lock", "stale.tmp"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	orphan := filepath.Join(dir, "or", "orphan")
	if err := os.MkdirAll(filepath.Dir(orphan), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphan, nil, 0600); err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "keep", testBlobMeta{Name: "keep"}, "ok")
	removed, err := cache.CleanStartup()
	if err != nil || removed != 3 {
		t.Fatalf("CleanStartup() = removed=%d err=%v, want 3", removed, err)
	}
	if _, _, _, err := cache.Open("keep"); err != nil {
		t.Fatalf("keep payload missing: err=%v", err)
	}
}

func TestBlobCacheStartupKeepsEmptyPublishedBody(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewBlobCache[testBlobMeta](dir)
	if err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "empty-payload", testBlobMeta{Name: "empty"}, "")
	orphan := filepath.Join(dir, "or", "orphan")
	if err := os.MkdirAll(filepath.Dir(orphan), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphan, nil, 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := cache.CleanStartup()
	if err != nil || removed != 1 {
		t.Fatalf("CleanStartup() = removed=%d err=%v, want 1", removed, err)
	}
	if _, _, body, err := cache.Open("empty-payload"); err != nil {
		t.Fatalf("published empty body missing: err=%v", err)
	} else {
		data, err := io.ReadAll(body)
		_ = body.Close()
		if err != nil || len(data) != 0 {
			t.Fatalf("empty payload = %q err=%v", data, err)
		}
	}
	meta, size, err := cache.ReadMeta("empty-payload")
	if err != nil || meta.Name != "empty" || size != 0 {
		t.Fatalf("published metadata = %#v size=%d err=%v", meta, size, err)
	}
}

func TestBlobCacheReadMetaRejectsSizeMismatch(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewBlobCache[testBlobMeta](dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "br", "broken")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	header, err := encodeBlobHeader(testBlobMeta{Name: "broken"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(header, []byte("payload")...), 0600); err != nil {
		t.Fatal(err)
	}
	if meta, size, e := cache.ReadMeta("broken"); e == nil || !apierr.IsNotFoundError(e) {
		t.Fatalf("ReadMeta() = meta=%#v size=%d err=%v, want NotFoundError", meta, size, e)
	}
	if _, _, _, e := cache.Open("broken"); e == nil || !apierr.IsNotFoundError(e) {
		t.Fatalf("corrupt payload still present: err=%v", e)
	}
}

func TestBlobCacheWriteReplacesRecord(t *testing.T) {
	cache, err := NewBlobCache[testBlobMeta](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "same", testBlobMeta{Name: "fail"}, "")
	putBlob(t, cache, "same", testBlobMeta{Name: "ok"}, "ok")
	meta, size, err := cache.ReadMeta("same")
	if err != nil || meta.Name != "ok" || size != 2 {
		t.Fatalf("ReadMeta() = %#v size=%d err=%v", meta, size, err)
	}
	_, _, body, err := cache.Open("same")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || string(data) != "ok" {
		t.Fatalf("payload = %q err=%v", data, err)
	}
}

func TestBlobCacheHeaderIsJSON(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewBlobCache[testBlobMeta](dir)
	if err != nil {
		t.Fatal(err)
	}
	putBlob(t, cache, "abcdef", testBlobMeta{Name: "visible"}, "")
	raw, err := os.ReadFile(filepath.Join(dir, "ab", "abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"name":"visible"`) {
		t.Fatalf("header = %q", raw)
	}
}

func TestBlobCacheHeaderLengthIsStable(t *testing.T) {
	zero, err := encodeBlobHeader(testBlobMeta{Name: "n"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	large, err := encodeBlobHeader(testBlobMeta{Name: "n"}, 1<<60)
	if err != nil {
		t.Fatal(err)
	}
	if len(zero) != len(large) {
		t.Fatalf("header length %d vs %d", len(zero), len(large))
	}
}

func putBlob(t *testing.T, cache *BlobCache[testBlobMeta], key string, meta testBlobMeta, body string) {
	t.Helper()
	writer, err := cache.Create(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteMeta(meta); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	if body != "" {
		if _, err := io.WriteString(writer, body); err != nil {
			writer.Abort()
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}
