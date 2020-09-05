package storage

import (
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
