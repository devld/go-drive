package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common/types"
)

type PathPermissionStorage struct {
	db *DB
}

func NewPathPermissionStorage(db *DB) (*PathPermissionStorage, error) {
	return &PathPermissionStorage{db}, nil
}

// GetByPaths query types.PathPermission by subjects and paths
func (p *PathPermissionStorage) GetByPaths(subjects, paths []string) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if len(subjects) == 0 || len(paths) == 0 {
		return r, nil
	}
	e := p.db.C().Find(&r, "subject IN (?) AND path IN (?)", subjects, paths).Error
	return r, e
}

func (p *PathPermissionStorage) GetChildrenByPath(subjects []string, path string, depth int8) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if len(subjects) == 0 {
		return r, nil
	}
	var e error = nil
	if depth == -1 {
		e = p.db.C().Find(&r, "path LIKE (? || '%') AND subject IN (?)", path, subjects).Error
	} else {
		e = p.db.C().Find(&r, "depth = ? AND path LIKE (? || '%') AND subject IN (?)", depth, path, subjects).Error
	}
	return r, e
}

func (p *PathPermissionStorage) save(item types.PathPermission, db *gorm.DB) error {
	old := types.PathPermission{}
	exists := true
	if e := db.First(&old, "path = ? AND subject = ?", item.Path, item.Subject).Error; e != nil {
		if !gorm.IsRecordNotFoundError(e) {
			return e
		}
		exists = false
	}
	var e error
	if exists {
		e = db.Save(item).Error
	} else {
		e = db.Create(&item).Error
	}
	return e
}

func (p *PathPermissionStorage) Save(items []types.PathPermission) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if e := p.save(item, tx); e != nil {
				return e
			}
		}
		return nil
	})
}
