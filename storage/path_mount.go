package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common/types"
)

type PathMountStorage struct {
	db *DB
}

func NewPathMountStorage(db *DB) (*PathMountStorage, error) {
	pd := PathMountStorage{db}
	return &pd, nil
}

func (p *PathMountStorage) GetMounts() ([]types.PathMount, error) {
	ms := make([]types.PathMount, 0)
	e := p.db.C().Find(&ms).Error
	return ms, e
}

func saveMount(db *gorm.DB, mount types.PathMount, override bool) error {
	e := db.Where("path = ? AND name = ?", mount.Path, mount.Name).Find(&types.PathMount{}).Error
	if e == nil {
		if override {
			// update
			return db.Save(&mount).Error
		}
		return nil
	}
	if !gorm.IsRecordNotFoundError(e) {
		return e
	}
	return db.Create(&mount).Error
}

func saveMounts(db *gorm.DB, mounts []types.PathMount, override bool) error {
	for _, m := range mounts {
		e := saveMount(db, m, override)
		if e != nil {
			return e
		}
	}
	return nil
}

func deleteMounts(db *gorm.DB, mounts []types.PathMount) error {
	for _, m := range mounts {
		e := db.Delete(&types.PathMount{}, "path = ? AND name = ?", m.Path, m.Name).Error
		if e != nil {
			return e
		}
	}
	return nil
}

func (p *PathMountStorage) SaveMounts(mounts []types.PathMount, override bool) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		return saveMounts(tx, mounts, override)
	})
}

func (p *PathMountStorage) DeleteMounts(mounts []types.PathMount) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		return deleteMounts(tx, mounts)
	})
}

func (p *PathMountStorage) DeleteAndSaveMounts(deletes []types.PathMount, newMounts []types.PathMount, override bool) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		if e := deleteMounts(tx, deletes); e != nil {
			return e
		}
		return saveMounts(tx, newMounts, override)
	})
}
