package drive

import (
	"go-drive/common"
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
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	if common.IsRootPath(path) {
		return &driveEntry{path: name, name: name, meta: drive.Meta()}, nil
	}
	entry, e := drive.Get(path)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, entry), nil
}

func (d *DispatcherDrive) Save(path string, reader io.Reader, progress types.OnProgress) (types.IEntry, error) {
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	save, e := drive.Save(path, reader, progress)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, save), nil
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
	return mapDriveEntry(name, dir), nil
}

func (d *DispatcherDrive) Copy(from types.IEntry, to string, progress types.OnProgress) (types.IEntry, error) {
	driveTo, pathTo, name, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	_, e = driveTo.Get(pathTo)
	if e == nil {
		return nil, common.NewNotAllowedMessageError("dst file exists")
	}
	if !common.IsNotFoundError(e) {
		return nil, e
	}
	entry, e := driveTo.Copy(from, pathTo, progress)
	if e == nil {
		return entry, nil
	}
	if !common.IsUnsupportedError(e) {
		return nil, e
	}
	content, ok := from.(types.IContent)
	if !ok {
		return nil, common.NewNotAllowedMessageError("not allowed")
	}
	file, e := common.CopyIContentToTempFile(content, progress)
	if e != nil {
		return nil, e
	}
	save, e := driveTo.Save(pathTo, file, progress)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, save), nil
}

func (d *DispatcherDrive) Move(from string, to string) (types.IEntry, error) {
	driveFrom, pathFrom, name, e := d.Resolve(from)
	if e != nil {
		return nil, e
	}
	driveTo, pathTo, _, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	if driveFrom != driveTo {
		return nil, common.NewNotAllowedError()
	}
	move, e := driveTo.Move(pathFrom, pathTo)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, move), nil
}

func (d *DispatcherDrive) List(path string) ([]types.IEntry, error) {
	if common.IsRootPath(path) {
		drives := make([]types.IEntry, 0, len(d.drives))
		for k, v := range d.drives {
			drives = append(drives, &driveEntry{path: k, name: k, meta: v.Meta()})
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
	return mapDriveEntries(name, list), nil
}

func (d *DispatcherDrive) Delete(path string) error {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return e
	}
	return drive.Delete(path)
}

func (d *DispatcherDrive) Upload(path string, size int64, overwrite bool) (*types.DriveUploadConfig, error) {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Upload(path, size, overwrite)
}

func mapDriveEntry(driveName string, entry types.IEntry) types.IEntry {
	return &entryWrapper{driveName: driveName, entry: entry}
}

func mapDriveEntries(driveName string, entries []types.IEntry) []types.IEntry {
	mappedEntries := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		mappedEntries = append(mappedEntries, mapDriveEntry(driveName, e))
	}
	return mappedEntries
}

type entryWrapper struct {
	driveName string
	entry     types.IEntry
}

func (d *entryWrapper) Path() string {
	path := d.entry.Path()
	for strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return d.driveName + "/" + path
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

type driveEntry struct {
	path string
	name string
	meta types.DriveMeta
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
	return types.EntryMeta{CanRead: true, CanWrite: d.meta.CanWrite, Props: d.meta.Props}
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
