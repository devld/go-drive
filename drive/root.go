package drive

import (
	"encoding/json"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/drive/gdrive"
	"go-drive/drive/onedrive"
	"go-drive/storage"
	"log"
	"sync"
)

var driveFactories = []drive_util.DriveFactoryConfig{
	{
		Type: "fs", DisplayName: i18n.T("drive.fs.name"),
		README: i18n.T("drive.fs.readme"),
		ConfigForm: []types.FormItem{
			{Field: "path", Label: i18n.T("drive.fs.form.path.label"), Type: "text", Required: true, Description: i18n.T("drive.fs.form.path.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewFsDrive},
	},
	{
		Type: "s3", DisplayName: i18n.T("drive.s3.name"),
		README: i18n.T("drive.s3.readme"),
		ConfigForm: []types.FormItem{
			{Field: "id", Label: i18n.T("drive.s3.form.ak.label"), Type: "text", Required: true},
			{Field: "secret", Label: i18n.T("drive.s3.form.sk.label"), Type: "password", Required: true},
			{Field: "bucket", Label: i18n.T("drive.s3.form.bucket.label"), Type: "text", Required: true},
			{Field: "path_style", Label: i18n.T("drive.s3.form.path_style.label"), Type: "checkbox", Description: i18n.T("drive.s3.form.path_style.description")},
			{Field: "region", Label: i18n.T("drive.s3.form.region.label"), Type: "text"},
			{Field: "endpoint", Label: i18n.T("drive.s3.form.endpoint.label"), Type: "text", Description: i18n.T("drive.s3.form.endpoint.description")},
			{Field: "proxy_upload", Label: i18n.T("drive.s3.form.proxy_in.label"), Type: "checkbox", Description: i18n.T("drive.s3.form.proxy_in.description")},
			{Field: "proxy_download", Label: i18n.T("drive.s3.form.proxy_out.label"), Type: "checkbox", Description: i18n.T("drive.s3.form.proxy_out.description")},
			{Field: "cache_ttl", Label: i18n.T("drive.s3.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.s3.form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewS3Drive},
	},
	{
		Type: "webdav", DisplayName: i18n.T("drive.webdav.name"),
		README: i18n.T("drive.webdav.readme"),
		ConfigForm: []types.FormItem{
			{Field: "url", Label: i18n.T("drive.webdav.form.url.label"), Type: "text", Required: true, Description: i18n.T("drive.webdav.form.url.description")},
			{Field: "username", Label: i18n.T("drive.webdav.form.username.label"), Type: "text", Description: i18n.T("drive.webdav.form.username.description")},
			{Field: "password", Label: i18n.T("drive.webdav.form.password.label"), Type: "password"},
			{Field: "cache_ttl", Label: i18n.T("drive.webdav.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.webdav.form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewWebDAVDrive},
	},
	{
		Type: "onedrive", DisplayName: i18n.T("drive.onedrive.name"),
		README: i18n.T("drive.onedrive.readme"),
		ConfigForm: []types.FormItem{
			{Field: "client_id", Label: i18n.T("drive.onedrive.form.client_id.label"), Type: "text", Required: true},
			{Field: "client_secret", Label: i18n.T("drive.onedrive.form.client_secret.label"), Type: "password", Required: true},
			{Field: "proxy_upload", Label: i18n.T("drive.onedrive.form.proxy_in.label"), Type: "checkbox", Description: i18n.T("drive.onedrive.form.proxy_in.description")},
			{Field: "proxy_download", Label: i18n.T("drive.onedrive.form.proxy_out.label"), Type: "checkbox", Description: i18n.T("drive.onedrive.form.proxy_out.description")},
			{Field: "cache_ttl", Label: i18n.T("drive.onedrive.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.onedrive.form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: onedrive.NewOneDrive, InitConfig: onedrive.InitConfig, Init: onedrive.Init},
	},
	{
		Type: "gdrive", DisplayName: i18n.T("drive.gdrive.name"),
		README: i18n.T("drive.gdrive.readme"),
		ConfigForm: []types.FormItem{
			{Field: "client_id", Label: i18n.T("drive.gdrive.form.client_id.label"), Type: "text", Required: true},
			{Field: "client_secret", Label: i18n.T("drive.gdrive.form.client_secret.label"), Type: "password", Required: true},
			{Field: "cache_ttl", Label: i18n.T("drive.gdrive.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.gdrive.form.cache_ttl.description"), DefaultValue: "4h"},
		},
		Factory: drive_util.DriveFactory{Create: gdrive.NewGDrive, InitConfig: gdrive.InitConfig, Init: gdrive.Init},
	},
}

var driveFactoriesMap = make(map[string]drive_util.DriveFactoryConfig)

func init() {
	for _, f := range driveFactories {
		driveFactoriesMap[f.Type] = f
	}
}

func GetDrives() []drive_util.DriveFactoryConfig {
	r := make([]drive_util.DriveFactoryConfig, len(driveFactories))
	copy(r, driveFactories)
	for i, f := range r {
		form := make([]types.FormItem, len(f.ConfigForm))
		copy(form, f.ConfigForm)
		r[i].ConfigForm = form
	}
	return r
}

func GetDrive(driveType string) *drive_util.DriveFactoryConfig {
	f, ok := driveFactoriesMap[driveType]
	if ok {
		return &f
	}
	return nil
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
	f, ok := driveFactoriesMap[dc.Type]
	if !ok {
		return nil, nil, err.NewBadRequestError(i18n.T("drive.root.invalid_drive_type", dc.Type))
	}
	config := make(types.SM)
	e := json.Unmarshal([]byte(dc.Config), &config)
	if e != nil {
		return nil, nil, err.NewBadRequestError(i18n.T("drive.root.invalid_drive_config", dc.Name))
	}
	return &f.Factory, config, nil
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
	return initConfig, e
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
