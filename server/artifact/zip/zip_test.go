package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
	"testing"
	"time"
)

func TestZipPackTTLDefaultsToOneMinute(t *testing.T) {
	handler, e := NewHandler(artifact.HandlerDeps{Config: common.Config{}})
	if e != nil {
		t.Fatal(e)
	}
	spec := handler.Spec()
	caches := spec.Caches
	if len(caches) != 1 || caches[0].Policy.TTL != time.Minute {
		t.Fatalf("pack cache = %#v, want TTL 1m", caches)
	}
	if spec.Concurrency != common.DefaultArchiveConcurrent {
		t.Fatalf("concurrency = %d, want %d", spec.Concurrency, common.DefaultArchiveConcurrent)
	}
}

func TestZipProducePacksDriveFiles(t *testing.T) {
	drive := newZipDrive()
	ctx := task.DummyContext()
	if _, e := drive.Save(ctx, "docs/a.txt", 1, true, bytes.NewReader([]byte("A"))); e != nil {
		t.Fatal(e)
	}
	if _, e := drive.Save(ctx, "other.txt", 1, true, bytes.NewReader([]byte("B"))); e != nil {
		t.Fatal(e)
	}
	drive.dirs["docs"] = struct{}{}
	source, e := drive.Get(context.Background(), "docs")
	if e != nil {
		t.Fatal(e)
	}
	handler, e := NewHandler(artifact.HandlerDeps{Config: common.Config{Archive: common.ArchiveConfig{PackTTL: time.Minute}}})
	if e != nil {
		t.Fatal(e)
	}
	args := `["a.txt"]`
	var buf bytes.Buffer
	if e := handler.(*Handler).Produce(ctx, artifact.Request{Source: source, Args: args}, bufferWriter{Buffer: &buf}); e != nil {
		t.Fatal(e)
	}
	reader, e := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if e != nil {
		t.Fatal(e)
	}
	if len(reader.File) != 1 || reader.File[0].Name != "a.txt" {
		t.Fatalf("entries = %#v", reader.File)
	}
}

func TestZipProduceReportsTotalWhileWalking(t *testing.T) {
	drive := newZipDrive()
	if _, e := drive.Save(task.DummyContext(), "a.txt", 1, true, bytes.NewReader([]byte("A"))); e != nil {
		t.Fatal(e)
	}
	source, e := drive.Get(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	handler, e := NewHandler(artifact.HandlerDeps{Config: common.Config{}})
	if e != nil {
		t.Fatal(e)
	}
	ctx := task.NewTaskContext(context.Background())
	if e := handler.(*Handler).Produce(ctx, artifact.Request{Source: source, Args: `["a.txt"]`}, bufferWriter{Buffer: &bytes.Buffer{}}); e != nil {
		t.Fatal(e)
	}
	if ctx.GetTotal() != 1 {
		t.Fatalf("total = %d, want 1", ctx.GetTotal())
	}
}

func TestZipKeepsMemberPathRelative(t *testing.T) {
	drive := newZipDrive()
	if _, e := drive.Save(task.DummyContext(), "docs/a.txt", 1, true, bytes.NewReader([]byte("A"))); e != nil {
		t.Fatal(e)
	}
	drive.dirs["docs"] = struct{}{}
	source, e := drive.Get(context.Background(), "docs")
	if e != nil {
		t.Fatal(e)
	}
	handler, e := NewHandler(artifact.HandlerDeps{Config: common.Config{}})
	if e != nil {
		t.Fatal(e)
	}
	args := `["docs/a.txt"]`
	errProduce := handler.(*Handler).Produce(task.DummyContext(), artifact.Request{Source: source, Args: args}, bufferWriter{Buffer: &bytes.Buffer{}})
	if !err.IsNotFoundError(errProduce) {
		t.Fatalf("err = %v, want the prefixed path joined under the directory", errProduce)
	}
}

type bufferWriter struct{ *bytes.Buffer }

func (bufferWriter) WriteMeta(artifact.Meta) error { return nil }

type zipDrive struct {
	files map[string][]byte
	dirs  map[string]struct{}
}

func newZipDrive() *zipDrive {
	return &zipDrive{files: map[string][]byte{}, dirs: map[string]struct{}{"": {}}}
}

func (d *zipDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}
func (d *zipDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	if _, ok := d.files[path]; ok {
		return &zipEntry{d: d, path: path}, nil
	}
	if _, ok := d.dirs[path]; ok {
		return &zipEntry{d: d, path: path, dir: true}, nil
	}
	return nil, err.NewNotFoundError()
}
func (d *zipDrive) Save(_ types.TaskCtx, path string, _ int64, _ bool, reader io.Reader) (types.IEntry, error) {
	body, e := io.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	d.files[path] = body
	return &zipEntry{d: d, path: path}, nil
}
func (d *zipDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *zipDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *zipDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}
func (d *zipDrive) List(_ context.Context, path string) ([]types.IEntry, error) {
	prefix := path
	if prefix != "" {
		prefix += "/"
	}
	var entries []types.IEntry
	seen := map[string]struct{}{}
	consider := func(name string, dir bool) {
		rest := name
		if prefix != "" {
			if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
				return
			}
			rest = name[len(prefix):]
		}
		if rest == "" || containsSlash(rest) {
			return
		}
		if _, ok := seen[rest]; ok {
			return
		}
		seen[rest] = struct{}{}
		child := prefix + rest
		if prefix == "" {
			child = rest
		}
		entries = append(entries, &zipEntry{d: d, path: child, dir: dir})
	}
	for name := range d.files {
		consider(name, false)
	}
	for name := range d.dirs {
		if name != "" {
			consider(name, true)
		}
	}
	return entries, nil
}
func (d *zipDrive) Delete(types.TaskCtx, string) error { return err.NewUnsupportedError() }
func (d *zipDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	return nil, err.NewUnsupportedError()
}

type zipEntry struct {
	d    *zipDrive
	path string
	dir  bool
}

func (e *zipEntry) Path() string { return e.path }
func (e *zipEntry) Name() string {
	if i := lastSlash(e.path); i >= 0 {
		return e.path[i+1:]
	}
	return e.path
}
func (e *zipEntry) Type() types.EntryType {
	if e.dir {
		return types.TypeDir
	}
	return types.TypeFile
}
func (e *zipEntry) Size() int64 {
	if e.dir {
		return 0
	}
	return int64(len(e.d.files[e.path]))
}
func (e *zipEntry) ModTime() int64 { return 0 }
func (e *zipEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}
func (e *zipEntry) Drive() types.IDrive { return e.d }
func (e *zipEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
func (e *zipEntry) GetReader(context.Context, types.ReaderRange) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(e.d.files[e.path])), nil
}

func lastSlash(path string) int {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return i
		}
	}
	return -1
}

func containsSlash(path string) bool { return lastSlash(path) >= 0 }

type fixedOption string

func (o fixedOption) GetValue(string) types.SV { return types.SV(o) }

func TestZipOptionLimitsSelection(t *testing.T) {
	drive := newZipDrive()
	if _, e := drive.Save(task.DummyContext(), "a.txt", 4, true, bytes.NewReader([]byte("abcd"))); e != nil {
		t.Fatal(e)
	}
	source, _ := drive.Get(context.Background(), "")
	handler, e := NewHandler(artifact.HandlerDeps{Config: common.Config{}, Options: fixedOption("1")})
	if e != nil {
		t.Fatal(e)
	}
	errProduce := handler.(*Handler).Produce(task.NewTaskContext(context.Background()), artifact.Request{
		Source: source,
		Args:   `["a.txt"]`,
	}, bufferWriter{Buffer: &bytes.Buffer{}})
	if errProduce == nil || i18n.T("api.zip.size_exceed", "1") != errProduce.Error() && !err.IsNotAllowedError(errProduce) {
		t.Fatalf("err = %v", errProduce)
	}
}
