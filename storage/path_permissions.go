package storage

import (
	"go-drive/common/registry"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type PathPermissionDAO struct {
	db *DB
}

func NewPathPermissionDAO(db *DB, ch *registry.ComponentsHolder) *PathPermissionDAO {
	dao := &PathPermissionDAO{db}
	ch.Add("pathPermissionDAO", dao)
	return dao
}

func (p *PathPermissionDAO) GetAll() ([]types.PathPermission, error) {
	pps := make([]types.PathPermission, 0)
	if e := p.db.C().Find(&pps).Error; e != nil {
		return nil, e
	}
	return pps, nil
}

func (p *PathPermissionDAO) GetByPath(path string) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if e := p.db.C().Find(&r, "`path` = ?", path).Error; e != nil {
		return nil, e
	}
	return r, nil
}

func (p *PathPermissionDAO) SavePathPermissions(path string, permissions []types.PathPermission) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Delete(&types.PathPermission{}, "`path` = ?", path).Error; e != nil {
			return e
		}
		for _, p := range permissions {
			p.Path = &path
			if e := tx.Create(&p).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

func (p *PathPermissionDAO) DeleteByPath(path string) error {
	return p.db.C().Delete(&types.PathPermission{}, "`path` = ?", path).Error
}
