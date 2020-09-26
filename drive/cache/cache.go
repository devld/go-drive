package cache

import "go-drive/common/types"

type CacheableEntry interface {
	Path() string
	Serialize() string
}

type DriveCache interface {
	PutEntry(entry CacheableEntry) error
	PutChildren(path string, entries []CacheableEntry) error
	Evict(path string) error
	GetEntry(path string) (types.IEntry, error)
	GetChildren(path string) ([]types.IEntry, error)
}

func DummyCache() DriveCache {
	return &dummyCache{}
}

type dummyCache struct {
}

func (d *dummyCache) PutEntry(CacheableEntry) error {
	return nil
}

func (d *dummyCache) PutChildren(string, []CacheableEntry) error {
	return nil
}

func (d *dummyCache) Evict(string) error {
	return nil
}

func (d *dummyCache) GetEntry(string) (types.IEntry, error) {
	return nil, nil
}

func (d *dummyCache) GetChildren(string) ([]types.IEntry, error) {
	return nil, nil
}
