package drive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"os"
	path2 "path"
	"time"

	"github.com/secsy/goftp"
)

func init() {
	t := i18n.TPrefix("drive.ftp.")
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "ftp",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{Label: t("form.host.label"), Type: "text", Field: "host", Required: true, Description: t("form.host.description")},
			{Label: t("form.port.label"), Type: "text", Field: "port", Required: true, Description: t("form.port.description"), DefaultValue: "21"},
			{Label: t("form.user.label"), Type: "text", Field: "user", Description: t("form.user.description")},
			{Label: t("form.password.label"), Type: "password", Field: "password", Description: t("form.password.description")},
			{Label: t("form.concurrent.label"), Type: "text", Field: "concurrent", Description: t("form.concurrent.description")},
			{Label: t("form.timeout.label"), Type: "text", Field: "timeout", Description: t("form.timeout.description")},
			{Label: t("form.cache_ttl.label"), Type: "text", Field: "cache_ttl", Description: t("form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewFtpDrive},
	})
}

func NewFtpDrive(ctx context.Context, config types.SM,
	driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	cacheTTL := config.GetDuration("cache_ttl", -1)

	client, e := goftp.DialConfig(
		goftp.Config{
			User:               config["user"],
			Password:           config["password"],
			ConnectionsPerHost: config.GetInt("concurrent", 5),
			Timeout:            config.GetDuration("timeout", 5*time.Second),
		},
		fmt.Sprintf("%s:%d", config["host"], config.GetInt("port", 21)),
	)
	if e != nil {
		return nil, e
	}
	ftp := &FTPDrive{
		c:        client,
		cacheTTL: cacheTTL,
	}

	if cacheTTL <= 0 {
		ftp.cache = drive_util.DummyCache()
	} else {
		ftp.cache = driveUtils.CreateCache(ftp.deserializeEntry, nil)
	}

	_, e = ftp.List(ctx, "")
	if e != nil {
		return nil, e
	}

	return ftp, nil
}

type FTPDrive struct {
	c        *goftp.Client
	cache    drive_util.DriveCache
	cacheTTL time.Duration
}

func (f *FTPDrive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (f *FTPDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return &ftpEntry{d: f, isDir: true, modTime: -1}, nil
	}
	if cached, _ := f.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	parentPath := utils.PathParent(path)
	name := utils.PathBase(path)
	entries, e := f.list(ctx, parentPath)
	if e != nil {
		return nil, e
	}
	for _, found := range entries {
		if utils.PathBase(found.Path()) == name {
			_ = f.cache.PutEntry(found, f.cacheTTL)
			return found, nil
		}
	}
	return nil, err.NewNotFoundError()
}

func (f *FTPDrive) Save(ctx types.TaskCtx, path string, _ int64, override bool, reader io.Reader) (types.IEntry, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	e := f.c.Store(path, reader)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(path, false)
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *FTPDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	_, e := f.c.Mkdir(path)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *FTPDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (f *FTPDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(f, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	fromEntry := from.(*ftpEntry)
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, to); e != nil {
			return nil, e
		}
	}
	e := f.c.Rename(fromEntry.path, to)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(to, true)
	_ = f.cache.Evict(utils.PathParent(to), false)
	_ = f.cache.Evict(fromEntry.path, true)
	_ = f.cache.Evict(utils.PathParent(fromEntry.path), false)
	return f.Get(ctx, to)
}

func (f *FTPDrive) list(_ context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	stats, e := f.c.ReadDir(path)
	if e != nil {
		return nil, mapError(e)
	}
	entries := make([]types.IEntry, len(stats))
	for i, s := range stats {
		entries[i] = f.newFTPEntry(path, s)
	}
	_ = f.cache.PutChildren(path, entries, f.cacheTTL)
	return entries, nil
}

func (f *FTPDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	entries, e := f.list(ctx, path)
	if e != nil {
		return nil, e
	}
	if len(entries) == 0 {
		// we need to check whether the folder exists
		_, e := f.Get(ctx, path)
		if e != nil {
			return nil, e
		}
	}
	return entries, nil
}

func (f *FTPDrive) Delete(ctx types.TaskCtx, path string) error {
	deleteRoot, e := f.Get(ctx, path)
	if e != nil {
		return e
	}
	tree, e := drive_util.BuildEntriesTree(ctx, deleteRoot, false)
	if e != nil {
		return e
	}
	entries := drive_util.FlattenEntriesTree(tree)

	for i := len(entries) - 1; i >= 0; i-- {
		var e error
		if entries[i].Type().IsDir() {
			e = f.c.Rmdir(entries[i].Path())
		} else {
			e = f.c.Delete(entries[i].Path())
		}
		if e != nil {
			return e
		}
		ctx.Progress(1, false)
	}

	_ = f.cache.Evict(utils.PathParent(path), false)
	_ = f.cache.Evict(path, true)
	return nil
}

func (f *FTPDrive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (f *FTPDrive) newFTPEntry(path string, stat os.FileInfo) *ftpEntry {
	return &ftpEntry{
		d:       f,
		path:    path2.Join(path, stat.Name()),
		size:    stat.Size(),
		isDir:   stat.IsDir(),
		modTime: utils.Millisecond(stat.ModTime()),
	}
}

func (f *FTPDrive) deserializeEntry(dat string) (types.IEntry, error) {
	ec, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	return &ftpEntry{path: ec.Path, d: f, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir()}, nil
}

type ftpEntry struct {
	d       *FTPDrive
	path    string
	size    int64
	isDir   bool
	modTime int64
}

func (f *ftpEntry) Path() string {
	return f.path
}

func (f *ftpEntry) Type() types.EntryType {
	if f.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (f *ftpEntry) Size() int64 {
	if f.Type().IsDir() {
		return -1
	}
	return f.size
}

func (f *ftpEntry) Meta() types.EntryMeta {
	return types.EntryMeta{CanRead: true, CanWrite: true}
}

func (f *ftpEntry) ModTime() int64 {
	return f.modTime
}

func (f *ftpEntry) Drive() types.IDrive {
	return f.d
}

func (f *ftpEntry) Name() string {
	return utils.PathBase(f.path)
}

func (f *ftpEntry) GetReader(context.Context) (io.ReadCloser, error) {
	return utils.NewLazyReader(func() (io.ReadCloser, error) {
		r, w := io.Pipe()
		go func() {
			if e := f.d.c.Retrieve(f.path, w); e != nil {
				_ = r.CloseWithError(e)
			}
			_ = w.Close()
		}()
		return r, nil
	}), nil
}

func (f *ftpEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}

func mapError(e error) error {
	fe, ok := e.(goftp.Error)
	if !ok {
		return e
	}
	return errors.New(fmt.Sprintf("[%d] %s", fe.Code(), fe.Message()))
}
