package driveutil

import (
	"context"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DriveFS is the process-wide file adapter. It owns the shared source cache
// used by WebDAV and archive preview. Bind scopes it to one drive.
type DriveFS struct {
	tempDir string
	cfp     *CacheFilePool
}

func NewDriveFS(config common.Config) (*DriveFS, error) {
	tempDir := config.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	dir := filepath.Join(tempDir, "drive-fs")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	items := utils.PositiveOr(config.VFS.CacheItems, common.DefaultVFSCacheItems)
	maxBytes := config.VFS.CacheSize.DataSize(common.DefaultVFSCacheSize.DataSize(0))
	pool, e := NewCacheFilePool(CacheFilePoolOptions{
		MaxEntries:   items,
		MaxBytes:     maxBytes,
		Dir:          dir,
		CleanStartup: true,
	})
	if e != nil {
		return nil, e
	}
	return &DriveFS{tempDir: tempDir, cfp: pool}, nil
}

func (fs *DriveFS) Bind(ctx context.Context, drive types.IDrive) *BoundDriveFS {
	if fs == nil {
		fs = &DriveFS{}
	}
	return &BoundDriveFS{drive: drive, tempDir: fs.tempDir, cfp: fs.cfp, ctx: ctx}
}

func (fs *DriveFS) Dispose() error {
	if fs == nil || fs.cfp == nil {
		return nil
	}
	e := fs.cfp.Dispose()
	fs.cfp = nil
	return e
}

type DriveFSFile interface {
	fs.File
	fs.ReadDirFile
	io.Seeker
	io.ReaderAt
	io.Writer
	Readdir(count int) ([]fs.FileInfo, error)
	GetURL(ctx context.Context) (string, error)
}

type BoundDriveFS struct {
	drive   types.IDrive
	tempDir string

	cfp *CacheFilePool
	// ctx is used by Open/ReadDir, which implement fs.FS and have no context argument.
	ctx context.Context
}

func (w *BoundDriveFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	entry, e := w.drive.Get(ctx, utils.CleanPath(name))
	if e != nil {
		return nil, mapError(e)
	}
	return entryFileInfo{entry}, nil
}

func (w *BoundDriveFS) Open(name string) (fs.File, error) {
	r, e := w.OpenFile(w.ctx, name, os.O_RDONLY, 0644)
	if e != nil {
		if err.IsNotFoundError(e) {
			e = os.ErrNotExist
		}
		return nil, &fs.PathError{Op: "open", Path: name, Err: e}
	}
	return r, nil
}

func (w *BoundDriveFS) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (DriveFSFile, error) {
	if flag&os.O_SYNC != 0 {
		return nil, os.ErrInvalid
	}

	name = utils.CleanPath(name)
	entry, e := w.drive.Get(ctx, name)
	if e != nil && !err.IsNotFoundError(e) {
		return nil, mapError(e)
	}

	if e == nil && entry.Type().IsDir() {
		return w.newDriveFSFile(ctx, entry, flag), nil
	}

	if e == nil && flag&os.O_EXCL != 0 {
		return nil, os.ErrExist
	}
	if e != nil && flag&os.O_CREATE == 0 {
		return nil, mapError(e)
	}
	if entry == nil {
		entry = &createdEntry{
			path:    name,
			drive:   w.drive,
			modTime: utils.Millisecond(time.Now()),
		}
	}

	return w.newDriveFSFile(ctx, entry, flag), nil
}

// OpenEntry opens an existing file entry. Read-only opens use a native
// *os.File when the entry already provides one, and otherwise use the cache
// pool. This is the same read path as OpenFile.
func (w *BoundDriveFS) OpenEntry(ctx context.Context, entry types.IEntry) (DriveFSFile, error) {
	if entry == nil || !entry.Type().IsFile() {
		return nil, os.ErrInvalid
	}
	return w.newDriveFSFile(ctx, entry, os.O_RDONLY), nil
}

func (w *BoundDriveFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = utils.CleanPath(name)
	entries, e := w.drive.List(w.ctx, name)
	if e != nil {
		return nil, mapError(e)
	}
	return utils.ArrayMap(
		entries,
		func(t *types.IEntry) fs.DirEntry { return entryFileInfo{*t} },
	), nil
}

func (w *BoundDriveFS) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	_, e := w.drive.MakeDir(ctx, utils.CleanPath(name))
	if err.IsPermissionDeniedNotFoundError(e) {
		return os.ErrPermission
	}
	return mapError(e)
}

func (w *BoundDriveFS) RemoveAll(ctx context.Context, name string) error {
	return mapError(w.drive.Delete(task.NewContextWrapper(ctx), utils.CleanPath(name)))
}

func (w *BoundDriveFS) Rename(ctx context.Context, oldName, newName string) error {
	from, e := w.drive.Get(ctx, utils.CleanPath(oldName))
	if e != nil {
		return mapError(e)
	}
	_, e = w.drive.Move(task.NewContextWrapper(ctx), from, utils.CleanPath(newName), false)
	return mapError(e)
}

func (w *BoundDriveFS) newDriveFSFile(ctx context.Context, e types.IEntry, flag int) *driveFSFile {
	var seekPos int64 = 0
	if flag&os.O_APPEND != 0 && flag&os.O_TRUNC == 0 {
		seekPos = e.Size()
	}
	return &driveFSFile{
		fs:       w,
		ctx:      ctx,
		e:        e,
		seekPos:  seekPos,
		mu:       sync.Mutex{},
		tempDir:  w.tempDir,
		openFlag: flag,
	}
}

type driveFSFile struct {
	fs  *BoundDriveFS
	ctx context.Context

	e      types.IEntry
	file   *os.File
	reader io.ReadSeekCloser

	children []types.IEntry
	dirPos   int
	seekPos  int64
	mu       sync.Mutex

	modified    bool
	replaced    bool
	writeFailed bool
	attempted   bool
	tempDir     string
	openFlag    int
}

func (w *driveFSFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.reader != nil {
		_ = w.reader.Close()
	}

	// A truncated or newly created file is saved even when Write was never
	// called, so an empty PUT still creates or clears the remote file.
	if !w.attempted && w.file == nil && w.replacesContent() {
		if e := w.getFile(); e != nil {
			return e
		}
	}
	if w.file == nil {
		return nil
	}
	file := utils.NewTempFile(w.file)

	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	if w.modified || (w.replaced && !w.writeFailed) {
		stat, e := file.Stat()
		if e != nil {
			return e
		}
		_, e = file.Seek(0, 0)
		if e != nil {
			return e
		}
		_, e = w.e.Drive().Save(task.NewContextWrapper(w.ctx),
			w.e.Path(), stat.Size(), true, file)
		if e != nil {
			return e
		}
	}
	return nil
}

func (w *driveFSFile) getFile() error {
	if w.file != nil || w.reader != nil {
		return nil
	}

	if w.openFlag == os.O_RDONLY {
		cacheKey, e := cacheFileKey(w.e)
		if e != nil {
			return e
		}
		if !w.fs.cfp.Has(cacheKey) {
			file, e := openLocalFile(w.ctx, w.e)
			if e != nil {
				return e
			}
			if file != nil {
				if w.seekPos != 0 {
					if _, e = file.Seek(w.seekPos, io.SeekStart); e != nil {
						_ = file.Close()
						return e
					}
				}
				w.reader = file
				return nil
			}
		}
		reader, e := w.fs.cfp.GetReader(w.ctx, cacheKey, w.e.Size(),
			func(ctx context.Context, start, size int64) (io.ReadCloser, error) {
				reader, e := GetIContentReader(ctx, w.e, start, size)
				if e != nil {
					return nil, e
				}
				progress, ok := w.ctx.(types.TaskCtx)
				if !ok {
					return reader, nil
				}
				return struct {
					io.Reader
					io.Closer
				}{ProgressReader(reader, progress), reader}, nil
			},
		)
		if e != nil {
			return e
		}
		_, e = reader.Seek(w.seekPos, io.SeekStart)
		if e != nil {
			_ = reader.Close()
			return e
		}
		w.reader = reader
		return nil
	}

	var file *os.File
	if w.replacesContent() {
		tempFile, e := os.CreateTemp(w.tempDir, "go-drive-temp")
		if e != nil {
			return e
		}
		file = tempFile
		w.replaced = true
	} else {
		tempFile, e := CopyIContentToTempFile(task.NewContextWrapper(w.ctx), w.e, w.tempDir)
		if e != nil {
			return e
		}
		_ = tempFile.Close()
		file, e = os.OpenFile(tempFile.Name(), w.openFlag, os.ModePerm)
		if e != nil {
			_ = os.Remove(tempFile.Name())
			return e
		}
	}

	_, e := file.Seek(w.seekPos, io.SeekStart)
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return e
	}
	w.file = file
	return nil
}

func (w *driveFSFile) Read(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsFile() {
		return 0, os.ErrInvalid
	}
	if e := w.getFile(); e != nil {
		return 0, e
	}
	if w.reader != nil {
		n, err = w.reader.Read(p)
	} else {
		n, err = w.file.Read(p)
	}
	return
}

func (w *driveFSFile) ReadAt(p []byte, off int64) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsFile() {
		return 0, os.ErrInvalid
	}
	if e := w.getFile(); e != nil {
		return 0, e
	}
	if ra, ok := w.reader.(io.ReaderAt); ok {
		return ra.ReadAt(p, off)
	}
	if w.file != nil {
		return w.file.ReadAt(p, off)
	}
	return 0, os.ErrInvalid
}

func (w *driveFSFile) Seek(offset int64, whence int) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.reader != nil {
		return w.reader.Seek(offset, whence)
	}
	if w.file == nil {
		// a fake file opened with flag = 0, used to get file size
		size := w.e.Size()
		if w.replacesContent() {
			size = 0
		}
		pos := w.seekPos

		switch whence {
		case io.SeekStart:
			pos = offset
		case io.SeekCurrent:
			pos += offset
		case io.SeekEnd:
			pos = size + offset
		default:
			pos = -1
		}
		if pos < 0 {
			return 0, os.ErrInvalid
		}
		w.seekPos = pos
		return w.seekPos, nil
	}
	return w.file.Seek(offset, whence)
}

func (w *driveFSFile) Readdir(count int) ([]fs.FileInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsDir() {
		return nil, os.ErrInvalid
	}
	if w.children == nil {
		entries, e := w.e.Drive().List(w.ctx, w.e.Path())
		if e != nil {
			return nil, mapError(e)
		}
		w.children = entries
	}
	pos := w.dirPos
	if pos >= len(w.children) {
		if count > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}
	if count <= 0 {
		return entriesToFileInfos(w.children), nil
	}
	w.dirPos += count
	if w.dirPos > len(w.children) {
		w.dirPos = len(w.children)
	}
	return entriesToFileInfos(w.children[pos:w.dirPos]), nil
}

func (w *driveFSFile) Stat() (fs.FileInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	info := entryFileInfo{w.e}
	if w.file != nil {
		stat, e := w.file.Stat()
		if e != nil {
			return nil, e
		}
		return fileStat{info, stat.Size(), stat.ModTime()}, nil
	}
	if w.replacesContent() {
		return fileStat{info, 0, info.ModTime()}, nil
	}
	return info, nil
}

func (w *driveFSFile) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsFile() {
		return 0, os.ErrInvalid
	}
	if !w.e.Meta().Writable {
		return 0, os.ErrPermission
	}
	w.attempted = true
	if e := w.getFile(); e != nil {
		return 0, e
	}
	n, err = w.file.Write(p)
	if err == nil {
		w.modified = true
	} else {
		w.writeFailed = true
	}
	return
}

// replacesContent reports whether this open starts from an empty file.
// O_TRUNC clears an existing file. A createdEntry has no content to copy.
func (w *driveFSFile) replacesContent() bool {
	if w.e == nil || !w.e.Type().IsFile() || w.openFlag == os.O_RDONLY {
		return false
	}
	if w.openFlag&os.O_TRUNC != 0 {
		return true
	}
	_, created := w.e.(*createdEntry)
	return created
}

func (w *driveFSFile) GetURL(ctx context.Context) (string, error) {
	if !w.e.Type().IsFile() {
		return "", os.ErrInvalid
	}
	u, e := w.e.GetURL(ctx)
	if err.IsUnsupportedError(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	if u.Proxy {
		return "", nil
	}
	return u.URL, nil
}

func (w *driveFSFile) ReadDir(n int) ([]fs.DirEntry, error) {
	r, e := w.Readdir(n)
	if e != nil {
		return nil, e
	}
	return utils.ArrayMap(
		r,
		func(t *fs.FileInfo) fs.DirEntry { return (*t).(entryFileInfo) },
	), nil
}

func cacheFileKey(entry types.IEntry) (string, error) {
	resolved, ok := IEntryAs[types.IDispatcherEntry](entry)
	if !ok || resolved.GetRealPath() == "" {
		return "", fmt.Errorf("entry %q has no real path", entry.Path())
	}
	return fmt.Sprintf("%s,m:%d,s:%d", resolved.GetRealPath(), entry.ModTime(), entry.Size()), nil
}

// openLocalFile returns a native *os.File when GetReader already provides one.
// URL-capable entries skip this probe so the range cache can fetch them. A
// non-file reader is closed and the caller should use the cache pool.
func openLocalFile(ctx context.Context, entry types.IEntry) (*os.File, error) {
	if _, err := entry.GetURL(ctx); err == nil {
		return nil, nil
	}
	reader, err := entry.GetReader(ctx, -1, -1)
	if err != nil {
		return nil, err
	}
	file, ok := reader.(*os.File)
	if !ok {
		_ = reader.Close()
		return nil, nil
	}
	return file, nil
}

func entriesToFileInfos(es []types.IEntry) []fs.FileInfo {
	fi := make([]fs.FileInfo, 0, len(es))
	for _, e := range es {
		fi = append(fi, entryFileInfo{e})
	}
	return fi
}

type entryFileInfo struct {
	e types.IEntry
}

// fileStat reports the size and modification time of the temp file being written.
type fileStat struct {
	entryFileInfo
	size    int64
	modTime time.Time
}

func (f fileStat) Size() int64 {
	return f.size
}

func (f fileStat) ModTime() time.Time {
	return f.modTime
}

func (e entryFileInfo) Name() string {
	return utils.PathBase(e.e.Path())
}

func (e entryFileInfo) Size() int64 {
	return e.e.Size()
}

func (e entryFileInfo) Mode() fs.FileMode {
	var p fs.FileMode = 0
	meta := e.e.Meta()
	if meta.Readable {
		p |= 04 // r
	}
	if meta.Writable {
		p |= 02 // w
	}
	if e.e.Type().IsDir() {
		p |= fs.ModeDir
		if meta.Readable {
			p |= 01 // x
		}
	}
	return p
}

func (e entryFileInfo) ModTime() time.Time {
	return utils.Time(e.e.ModTime())
}

func (e entryFileInfo) IsDir() bool {
	return e.e.Type().IsDir()
}

func (e entryFileInfo) Sys() any {
	return e.e
}

func (e entryFileInfo) Info() (fs.FileInfo, error) {
	return e, nil
}

func (e entryFileInfo) Type() fs.FileMode {
	return e.Mode()
}

func mapError(e error) error {
	if err.IsNotFoundError(e) {
		return os.ErrNotExist
	}
	if err.IsNotAllowedError(e) {
		return os.ErrPermission
	}
	return e
}

var _ types.IEntry = (*createdEntry)(nil)

type createdEntry struct {
	path    string
	drive   types.IDrive
	modTime int64
}

func (c *createdEntry) Path() string {
	return c.path
}

func (c *createdEntry) Type() types.EntryType {
	return types.TypeFile
}

func (c *createdEntry) Size() int64 {
	return 0
}

func (c *createdEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}

func (c *createdEntry) ModTime() int64 {
	return c.modTime
}

func (c *createdEntry) Drive() types.IDrive {
	return c.drive
}

func (c *createdEntry) Name() string {
	return utils.PathBase(c.path)
}

func (c *createdEntry) GetReader(_ context.Context, _, _ int64) (io.ReadCloser, error) {
	return nil, err.NewNotAllowedError()
}

func (c *createdEntry) GetURL(_ context.Context) (*types.ContentURL, error) {
	return nil, err.NewNotAllowedError()
}
