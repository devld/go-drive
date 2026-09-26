package storage

import (
	"errors"
	"go-drive/common/driveutil"
	"go-drive/common/secretbox"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type DriveDataDAO struct {
	db      *DB
	secrets *secretbox.Box
}

func NewDriveDataDAO(db *DB, secrets *secretbox.Box) *DriveDataDAO {
	return &DriveDataDAO{db: db, secrets: secrets}
}

func (d *DriveDataDAO) GetDataStore(ns string) driveutil.DriveDataStore {
	return &dbDriveNamespacedDataStore{db: d.db, ns: ns, secrets: d.secrets}
}

func (d *DriveDataDAO) Remove(ns string) error {
	return d.db.C().Delete(&types.DriveData{}, "`drive` = ?", ns).Error
}

type dbDriveNamespacedDataStore struct {
	ns      string
	db      *DB
	secrets *secretbox.Box
}

func (d *dbDriveNamespacedDataStore) save(db *gorm.DB, key string, value string) error {
	e := db.Where("`drive` = ? AND `data_key` = ?", d.ns, key).Take(&types.DriveData{}).Error
	if e == nil {
		if value == "" {
			return db.Delete(&types.DriveData{}, "`drive` = ? AND `data_key` = ?", d.ns, key).Error
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

func (d *dbDriveNamespacedDataStore) SaveEncrypted(m types.SM) error {
	sealed := make(types.SM, len(m))
	for key, val := range m {
		encrypted, e := d.secrets.Encrypt(val)
		if e != nil {
			return e
		}
		sealed[key] = encrypted
	}
	return d.Save(sealed)
}

func (d *dbDriveNamespacedDataStore) Load(key string, keys ...string) (types.SM, error) {
	items := make([]types.DriveData, 0)
	query := d.db.C().Where("`drive` = ?", d.ns)
	loadKeys := append([]string{key}, keys...)
	query = query.Where("`data_key` IN (?)", loadKeys)
	e := query.Find(&items).Error
	if e != nil {
		return nil, e
	}
	r := make(types.SM, len(items))
	for _, i := range items {
		opened, e := d.secrets.Decrypt(i.Value)
		if e != nil {
			return nil, e
		}
		r[i.Key] = opened
	}
	return r, nil
}

func (d *dbDriveNamespacedDataStore) Clear() error {
	return d.db.C().Delete(&types.DriveData{}, "`drive` = ?", d.ns).Error
}
