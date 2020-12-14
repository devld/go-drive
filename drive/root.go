package drive

import (
	"context"
	"encoding/json"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	_ "go-drive/drive/gdrive"
	_ "go-drive/drive/onedrive"
	"go-drive/storage"
	"log"
	"sync"
)

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
	ctx context.Context,
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
	if e := r.ReloadDrive(ctx, true); e != nil {
		return nil, e
	}
	return r, nil
}

func (d *RootDrive) Get() types.IDrive {
	return d.root
}

func checkAndParseConfig(dc types.Drive) (*drive_util.DriveFactory, types.SM, error) {
	f := drive_util.GetDrive(dc.Type)
	if f == nil {
		return nil, nil, err.NewBadRequestError(i18n.T("drive.root.invalid_drive_type", dc.Type))
	}
	config := make(types.SM)
	e := json.Unmarshal([]byte(dc.Config), &config)
	if e != nil {
		return nil, nil, err.NewBadRequestError(i18n.T("drive.root.invalid_drive_config", dc.Name))
	}
	return &f.Factory, config, nil
}

func (d *RootDrive) ReloadDrive(ctx context.Context, ignoreFailure bool) error {
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
		iDrive, e := factory.Create(ctx, config, d.createDriveUtils(dc.Name))
		if e != nil {
			if ignoreFailure {
				log.Printf("[%s]: %v", dc.Name, e)
				continue
			}
			return err.NewBadRequestError(i18n.T("drive.root.error_create_drive", dc.Name, e.Error()))
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

func (d *RootDrive) DriveInitConfig(ctx context.Context, name string) (*drive_util.DriveInitConfig, error) {
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
	initConfig, e := factory.InitConfig(ctx, config, d.createDriveUtils(name))
	return initConfig, e
}

func (d *RootDrive) DriveInit(ctx context.Context, name string, data types.SM) error {
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
	return factory.Init(ctx, data, config, d.createDriveUtils(name))
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
