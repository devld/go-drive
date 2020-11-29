package drive

import (
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FsDrive struct {
	path string
}

type fsFile struct {
	drive *FsDrive
	path  string

	size  int64
	isDir bool

	modTime int64
}

// NewFsDrive creates a file system drive
// params:
//   - path: root key of this drive
func NewFsDrive(config drive_util.DriveConfig, utils drive_util.DriveUtils) (types.IDrive, error) {
	path := config["path"]
	if common.CleanPath(path) == "" {
		return nil, common.NewNotAllowedMessageError("invalid root path")
	}

	localRoot, e := utils.Config.GetLocalFsDir()
	if e != nil {
		return nil, e
	}

	path, e = filepath.Abs(filepath.Join(localRoot, path))
	if e != nil {
		return nil, e
	}
	if exists, _ := common.FileExists(path); !exists {
		return nil, common.NewNotFoundMessageError("root path not exists")
	}
	return &FsDrive{path}, nil
}

func (f *FsDrive) newFsFile(path string, file os.FileInfo) (types.IEntry, error) {
	path, e := filepath.Abs(path)
	if e != nil {
		return nil, common.NewNotFoundMessageError("invalid key")
	}
	if !strings.HasPrefix(path, f.path) {
		panic("invalid file key")
	}
	path = strings.ReplaceAll(path, "\\", "/")
	path = path[len(f.path):]
	for strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return &fsFile{
		drive:   f,
		path:    path,
		size:    file.Size(),
		isDir:   file.IsDir(),
		modTime: common.Millisecond(file.ModTime()),
	}, nil
}

func (f *FsDrive) getPath(path string) string {
	path = filepath.Clean(path)
	return filepath.Join(f.path, path)
}

func (f *FsDrive) isRootPath(path string) bool {
	return filepath.Clean(path) == f.path
}

func (f *FsDrive) Get(path string) (types.IEntry, error) {
	path = f.getPath(path)
	stat, e := os.Stat(path)
	if os.IsNotExist(e) {
		return nil, common.NewNotFoundError()
	}
	if e != nil {
		return nil, e
	}
	return f.newFsFile(path, stat)
}

func (f *FsDrive) Save(path string, _ int64, override bool,
	reader io.Reader, ctx types.TaskCtx) (types.IEntry, error) {
	path = f.getPath(path)
	if !override {
		if e := requireFile(path, false); e != nil {
			return nil, e
		}
	}
	file, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if e != nil {
		return nil, e
	}
	defer func() { _ = file.Close() }()
	_, e = drive_util.Copy(file, reader, task.NewProgressCtxWrapper(ctx))
	if e != nil {
		return nil, e
	}
	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}
	return f.newFsFile(path, stat)
}

func (f *FsDrive) MakeDir(path string) (types.IEntry, error) {
	path = f.getPath(path)
	if e := requireFile(path, false); e != nil {
		return nil, e
	}
	if e := os.Mkdir(path, 0755); e != nil {
		return nil, e
	}
	stat, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	return f.newFsFile(path, stat)
}

func (f *FsDrive) Copy(types.IEntry, string, bool, types.TaskCtx) (types.IEntry, error) {
	return nil, common.NewUnsupportedError()
}

func (f *FsDrive) isSelf(entry types.IEntry) bool {
	if fe, ok := entry.(*fsFile); ok {
		return fe.drive == f
	}
	return false
}

func (f *FsDrive) Move(from types.IEntry, to string, override bool, _ types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, f.isSelf)
	if from == nil {
		return nil, common.NewUnsupportedError()
	}
	fromPath := f.getPath(from.(*fsFile).path)
	toPath := f.getPath(to)
	if f.isRootPath(fromPath) || f.isRootPath(toPath) {
		return nil, common.NewNotAllowedError()
	}
	if e := requireFile(fromPath, true); e != nil {
		return nil, e
	}
	exists, e := common.FileExists(toPath)
	if e != nil {
		return nil, e
	}
	if exists {
		if !override {
			return nil, common.NewNotAllowedMessageError("file exists")
		}
		if e := f.Delete(to, task.DummyContext()); e != nil {
			return nil, e
		}
	}
	if e := os.Rename(fromPath, toPath); e != nil {
		return nil, e
	}
	stat, e := os.Stat(toPath)
	if e != nil {
		return nil, e
	}
	return f.newFsFile(toPath, stat)
}

func (f *FsDrive) List(path string) ([]types.IEntry, error) {
	path = f.getPath(path)
	isDir, e := common.IsDir(path)
	if os.IsNotExist(e) {
		return nil, common.NewNotFoundError()
	}
	if !isDir {
		return nil, common.NewNotAllowedMessageError("cannot list on file")
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	entries := make([]types.IEntry, len(files))
	for i, file := range files {
		entry, e := f.newFsFile(filepath.Join(path, file.Name()), file)
		if e != nil {
			return nil, e
		}
		entries[i] = entry
	}
	return entries, nil
}

func (f *FsDrive) Delete(path string, _ types.TaskCtx) error {
	path = f.getPath(path)
	if f.isRootPath(path) {
		return common.NewNotAllowedMessageError("root cannot be deleted")
	}
	if e := requireFile(path, true); e != nil {
		return e
	}
	return os.RemoveAll(path)
}

func (f *FsDrive) Upload(path string, size int64, override bool,
	_ types.SM) (*types.DriveUploadConfig, error) {
	path = f.getPath(path)
	if !override {
		if e := requireFile(path, false); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func requireFile(path string, requireExists bool) error {
	exists, e := common.FileExists(path)
	if e != nil {
		return e
	}
	if requireExists && !exists {
		return common.NewNotFoundMessageError("file does not exist")
	}
	if !requireExists && exists {
		return common.NewNotAllowedMessageError("file exists")
	}
	return nil
}

func (f *FsDrive) Meta() types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (f *fsFile) Path() string {
	return f.path
}

func (f *fsFile) Type() types.EntryType {
	if f.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (f *fsFile) Size() int64 {
	if f.Type().IsDir() {
		return -1
	}
	return f.size
}

func (f *fsFile) Meta() types.EntryMeta {
	return types.EntryMeta{CanRead: true, CanWrite: true}
}

func (f *fsFile) ModTime() int64 {
	return f.modTime
}

func (f *fsFile) Drive() types.IDrive {
	return f.drive
}

func (f *fsFile) Name() string {
	return common.PathBase(f.path)
}

func (f *fsFile) GetReader() (io.ReadCloser, error) {
	if !f.Type().IsFile() {
		return nil, common.NewNotAllowedError()
	}
	path := f.drive.getPath(f.path)
	exists, e := common.FileExists(path)
	if e != nil {
		return nil, e
	}
	if !exists {
		return nil, common.NewNotFoundMessageError("file does not exist")
	}
	return os.Open(path)
}

func (f *fsFile) GetURL() (*types.ContentURL, error) {
	return nil, common.NewUnsupportedError()
}
