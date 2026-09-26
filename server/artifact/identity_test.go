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
func (e *identityTestEntry) GetReader(context.Context, types.ReaderRange) (io.ReadCloser, error) {
	return nil, nil
}
func (e *identityTestEntry) GetURL(context.Context) (*types.ContentURL, error) { return nil, nil }
func (e *identityTestEntry) GetDispatchedDrive() (string, types.IDrive)        { return "drive", nil }
func (e *identityTestEntry) GetRealPath() string                               { return e.real }

func TestBindSourceKeepsHandlerFields(t *testing.T) {
	handler := ResolvedRequest{Key: "member", Fingerprint: "v1", Cache: "content"}
	entry := &identityTestEntry{path: "photos/a.png", real: "drive/photos/a.png", size: 10, modTime: 1}
	got := bindSource(entry, handler)
	if got.ResolvedRequest != handler {
		t.Fatalf("handler fields = %#v, want %#v", got.ResolvedRequest, handler)
	}
	if got.fullKey != "drive/photos/a.png|member" {
		t.Fatalf("fullKey = %q", got.fullKey)
	}
	wantFP := "path=photos/a.png|type=file|size=10|mod=1|v1"
	if got.fullFingerprint != wantFP {
		t.Fatalf("fullFingerprint = %q, want %q", got.fullFingerprint, wantFP)
	}
	entry.size++
	after := bindSource(entry, handler)
	if after.ResolvedRequest != handler {
		t.Fatalf("handler fields changed after source update: %#v", after.ResolvedRequest)
	}
	if after.fullKey != got.fullKey {
		t.Fatalf("fullKey changed after source update: %q -> %q", got.fullKey, after.fullKey)
	}
	if after.fullFingerprint == got.fullFingerprint {
		t.Fatal("fullFingerprint did not change after source update")
	}
}
