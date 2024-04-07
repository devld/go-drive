package drive_util

import (
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"time"
)

func NewMemDriveCacheManager(cleanPeriod time.Duration) *MemDriveCacheManager {
	mm := &MemDriveCacheManager{
		cache: utils.NewPathTreeNode[memCacheData](""),
	}
	if cleanPeriod > 0 {
		mm.timerStop = utils.TimeTick(mm.clean, cleanPeriod)
	}
	return mm
}

type MemDriveCacheManager struct {
	cache     *utils.PathTreeNode[memCacheData]
	timerStop func()
}

func (m *MemDriveCacheManager) GetCacheStore(ns string, deserialize EntryDeserialize) DriveCache {
	return &memDriveCache{mgr: m, ns: ns, deserialize: deserialize}
}

func (m *MemDriveCacheManager) EvictCacheStore(ns string) error {
	node, parent := m.cache.Get(ns)
	if node == nil {
		return nil
	}
	node.L().Lock()
	node.Data.EntryCacheItem = nil
	node.L().Unlock()
	if parent != nil {
		parent.RemoveChild(node.Key())
	}
	return nil
}

func (m *MemDriveCacheManager) _cleanNode(node, parent *utils.PathTreeNode[memCacheData], cleaned, total *int) {
	children := node.Children()
	for _, v := range children {
		m._cleanNode(v, node, cleaned, total)
	}
	*total++
	if parent == nil || len(children) != 0 {
		return
	}
	node.L().RLock()
	if node.Data.isInvalid() {
		*cleaned++
		parent.RemoveChild(node.Key())
	}
	node.L().RUnlock()
}

func (m *MemDriveCacheManager) clean() {
	cleaned, total := 0, 0
	t := time.Now().UnixMicro()
	m._cleanNode(m.cache, nil, &cleaned, &total)
	log.Printf("[MemDriveCacheManager] %d of %d cache items cleaned(%fms)",
		cleaned, total, float64(time.Now().UnixMicro()-t)/1000)
}

func (m *MemDriveCacheManager) Dispose() error {
	if m.timerStop != nil {
		m.timerStop()
	}
	return nil
}

type memDriveCache struct {
	mgr         *MemDriveCacheManager
	ns          string
	deserialize EntryDeserialize
}

func (m *memDriveCache) addNs(path string) string {
	if utils.IsRootPath(path) {
		return m.ns
	}
	return m.ns + "/" + path
}

func (m *memDriveCache) Evict(path string, descendants bool) error {
	node, parent := m.mgr.cache.Get(m.addNs(path))
	if node == nil {
		return nil
	}
	node.L().Lock()
	node.Data.EntryCacheItem = nil
	node.Data.childrenNames = nil
	node.L().Unlock()

	if descendants {
		parent.RemoveChild(node.Key())
	}
	return nil
}

func (m *memDriveCache) EvictAll() error {
	return m.Evict("", true)
}

func (m *memDriveCache) GetChildren(path string) ([]types.IEntry, error) {
	raw, e := m.GetChildrenRaw(path)
	if raw == nil {
		return nil, e
	}
	return utils.ArrayMapWithError(raw, func(t *EntryCacheItem) (types.IEntry, error) {
		return m.deserialize(*t)
	})
}

func (m *memDriveCache) GetChildrenRaw(path string) ([]EntryCacheItem, error) {
	node, _ := m.mgr.cache.Get(m.addNs(path))
	if node == nil {
		return nil, nil
	}
	node.L().RLock()
	if node.Data.childrenNames == nil {
		node.L().RUnlock()
		return nil, nil
	}
	childrenNames := node.Data.childrenNames
	node.L().RUnlock()

	data := make([]EntryCacheItem, 0, len(childrenNames))
	for _, key := range node.Data.childrenNames {
		t, _ := node.Get(key)
		if t == nil {
			return nil, nil
		}
		t.L().RLock()
		if t.Data.isInvalid() {
			t.L().RUnlock()
			return nil, nil
		}
		data = append(data, *t.Data.EntryCacheItem)
		t.L().RUnlock()
	}
	return data, nil
}

func (m *memDriveCache) GetEntry(path string) (types.IEntry, error) {
	raw, e := m.GetEntryRaw(path)
	if raw == nil {
		return nil, e
	}
	return m.deserialize(*raw)
}

func (m *memDriveCache) GetEntryRaw(path string) (*EntryCacheItem, error) {
	node, _ := m.mgr.cache.Get(m.addNs(path))
	if node == nil {
		return nil, nil
	}
	node.L().RLock()
	defer node.L().RUnlock()
	if node.Data.isInvalid() {
		return nil, nil
	}
	return node.Data.EntryCacheItem, nil
}

func (m *memDriveCache) PutChildren(parentPath string, entries []types.IEntry, ttl time.Duration) error {
	node := m.mgr.cache.Create(m.addNs(parentPath))

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	childrenNames := make([]string, 0, len(entries))
	children := make(map[string]memCacheData, len(entries))

	for _, i := range entries {
		name := utils.PathBase(i.Path())
		childrenNames = append(childrenNames, name)
		cacheData := SerializeEntry(i)
		children[name] = memCacheData{
			EntryCacheItem: &cacheData,
			expiresAt:      expiresAt,
		}
	}

	node.AddChildren(children)
	node.L().Lock()
	node.Data.childrenNames = childrenNames
	node.L().Unlock()
	return nil
}

func (m *memDriveCache) PutEntries(entries []types.IEntry, ttl time.Duration) error {
	for _, i := range entries {
		if e := m.PutEntry(i, ttl); e != nil {
			return e
		}
	}
	return nil
}

func (m *memDriveCache) PutEntry(entry types.IEntry, ttl time.Duration) error {
	node := m.mgr.cache.Create(m.addNs(entry.Path()))

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	node.L().Lock()
	defer node.L().Unlock()

	cacheData := SerializeEntry(entry)
	node.Data.EntryCacheItem = &cacheData
	node.Data.expiresAt = expiresAt
	return nil
}

func (m *memDriveCache) Dispose() error {
	return m.EvictAll()
}

type memCacheData struct {
	*EntryCacheItem
	childrenNames []string
	expiresAt     *time.Time
}

func (mcd memCacheData) isExpired() bool {
	return (mcd.expiresAt != nil && mcd.expiresAt.Before(time.Now()))
}

func (mcd memCacheData) isInvalid() bool {
	return mcd.EntryCacheItem == nil || mcd.isExpired()
}

var _ DriveCache = (*memDriveCache)(nil)
var _ types.IDisposable = (*MemDriveCacheManager)(nil)
