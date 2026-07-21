package server

import (
	"context"
	"io"
	"testing"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/types"
)

func TestCheckCopyOrMovePathBoundary(t *testing.T) {
	if e := checkCopyOrMove("drive/dir", "drive/dir/child"); e == nil {
		t.Fatal("moving to a child path should be rejected")
	}
	if e := checkCopyOrMove("drive/dir", "drive/directory/child"); e != nil {
		t.Fatalf("a segment prefix is not a child path: %v", e)
	}
}

func TestFindNonExistsEntryNameWithReservedReservesBatchNames(t *testing.T) {
	drive := &mountNameTestDrive{existing: map[string]bool{"drive/foo.txt": true}}
	reserved := map[string]bool{"drive/foo_1.txt": true}
	got, e := driveutil.FindNonExistsEntryNameWithReserved(context.Background(), drive, "drive/foo.txt", reserved)
	if e != nil {
		t.Fatalf("FindNonExistsEntryNameWithReserved: %v", e)
	}
	if got != "drive/foo_2.txt" {
		t.Fatalf("path=%q, want drive/foo_2.txt", got)
	}
}

type mountNameTestDrive struct {
	existing map[string]bool
}

func (d *mountNameTestDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{}, nil
}
func (d *mountNameTestDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	if d.existing[path] {
		return &mountNameTestEntry{path: path, drive: d}, nil
	}
	return nil, err.NewNotFoundError()
}
func (d *mountNameTestDrive) Save(types.TaskCtx, string, int64, bool, io.Reader) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) List(_ context.Context, path string) ([]types.IEntry, error) {
	entries := make([]types.IEntry, 0)
	for existing := range d.existing {
		if existing != path {
			entries = append(entries, &mountNameTestEntry{path: existing, drive: d})
		}
	}
	return entries, nil
}
func (d *mountNameTestDrive) Delete(types.TaskCtx, string) error { panic("not used") }
func (d *mountNameTestDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	panic("not used")
}

type mountNameTestEntry struct {
	path  string
	drive types.IDrive
}

func (e *mountNameTestEntry) Path() string          { return e.path }
func (e *mountNameTestEntry) Name() string          { return e.path }
func (e *mountNameTestEntry) Type() types.EntryType { return types.TypeFile }
func (e *mountNameTestEntry) Size() int64           { return 0 }
func (e *mountNameTestEntry) Meta() types.EntryMeta { return types.EntryMeta{} }
func (e *mountNameTestEntry) ModTime() int64        { return 0 }
func (e *mountNameTestEntry) Drive() types.IDrive   { return e.drive }
func (e *mountNameTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	panic("not used")
}
func (e *mountNameTestEntry) GetURL(context.Context) (*types.ContentURL, error) {
	panic("not used")
}
