package drive

import (
	"go-drive/common"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var pathRegexp = regexp.MustCompile(`^/?([^/]+)(/(.*))?$`)

// Drive splits drive name and path from the raw path.
// Then dispatch request to the specified drive.
type Drive map[string]common.IDrive

func NewDrive() Drive {
	return make(Drive)
}

func (d Drive) Meta() common.IDriveMeta {
	panic("not supported")
}

func (d Drive) AddDrive(name string, drive common.IDrive) {
	d[name] = drive
}

func (d Drive) Resolve(path string) (common.IDrive, string, string, error) {
	paths := pathRegexp.FindStringSubmatch(path)
	if paths == nil {
		return nil, "", "", common.NewNotFoundError("not found")
	}
	driveName := paths[1]
	entryPath := "/" + paths[3]
	drive, ok := d[driveName]
	if !ok {
		return nil, "", "", common.NewNotFoundError("not found")
	}
	return drive, entryPath, driveName, nil
}

func (d Drive) Get(path string) (common.IEntry, error) {
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

func (d Drive) Save(path string, reader io.Reader, progress common.OnProgress) (common.IEntry, error) {
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

func (d Drive) MakeDir(path string) (common.IEntry, error) {
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

func (d Drive) Copy(from common.IEntry, to string, progress common.OnProgress) (common.IEntry, error) {
	driveTo, pathTo, name, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	_, e = driveTo.Get(pathTo)
	if e == nil {
		return nil, common.NewNotAllowedError("dst file exists")
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
	// copy file
	download, ok := from.(common.IDownloadable)
	if ok {
		url, _, e := download.GetURL()
		if e != nil {
			return nil, e
		}
		resp, e := http.Get(url)
		if e != nil {
			return nil, e
		}
		if resp.StatusCode != 200 {
			return nil, common.NewRemoteApiError(resp.StatusCode, "failed to copy file")
		}
		defer func() { _ = resp.Body.Close() }()
		save, e := driveTo.Save(pathTo, resp.Body, progress)
		if e != nil {
			return nil, e
		}
		return mapDriveEntry(name, save), nil
	}
	readable, ok := from.(common.IReadable)
	if ok {
		reader, e := readable.GetReader()
		if e != nil {
			return nil, e
		}
		defer func() { _ = reader.Close() }()
		save, e := driveTo.Save(pathTo, reader, progress)
		if e != nil {
			return nil, e
		}
		return mapDriveEntry(name, save), nil
	}
	return nil, common.NewNotAllowedError("source file is not readable")
}

func (d Drive) Move(from string, to string) (common.IEntry, error) {
	driveFrom, pathFrom, name, e := d.Resolve(from)
	if e != nil {
		return nil, e
	}
	driveTo, pathTo, _, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	if driveFrom != driveTo {
		return nil, common.NewNotAllowedError("not allowed")
	}
	move, e := driveTo.Move(pathFrom, pathTo)
	if e != nil {
		return nil, e
	}
	return mapDriveEntry(name, move), nil
}

func (d Drive) List(path string) ([]common.IEntry, error) {
	if path == "" || path == "/" {
		drives := make([]common.IEntry, 0, len(d))
		for k, v := range d {
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

func (d Drive) Delete(path string) error {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return e
	}
	return drive.Delete(path)
}

func (d Drive) Upload(path string, overwrite bool) (*common.DriveUploadConfig, error) {
	drive, path, _, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Upload(path, overwrite)
}

func mapDriveEntry(driveName string, entry common.IEntry) common.IEntry {
	_, downloadable := entry.(common.IDownloadable)
	_, readable := entry.(common.IReadable)
	entryWrapper := driveEntryWrapper{driveName: driveName, entry: entry}
	if downloadable && readable {
		return &driveEntryWrapperDownloadableReadable{entryWrapper}
	} else if downloadable {
		return &driveEntryWrapperDownloadable{entryWrapper}
	} else if readable {
		return &driveEntryWrapperReadable{entryWrapper}
	}
	return &entryWrapper
}

func mapDriveEntries(driveName string, entries []common.IEntry) []common.IEntry {
	mappedEntries := make([]common.IEntry, 0, len(entries))
	for _, e := range entries {
		mappedEntries = append(mappedEntries, mapDriveEntry(driveName, e))
	}
	return mappedEntries
}

type driveEntryWrapper struct {
	driveName string
	entry     common.IEntry
}

type driveEntryWrapperDownloadable struct {
	driveEntryWrapper
}

type driveEntryWrapperReadable struct {
	driveEntryWrapper
}

type driveEntryWrapperDownloadableReadable struct {
	driveEntryWrapper
}

func (d driveEntryWrapperDownloadable) GetURL() (string, bool, error) {
	downloadable := d.entry.(common.IDownloadable)
	return downloadable.GetURL()
}

func (d driveEntryWrapperReadable) GetReader() (io.ReadCloser, error) {
	readable := d.entry.(common.IReadable)
	return readable.GetReader()
}

func (d *driveEntryWrapper) Path() string {
	path := d.entry.Path()
	for strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return d.driveName + "/" + path
}

func (d *driveEntryWrapper) Name() string {
	return d.entry.Name()
}

func (d *driveEntryWrapper) Type() common.EntryType {
	return d.entry.Type()
}

func (d *driveEntryWrapper) Size() int64 {
	return d.entry.Size()
}

func (d *driveEntryWrapper) Meta() common.IEntryMeta {
	return d.entry.Meta()
}

func (d *driveEntryWrapper) CreatedAt() int64 {
	return d.entry.CreatedAt()
}

func (d *driveEntryWrapper) UpdatedAt() int64 {
	return d.entry.UpdatedAt()
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
	props := make(map[string]interface{})
	for k, v := range d.meta.Props() {
		props[k] = v
	}
	return driveEntryMeta{canWrite: d.meta.CanWrite(), props: props}
}

func (d *driveEntry) CreatedAt() int64 {
	return -1
}

func (d *driveEntry) UpdatedAt() int64 {
	return -1
}

type driveEntryMeta struct {
	canWrite bool
	props    map[string]interface{}
}

func (d driveEntryMeta) CanWrite() bool {
	return d.canWrite
}

func (d driveEntryMeta) Props() map[string]interface{} {
	return d.props
}
