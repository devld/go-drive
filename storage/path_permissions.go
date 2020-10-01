package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common"
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

func (p *PathPermissionStorage) GetByPath(path string) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if e := p.db.C().Find(&r, "path = ?", path).Error; e != nil {
		return nil, e
	}
	return r, nil
}

func (p *PathPermissionStorage) SavePathPermissions(path string, permissions []types.PathPermission) error {
	return p.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Delete(&types.PathPermission{}, "path = ?", path).Error; e != nil {
			return e
		}
		for _, p := range permissions {
			p.Path = &path
			p.Depth = uint8(common.PathDepth(path))
			if e := tx.Create(&p).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

func (p *PathPermissionStorage) ResolvePathPermission(subjects []string, path string) (types.Permission, error) {
	paths := common.PathParentTree(path)
	items, e := p.GetByPaths(subjects, paths)
	if e != nil {
		return types.PermissionEmpty, e
	}
	return common.ResolveAcceptedPermissions(items), nil
}

func (p *PathPermissionStorage) ResolvePathChildrenPermission(subjects []string, parentPath string) (map[string]types.Permission, error) {
	permissions, e := p.GetChildrenByPath(subjects, parentPath, int8(common.PathDepth(parentPath)+1))
	if e != nil {
		return nil, e
	}
	return makePermissionsMap(permissions), nil
}

func (p *PathPermissionStorage) ResolvePathAndDescendantPermission(subjects []string, parentPath string) (map[string]types.Permission, error) {
	permissions, e := p.GetChildrenByPath(subjects, parentPath, -1)
	if e != nil {
		return nil, e
	}
	return makePermissionsMap(permissions), nil
}

func makePermissionsMap(permissions []types.PathPermission) map[string]types.Permission {
	pMap := make(map[string][]types.PathPermission)
	for _, p := range permissions {
		ps, ok := pMap[*p.Path]
		if !ok {
			ps = make([]types.PathPermission, 0, 1)
		}
		ps = append(ps, p)
		pMap[*p.Path] = ps
	}
	result := make(map[string]types.Permission, len(pMap))
	for k, v := range pMap {
		result[k] = common.ResolveAcceptedPermissions(v)
	}
	return result
}
