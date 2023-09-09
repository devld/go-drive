package fs

import (
	"context"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var fsT = i18n.TPrefix("drive.fs.")

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "fs",
		DisplayName: fsT("name"),
		README:      fsT("readme"),
		ConfigForm: []types.FormItem{
			{Field: "path", Label: fsT("form.path.label"), Type: "text", Required: true, Description: fsT("form.path.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewDrive},
	})
}

type Drive struct {
	path string
}

type fsFile struct {
	drive *Drive
	path  string

	size  int64
	isDir bool

	modTime int64
}

// NewDrive creates a file system drive
func NewDrive(_ context.Context, config types.SM,
	driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	path := config["path"]
	if utils.CleanPath(path) == "" {
		return nil, err.NewNotAllowedMessageError(fsT("invalid_root_path"))
	}

	localRoot, e := driveUtils.Config.GetLocalFsDir()
	if e != nil {
		return nil, e
	}

	path, e = filepath.Abs(filepath.Join(localRoot, path))
	if e != nil {
		return nil, e
	}
	if exists, _ := utils.FileExists(path); !exists {
		return nil, err.NewNotFoundMessageError(fsT("root_path_not_exists"))
	}
	return &Drive{path}, nil
}

func (f *Drive) newFsFile(path string, file os.FileInfo) (types.IEntry, error) {
	path, e := filepath.Abs(path)
	if e != nil {
		return nil, err.NewNotFoundMessageError(i18n.T("drive.invalid_path"))
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
		modTime: utils.Millisecond(file.ModTime()),
	}, nil
}

func (f *Drive) getPath(path string) string {
	path = filepath.Clean(path)
	return filepath.Join(f.path, path)
}

func (f *Drive) isRootPath(path string) bool {
	return filepath.Clean(path) == f.path
}

func (f *Drive) Get(_ context.Context, path string) (types.IEntry, error) {
	path = f.getPath(path)
	stat, e := os.Stat(path)
	if os.IsNotExist(e) {
		return nil, err.NewNotFoundError()
	}
	if e != nil {
		return nil, e
	}
	return f.newFsFile(path, stat)
}

func (f *Drive) Save(ctx types.TaskCtx, path string, _ int64, override bool, reader io.Reader) (types.IEntry, error) {
	var fileMode os.FileMode = 0644
	path = f.getPath(path)
	var e error

	fileInfo, e := os.Stat(path)
	if e != nil {
		if !os.IsNotExist(e) {
			return nil, e
		}
		e = nil
	}

	if fileInfo != nil {
		fileMode = fileInfo.Mode()
	}

	if !override {
		if fileInfo != nil {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
	}
	fileMoved := false
	if tf, ok := reader.(*utils.TempFile); ok {
		fileMoved, e = tf.TransferTo(path)
		if e != nil {
			return nil, e
		}
		if fileMoved {
			_ = os.Chmod(path, fileMode)
		}
	}
	if !fileMoved {
		file, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fileMode)
		if e != nil {
			return nil, e
		}
		defer func() { _ = file.Close() }()
		_, e = drive_util.Copy(ctx, file, reader)
		if e != nil {
			return nil, e
		}
	}
	stat, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	return f.newFsFile(path, stat)
}

func (f *Drive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	fPath := f.getPath(path)
	stat, e := os.Stat(fPath)
	if e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	if e == nil {
		if !stat.IsDir() {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		return f.newFsFile(fPath, stat)
	}
	if e := os.Mkdir(fPath, 0755); e != nil {
		return nil, e
	}
	stat, e = os.Stat(fPath)
	if e != nil {
		return nil, e
	}
	return f.newFsFile(fPath, stat)
}

func (f *Drive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (f *Drive) Move(_ types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(f, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	fromPath := f.getPath(from.(*fsFile).path)
	toPath := f.getPath(to)
	if f.isRootPath(fromPath) || f.isRootPath(toPath) {
		return nil, err.NewNotAllowedError()
	}
	if e := requireFile(fromPath, true); e != nil {
		return nil, e
	}
	exists, e := utils.FileExists(toPath)
	if e != nil {
		return nil, e
	}
	if exists {
		if !override {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		if e := f.Delete(task.DummyContext(), to); e != nil {
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

func (f *Drive) List(_ context.Context, path string) ([]types.IEntry, error) {
	path = f.getPath(path)
	isDir, e := utils.IsDir(path)
	if os.IsNotExist(e) {
		return nil, err.NewNotFoundError()
	}
	if !isDir {
		return nil, err.NewNotAllowedMessageError(fsT("cannot_list_file"))
	}
	files, ee := os.ReadDir(path)
	if ee != nil {
		return nil, ee
	}
	entries := make([]types.IEntry, len(files))
	for i, file := range files {
		info, e := file.Info()
		if e != nil {
			return nil, e
		}
		entry, e := f.newFsFile(filepath.Join(path, file.Name()), info)
		if e != nil {
			return nil, e
		}
		entries[i] = entry
	}
	return entries, nil
}

func (f *Drive) Delete(_ types.TaskCtx, path string) error {
	path = f.getPath(path)
	if f.isRootPath(path) {
		return err.NewNotAllowedMessageError(fsT("cannot_delete_root"))
	}
	if e := requireFile(path, true); e != nil {
		return e
	}
	return os.RemoveAll(path)
}

func (f *Drive) Upload(_ context.Context, path string, size int64,
	override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	path = f.getPath(path)
	if !override {
		if e := requireFile(path, false); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func requireFile(path string, requireExists bool) error {
	exists, e := utils.FileExists(path)
	if e != nil {
		return e
	}
	if requireExists && !exists {
		return err.NewNotFoundMessageError(i18n.T("drive.file_not_exists"))
	}
	if !requireExists && exists {
		return err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	return nil
}

func (f *Drive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
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
	return types.EntryMeta{Readable: true, Writable: true}
}

func (f *fsFile) ModTime() int64 {
	return f.modTime
}

func (f *fsFile) Drive() types.IDrive {
	return f.drive
}

func (f *fsFile) Name() string {
	return utils.PathBase(f.path)
}

func (f *fsFile) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	if !f.Type().IsFile() {
		return nil, err.NewNotAllowedError()
	}
	path := f.drive.getPath(f.path)
	exists, e := utils.FileExists(path)
	if e != nil {
		return nil, e
	}
	if !exists {
		return nil, err.NewNotFoundMessageError(i18n.T("drive.file_not_exists"))
	}
	file, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	if start >= 0 {
		_, e = file.Seek(start, io.SeekStart)
		if e != nil {
			return nil, e
		}
		if size > 0 {
			return drive_util.LimitReadCloser(file, size), nil
		}
	}
	return file, nil
}

func (f *fsFile) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
