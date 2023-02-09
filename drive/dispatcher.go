package drive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"io"
	path2 "path"
	"regexp"
	"strings"
	"sync"
)

var pathRegexp = regexp.MustCompile(`^/?([^/]+)(/(.*))?$`)

const maxMountDepth = 10

// DispatcherDrive splits drive name and key from the raw key.
// Then dispatch request to the specified drive.
type DispatcherDrive struct {
	_drives map[string]types.IDrive
	_mounts map[string]map[string]types.PathMount

	_drivesMux *sync.RWMutex
	_mountsMux *sync.RWMutex

	tempDir string

	mountStorage *storage.PathMountDAO

	drivesLock *utils.KeyLock
}

func NewDispatcherDrive(mountStorage *storage.PathMountDAO, config common.Config) *DispatcherDrive {
	return &DispatcherDrive{
		_drives:      make(map[string]types.IDrive),
		_drivesMux:   &sync.RWMutex{},
		_mountsMux:   &sync.RWMutex{},
		mountStorage: mountStorage,
		tempDir:      config.TempDir,
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

func (d *DispatcherDrive) reloadMounts() error {
	d._mountsMux.Lock()
	defer d._mountsMux.Unlock()
	mounts, e := d.mountStorage.GetMounts()
	if e != nil {
		return e
	}
	ms := make(map[string]map[string]types.PathMount, 0)
	for _, m := range mounts {
		t := ms[*m.Path]
		if t == nil {
			t = make(map[string]types.PathMount, 0)
		}
		t[m.Name] = m
		ms[*m.Path] = t
	}

	d._mounts = ms
	return nil
}

func (d *DispatcherDrive) drives() map[string]types.IDrive {
	d._drivesMux.RLock()
	defer d._drivesMux.RUnlock()
	return d._drives
}

func (d *DispatcherDrive) mounts() map[string]map[string]types.PathMount {
	d._mountsMux.RLock()
	defer d._mountsMux.RUnlock()
	return d._mounts
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
	panic("not supported")
}

func (d *DispatcherDrive) resolve(path string) (string, types.IDrive, string, error) {
	mountDepth := 0
	for {
		targetPath := d.resolveMount(path)
		if targetPath == "" {
			break
		}
		mountDepth++
		path = targetPath
		if mountDepth > maxMountDepth {
			return "", nil, "", errors.New("maximum mounting depth exceeded")
		}
	}
	paths := pathRegexp.FindStringSubmatch(path)
	if paths == nil {
		return "", nil, "", err.NewNotFoundError()
	}
	driveName := paths[1]
	entryPath := paths[3]
	drive, ok := d.drives()[driveName]
	if !ok {
		if utils.IsRootPath(utils.PathParent(path)) {
			return "", nil, "", err.NewNotFoundMessageError(i18n.T("error.root_not_writable"))
		}
		return "", nil, "", err.NewNotFoundError()
	}
	return driveName, drive, entryPath, nil
}

func (d *DispatcherDrive) resolveMount(path string) string {
	tree := utils.PathParentTree(path)
	var mountAt, prefix string
	mounts := d.mounts()
	for _, p := range tree {
		dir := utils.PathParent(p)
		name := utils.PathBase(p)
		temp := mounts[dir]
		if temp != nil {
			mountAt = temp[name].MountAt
			if mountAt != "" {
				prefix = p
				break
			}
		}
	}
	if mountAt == "" {
		return ""
	}

	return path2.Join(
		mountAt,
		utils.CleanPath(path[len(prefix):]),
	)
}

func (d *DispatcherDrive) resolveMountedChildren(path string) ([]types.PathMount, bool) {
	result := make([]types.PathMount, 0)
	isSelf := false
	for mountParent, mounts := range d.mounts() {
		for mountName, m := range mounts {
			mountPath := path2.Join(mountParent, mountName)
			if mountPath == path || utils.IsPathParent(mountPath, path) {
				result = append(result, m)
				if !isSelf && path2.Join(*m.Path, m.Name) == path {
					isSelf = true
				}
			}
		}
	}
	return result, isSelf
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

func (d *DispatcherDrive) FindNonExistsEntryName(ctx context.Context, drive types.IDrive, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	_, e := drive.Get(ctx, path)
	if e != nil && !err.IsNotFoundError(e) {
		return "", e
	}
	if e != nil {
		return path, nil
	}
	parentPath := utils.PathParent(path)
	pathBaseName := utils.PathName(path)
	pathExt := path2.Ext(path)
	siblings, e := drive.List(ctx, parentPath)
	if e != nil {
		return "", e
	}
	seq := 1
	var newPath string
	for newPath == "" {
		newPath = utils.CleanPath(path2.Join(parentPath, fmt.Sprintf("%s_%d%s", pathBaseName, seq, pathExt)))
		for _, i := range siblings {
			if newPath == i.Path() {
				newPath = ""
				break
			}
		}
		seq++
	}
	return newPath, nil
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
		realPath, e = d.FindNonExistsEntryName(ctx, drive, realPath)
		if e != nil {
			return nil, e
		}
	}
	save, e := drive.Save(ctx, realPath, size, override, reader)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(path2.Join(driveName, realPath), driveName, save), nil
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
	mounts, _ := d.resolveMountedChildren(from.Path())
	if len(mounts) == 0 {
		// if `from` has no mounted children, then copy
		if !override {
			pathTo, e = d.FindNonExistsEntryName(ctx, driveTo, pathTo)
			if e != nil {
				return nil, e
			}
		}
		entry, e := driveTo.Copy(ctx, from, pathTo, override)
		if e == nil {
			return entry, nil
		}
		if !err.IsUnsupportedError(e) {
			return nil, e
		}
	}
	// if `from` has mounted children, we need to copy them
	e = drive_util.CopyAll(ctx, from, d, to,
		func(from types.IEntry, _ types.IDrive, to string, ctx types.TaskCtx) error {
			_, driveTo, pathTo, e := d.resolve(to)
			ctxWrapper := task.NewCtxWrapper(ctx, true, false)
			if e != nil {
				return e
			}

			if !override {
				pathTo, e = d.FindNonExistsEntryName(ctx, driveTo, pathTo)
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
			return drive_util.CopyEntry(ctxWrapper, from, driveTo, pathTo, true, d.tempDir)
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
	return d.mapDriveEntry(path2.Join(driveName, pathTo), driveName, copied), nil
}

func (d *DispatcherDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	driveName, driveTo, pathTo, e := d.resolve(to)
	// if path depth is 1, move mounts
	if e != nil && utils.PathDepth(to) != 1 {
		return nil, e
	}
	fromPath := from.Path()
	children, isSelf := d.resolveMountedChildren(fromPath)
	if len(children) > 0 {
		movedMounts := make([]types.PathMount, 0, len(children))
		for _, m := range children {
			t := path2.Join(
				to,
				path2.Join(*m.Path, m.Name)[len(fromPath):],
			)

			if !override {
				t, e = d.FindNonExistsEntryName(ctx, d, t)
				if e != nil {
					return nil, e
				}
			}

			mPath := utils.PathParent(t)
			m.Path = &mPath
			m.Name = utils.PathBase(t)
			movedMounts = append(movedMounts, m)
		}
		if e := d.mountStorage.DeleteAndSaveMounts(children, movedMounts, true); e != nil {
			return nil, e
		}
		_ = d.reloadMounts()
		if isSelf {
			return d.Get(ctx, path2.Join(driveName, pathTo))
		}
	} else {
		// no mounts matched and toPath is in root or trying to move drive
		if driveTo == nil {
			return nil, err.NewNotAllowedError()
		}
	}
	if driveTo != nil {

		if !override {
			pathTo, e = d.FindNonExistsEntryName(ctx, driveTo, pathTo)
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
		return d.mapDriveEntry(path2.Join(driveName, pathTo), driveName, move), nil
	}
	return d.Get(ctx, path2.Join(driveName, pathTo))
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

	ms := d.mounts()[path]
	if ms != nil {
		mountedMap := make(map[string]types.IEntry, len(entries))
		for name, m := range ms {
			_, drive, entryPath, e := d.resolve(m.MountAt)
			if e != nil {
				continue
			}
			entry, e := drive.Get(ctx, entryPath)
			if e != nil {
				if err.IsNotFoundError(e) {
					continue
				}
				return nil, e
			}
			mountedMap[name] = &entryWrapper{d: d, path: path2.Join(path, name), IEntry: entry, mountAt: m.MountAt}
		}

		newEntries := make([]types.IEntry, 0, len(entries)+len(mountedMap))
		for _, e := range entries {
			if mountedMap[utils.PathBase(e.Path())] == nil {
				newEntries = append(newEntries, e)
			}
		}
		for _, e := range mountedMap {
			newEntries = append(newEntries, e)
		}
		entries = newEntries
	}
	return entries, nil
}

func (d *DispatcherDrive) Delete(ctx types.TaskCtx, path string) error {
	children, isSelf := d.resolveMountedChildren(path)
	if len(children) > 0 {
		e := d.mountStorage.DeleteMounts(children)
		if e != nil {
			return e
		}
		_ = d.reloadMounts()
		if isSelf {
			return nil
		}
	}
	_, drive, path, e := d.resolve(path)
	if e != nil {
		return e
	}
	if utils.IsRootPath(path) {
		return err.NewNotAllowedError()
	}
	return drive.Delete(ctx, path)
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
		p, e := d.FindNonExistsEntryName(ctx, drive, realPath)
		if e != nil {
			return nil, e
		}
		if realPath != p {
			newPath = path2.Join(driveName, p)
		}
		realPath = p
	}
	if size == 0 {
		return types.UseLocalProvider(0), nil
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

type entryWrapper struct {
	types.IEntry
	d         *DispatcherDrive
	path      string
	mountAt   string
	driveName string
}

func (d *entryWrapper) Path() string {
	return d.path
}

func (d *entryWrapper) Meta() types.EntryMeta {
	meta := d.IEntry.Meta()
	if d.mountAt != "" {
		meta.Props = utils.MapCopy(meta.Props, nil)
		meta.Props["mountAt"] = d.mountAt
	}
	return meta
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
	return types.EntryMeta{Readable: true, Writable: true, Props: d.meta.Props}
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
