package storage

import (
	"errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"sort"
	"time"

	"gorm.io/gorm"
)

type PathMetaDAO struct {
	db    *DB
	cache *utils.KVCache[*types.PathMeta]
}

func NewPathMetaDAO(db *DB, ch *registry.ComponentsHolder) *PathMetaDAO {
	dao := &PathMetaDAO{db: db, cache: utils.NewKVCache[*types.PathMeta](1 * time.Hour)}
	ch.Add("pathMetaDAO", dao)
	return dao
}

func (d *PathMetaDAO) getCached(path string) (*types.PathMeta, bool) {
	v, ok := d.cache.Get(path)
	if ok {
		if v == nil {
			return nil, true
		}
		vv := *v // make copy
		return &vv, true
	}
	return nil, false
}

func (d *PathMetaDAO) Get(path string) (*types.PathMeta, error) {
	if pm, ok := d.getCached(path); ok {
		return pm, nil
	}
	pm := types.PathMeta{}
	e := d.db.C().Where("path = ?", path).Take(&pm).Error
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			d.cache.Set(path, nil, 2*time.Hour)
			return nil, nil
		}
		return nil, e
	}
	d.cache.Set(path, &pm, 2*time.Hour)
	return &pm, nil
}

func (d *PathMetaDAO) Gets(path string) ([]*types.PathMeta, error) {
	missed := []string{}
	paths := utils.PathParentTree(path)
	result := make([]*types.PathMeta, 0, len(paths))
	for _, p := range paths {
		if pm, ok := d.getCached(p); ok {
			if pm != nil {
				result = append(result, pm)
			}
		} else {
			missed = append(missed, p)
		}
	}
	if len(missed) == 0 {
		return result, nil
	}
	items := make([]types.PathMeta, 0, len(missed))
	if e := d.db.C().Where("path in ?", missed).Find(&items).Error; e != nil {
		return nil, e
	}
	itemsMap := utils.ArrayKeyBy(items, func(m types.PathMeta, i int) string { return *m.Path })
	for _, missedPath := range missed {
		item, ok := itemsMap[missedPath]
		if ok {
			result = append(result, &item)
			d.cache.Set(missedPath, &item, 2*time.Hour)
		} else {
			d.cache.Set(missedPath, nil, 2*time.Hour)
		}
	}
	sort.Slice(result, func(i, j int) bool { return len(*result[j].Path) > len(*result[i].Path) })
	return result, nil
}

func (d *PathMetaDAO) GetMerged(path string) (*types.MergedPathMeta, error) {
	items, e := d.Gets(path)
	if e != nil || len(items) == 0 {
		return nil, e
	}
	r := types.MergedPathMeta{}
	for _, item := range items {
		same := *item.Path == path
		if r.Password.V == "" && (same || (item.Recursive&(1<<0)) != 0) {
			r.Password.V = item.Password
			r.Password.Path = *item.Path
		}
		if r.DefaultSort.V == "" && (same || (item.Recursive&(1<<1)) != 0) {
			r.DefaultSort.V = item.DefaultSort
			r.DefaultSort.Path = *item.Path
		}
		if r.DefaultMode.V == "" && (same || (item.Recursive&(1<<2)) != 0) {
			r.DefaultMode.V = item.DefaultMode
			r.DefaultMode.Path = *item.Path
		}
		if r.HiddenPattern.V == "" && (same || (item.Recursive&(1<<3)) != 0) {
			r.HiddenPattern.V = item.HiddenPattern
			r.HiddenPattern.Path = *item.Path
		}
	}
	return &r, nil
}

func (d *PathMetaDAO) Set(data types.PathMeta) error {
	updates := map[string]interface{}{
		"password":       data.Password,
		"default_sort":   data.DefaultSort,
		"default_mode":   data.DefaultMode,
		"hidden_pattern": data.HiddenPattern,
		"recursive":      data.Recursive,
	}
	e := d.db.C().Transaction(func(tx *gorm.DB) error {
		update := tx.Model(types.PathMeta{}).Where("path = ?", data.Path).Updates(updates)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 0 {
			return nil
		}
		return tx.Create(&data).Error
	})
	if e == nil {
		d.cache.Remove(*data.Path)
	}
	return e
}

func (d *PathMetaDAO) Delete(path string) error {
	e := d.db.C().Where("path = ?", path).Delete(&types.PathMeta{}).Error
	if e == nil {
		d.cache.Remove(path)
	}
	return e
}

func (d *PathMetaDAO) GetAll() ([]types.PathMeta, error) {
	result := make([]types.PathMeta, 0)
	if e := d.db.C().Order("path").Find(&result).Error; e != nil {
		return nil, e
	}
	return result, nil
}
