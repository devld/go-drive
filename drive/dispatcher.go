package drive

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	path2 "path"
	"regexp"
	"strings"
	"sync"
)

var pathRegexp = regexp.MustCompile(`^/?([^/]+)(/(.*))?$`)

var _ types.IDrive = (*DispatcherDrive)(nil)

// DispatcherDrive splits drive name and key from the raw key.
// Then dispatch request to the specified drive.
type DispatcherDrive struct {
	_drives map[string]types.IDrive

	_drivesMux *sync.RWMutex

	tempDir string

	drivesLock *utils.KeyLock
}

func NewDispatcherDrive(config common.Config) *DispatcherDrive {
	return &DispatcherDrive{
		_drives:    make(map[string]types.IDrive),
		_drivesMux: &sync.RWMutex{},
		tempDir:    config.TempDir,
	}
}

func (d *DispatcherDrive) setDrives(drives map[string]types.IDrive) {
	d._drivesMux.Lock()
	defer d._drivesMux.Unlock()
	for _, d := range d._drives {
		if disposable, ok := d.(types.IDisposable); ok {
			_ = disposable.Dispose()
		}
	}
	newDrives := make(map[string]types.IDrive, len(drives))
	for k, v := range drives {
		newDrives[k] = v
	}
	d._drives = newDrives
	d.drivesLock = utils.NewKeyLock(len(drives))
}

func (d *DispatcherDrive) drives() map[string]types.IDrive {
	d._drivesMux.RLock()
	defer d._drivesMux.RUnlock()
	return d._drives
}

func (d *DispatcherDrive) Dispose() error {
	d._drivesMux.Lock()
	defer d._drivesMux.Unlock()
	for _, d := range d._drives {
		if disposable, ok := d.(types.IDisposable); ok {
			_ = disposable.Dispose()
		}
	}
	return nil
}

func (d *DispatcherDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: false}, nil
}

func (d *DispatcherDrive) resolve(path string) (string, types.IDrive, string, error) {
	paths := pathRegexp.FindStringSubmatch(path)
	if paths == nil {
		return "", nil, "", err.NewNotFoundError()
	}
	driveName := paths[1]
	entryPath := paths[3]
	drive, ok := d.drives()[driveName]
	if !ok {
		return "", nil, "", err.NewNotFoundError()
	}
	return driveName, drive, entryPath, nil
}

func (d *DispatcherDrive) mapResultEntry(requestedPath, driveName string, entry types.IEntry) types.IEntry {
	virtualPath := path2.Join(utils.PathParent(requestedPath), utils.PathBase(entry.Path()))
	return d.mapDriveEntry(virtualPath, driveName, entry)
}

func (d *DispatcherDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return &driveEntry{d: d, path: "", name: "", meta: types.DriveMeta{
			Writable: false,
		}}, nil
	}
	driveName, drive, realPath, e := d.resolve(path)
	if e != nil {
		return nil, e
	}
	entry, e := drive.Get(ctx, realPath)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(path, driveName, entry), nil
}

func (d *DispatcherDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	driveName, drive, realPath, e := d.resolve(path)
	if e != nil {
		return nil, e
	}
	if e := d.ensureDir(ctx, driveName, drive, utils.PathParent(realPath)); e != nil {
		return nil, e
	}
	if !override {
		realPath, e = driveutil.FindNonExistsEntryName(ctx, drive, realPath)
		if e != nil {
			return nil, e
		}
	}
	save, e := drive.Save(ctx, realPath, size, override, reader)
	if e != nil {
		return nil, e
	}
	virtualPath := path2.Join(utils.PathParent(path), utils.PathBase(realPath))
	return d.mapDriveEntry(virtualPath, driveName, save), nil
}

func (d *DispatcherDrive) ensureDir(ctx context.Context, driveName string, drive types.IDrive, path string) error {
	d.drivesLock.Lock(driveName)
	defer d.drivesLock.UnLock(driveName)

	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return nil
	}
	segments := strings.Split(path, "/")
	path = ""
	for _, s := range segments {
		path = path2.Join(path, s)
		_, e := drive.Get(ctx, path)
		if e == nil {
			continue
		}
		if err.IsNotFoundError(e) {
			_, e := drive.MakeDir(ctx, path)
			if e != nil {
				return e
			}
		} else {
			return e
		}
	}
	return nil
}

func (d *DispatcherDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	driveName, drive, realPath, e := d.resolve(path)
	if e != nil {
		return nil, e
	}
	if e := d.ensureDir(ctx, driveName, drive, utils.PathParent(realPath)); e != nil {
		return nil, e
	}
	dir, e := drive.MakeDir(ctx, realPath)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(path, driveName, dir), nil
}

func (d *DispatcherDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string,
	override bool) (types.IEntry, error) {
	driveName, driveTo, pathTo, e := d.resolve(to)
	if e != nil {
		return nil, e
	}
	if !override {
		pathTo, e = driveutil.FindNonExistsEntryName(ctx, driveTo, pathTo)
		if e != nil {
			return nil, e
		}
	}
	entry, e := driveTo.Copy(ctx, from, pathTo, override)
	if e == nil {
		return d.mapResultEntry(to, driveName, entry), nil
	}
	if !err.IsUnsupportedError(e) {
		return nil, e
	}
	e = driveutil.CopyAll(ctx, from, d, to,
		func(from types.IEntry, _ types.IDrive, to string, ctx types.TaskCtx) error {
			_, driveTo, pathTo, e := d.resolve(to)
			ctxWrapper := task.NewCtxWrapper(ctx, true, false)
			if e != nil {
				return e
			}

			if !override {
				pathTo, e = driveutil.FindNonExistsEntryName(ctx, driveTo, pathTo)
				if e != nil {
					return e
				}
			}
			_, e = driveTo.Copy(ctxWrapper, from, pathTo, true)
			if e == nil {
				return nil
			}
			if !err.IsUnsupportedError(e) {
				return e
			}
			return driveutil.CopyEntry(ctxWrapper, from, driveTo, pathTo, true, d.tempDir)
		},
		nil,
	)
	if e != nil {
		return nil, e
	}
	copied, e := driveTo.Get(ctx, pathTo)
	if e != nil {
		return nil, e
	}
	return d.mapResultEntry(to, driveName, copied), nil
}

func (d *DispatcherDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	driveName, driveTo, pathTo, e := d.resolve(to)
	if e != nil {
		return nil, e
	}
	if !override {
		pathTo, e = driveutil.FindNonExistsEntryName(ctx, driveTo, pathTo)
		if e != nil {
			return nil, e
		}
	}
	move, e := driveTo.Move(ctx, from, pathTo, override)
	if e != nil {
		if err.IsUnsupportedError(e) {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.dispatcher.move_across_not_supported"))
		}
		return nil, e
	}
	return d.mapResultEntry(to, driveName, move), nil
}

func (d *DispatcherDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	var entries []types.IEntry
	if utils.IsRootPath(path) {
		drives := d.drives()
		driveEntries := make([]types.IEntry, 0, len(drives))
		for k, v := range drives {
			meta, e := v.Meta(ctx)
			if e != nil {
				return nil, e
			}
			driveEntries = append(driveEntries, &driveEntry{d: d, path: k, name: k, meta: meta})
		}
		entries = driveEntries
	} else {
		driveName, drive, realPath, e := d.resolve(path)
		if e != nil {
			return nil, e
		}
		list, e := drive.List(ctx, realPath)
		if e != nil {
			return nil, e
		}
		entries = d.mapDriveEntries(path, driveName, list)
	}

	return entries, nil
}

func (d *DispatcherDrive) Delete(ctx types.TaskCtx, path string) error {
	_, drive, resolvedPath, e := d.resolve(path)
	if e != nil {
		return e
	}
	if utils.IsRootPath(resolvedPath) {
		return err.NewNotAllowedError()
	}
	return drive.Delete(ctx, resolvedPath)
}

func (d *DispatcherDrive) Upload(ctx context.Context, path string, size int64,
	override bool, config types.SM) (*types.DriveUploadConfig, error) {
	driveName, drive, realPath, e := d.resolve(path)
	if e != nil {
		return nil, e
	}
	if e := d.ensureDir(ctx, driveName, drive, utils.PathParent(realPath)); e != nil {
		return nil, e
	}
	var newPath string
	if !override {
		p, e := driveutil.FindNonExistsEntryName(ctx, drive, realPath)
		if e != nil {
			return nil, e
		}
		if realPath != p {
			newPath = path2.Join(utils.PathParent(path), utils.PathBase(p))
		}
		realPath = p
	}
	if size == 0 {
		r := types.UseLocalProvider(0)
		if newPath != "" {
			r.Path = newPath
		}
		return r, nil
	}
	r, e := drive.Upload(ctx, realPath, size, override, config)
	if e != nil {
		return r, e
	}
	if r != nil && newPath != "" {
		rr := *r
		rr.Path = newPath
		r = &rr
	}
	return r, nil
}

func (d *DispatcherDrive) mapDriveEntry(path string, driveName string, entry types.IEntry) types.IEntry {
	return &entryWrapper{d: d, path: path, IEntry: entry, driveName: driveName}
}

func (d *DispatcherDrive) mapDriveEntries(dir string, driveName string, entries []types.IEntry) []types.IEntry {
	mappedEntries := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		path := e.Path()
		mappedEntries = append(
			mappedEntries,
			d.mapDriveEntry(path2.Join(dir, utils.PathBase(path)), driveName, e),
		)
	}
	return mappedEntries
}

var _ types.IEntryWrapper = (*entryWrapper)(nil)
var _ types.IDispatcherEntry = (*entryWrapper)(nil)

type entryWrapper struct {
	types.IEntry
	d         *DispatcherDrive
	path      string
	driveName string
}

func (d *entryWrapper) Path() string {
	return d.path
}

func (d *entryWrapper) Name() string {
	return utils.PathBase(d.path)
}

func (d *entryWrapper) Meta() types.EntryMeta {
	return d.IEntry.Meta()
}

func (d *entryWrapper) Drive() types.IDrive {
	return d.d
}

func (d *entryWrapper) GetIEntry() types.IEntry {
	return d.IEntry
}

func (d *entryWrapper) GetDispatchedDrive() (string, types.IDrive) {
	return d.driveName, d.IEntry.Drive()
}

func (d *entryWrapper) GetRealPath() string {
	return path2.Join(d.driveName, d.IEntry.Path())
}

var _ types.IEntry = (*driveEntry)(nil)

type driveEntry struct {
	d    *DispatcherDrive
	path string
	name string
	meta types.DriveMeta
}

func (d *driveEntry) Path() string {
	return d.path
}

func (d *driveEntry) Type() types.EntryType {
	return types.TypeDir
}

func (d *driveEntry) Size() int64 {
	return -1
}

func (d *driveEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: d.meta.Writable, Props: d.meta.Props}
}

func (d *driveEntry) ModTime() int64 {
	return -1
}

func (d *driveEntry) Name() string {
	return d.name
}

func (d *driveEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, err.NewNotAllowedError()
}

func (d *driveEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewNotAllowedError()
}

func (d *driveEntry) Drive() types.IDrive {
	return d.d
}
