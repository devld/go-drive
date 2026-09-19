package artifact

import (
	"context"
	"go-drive/common/types"
	"io"
	"testing"
)

type identityTestEntry struct {
	path    string
	real    string
	size    int64
	modTime int64
}

func (e *identityTestEntry) Path() string          { return e.path }
func (e *identityTestEntry) Name() string          { return e.path }
func (e *identityTestEntry) Type() types.EntryType { return types.TypeFile }
func (e *identityTestEntry) Size() int64           { return e.size }
func (e *identityTestEntry) ModTime() int64        { return e.modTime }
func (e *identityTestEntry) Meta() types.EntryMeta { return types.EntryMeta{Readable: true} }
func (e *identityTestEntry) Drive() types.IDrive   { return nil }
func (e *identityTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, nil
}
func (e *identityTestEntry) GetURL(context.Context) (*types.ContentURL, error) { return nil, nil }
func (e *identityTestEntry) GetDispatchedDrive() (string, types.IDrive)        { return "drive", nil }
func (e *identityTestEntry) GetRealPath() string                               { return e.real }

func TestBindSourceComposesHandlerIdentity(t *testing.T) {
	entry := &identityTestEntry{path: "photos/a.png", real: "drive/photos/a.png", size: 10, modTime: 1}
	got := bindSource(entry, ResolvedRequest{Key: "member", Fingerprint: "v1"})
	wantKey := "drive/photos/a.png|member"
	if got.Key != wantKey {
		t.Fatalf("Key = %q, want %q", got.Key, wantKey)
	}
	wantFP := "path=photos/a.png|type=file|size=10|mod=1|v1"
	if got.Fingerprint != wantFP {
		t.Fatalf("Fingerprint = %q, want %q", got.Fingerprint, wantFP)
	}
	entry.size++
	after := bindSource(entry, ResolvedRequest{Key: "member", Fingerprint: "v1"})
	if after.Key != got.Key {
		t.Fatalf("Key changed after source update: %q -> %q", got.Key, after.Key)
	}
	if after.Fingerprint == got.Fingerprint {
		t.Fatal("Fingerprint did not change after source update")
	}
}

func TestBindSourceSkipsNilEntry(t *testing.T) {
	got := bindSource(nil, ResolvedRequest{Key: "k", Fingerprint: "fp"})
	if got.Key != "k" || got.Fingerprint != "fp" {
		t.Fatalf("bindSource(nil) = %#v", got)
	}
}
