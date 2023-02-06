package drive_util

import (
	"encoding/json"
	"go-drive/common/types"
	"time"
)

type DriveCacheManager interface {
	types.IDisposable
	GetCacheStore(ns string, deserialize EntryDeserialize) DriveCache
	EvictCacheStore(ns string) error
}

type EntrySerialize = func(types.IEntry) EntryCacheItem
type EntryDeserialize = func(EntryCacheItem) (types.IEntry, error)

type DriveCache interface {
	// PutEntries cache the entries, is ttl <= 0, the cache won't expire
	PutEntries(entries []types.IEntry, ttl time.Duration) error
	// PutEntry cache the entry, is ttl <= 0, the cache won't expire
	PutEntry(entry types.IEntry, ttl time.Duration) error
	// PutChildren cache the children of parentPath, is ttl <= 0, these caches won't expire
	PutChildren(parentPath string, entries []types.IEntry, ttl time.Duration) error
	// Evict evicts both the entry and it's children
	Evict(path string, descendants bool) error
	EvictAll() error
	GetEntry(path string) (types.IEntry, error)
	GetChildren(path string) ([]types.IEntry, error)

	GetEntryRaw(path string) (*EntryCacheItem, error)
	GetChildrenRaw(path string) ([]EntryCacheItem, error)
}

type CacheableEntry interface {
	EntryData() types.SM
}

type EntryCacheItem struct {
	ModTime int64           `json:"m"`
	Size    int64           `json:"s"`
	Path    string          `json:"p"`
	Type    types.EntryType `json:"t"`
	Data    types.SM        `json:"d"`
}

func SerializeEntry(e types.IEntry) EntryCacheItem {
	dat := EntryCacheItem{
		ModTime: e.ModTime(),
		Size:    e.Size(),
		Path:    e.Path(),
		Type:    e.Type(),
	}
	if ed, ok := e.(CacheableEntry); ok {
		dat.Data = ed.EntryData()
	}
	return dat
}

func DeserializeEntry(dat string) (*EntryCacheItem, error) {
	if dat == "" {
		return nil, nil
	}
	ec := EntryCacheItem{}
	e := json.Unmarshal([]byte(dat), &ec)
	return &ec, e
}

func DummyCache() DriveCache {
	return &dummyCache{}
}

type dummyCache struct {
}

func (d *dummyCache) PutEntries([]types.IEntry, time.Duration) error {
	return nil
}

func (d *dummyCache) PutEntry(types.IEntry, time.Duration) error {
	return nil
}

func (d *dummyCache) PutChildren(string, []types.IEntry, time.Duration) error {
	return nil
}

func (d *dummyCache) Evict(string, bool) error {
	return nil
}

func (d *dummyCache) EvictAll() error {
	return nil
}

func (d *dummyCache) GetEntry(string) (types.IEntry, error) {
	return nil, nil
}

func (d *dummyCache) GetChildren(string) ([]types.IEntry, error) {
	return nil, nil
}
func (d *dummyCache) GetEntryRaw(path string) (*EntryCacheItem, error) {
	return nil, nil
}

func (d *dummyCache) GetChildrenRaw(path string) ([]EntryCacheItem, error) {
	return nil, nil
}
