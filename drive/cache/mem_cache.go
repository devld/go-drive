package cache

import (
	"encoding/json"
	"github.com/bluele/gcache"
	jsoniter "github.com/json-iterator/go"
	"go-drive/common/types"
	"time"
)

type EntryDeserialize = func(string) (types.IEntry, error)

type MemCache struct {
	c           gcache.Cache
	ttl         time.Duration
	deserialize EntryDeserialize
}

func NewMemCache(size int, ttl time.Duration, deserialize EntryDeserialize) *MemCache {
	return &MemCache{
		c:           gcache.New(size).LRU().Build(),
		ttl:         ttl,
		deserialize: deserialize,
	}
}

func (m *MemCache) PutEntry(entry CacheableEntry) error {
	key := "e" + entry.Path()
	if m.ttl <= 0 {
		return m.c.Set(key, entry.Serialize())
	}
	return m.c.SetWithExpire(key, entry.Serialize(), m.ttl)
}

func (m *MemCache) PutChildren(path string, entries []CacheableEntry) error {
	key := "c" + path
	data, e := childrenToString(entries)
	if e != nil {
		return e
	}
	if m.ttl <= 0 {
		return m.c.Set(key, data)
	}
	return m.c.SetWithExpire(key, data, m.ttl)
}

func (m *MemCache) Evict(path string) error {
	m.c.Remove("e" + path)
	m.c.Remove("c" + path)
	return nil
}

func (m *MemCache) GetEntry(path string) (types.IEntry, error) {
	get, e := m.c.Get("e" + path)
	if e == gcache.KeyNotFoundError {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return m.deserialize(get.(string))
}

func (m *MemCache) GetChildren(path string) ([]types.IEntry, error) {
	get, e := m.c.Get("c" + path)
	if e == gcache.KeyNotFoundError {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return m.stringToEntries(get.(string))
}

func (m *MemCache) stringToEntries(j string) ([]types.IEntry, error) {
	data := make([]string, 0)
	e := json.Unmarshal([]byte(j), &data)
	if e != nil {
		return nil, e
	}
	entries := make([]types.IEntry, len(data))
	for i, s := range data {
		entry, e := m.deserialize(s)
		if e != nil {
			return nil, e
		}
		entries[i] = entry
	}
	return entries, nil
}

func childrenToString(c []CacheableEntry) (string, error) {
	data := make([]string, len(c))
	for i, e := range c {
		data[i] = e.Serialize()
	}
	return jsoniter.MarshalToString(data)
}
