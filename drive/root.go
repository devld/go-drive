package drive

import (
	"context"
	"encoding/json"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"
	"sync"
	"time"
)

type RootDrive struct {
	root             *PathMountOverlayDrive
	dispatcher       *DispatcherDrive
	driveStorage     *storage.DriveDAO
	mountStorage     *storage.PathMountDAO
	driveDataStorage *storage.DriveDataDAO

	driveCacheMgr driveutil.DriveCacheManager
	driveRegistry *driveutil.DriveRegistry

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
	driveRegistry := ch.Get(registry.KeyDriveRegistry).(*driveutil.DriveRegistry)
	dispatcher := NewDispatcherDrive(config)
	root := NewPathMountOverlayDrive(dispatcher, mountStorage)
	r := &RootDrive{
		root:             root,
		dispatcher:       dispatcher,
		driveStorage:     driveStorage,
		mountStorage:     mountStorage,
		driveDataStorage: dataStorage,
		driveRegistry:    driveRegistry,
		config:           config,
		mux:              &sync.Mutex{},
	}

	switch config.Cache.Type {
	case "db":
		r.driveCacheMgr = driveCacheStorage
		driveCacheStorage.StartCleaner(config.Cache.CleanPeriod)
	default:
		r.driveCacheMgr = driveutil.NewMemDriveCacheManager(config.Cache.CleanPeriod)
	}

	if e := r.ReloadMounts(); e != nil {
		return nil, e
	}
	if e := r.ReloadDrive(ctx, true); e != nil {
		return nil, e
	}
	ch.Add(registry.KeyRootDrive, r)
	return r, nil
}

func (d *RootDrive) Get() types.IDrive {
	return d.root
}

func checkAndParseConfig(dc types.Drive, driveRegistry *driveutil.DriveRegistry) (*driveutil.DriveFactory, types.SM, error) {
	f := driveRegistry.GetDrive(dc.Type)
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
	started := time.Now()

	drivesConfig, e := d.driveStorage.GetDrives()
	if e != nil {
		logging.For("drive").Errorf("drive reload failed: %v", e)
		return e
	}

	driveLog := logging.For("drive")
	driveLog.Debugf("drive reload started configured=%d ignore_failure=%t", len(drivesConfig), ignoreFailure)
	drives := make(map[string]types.IDrive, len(drivesConfig))
	ok := false
	defer func() {
		if !ok {
			for name, d := range drives {
				if disposable, ok := d.(types.IDisposable); ok {
					if e := disposable.Dispose(); e != nil {
						driveLog.Warnf("drive cleanup failed name=%s: %v", logging.Sanitize(name), e)
					}
				}
			}
		}
	}()
	for _, dc := range drivesConfig {
		if !dc.Enabled {
			continue
		}
		factory, config, e := checkAndParseConfig(dc, d.driveRegistry)
		if e != nil {
			if ignoreFailure {
				driveLog.Warnf("error parsing drive config for '%s' (%s): %v",
					logging.Sanitize(dc.Name), logging.Sanitize(dc.Type), e)
				continue
			}
			return e
		}
		driveLog.Infof("creating drive '%s' (%s)", logging.Sanitize(dc.Name), logging.Sanitize(dc.Type))
		iDrive, e := factory.Create(ctx, config, d.createDriveUtils(dc.Name))
		if e != nil {
			if ignoreFailure {
				driveLog.Warnf("error creating drive '%s' (%s): %v",
					logging.Sanitize(dc.Name), logging.Sanitize(dc.Type), e)
				continue
			}
			return err.NewBadRequestError(i18n.T("drive.root.error_create_drive", dc.Name, e.Error()))
		}
		driveLog.Infof("created drive '%s' (%s)", logging.Sanitize(dc.Name), logging.Sanitize(dc.Type))
		drives[dc.Name] = iDrive
	}
	d.dispatcher.setDrives(drives)
	ok = true

	driveLog.Infof("reloaded drives configured=%d active=%d duration=%s", len(drivesConfig), len(drives), time.Since(started))
	return nil
}

func (d *RootDrive) ReloadMounts() error {
	started := time.Now()
	if e := d.root.reloadMounts(); e != nil {
		logging.For("drive").Errorf("mount reload failed: %v", e)
		return e
	}
	logging.For("drive").Debugf("mount reload completed duration=%s", time.Since(started))
	return nil
}

func (d *RootDrive) ClearDriveCache(ns string) error {
	return d.driveCacheMgr.EvictCacheStore(ns)
}

func (d *RootDrive) Dispose() error {
	_ = d.driveCacheMgr.Dispose()
	return d.dispatcher.Dispose()
}

func (d *RootDrive) DriveInitConfig(ctx context.Context, name string) (*driveutil.DriveInitConfig, error) {
	dc, e := d.driveStorage.GetDrive(name)
	if e != nil {
		return nil, e
	}
	factory, config, e := checkAndParseConfig(dc, d.driveRegistry)
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
	factory, config, e := checkAndParseConfig(dc, d.driveRegistry)
	if e != nil {
		return e
	}
	if factory.Init == nil {
		return nil
	}
	return factory.Init(ctx, data, config, d.createDriveUtils(name))
}

func (d *RootDrive) createDriveUtils(name string) driveutil.DriveUtils {
	return driveutil.DriveUtils{
		Data: d.driveDataStorage.GetDataStore(name),
		CreateCache: func(de driveutil.EntryDeserialize) driveutil.DriveCache {
			return d.driveCacheMgr.GetCacheStore(name, de)
		},
		Config: d.config,
	}
}
