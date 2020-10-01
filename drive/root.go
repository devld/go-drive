package drive

import (
	"encoding/json"
	"fmt"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/storage"
)

var drivesFactory = map[string]types.DriveCreator{
	"fs": NewFsDrive,
	"s3": NewS3Drive,
}

type RootDrive struct {
	root         *DispatcherDrive
	driveStorage *storage.DriveStorage
	mountStorage *storage.PathMountStorage
}

func NewRootDrive(driveStorage *storage.DriveStorage, mountStorage *storage.PathMountStorage) (*RootDrive, error) {
	root := NewDispatcherDrive(mountStorage)
	r := &RootDrive{
		root:         root,
		driveStorage: driveStorage,
		mountStorage: mountStorage,
	}
	if e := r.ReloadMounts(); e != nil {
		return nil, e
	}
	if e := r.ReloadDrive(); e != nil {
		return nil, e
	}
	return r, nil
}

func (d *RootDrive) Get() types.IDrive {
	return d.root
}

func (d *RootDrive) ReloadDrive() error {
	drivesConfig, e := d.driveStorage.GetDrives()
	if e != nil {
		return e
	}
	drives := make(map[string]types.IDrive, len(drivesConfig))
	ok := false
	defer func() {
		if !ok {
			for _, d := range drives {
				if disposable, ok := d.(types.IDisposable); ok {
					_ = disposable.Dispose()
				}
			}
		}
	}()
	for _, d := range drivesConfig {
		create, ok := drivesFactory[d.Type]
		if !ok {
			return common.NewBadRequestError(fmt.Sprintf("invalid drive type '%s'", d.Type))
		}
		config := make(map[string]string)
		e := json.Unmarshal([]byte(d.Config), &config)
		if e != nil {
			return common.NewBadRequestError(fmt.Sprintf("invalid drive config of '%s'", d.Name))
		}
		iDrive, e := create(config)
		if e != nil {
			return common.NewBadRequestError(fmt.Sprintf("error when creating drive '%s': %s", d.Name, e.Error()))
		}
		drives[d.Name] = iDrive
	}
	d.root.setDrives(drives)
	ok = true
	return nil
}

func (d *RootDrive) ReloadMounts() error {
	return d.root.reloadMounts()
}
