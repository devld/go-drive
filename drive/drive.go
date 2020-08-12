package drive

import (
	"go-drive/common"
	"io"
	"regexp"
	"strings"
	"sync"
)

var pathRegexp = regexp.MustCompile(`^/?([^/]+)(/(.*))?$`)

// Drive splits drive name and path from the raw path.
// Then dispatch request to the specified drive.
type Drive struct {
	drives map[string]common.IDrive
	mux    *sync.Mutex
}

func NewDrive() *Drive {
	return &Drive{drives: make(map[string]common.IDrive), mux: &sync.Mutex{}}
}

func (d *Drive) SetDrives(drives map[string]common.IDrive) {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.drives = drives
	newDrives := make(map[string]common.IDrive, len(drives))
	for k, v := range drives {
		newDrives[k] = v
	}
	d.drives = newDrives
}

func (d *Drive) Meta() common.IDriveMeta {
	panic("not supported")
}

func (d *Drive) Resolve(path string) (common.IDrive, string, string, error) {
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

func (d *Drive) Get(path string) (common.IEntry, error) {
	drive, path, name, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	if path == "" || path == "/" {
		return &driveEntry{path: name, name: name, meta: drive.Meta()}, nil
	}
	entry, e := drive.Get(path)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, entry), nil
}

func (d *Drive) Save(path string, reader io.Reader, progress common.OnProgress) (common.IEntry, error) {
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

func (d *Drive) MakeDir(path string) (common.IEntry, error) {
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

func (d *Drive) Copy(from common.IEntry, to string, progress common.OnProgress) (common.IEntry, error) {
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
	if !common.IsNotSupportedError(e) {
		return nil, e
	}
	content, ok := from.(common.IContent)
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

func (d *Drive) Move(from string, to string) (common.IEntry, error) {
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

func (d *Drive) List(path string) ([]common.IEntry, error) {
	if path == "" || path == "/" {
		drives := make([]common.IEntry, 0, len(d.drives))
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

func (d *Drive) Delete(path string) error {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return e
	}
	return drive.Delete(path)
}

func (d *Drive) Upload(path string, size int64, overwrite bool) (*common.DriveUploadConfig, error) {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Upload(path, size, overwrite)
}

func mapDriveEntry(driveName string, entry common.IEntry) common.IEntry {
	return &entryWrapper{driveName: driveName, entry: entry}
}

func mapDriveEntries(driveName string, entries []common.IEntry) []common.IEntry {
	mappedEntries := make([]common.IEntry, 0, len(entries))
	for _, e := range entries {
		mappedEntries = append(mappedEntries, mapDriveEntry(driveName, e))
	}
	return mappedEntries
}

type entryWrapper struct {
	driveName string
	entry     common.IEntry
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

func (d *entryWrapper) Type() common.EntryType {
	return d.entry.Type()
}

func (d *entryWrapper) Size() int64 {
	return d.entry.Size()
}

func (d *entryWrapper) Meta() common.IEntryMeta {
	return d.entry.Meta()
}

func (d *entryWrapper) CreatedAt() int64 {
	return d.entry.CreatedAt()
}

func (d *entryWrapper) UpdatedAt() int64 {
	return d.entry.UpdatedAt()
}

func (d *entryWrapper) GetReader() (io.ReadCloser, error) {
	content, ok := d.entry.(common.IContent)
	if !ok {
		return nil, common.NewNotAllowedError()
	}
	return content.GetReader()
}

func (d *entryWrapper) GetURL() (string, bool, error) {
	content, ok := d.entry.(common.IContent)
	if !ok {
		return "", false, common.NewNotAllowedError()
	}
	return content.GetURL()
}

type driveEntry struct {
	path string
	name string
	meta common.IDriveMeta
}

func (d *driveEntry) Path() string {
	return d.path
}

func (d *driveEntry) Name() string {
	return d.name
}

func (d *driveEntry) Type() common.EntryType {
	return common.TypeDir
}

func (d *driveEntry) Size() int64 {
	return -1
}

func (d *driveEntry) Meta() common.IEntryMeta {
	return driveEntryMeta{d.meta.CanWrite(), d.meta.Props()}
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

type driveEntryMeta struct {
	canWrite bool
	props    map[string]interface{}
}

func (d driveEntryMeta) CanWrite() bool {
	return d.canWrite
}

func (d driveEntryMeta) CanRead() bool {
	return true
}

func (d driveEntryMeta) Props() map[string]interface{} {
	return d.props
}
