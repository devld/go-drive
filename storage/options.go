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

func (d *OptionsDAO) Set(key, value string) error {
	return d.set(d.db.C(), key, value)
}

func (d *OptionsDAO) Sets(options map[string]string) error {
	return d.db.C().Transaction(func(tx *gorm.DB) error {
		for key, value := range options {
			if e := d.set(tx, key, value); e != nil {
				return e
			}
		}
		return nil
	})
}

func (d *OptionsDAO) set(db *gorm.DB, key, value string) error {
	if key == "" {
		panic(errors.New("key is empty"))
	}
	o, e := d.get(key, false)
	if e != nil {
		return e
	}
	if o.Key == "" {
		return db.Create(&types.Option{Key: key, Value: value}).Error
	}
	e = db.Model(&types.Option{}).Where("key = ?", key).
		Update("value", value).Error
	if e == nil {
		d.cache.Remove(key)
	}
	return e
}

func (d *OptionsDAO) Get(key string) (string, error) {
	o, e := d.get(key, true)
	if e != nil {
		return "", e
	}
	return o.Value, nil
}

func (d *OptionsDAO) Gets(keys ...string) (map[string]string, error) {
	options := make(map[string]string)
	for _, key := range keys {
		o, e := d.get(key, true)
		if e != nil {
			return nil, e
		}
		options[key] = o.Value
	}
	return options, nil
}

func (d *OptionsDAO) GetOrDefault(key, defVal string) (string, error) {
	o, e := d.get(key, true)
	if e != nil {
		return "", nil
	}
	if o.Key == "" {
		return defVal, nil
	}
	return o.Value, nil
}

func (d *OptionsDAO) get(key string, getCache bool) (types.Option, error) {
	if getCache {
		o, ok := d.cache.Get(key)
		if ok {
			return o.(types.Option), nil
		}
	}
	var option types.Option
	e := d.db.C().Where("key = ?", key).Take(&option).Error
	if e == nil {
		d.cache.Set(key, option)
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = nil
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
