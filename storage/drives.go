package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type DriveDAO struct {
	db *DB
}

func NewDriveDAO(db *DB, ch *registry.ComponentsHolder) *DriveDAO {
	dao := &DriveDAO{db: db}
	ch.Add("drivesDAO", dao)
	return dao
}

func (d *DriveDAO) GetDrives() ([]types.Drive, error) {
	var drivesConfig []types.Drive
	e := d.db.C().Find(&drivesConfig).Error
	return drivesConfig, e
}

func (d *DriveDAO) GetDrive(name string) (types.Drive, error) {
	var config types.Drive
	e := d.db.C().Where("`name` = ?", name).Take(&config).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return config, err.NewNotFoundError()
	}
	return config, e
}

func (d *DriveDAO) AddDrive(drive types.Drive) (types.Drive, error) {
	e := d.db.C().Where("`name` = ?", drive.Name).Take(&types.Drive{}).Error
	if e == nil {
		return types.Drive{},
			err.NewNotAllowedMessageError(i18n.T("storage.drives.drive_exists", drive.Name))
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return types.Drive{}, e
	}
	e = d.db.C().Create(&drive).Error
	return drive, e
}

func (d *DriveDAO) UpdateDrive(name string, drive types.Drive) error {
	drive.Name = name
	return d.db.C().Save(drive).Error
}

func (d *DriveDAO) DeleteDrive(name string) error {
	return d.db.C().Delete(&types.Drive{}, "`name` = ?", name).Error
}
