package drive

import (
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"regexp"
	"strings"
	"sync"
)

var pathRegexp = regexp.MustCompile(`^/?([^/]+)(/(.*))?$`)

// DispatcherDrive splits drive name and path from the raw path.
// Then dispatch request to the specified drive.
type DispatcherDrive struct {
	drives map[string]types.IDrive
	mux    *sync.Mutex
}

func NewDispatcherDrive() *DispatcherDrive {
	return &DispatcherDrive{drives: make(map[string]types.IDrive), mux: &sync.Mutex{}}
}

func (d *DispatcherDrive) SetDrives(drives map[string]types.IDrive) {
	d.mux.Lock()
	defer d.mux.Unlock()
	for _, d := range d.drives {
		if disposable, ok := d.(types.IDisposable); ok {
			_ = disposable.Dispose()
		}
	}
	newDrives := make(map[string]types.IDrive, len(drives))
	for k, v := range drives {
		newDrives[k] = v
	}
	d.drives = newDrives
}

func (d *DispatcherDrive) Meta() types.DriveMeta {
	panic("not supported")
}

func (d *DispatcherDrive) Resolve(path string) (types.IDrive, string, string, error) {
	paths := pathRegexp.FindStringSubmatch(path)
	if paths == nil {
		return nil, "", "", common.NewNotFoundError("not found")
	}
	driveName := paths[1]
	entryPath := "/" + paths[3]
	drive, ok := d.drives[driveName]
	if !ok {
		return nil, "", "", common.NewNotFoundError("not found")
	}
	return drive, entryPath, driveName, nil
}

func (d *DispatcherDrive) Get(path string) (types.IEntry, error) {
	if common.IsRootPath(path) {
		return &driveEntry{d: d, path: "", name: "", meta: types.DriveMeta{
			CanWrite: false,
		}}, nil
	}
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	entry, e := drive.Get(path)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(name, entry), nil
}

func (d *DispatcherDrive) Save(path string, reader io.Reader, ctx task.Context) (types.IEntry, error) {
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	save, e := drive.Save(path, reader, ctx)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(name, save), nil
}

func (d *DispatcherDrive) MakeDir(path string) (types.IEntry, error) {
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	dir, e := drive.MakeDir(path)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(name, dir), nil
}

func (d *DispatcherDrive) Copy(from types.IEntry, to string, override bool, ctx task.Context) (types.IEntry, error) {
	driveTo, pathTo, name, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	entry, e := driveTo.Copy(from, pathTo, override, ctx)
	if e == nil {
		return entry, nil
	}
	if !common.IsUnsupportedError(e) {
		return nil, e
	}
	e = common.CopyAll(from, d, to, override, ctx, nil)
	if e != nil {
		return nil, e
	}
	copied, e := driveTo.Get(pathTo)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(name, copied), nil
}

func (d *DispatcherDrive) Move(from types.IEntry, to string, override bool, ctx task.Context) (types.IEntry, error) {
	driveTo, pathTo, name, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	move, e := driveTo.Move(from, pathTo, override, ctx)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntry(name, move), nil
}

func (d *DispatcherDrive) List(path string) ([]types.IEntry, error) {
	if common.IsRootPath(path) {
		drives := make([]types.IEntry, 0, len(d.drives))
		for k, v := range d.drives {
			drives = append(drives, &driveEntry{d: d, path: k, name: k, meta: v.Meta(), isDrive: true})
		}
		return drives, nil
	}
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	list, e := drive.List(path)
	if e != nil {
		return nil, e
	}
	return d.mapDriveEntries(name, list), nil
}

func (d *DispatcherDrive) Delete(path string) error {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return e
	}
	if path == "/" {
		return common.NewNotAllowedError()
	}
	return drive.Delete(path)
}

func (d *DispatcherDrive) Upload(path string, size int64, override bool) (*types.DriveUploadConfig, error) {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Upload(path, size, override)
}

func (d *DispatcherDrive) mapDriveEntry(driveName string, entry types.IEntry) types.IEntry {
	return &entryWrapper{d: d, driveName: driveName, entry: entry}
}

func (d *DispatcherDrive) mapDriveEntries(driveName string, entries []types.IEntry) []types.IEntry {
	mappedEntries := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		mappedEntries = append(mappedEntries, d.mapDriveEntry(driveName, e))
	}
	return mappedEntries
}

type entryWrapper struct {
	d         *DispatcherDrive
	driveName string
	entry     types.IEntry
}

func (d *entryWrapper) Path() string {
	path := d.entry.Path()
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return d.driveName + path
}

func (d *entryWrapper) Name() string {
	return d.entry.Name()
}

func (d *entryWrapper) Type() types.EntryType {
	return d.entry.Type()
}

func (d *entryWrapper) Size() int64 {
	return d.entry.Size()
}

func (d *entryWrapper) Meta() types.EntryMeta {
	return d.entry.Meta()
}

func (d *entryWrapper) CreatedAt() int64 {
	return d.entry.CreatedAt()
}

func (d *entryWrapper) UpdatedAt() int64 {
	return d.entry.UpdatedAt()
}

func (d *entryWrapper) GetReader() (io.ReadCloser, error) {
	if content, ok := d.entry.(types.IContent); ok {
		return content.GetReader()
	}
	return nil, common.NewNotAllowedError()
}

func (d *entryWrapper) GetURL() (string, bool, error) {
	if content, ok := d.entry.(types.IContent); ok {
		return content.GetURL()
	}
	return "", false, common.NewNotAllowedError()
}

func (d *entryWrapper) Drive() types.IDrive {
	return d.d
}

func (d *entryWrapper) GetIEntry() types.IEntry {
	return d.entry
}

type driveEntry struct {
	d       *DispatcherDrive
	path    string
	name    string
	meta    types.DriveMeta
	isDrive bool
}

func (d *driveEntry) Path() string {
	return d.path
}

func (d *driveEntry) Name() string {
	return d.name
}

func (d *driveEntry) Type() types.EntryType {
	return types.TypeDir
}

func (d *driveEntry) Size() int64 {
	return -1
}

func (d *driveEntry) Meta() types.EntryMeta {
	return types.EntryMeta{CanRead: true, CanWrite: d.meta.CanWrite && !d.isDrive, Props: d.meta.Props}
}

func (d *driveEntry) CreatedAt() int64 {
	return -1
}

func (d *driveEntry) UpdatedAt() int64 {
	return -1
}

func (d *driveEntry) GetReader() (io.ReadCloser, error) {
	return nil, common.NewNotAllowedError()
}

func (d *driveEntry) GetURL() (string, bool, error) {
	return "", false, common.NewNotAllowedError()
}

func (d *driveEntry) Drive() types.IDrive {
	return d.d
}
