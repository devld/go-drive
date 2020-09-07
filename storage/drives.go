package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common/types"
)

type DriveStorage struct {
	db *DB
}

func NewDriveStorage(db *DB) (*DriveStorage, error) {
	ds := DriveStorage{db: db}
	return &ds, nil
}

func (d *DriveStorage) GetDrives() ([]types.Drive, error) {
	var drivesConfig []types.Drive
	e := d.db.C().Find(&drivesConfig).Error
	return drivesConfig, e
}

func (d *DriveStorage) SaveDrives(drives []types.Drive) error {
	return d.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Delete(&types.Drive{}).Error; e != nil {
			return e
		}
		return tx.Create(&drives).Error
	})
}
