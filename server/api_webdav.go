package server

import (
	"context"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/webdav"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var webdavHTTPMethods = []string{
	"OPTIONS", "GET", "HEAD", "POST", "DELETE", "PUT",
	"MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", "PROPFIND", "PROPPATCH",
}

func InitWebdavAccess(router gin.IRouter, config common.Config,
	access *drive.Access, userAuth *UserAuth) error {

	wa := &webdavAccess{
		access:  access,
		config:  config,
		lockSys: webdav.NewMemLS(),
	}

	withAuth := router.Group(config.WebDav.Prefix, BasicAuth(userAuth, "webdav", config.WebDav.AllowAnonymous))
	withoutAuth := router.Group(config.WebDav.Prefix)

	for _, method := range webdavHTTPMethods {
		r := withAuth
		if method == "OPTIONS" {
			r = withoutAuth
		}
		r.Handle(method, "/*path", wa.ServeHTTP)
	}
	return nil
}

type webdavAccess struct {
	access  *drive.Access
	lockSys webdav.LockSystem
	config  common.Config
}

func (w *webdavAccess) ServeHTTP(c *gin.Context) {
	session := GetSession(c)

	drive, e := w.access.GetDrive(session, nil)
	if e != nil {
		log.Printf("GetDrive error: %v", e)
		c.AbortWithError(http.StatusInternalServerError, e)
		return
	}

	handler := webdav.Handler{
		Prefix: w.config.WebDav.Prefix,
		FileSystem: &webdavFs{
			drive:   drive,
			tempDir: w.config.TempDir,
		},
		LockSystem: w.lockSys,
	}
	handler.ServeHTTP(c.Writer, c.Request)
}

type webdavFs struct {
	drive   types.IDrive
	tempDir string
}

func (w *webdavFs) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	_, e := w.drive.MakeDir(ctx, utils.CleanPath(name))
	return mapError(e)
}

func (w *webdavFs) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (webdav.File, error) {
	if flag&os.O_SYNC != 0 {
		return nil, os.ErrInvalid
	}

	name = utils.CleanPath(name)
	entry, e := w.drive.Get(ctx, name)
	if e != nil && !err.IsNotFoundError(e) {
		return nil, mapError(e)
	}

	if e == nil && entry.Type().IsDir() {
		return w.newWebDavFile(entry, flag), nil
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

	return w.newWebDavFile(entry, flag), nil
}

func (w *webdavFs) RemoveAll(ctx context.Context, name string) error {
	return mapError(w.drive.Delete(task.NewContextWrapper(ctx), utils.CleanPath(name)))
}

func (w *webdavFs) Rename(ctx context.Context, oldName, newName string) error {
	from, e := w.drive.Get(ctx, utils.CleanPath(oldName))
	if e != nil {
		return mapError(e)
	}
	_, e = w.drive.Move(task.NewContextWrapper(ctx), from, utils.CleanPath(newName), false)
	return mapError(e)
}

func (w *webdavFs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	entry, e := w.drive.Get(ctx, utils.CleanPath(name))
	if e != nil {
		return nil, mapError(e)
	}
	return entryToFileInfo(entry), nil
}

func (w *webdavFs) newWebDavFile(e types.IEntry, flag int) *webdavFile {
	var seekPos int64 = 0
	if flag&os.O_APPEND != 0 {
		seekPos = e.Size()
	}
	return &webdavFile{
		e:        e,
		seekPos:  seekPos,
		mu:       sync.Mutex{},
		tempDir:  w.tempDir,
		openFlag: flag,
	}
}

type webdavFile struct {
	e         types.IEntry
	file      *os.File
	cacheFile *utils.CacheFile

	children []types.IEntry
	dirPos   int
	seekPos  int64
	mu       sync.Mutex

	modified bool
	tempDir  string
	openFlag int
}

func (w *webdavFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cacheFile != nil {
		_ = w.cacheFile.Close()
	}

	if w.file == nil {
		return nil
	}
	file := utils.NewTempFile(w.file)

	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	if w.modified {
		stat, e := file.Stat()
		if e != nil {
			return e
		}
		_, e = file.Seek(0, 0)
		if e != nil {
			return e
		}
		_, e = w.e.Drive().Save(task.NewContextWrapper(context.Background()),
			w.e.Path(), stat.Size(), true, file)
		if e != nil {
			return e
		}
	}
	return nil
}

func (w *webdavFile) getFile() error {
	if w.file != nil || w.cacheFile != nil {
		return nil
	}

	if w.openFlag == os.O_RDONLY {
		cf, e := utils.NewCacheFile(w.e.Size(), w.tempDir, "webdav-tmp")
		if e != nil {
			return e
		}
		_, e = cf.Seek(w.seekPos, io.SeekStart)
		if e != nil {
			_ = cf.Close()
			return e
		}
		if w.e.Size() > 0 {
			go func() {
				e := drive_util.CopyIContent(task.NewContextWrapper(context.Background()), w.e, cf)
				if e != nil {
					log.Printf("error copy file: %v", e)
					_ = cf.Close()
				}
			}()
		}
		w.cacheFile = cf
		return nil
	}

	var file *os.File
	if w.openFlag&os.O_CREATE != 0 || w.openFlag&os.O_TRUNC != 0 {
		tempFile, e := ioutil.TempFile(w.tempDir, "webdav-temp")
		if e != nil {
			return e
		}
		file = tempFile
	} else {
		tempFile, e := drive_util.CopyIContentToTempFile(task.NewContextWrapper(context.Background()), w.e, w.tempDir)
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

func (w *webdavFile) Read(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsFile() {
		return 0, os.ErrInvalid
	}
	if e := w.getFile(); e != nil {
		return 0, e
	}
	if w.cacheFile != nil {
		n, err = w.cacheFile.Read(p)
	} else {
		n, err = w.file.Read(p)
	}
	return
}

func (w *webdavFile) Seek(offset int64, whence int) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cacheFile != nil {
		return w.cacheFile.Seek(offset, whence)
	}
	if w.file == nil {
		// a fake file opened with flag = 0, used to get file size
		size := w.e.Size()
		pos := w.seekPos

		switch whence {
		case io.SeekStart:
			pos = offset
		case io.SeekCurrent:
			pos += offset
		case io.SeekEnd:
			pos += size + offset
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

func (w *webdavFile) Readdir(count int) ([]fs.FileInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsDir() {
		return nil, os.ErrInvalid
	}
	if w.children == nil {
		entries, e := w.e.Drive().List(context.Background(), w.e.Path())
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

func (w *webdavFile) Stat() (fs.FileInfo, error) {
	return entryToFileInfo(w.e), nil
}

func (w *webdavFile) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.e.Type().IsFile() {
		return 0, os.ErrInvalid
	}
	if !w.e.Meta().Writable {
		return 0, os.ErrPermission
	}
	if e := w.getFile(); e != nil {
		return 0, e
	}
	n, err = w.file.Write(p)
	if err == nil {
		w.modified = true
	}
	return
}

func (w *webdavFile) GetURL(ctx context.Context) (string, error) {
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

func entryToFileInfo(e types.IEntry) fs.FileInfo {
	return entryFileInfo{e}
}

func entriesToFileInfos(es []types.IEntry) []fs.FileInfo {
	fi := make([]fs.FileInfo, 0, len(es))
	for _, e := range es {
		fi = append(fi, entryToFileInfo(e))
	}
	return fi
}

type entryFileInfo struct {
	e types.IEntry
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

func (e entryFileInfo) Sys() interface{} {
	return e.e
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

func (c *createdEntry) GetReader(_ context.Context) (io.ReadCloser, error) {
	return nil, err.NewNotAllowedError()
}

func (c *createdEntry) GetURL(_ context.Context) (*types.ContentURL, error) {
	return nil, err.NewNotAllowedError()
}
