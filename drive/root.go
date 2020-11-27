package drive

import (
	"encoding/json"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"go-drive/drive/onedrive"
	"go-drive/storage"
	"log"
	"sync"
)

var driveFactories = map[string]drive_util.DriveFactory{
	"fs":       {Create: NewFsDrive},
	"s3":       {Create: NewS3Drive},
	"onedrive": {Create: onedrive.NewOneDrive, InitConfig: onedrive.InitConfig, Init: onedrive.Init},
	"webdav":   {Create: NewWebDAVDrive},
}

type RootDrive struct {
	root              *DispatcherDrive
	driveStorage      *storage.DriveDAO
	mountStorage      *storage.PathMountDAO
	driveDataStorage  *storage.DriveDataDAO
	driveCacheStorage *storage.DriveCacheDAO

	config common.Config

	mux *sync.Mutex
}

func NewRootDrive(
	config common.Config,
	driveStorage *storage.DriveDAO,
	mountStorage *storage.PathMountDAO,
	dataStorage *storage.DriveDataDAO,
	driveCacheStorage *storage.DriveCacheDAO) (*RootDrive, error) {
	root := NewDispatcherDrive(mountStorage, config)
	r := &RootDrive{
		root:              root,
		driveStorage:      driveStorage,
		mountStorage:      mountStorage,
		driveDataStorage:  dataStorage,
		driveCacheStorage: driveCacheStorage,
		config:            config,
		mux:               &sync.Mutex{},
	}
	if e := r.ReloadMounts(); e != nil {
		return nil, e
	}
	if e := r.ReloadDrive(true); e != nil {
		return nil, e
	}
	return r, nil
}

func (d *RootDrive) Get() types.IDrive {
	return d.root
}

func checkAndParseConfig(dc types.Drive) (*drive_util.DriveFactory, types.SM, error) {
	factory, ok := driveFactories[dc.Type]
	if !ok {
		return nil, nil, common.NewBadRequestError(fmt.Sprintf("invalid drive type '%s'", dc.Type))
	}
	config := make(types.SM)
	e := json.Unmarshal([]byte(dc.Config), &config)
	if e != nil {
		return nil, nil, common.NewBadRequestError(fmt.Sprintf("invalid drive config of '%s'", dc.Name))
	}
	return &factory, config, nil
}

func (d *RootDrive) ReloadDrive(ignoreFailure bool) error {
	d.mux.Lock()
	defer d.mux.Unlock()

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
	for _, dc := range drivesConfig {
		if !dc.Enabled {
			continue
		}
		factory, config, e := checkAndParseConfig(dc)
		if e != nil {
			if ignoreFailure {
				log.Printf("[%s]: %v", dc.Name, e)
				continue
			}
			return e
		}
		iDrive, e := factory.Create(config, d.createDriveUtils(dc.Name))
		if e != nil {
			if ignoreFailure {
				log.Printf("[%s]: %v", dc.Name, e)
				continue
			}
			return common.NewBadRequestError(fmt.Sprintf("error when creating drive '%s': %s", dc.Name, e.Error()))
		}
		drives[dc.Name] = iDrive
	}
	d.root.setDrives(drives)
	ok = true
	return nil
}

func (d *RootDrive) ReloadMounts() error {
	return d.root.reloadMounts()
}

func (d *RootDrive) DriveInitConfig(name string) (*drive_util.DriveInitConfig, error) {
	dc, e := d.driveStorage.GetDrive(name)
	if e != nil {
		return nil, e
	}
	factory, config, e := checkAndParseConfig(dc)
	if e != nil {
		return nil, e
	}
	if factory.InitConfig == nil {
		return nil, nil
	}
	initConfig, e := factory.InitConfig(config, d.createDriveUtils(name))
	return &initConfig, e
}

func (d *RootDrive) DriveInit(name string, data types.SM) error {
	dc, e := d.driveStorage.GetDrive(name)
	if e != nil {
		return e
	}
	factory, config, e := checkAndParseConfig(dc)
	if e != nil {
		return e
	}
	if factory.Init == nil {
		return nil
	}
	return factory.Init(data, config, d.createDriveUtils(name))
}

func (d *RootDrive) createDriveUtils(name string) drive_util.DriveUtils {
	return drive_util.DriveUtils{
		Data: d.driveDataStorage.GetDataStore(name),
		CreateCache: func(de drive_util.EntryDeserialize, s drive_util.EntrySerialize) drive_util.DriveCache {
			if s == nil {
				s = drive_util.SerializeEntry
			}
			return d.driveCacheStorage.GetCacheStore(name, s, de)
		},
		Config: d.config,
	}
}
