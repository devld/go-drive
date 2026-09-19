package artifact

import (
	"encoding/json"
	"errors"
	apierr "go-drive/common/errors"
	"io"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T, policies map[ArtifactType]Policy) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []ArtifactType{"thumbnail", "archive-index", "archive-content"} {
		if err := store.registerType(typ, policies[typ]); err != nil {
			t.Fatalf("register artifact type %q: %v", typ, err)
		}
	}
	return store
}

func writeArtifact(t *testing.T, store *Store, typ ArtifactType, key, fingerprint string, meta Meta, body string) Info {
	t.Helper()
	return writeArtifactAt(t, store, typ, key, fingerprint, time.Time{}, meta, body)
}

func writeArtifactAt(t *testing.T, store *Store, typ ArtifactType, key, fingerprint string, createdAt time.Time, meta Meta, body string) Info {
	t.Helper()
	writer, err := store.Create(typ, key, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if err := writer.writeMeta(meta, createdAt); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, body); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.Info()
}

func TestStoreWritesRawBodyAndHidesFingerprint(t *testing.T) {
	store := newTestStore(t, nil)
	info := writeArtifact(t, store, "thumbnail", "drive/image.png", "private/source/fingerprint", Meta{
		Name:     "image.png",
		MimeType: "image/png",
		ModTime:  time.UnixMilli(42),
	}, "raw-payload")
	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private/source/fingerprint") {
		t.Fatalf("artifact info leaked fingerprint: %s", encoded)
	}
	if strings.Contains(string(encoded), `"id"`) {
		t.Fatalf("artifact info leaked storage id: %s", encoded)
	}
	if info.Type != "thumbnail" || info.Name != "image.png" || info.Size != int64(len("raw-payload")) {
		t.Fatalf("Write() info = %#v", info)
	}

	lookedUp, err := store.Lookup("thumbnail", "drive/image.png", "private/source/fingerprint", time.Hour)
	if err != nil {
		t.Fatalf("Lookup() = info=%v err=%v", lookedUp, err)
	}
	opened, err := store.Open("thumbnail", "drive/image.png")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(opened.Body)
	_ = opened.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "raw-payload" {
		t.Fatalf("payload = %q, want raw-payload", data)
	}
}

func TestStoreFailureAndFingerprintInvalidation(t *testing.T) {
	store := newTestStore(t, nil)
	if err := store.WriteFailure("archive-index", "archive.zip", "fingerprint-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Lookup("archive-index", "archive.zip", "fingerprint-a", time.Hour); err == nil || !apierr.IsNotFoundError(err) {
		t.Fatalf("Lookup() err=%v, want NotFoundError", err)
	}
	if _, err := store.Lookup("archive-index", "archive.zip", "fingerprint-b", time.Hour); !errors.Is(err, errCacheMiss) {
		t.Fatalf("stale failure err=%v, want cache miss", err)
	}
	writeArtifact(t, store, "archive-index", "archive.zip", "fingerprint-a", Meta{}, "index")
	if got, err := store.Lookup("archive-index", "archive.zip", "fingerprint-a", time.Hour); err != nil {
		t.Fatalf("failure survived successful write: info=%v err=%v", got, err)
	}
	opened, err := store.Open("archive-index", "archive.zip")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(opened.Body)
	_ = opened.Body.Close()
	if err != nil || string(data) != "index" {
		t.Fatalf("payload = %q err=%v", data, err)
	}
}

func TestStoreEvictsOldestPayloadAndProtectsActiveReader(t *testing.T) {
	store := newTestStore(t, nil)
	writeArtifactAt(t, store, "archive-content", "old", "old", time.Now().Add(-time.Hour), Meta{Name: "old"}, "1234")
	writeArtifactAt(t, store, "archive-content", "new", "new", time.Now(), Meta{Name: "new"}, "5678")
	writeArtifactAt(t, store, "archive-content", "latest", "latest", time.Now().Add(time.Minute), Meta{Name: "latest"}, "9012")
	active, err := store.Open("archive-content", "old")
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := store.Clean("archive-content", time.Hour*2, 8); err != nil || removed == 0 {
		t.Fatalf("Clean() with active reader = removed:%d err:%v, want another item evicted", removed, err)
	}
	if _, err := store.Open("archive-content", "latest"); err != nil {
		t.Fatalf("latest artifact was unexpectedly removed: %v", err)
	}
	_ = active.Body.Close()
	if removed, err := store.Clean("archive-content", time.Hour*2, 4); err != nil || removed == 0 {
		t.Fatalf("Clean() after reader close = removed:%d err:%v, want eviction", removed, err)
	}
	if _, err := store.Open("archive-content", "old"); err == nil {
		t.Fatal("old artifact still available after byte eviction")
	}
	if _, err := store.Open("archive-content", "new"); err == nil {
		t.Fatal("evicted intermediate artifact still available")
	}
}

func TestStoreServesOversizedPayloadOnce(t *testing.T) {
	store := newTestStore(t, map[ArtifactType]Policy{"archive-content": {MaxBytes: 3}})
	writeArtifact(t, store, "archive-content", "oversized", "oversized", Meta{Name: "oversized"}, "1234")
	opened, err := store.Open("archive-content", "oversized")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(opened.Body)
	_ = opened.Body.Close()
	if err != nil || string(data) != "1234" {
		t.Fatalf("oversized payload = %q err=%v", data, err)
	}
	if _, err := store.Open("archive-content", "oversized"); err == nil {
		t.Fatal("oversized payload remained after its first reader closed")
	}
}
