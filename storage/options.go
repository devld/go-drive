package storage

import (
	"errors"
	cmap "github.com/orcaman/concurrent-map"
	"go-drive/common/types"
	"gorm.io/gorm"
)

type OptionsDAO struct {
	db    *DB
	cache cmap.ConcurrentMap
}

func NewOptionsDAO(db *DB) *OptionsDAO {
	return &OptionsDAO{db: db, cache: cmap.New()}
}

func (d *OptionsDAO) Set(key string, value string) error {
	if key == "" {
		panic(errors.New("key is empty"))
	}
	o, e := d.get(key)
	if e != nil {
		return e
	}
	if o.Key == "" {
		return d.db.C().Create(&types.Option{Key: key, Value: value}).Error
	}
	e = d.db.C().Model(&types.Option{}).Where("key = ?", key).
		Update("value", value).Error
	if e == nil {
		d.cache.Remove(key)
	}
	return e
}

func (d *OptionsDAO) Get(key string) (string, error) {
	o, e := d.get(key)
	if e != nil {
		return "", e
	}
	return o.Value, nil
}

func (d *OptionsDAO) GetOrDefault(key, defVal string) (string, error) {
	o, e := d.get(key)
	if e != nil {
		return "", nil
	}
	if o.Key == "" {
		return defVal, nil
	}
	return o.Value, nil
}

func (d *OptionsDAO) get(key string) (types.Option, error) {
	o, ok := d.cache.Get(key)
	if ok {
		return o.(types.Option), nil
	}
	var option types.Option
	e := d.db.C().Where("key = ?", key).Take(&option).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = nil
	}
	if e == nil {
		d.cache.Set(key, option)
	}
	return option, e
}

func (d *OptionsDAO) Delete(key string) error {
	e := d.db.C().Delete(&types.Option{}, "key = ?", key).Error
	if e == nil {
		d.cache.Remove(key)
	}
	return e
}
