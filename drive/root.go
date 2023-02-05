package drive

import (
	"context"
	"encoding/json"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"
	"log"
	"sync"
)

type RootDrive struct {
	root             *DispatcherDrive
	driveStorage     *storage.DriveDAO
	mountStorage     *storage.PathMountDAO
	driveDataStorage *storage.DriveDataDAO

	driveCacheMgr drive_util.DriveCacheManager

	config common.Config

	mux *sync.Mutex
}

func NewRootDrive(
	ctx context.Context,
	config common.Config,
	driveStorage *storage.DriveDAO,
	mountStorage *storage.PathMountDAO,
	dataStorage *storage.DriveDataDAO,
	driveCacheStorage *storage.DriveCacheDAO,
	ch *registry.ComponentsHolder) (*RootDrive, error) {
	root := NewDispatcherDrive(mountStorage, config)
	r := &RootDrive{
		root:             root,
		driveStorage:     driveStorage,
		mountStorage:     mountStorage,
		driveDataStorage: dataStorage,
		config:           config,
		mux:              &sync.Mutex{},
	}

	switch config.Cache.Type {
	case "db":
		r.driveCacheMgr = driveCacheStorage
		driveCacheStorage.StartCleaner(config.Cache.CleanPeriod)
	default:
		r.driveCacheMgr = drive_util.NewMemDriveCacheManager(config.Cache.CleanPeriod)
	}

	if e := r.ReloadMounts(); e != nil {
		return nil, e
	}
	if e := r.ReloadDrive(ctx, true); e != nil {
		return nil, e
	}
	ch.Add("rootDrive", r)
	return r, nil
}

func (d *RootDrive) Get() types.IDispatcherDrive {
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

	log.Println("Reloading drives...")
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
		log.Println("Creating drive:", dc.Name)
		iDrive, e := factory.Create(ctx, config, d.createDriveUtils(dc.Name))
		if e != nil {
			if ignoreFailure {
				log.Printf("[%s]: %v", dc.Name, e)
				continue
			}
			return err.NewBadRequestError(i18n.T("drive.root.error_create_drive", dc.Name, e.Error()))
		}
		log.Println("Created drive:", dc.Name)
		drives[dc.Name] = iDrive
	}
	d.root.setDrives(drives)
	ok = true

	log.Println("Reloading drives done.")
	return nil
}

func (d *RootDrive) ReloadMounts() error {
	return d.root.reloadMounts()
}

func (d *RootDrive) Dispose() error {
	_ = d.driveCacheMgr.Dispose()
	return d.root.Dispose()
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
		CreateCache: func(de drive_util.EntryDeserialize) drive_util.DriveCache {
			return d.driveCacheMgr.GetCacheStore(name, de)
		},
		Config: d.config,
	}
}
