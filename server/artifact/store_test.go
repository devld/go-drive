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

func newTestStore(t *testing.T, policies map[string]Policy) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range []string{"thumbnail", "archive-index", "archive-content"} {
		if err := store.registerType(bucket, policies[bucket]); err != nil {
			t.Fatalf("register artifact bucket %q: %v", bucket, err)
		}
	}
	return store
}

func writeArtifact(t *testing.T, store *Store, bucket, key, fingerprint string, meta Meta, body string) Info {
	t.Helper()
	writer, err := store.Create(bucket, key, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteMeta(meta); err != nil {
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
	return writer.info()
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
	var public map[string]any
	if err := json.Unmarshal(encoded, &public); err != nil {
		t.Fatal(err)
	}
	if _, ok := public["type"]; ok {
		t.Fatalf("artifact info leaked handler: %s", encoded)
	}
	if info.Name != "image.png" || info.Size != int64(len("raw-payload")) {
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
	writeArtifact(t, store, "archive-content", "old", "old", Meta{Name: "old"}, "1234")
	writeArtifact(t, store, "archive-content", "new", "new", Meta{Name: "new"}, "5678")
	writeArtifact(t, store, "archive-content", "latest", "latest", Meta{Name: "latest"}, "9012")
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
	store := newTestStore(t, map[string]Policy{"archive-content": {MaxBytes: 3}})
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

func TestCleanDropsIdleArtifactWhenTTLIsZero(t *testing.T) {
	store := newTestStore(t, nil)
	writeArtifact(t, store, "archive-content", "pack", "pack", Meta{Name: "pack.zip"}, "zip")
	opened, err := store.Open("archive-content", "pack")
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := store.Clean("archive-content", 0, 0); err != nil || removed != 0 {
		t.Fatalf("Clean() while reading = removed:%d err:%v, want the open artifact kept", removed, err)
	}
	_ = opened.Body.Close()
	if _, err := store.Lookup("archive-content", "pack", "pack", 0); err != nil {
		t.Fatalf("closed artifact was removed before the next cleanup: %v", err)
	}
	if removed, err := store.Clean("archive-content", 0, 0); err != nil || removed != 1 {
		t.Fatalf("Clean() after close = removed:%d err:%v, want the idle artifact removed", removed, err)
	}
	if _, err := store.Open("archive-content", "pack"); err == nil {
		t.Fatal("idle artifact remained after cleanup")
	}
}

func TestCleanKeepsExpiredArtifactWhileItIsOpen(t *testing.T) {
	store := newTestStore(t, nil)
	writeArtifact(t, store, "archive-content", "live", "live", Meta{Name: "live"}, "abc")
	opened, err := store.Open("archive-content", "live")
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := store.Clean("archive-content", time.Nanosecond, 0); err != nil || removed != 0 {
		t.Fatalf("Clean() while reading = removed:%d err:%v, want the open artifact kept", removed, err)
	}
	probe, err := store.Open("archive-content", "live")
	if err != nil {
		t.Fatalf("open artifact was removed: %v", err)
	}
	_ = probe.Body.Close()
	_ = opened.Body.Close()
	if removed, err := store.Clean("archive-content", time.Nanosecond, 0); err != nil || removed == 0 {
		t.Fatalf("Clean() after close = removed:%d err:%v, want expiry", removed, err)
	}
}
