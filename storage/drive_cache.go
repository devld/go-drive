package storage

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"log"
	"time"
)

type DriveCacheStorage struct {
	db        *DB
	timerStop func()
}

func NewDriveCacheStorage(db *DB) (*DriveCacheStorage, error) {
	c := &DriveCacheStorage{db: db}
	c.timerStop = common.TimeTick(c.cleanExpired, 60*time.Second)
	return c, nil
}

func (d *DriveCacheStorage) cleanExpired() {
	now := time.Now().Unix()
	rows := d.db.C().Delete(&types.DriveCache{}, "expires_at > 0 AND expires_at < ?", now).RowsAffected
	if common.IsDebugOn() && rows > 0 {
		log.Printf("%d expired caches item cleaned", rows)
	}
}

func (d *DriveCacheStorage) Dispose() error {
	d.timerStop()
	return nil
}

func (d *DriveCacheStorage) GetCacheStore(ns string, serialize drive_util.EntrySerialize, deserialize drive_util.EntryDeserialize) drive_util.DriveCache {
	return &dbDriveNamespacedCacheStore{db: d.db, ns: ns, s: serialize, d: deserialize}
}

func (d *DriveCacheStorage) Remove(ns string) error {
	return d.db.C().Delete(&types.DriveCache{}, "drive = ?", ns).Error
}

type dbDriveNamespacedCacheStore struct {
	ns string
	db *DB
	s  drive_util.EntrySerialize
	d  drive_util.EntryDeserialize
}

func (d *dbDriveNamespacedCacheStore) put(db *gorm.DB, path string, cacheType uint8, val string, ttl time.Duration) error {
	expiresAt := int64(-1)
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).Unix()
	}
	depth := uint8(common.PathDepth(path))
	path = path + "/"
	c := 0
	e := db.Model(&types.DriveCache{}).Where(
		"drive = ? AND path = ? AND depth = ? AND type = ?",
		d.ns, path, depth, cacheType,
	).Count(&c).Error
	if e != nil {
		return e
	}
	item := &types.DriveCache{
		Drive: d.ns, Path: path, Depth: &depth, Type: cacheType,
		Value: val, ExpiresAt: expiresAt,
	}
	if c == 0 {
		return db.Create(&item).Error
	}
	return db.Save(&item).Error
}

func (d *dbDriveNamespacedCacheStore) get(path string, cacheType uint8) (string, error) {
	depth := common.PathDepth(path)
	path = path + "/"
	c := types.DriveCache{}
	e := d.db.C().Find(&c,
		"drive = ? AND path = ? AND depth = ? AND type = ?",
		d.ns, path, depth, cacheType,
	).Error
	if e != nil {
		return "", e
	}
	if c.ExpiresAt > 0 && c.ExpiresAt < time.Now().Unix() {
		return "", nil
	}
	return c.Value, nil
}

func pathLike(path string) string {
	if common.IsRootPath(path) {
		return path
	}
	return path + "/"
}

func (d *dbDriveNamespacedCacheStore) delete(db *gorm.DB, path string, descendants bool) error {
	depth := common.PathDepth(path)
	if descendants {
		return db.Delete(&types.DriveCache{},
			"drive = ? AND depth >= ? AND path LIKE (? || '%')", d.ns, depth, pathLike(path)).Error
	} else {
		return db.Delete(&types.DriveCache{},
			"drive = ? AND path = ? AND depth = ?", d.ns, path+"/", depth).Error
	}
}

func (d *dbDriveNamespacedCacheStore) PutEntries(entries []types.IEntry, ttl time.Duration) error {
	return d.db.C().Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			if err := d.put(tx, e.Path(), types.CacheEntry, d.s(e), ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *dbDriveNamespacedCacheStore) PutEntry(entry types.IEntry, ttl time.Duration) error {
	return d.put(d.db.C(), entry.Path(), types.CacheEntry, d.s(entry), ttl)
}

func (d *dbDriveNamespacedCacheStore) PutChildren(parentPath string, entries []types.IEntry, ttl time.Duration) error {
	e := d.PutEntries(entries, ttl)
	if e != nil {
		return e
	}
	childrenCache := make([]string, len(entries))
	for i, e := range entries {
		childrenCache[i] = e.Path()
	}
	dat, _ := json.Marshal(childrenCache)
	e = d.put(d.db.C(), parentPath, types.CacheChildren, string(dat), ttl)
	return e
}

func (d *dbDriveNamespacedCacheStore) Evict(path string, descendants bool) error {
	return d.delete(d.db.C(), path, descendants)
}

func (d *dbDriveNamespacedCacheStore) EvictAll() error {
	return d.db.C().Delete(&types.DriveCache{}, "drive = ?", d.ns).Error
}

func (d *dbDriveNamespacedCacheStore) GetEntry(path string) (types.IEntry, error) {
	v, e := d.get(path, types.CacheEntry)
	if e != nil {
		return nil, e
	}
	if v == "" {
		return nil, nil
	}
	return d.d(v)
}

func (d *dbDriveNamespacedCacheStore) GetChildren(path string) ([]types.IEntry, error) {
	depth := common.PathDepth(path)
	v, e := d.get(path, types.CacheChildren)
	if e != nil {
		return nil, e
	}
	if v == "" {
		return nil, nil
	}
	childrenPath := make([]string, 0)
	if e := json.Unmarshal([]byte(v), &childrenPath); e != nil {
		return nil, e
	}

	items := make([]types.DriveCache, 0)
	if e := d.db.C().Find(&items,
		"drive = ? AND type = ? AND depth = ? AND path LIKE (? || '%')",
		d.ns, types.CacheEntry, depth+1, pathLike(path),
	).Error; e != nil {
		return nil, e
	}
	itemsMap := make(map[string]types.DriveCache)
	for _, c := range items {
		itemsMap[c.Path] = c
	}

	entries := make([]types.IEntry, len(childrenPath))
	for i, path := range childrenPath {
		item, ok := itemsMap[path+"/"]
		if !ok || item.ExpiresAt > 0 && item.ExpiresAt < time.Now().Unix() {
			return nil, nil
		}
		entry, e := d.d(item.Value)
		if e != nil {
			return nil, e
		}
		entries[i] = entry
	}
	return entries, nil
}
