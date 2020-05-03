package drive

import (
	"go-drive/common"
	"io"
	"net/http"
	"regexp"
)

var pathRegexp = regexp.MustCompile(`^/([^/]+)(/(.*))?$`)

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

func (d Drive) Resolve(path string) (common.IDrive, string, error) {
	paths := pathRegexp.FindStringSubmatch(path)
	if paths == nil {
		return nil, "", common.NewNotFoundError("not found")
	}
	driveName := paths[1]
	entryPath := "/" + paths[3]
	drive, ok := d[driveName]
	if !ok {
		return nil, "", common.NewNotFoundError("not found")
	}
	return drive, entryPath, nil
}

func (d Drive) Get(path string) (common.IEntry, error) {
	drive, path, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Get(path)
}

func (d Drive) Save(path string, reader io.Reader, progress common.OnProgress) (common.IEntry, error) {
	drive, path, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.Save(path, reader, progress)
}

func (d Drive) MakeDir(path string) (common.IEntry, error) {
	drive, path, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.MakeDir(path)
}

func (d Drive) Copy(from common.IEntry, to string, progress common.OnProgress) (common.IEntry, error) {
	driveTo, pathTo, e := d.Resolve(to)
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
		defer func() { _ = resp.Body.Close() }()
		return driveTo.Save(pathTo, resp.Body, progress)
	}
	readable, ok := from.(common.IReadable)
	if ok {
		reader, e := readable.GetReader()
		if e != nil {
			return nil, e
		}
		defer func() { _ = reader.Close() }()
		return driveTo.Save(pathTo, reader, progress)
	}
	return nil, common.NewNotAllowedError("source file is not readable")
}

func (d Drive) Move(from string, to string) (common.IEntry, error) {
	driveFrom, pathFrom, e := d.Resolve(from)
	if e != nil {
		return nil, e
	}
	driveTo, pathTo, e := d.Resolve(to)
	if e != nil {
		return nil, e
	}
	if driveFrom != driveTo {
		return nil, common.NewNotAllowedError("not allowed")
	}
	return driveTo.Move(pathFrom, pathTo)
}

func (d Drive) List(path string) ([]common.IEntry, error) {
	if path == "" || path == "/" {
		drives := make([]common.IEntry, 0, len(d))
		for k, v := range d {
			drives = append(drives, driveEntry{name: k, meta: v.Meta()})
		}
		return drives, nil
	}
	drive, path, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	return drive.List(path)
}

func (d Drive) Delete(path string) error {
	drive, path, e := d.Resolve(path)
	if e != nil {
		return e
	}
	return drive.Delete(path)
}

func (d Drive) Upload(path string) (interface{}, error) {
	drive, path, e := d.Resolve(path)
	if e != nil {
		return nil, e
	}
	driveUpload, ok := drive.(common.IDriveUpload)
	if !ok {
		return nil, common.NewNotAllowedError("not supported")
	}
	return driveUpload.Upload(path)
}

type driveEntry struct {
	name string
	meta common.IDriveMeta
}

func (d driveEntry) Name() string {
	return d.name
}

func (d driveEntry) Type() common.EntryType {
	return "drive"
}

func (d driveEntry) Size() int64 {
	return -1
}

func (d driveEntry) Meta() common.IEntryMeta {
	props := make(map[string]interface{})
	props["direct_upload"] = d.meta.DirectlyUpload()
	for k, v := range d.meta.Props() {
		props[k] = v
	}
	return driveEntryMeta{canWrite: d.meta.CanWrite(), props: props}
}

func (d driveEntry) CreatedAt() int64 {
	return 0
}

func (d driveEntry) UpdatedAt() int64 {
	return 0
}

type driveEntryMeta struct {
	canWrite bool
	props    map[string]interface{}
}

func (d driveEntryMeta) CanRead() bool {
	return true
}

func (d driveEntryMeta) CanWrite() bool {
	return d.canWrite
}

func (d driveEntryMeta) Props() map[string]interface{} {
	return d.props
}
