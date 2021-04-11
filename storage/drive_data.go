package storage

import (
	"errors"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"gorm.io/gorm"
)

type DriveDataDAO struct {
	db *DB
}

func NewDriveDataDAO(db *DB) *DriveDataDAO {
	return &DriveDataDAO{db}
}

func (d *DriveDataDAO) GetDataStore(ns string) drive_util.DriveDataStore {
	return &dbDriveNamespacedDataStore{db: d.db, ns: ns}
}

func (d *DriveDataDAO) Remove(ns string) error {
	return d.db.C().Delete(&types.DriveData{}, "drive = ?", ns).Error
}

type dbDriveNamespacedDataStore struct {
	ns string
	db *DB
}

func (d *dbDriveNamespacedDataStore) save(db *gorm.DB, key string, value string) error {
	e := db.Where("drive = ? AND data_key = ?", d.ns, key).Take(&types.DriveData{}).Error
	if e == nil {
		if value == "" {
			return db.Delete(&types.DriveData{}, "drive = ? AND data_key = ?", d.ns, key).Error
		}
		return db.Save(&types.DriveData{Drive: d.ns, Key: key, Value: value}).Error
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return e
	}
	if value == "" {
		return nil
	}
	return db.Create(&types.DriveData{Drive: d.ns, Key: key, Value: value}).Error
}

func (d *dbDriveNamespacedDataStore) Save(m types.SM) error {
	return d.db.C().Transaction(func(tx *gorm.DB) error {
		for key, val := range m {
			if e := d.save(tx, key, val); e != nil {
				return e
			}
		}
		return nil
	})
}

func (d *dbDriveNamespacedDataStore) Load(keys ...string) (types.SM, error) {
	items := make([]types.DriveData, 0)
	e := d.db.C().Where("drive = ? AND data_key IN (?)", d.ns, keys).Find(&items).Error
	if e != nil {
		return nil, e
	}
	r := make(types.SM, len(items))
	for _, i := range items {
		r[i.Key] = i.Value
	}
	return r, nil
}
