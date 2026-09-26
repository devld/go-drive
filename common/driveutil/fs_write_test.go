package driveutil

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	err "go-drive/common/errors"
	"go-drive/common/types"
)

func TestDriveFSWriteSavesEmptyCreateAndTruncate(t *testing.T) {
	d := newWriteDrive(map[string][]byte{"old.txt": []byte("hello")})
	fs := (&DriveFS{tempDir: t.TempDir()}).Bind(context.Background(), d)

	created, e := fs.OpenFile(context.Background(), "new.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		t.Fatal(e)
	}
	info, e := created.Stat()
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() != 0 {
		t.Fatalf("new file size = %d", info.Size())
	}
	if e := created.Close(); e != nil {
		t.Fatal(e)
	}
	if got := d.saved["new.txt"]; got == nil || len(got) != 0 {
		t.Fatalf("empty create saved %#v", got)
	}

	truncated, e := fs.OpenFile(context.Background(), "old.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		t.Fatal(e)
	}
	info, e = truncated.Stat()
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() != 0 {
		t.Fatalf("truncated size before write = %d", info.Size())
	}
	if _, e := truncated.Write([]byte("ab")); e != nil {
		t.Fatal(e)
	}
	info, e = truncated.Stat()
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() != 2 {
		t.Fatalf("size after write = %d", info.Size())
	}
	if e := truncated.Close(); e != nil {
		t.Fatal(e)
	}
	if got := string(d.saved["old.txt"]); got != "ab" {
		t.Fatalf("truncated save = %q", got)
	}
}

func TestDriveFSCreateWithoutTruncKeepsExistingBytes(t *testing.T) {
	d := newWriteDrive(map[string][]byte{"old.txt": []byte("hello")})
	fs := (&DriveFS{tempDir: t.TempDir()}).Bind(context.Background(), d)

	f, e := fs.OpenFile(context.Background(), "old.txt", os.O_RDWR|os.O_CREATE, 0644)
	if e != nil {
		t.Fatal(e)
	}
	info, e := f.Stat()
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() != 5 {
		t.Fatalf("size before write = %d", info.Size())
	}
	if _, e := f.Write([]byte("XY")); e != nil {
		t.Fatal(e)
	}
	info, e = f.Stat()
	if e != nil {
		t.Fatal(e)
	}
	if info.Size() != 5 {
		t.Fatalf("size after overwrite = %d", info.Size())
	}
	if e := f.Close(); e != nil {
		t.Fatal(e)
	}
	if got := string(d.saved["old.txt"]); got != "XYllo" {
		t.Fatalf("saved = %q", got)
	}
}

func TestDriveFSReadWriteCloseWithoutWriteDoesNotSave(t *testing.T) {
	d := newWriteDrive(map[string][]byte{"old.txt": []byte("hello")})
	fs := (&DriveFS{tempDir: t.TempDir()}).Bind(context.Background(), d)
	f, e := fs.OpenFile(context.Background(), "old.txt", os.O_RDWR, 0644)
	if e != nil {
		t.Fatal(e)
	}
	if e := f.Close(); e != nil {
		t.Fatal(e)
	}
	if _, ok := d.saved["old.txt"]; ok {
		t.Fatal("close without write saved the file")
	}
}

type writeDrive struct {
	files map[string]*writeEntry
	saved map[string][]byte
}

func newWriteDrive(files map[string][]byte) *writeDrive {
	d := &writeDrive{files: map[string]*writeEntry{}, saved: map[string][]byte{}}
	for path, data := range files {
		copied := append([]byte(nil), data...)
		d.files[path] = &writeEntry{path: path, data: copied, drive: d}
	}
	return d
}

func (d *writeDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}

func (d *writeDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	if e, ok := d.files[path]; ok {
		return e, nil
	}
	return nil, err.NewNotFoundError()
}

func (d *writeDrive) Save(_ types.TaskCtx, path string, size int64, _ bool, reader io.Reader) (types.IEntry, error) {
	buf := make([]byte, size)
	if _, e := io.ReadFull(reader, buf); e != nil {
		return nil, e
	}
	copied := append([]byte(nil), buf...)
	if copied == nil {
		copied = []byte{}
	}
	d.saved[path] = copied
	entry := &writeEntry{path: path, data: buf, drive: d}
	d.files[path] = entry
	return entry, nil
}

func (d *writeDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *writeDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *writeDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *writeDrive) List(context.Context, string) ([]types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *writeDrive) Delete(types.TaskCtx, string) error { return err.NewUnsupportedError() }
func (d *writeDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	return nil, err.NewUnsupportedError()
}

type writeEntry struct {
	path  string
	data  []byte
	drive *writeDrive
}

func (e *writeEntry) Path() string          { return e.path }
func (e *writeEntry) Name() string          { return e.path }
func (e *writeEntry) Type() types.EntryType { return types.TypeFile }
func (e *writeEntry) Size() int64           { return int64(len(e.data)) }
func (e *writeEntry) ModTime() int64        { return 1 }
func (e *writeEntry) Drive() types.IDrive   { return e.drive }
func (e *writeEntry) Meta() types.EntryMeta { return types.EntryMeta{Readable: true, Writable: true} }
func (e *writeEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
func (e *writeEntry) GetReader(_ context.Context, start, size int64) (io.ReadCloser, error) {
	data := e.data
	if start > 0 {
		if start > int64(len(data)) {
			start = int64(len(data))
		}
		data = data[start:]
	}
	if size >= 0 && size < int64(len(data)) {
		data = data[:size]
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
