package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common"
	"go-drive/common/types"
	"sort"
)

type PathPermissionDAO struct {
	db *DB
}

func NewPathPermissionDAO(db *DB) *PathPermissionDAO {
	return &PathPermissionDAO{db}
}

func (p *PathPermissionDAO) GetAll() ([]types.PathPermission, error) {
	pps := make([]types.PathPermission, 0)
	if e := p.db.C().Find(&pps).Error; e != nil {
		return nil, e
	}
	return pps, nil
}

// GetByPaths query types.PathPermission by subjects and paths
func (p *PathPermissionDAO) GetByPaths(subjects, paths []string) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if len(subjects) == 0 || len(paths) == 0 {
		return r, nil
	}
	e := p.db.C().Find(&r, "subject IN (?) AND path IN (?)", subjects, paths).Error
	return r, e
}

func (p *PathPermissionDAO) GetChildrenByPath(subjects []string, path string, depth int8) ([]types.PathPermission, error) {
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

func (p *PathPermissionDAO) GetByPath(path string) ([]types.PathPermission, error) {
	r := make([]types.PathPermission, 0)
	if e := p.db.C().Find(&r, "path = ?", path).Error; e != nil {
		return nil, e
	}
	return r, nil
}

func (p *PathPermissionDAO) SavePathPermissions(path string, permissions []types.PathPermission) error {
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

func (p *PathPermissionDAO) DeleteByPath(path string) error {
	return p.db.C().Delete(&types.PathPermission{}, "path = ?", path).Error
}

func (p *PathPermissionDAO) ResolvePathPermission(subjects []string, path string) (types.Permission, error) {
	paths := common.PathParentTree(path)
	items, e := p.GetByPaths(subjects, paths)
	if e != nil {
		return types.PermissionEmpty, e
	}
	return ResolveAcceptedPermissions(items), nil
}

func (p *PathPermissionDAO) ResolvePathChildrenPermission(subjects []string, parentPath string) (map[string]types.Permission, error) {
	permissions, e := p.GetChildrenByPath(subjects, parentPath, int8(common.PathDepth(parentPath)+1))
	if e != nil {
		return nil, e
	}
	return makePermissionsMap(permissions), nil
}

func (p *PathPermissionDAO) ResolvePathAndDescendantPermission(subjects []string, parentPath string) (map[string]types.Permission, error) {
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
		result[k] = ResolveAcceptedPermissions(v)
	}
	return result
}

func pathPermissionLess(a, b types.PathPermission) bool {
	if a.Depth != b.Depth {
		return a.Depth > b.Depth
	}
	if a.IsForAnonymous() {
		if b.IsForAnonymous() {
			return a.Policy < b.Policy
		} else {
			return false
		}
	} else {
		if b.IsForAnonymous() {
			return true
		} else {
			if a.IsForUser() {
				if b.IsForUser() {
					return a.Policy < b.Policy
				} else {
					return true
				}
			} else {
				if b.IsForUser() {
					return false
				} else {
					return a.Policy < b.Policy
				}
			}
		}
	}
}

func ResolveAcceptedPermissions(items []types.PathPermission) types.Permission {
	sort.Slice(items, func(i, j int) bool { return pathPermissionLess(items[i], items[j]) })
	acceptedPermission := types.PermissionEmpty
	rejectedPermission := types.PermissionEmpty
	for _, item := range items {
		if item.IsAccept() {
			acceptedPermission |= item.Permission & ^rejectedPermission
		}
		if item.IsReject() {
			// acceptedPermission - ( item.Permission(reject) - acceptedPermission )
			acceptedPermission &= ^(item.Permission & (^acceptedPermission))
			rejectedPermission |= item.Permission
		}
	}
	return acceptedPermission
}
