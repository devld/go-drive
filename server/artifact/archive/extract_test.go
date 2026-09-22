package archive

import (
	"bytes"
	"context"
	apierr "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"io"
	"os"
	"testing"
)

func TestMemberMatchesUsesPrefixNotNamePrefix(t *testing.T) {
	selected := map[string]struct{}{"docs": {}}
	if !memberMatches("docs", selected) || !memberMatches("docs/info.md", selected) {
		t.Fatal("docs selection should include the directory and its children")
	}
	if memberMatches("docs-extra.txt", selected) {
		t.Fatal("docs should not match docs-extra.txt")
	}
}

func TestJoinExtractPathKeepsStructureAndBlocksEscape(t *testing.T) {
	got, err := joinExtractPath("out", "docs/info.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != "out/docs/info.md" {
		t.Fatalf("join = %q", got)
	}
	if _, err := joinExtractPath("out", "../secret"); err == nil {
		t.Fatal("expected escaped member to fail")
	}
}

func TestExtractMembersPreservesPathsAndDirectorySelection(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	dest := newExtractTestDrive()
	if _, err := dest.MakeDir(context.Background(), "out"); err != nil {
		t.Fatal(err)
	}
	source := &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size}

	err := previewer.ExtractMembers(task.DummyContext(), dest, source, "out", []string{"docs"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := dest.files["out/docs/info.md"]; string(got) != "nested content" {
		t.Fatalf("extracted nested file = %q", got)
	}
	if _, ok := dest.files["out/README.txt"]; ok {
		t.Fatal("unselected README.txt was extracted")
	}
}

func TestExtractMembersHonorsPermissionWrapper(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	inner := newExtractTestDrive()
	if _, err := inner.MakeDir(context.Background(), "out"); err != nil {
		t.Fatal(err)
	}

	root := ""
	denied := "out/docs"
	pm := utils.NewPermMap([]types.PathPermission{
		{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &denied, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyReject},
	})
	wrapped := drive.NewPermissionWrapperDrive(inner, pm)
	source := &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size}

	err := previewer.ExtractMembers(task.DummyContext(), wrapped, source, "out", []string{"docs"}, false)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !apierr.IsNotFoundError(err) && !apierr.IsNotAllowedError(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractMembersUsesChrootDestination(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	inner := newExtractTestDrive()
	if _, err := inner.MakeDir(context.Background(), "jail"); err != nil {
		t.Fatal(err)
	}
	if _, err := inner.MakeDir(context.Background(), "jail/out"); err != nil {
		t.Fatal(err)
	}
	wrapped := drive.NewChrootWrapper(inner, drive.NewChroot("jail", nil))
	source := &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size}

	err := previewer.ExtractMembers(task.DummyContext(), wrapped, source, "out", []string{"README.txt"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := inner.files["jail/out/README.txt"]; string(got) != "archive preview" {
		t.Fatalf("chrooted extract = %q, files=%v", got, inner.files)
	}
	if _, ok := inner.files["out/README.txt"]; ok {
		t.Fatal("wrote outside chroot")
	}
}

type extractTestDrive struct {
	dirs  map[string]struct{}
	files map[string][]byte
}

func newExtractTestDrive() *extractTestDrive {
	return &extractTestDrive{
		dirs:  map[string]struct{}{"": {}},
		files: map[string][]byte{},
	}
}

func (d *extractTestDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}

func (d *extractTestDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	path = utils.CleanPath(path)
	if _, ok := d.files[path]; ok {
		return &extractTestEntry{d: d, path: path}, nil
	}
	if _, ok := d.dirs[path]; ok {
		return &extractTestEntry{d: d, path: path, dir: true}, nil
	}
	return nil, apierr.NewNotFoundError()
}

func (d *extractTestDrive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	path = utils.CleanPath(path)
	if !override {
		if _, ok := d.files[path]; ok {
			return nil, apierr.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if size >= 0 {
		ctx.Progress(int64(len(body)), false)
	}
	d.files[path] = body
	return &extractTestEntry{d: d, path: path}, nil
}

func (d *extractTestDrive) MakeDir(_ context.Context, path string) (types.IEntry, error) {
	path = utils.CleanPath(path)
	if _, ok := d.files[path]; ok {
		return nil, apierr.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	d.dirs[path] = struct{}{}
	return &extractTestEntry{d: d, path: path, dir: true}, nil
}

func (d *extractTestDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, apierr.NewUnsupportedError()
}
func (d *extractTestDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, apierr.NewUnsupportedError()
}
func (d *extractTestDrive) List(context.Context, string) ([]types.IEntry, error) {
	return nil, apierr.NewUnsupportedError()
}
func (d *extractTestDrive) Delete(types.TaskCtx, string) error {
	return apierr.NewUnsupportedError()
}
func (d *extractTestDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	return nil, apierr.NewUnsupportedError()
}

type extractTestEntry struct {
	d    *extractTestDrive
	path string
	dir  bool
}

func (e *extractTestEntry) Path() string { return e.path }
func (e *extractTestEntry) Name() string { return utils.PathBase(e.path) }
func (e *extractTestEntry) Type() types.EntryType {
	if e.dir {
		return types.TypeDir
	}
	return types.TypeFile
}
func (e *extractTestEntry) Size() int64 {
	if e.dir {
		return -1
	}
	return int64(len(e.d.files[e.path]))
}
func (e *extractTestEntry) ModTime() int64 { return 0 }
func (e *extractTestEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}
func (e *extractTestEntry) Drive() types.IDrive { return e.d }
func (e *extractTestEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, apierr.NewUnsupportedError()
}
func (e *extractTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(e.d.files[e.path])), nil
}

func TestOpenLocalFileUsesOsFile(t *testing.T) {
	filename, size := makeZip(t)
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	entry := &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size}
	opened, err := openLocalFile(context.Background(), entry)
	if err != nil {
		t.Fatal(err)
	}
	if opened == nil {
		t.Fatal("expected *os.File")
	}
	_ = opened.Close()
}
