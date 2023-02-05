package drive_util

import (
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"
	"strings"
	"sync"
	"time"
)

func NewMemDriveCacheManager(cleanPeriod time.Duration) *MemDriveCacheManager {
	mm := &MemDriveCacheManager{
		cache: &memDriveCacheNode{
			key:      "", // root
			children: make(map[string]*memDriveCacheNode),
		},
	}
	if cleanPeriod > 0 {
		mm.timerStop = utils.TimeTick(mm.clean, cleanPeriod)
	}
	return mm
}

type MemDriveCacheManager struct {
	cache     *memDriveCacheNode
	timerStop func()
}

func (m *MemDriveCacheManager) GetCacheStore(ns string, deserialize EntryDeserialize) DriveCache {
	return &memDriveCache{mgr: m, ns: ns, deserialize: deserialize}
}

func (m *MemDriveCacheManager) findNode(path string) (*memDriveCacheNode, *memDriveCacheNode) {
	var node *memDriveCacheNode = m.cache
	var parent *memDriveCacheNode = nil
	if utils.IsRootPath(path) {
		return node, parent
	}
	for _, i := range strings.Split(utils.CleanPath(path), "/") {
		node.mu.RLock()
		t, ok := node.children[i]
		if !ok {
			node.mu.RUnlock()
			return nil, nil
		}
		parent = node
		node.mu.RUnlock()
		node = t
	}
	return node, parent
}

func (m *MemDriveCacheManager) createNodeIfNotExists(path string) *memDriveCacheNode {
	node := m.cache
	for _, i := range strings.Split(utils.CleanPath(path), "/") {
		node.mu.Lock()
		t, ok := node.children[i]
		if !ok {
			t = &memDriveCacheNode{key: i, children: make(map[string]*memDriveCacheNode)}
			node.children[i] = t
		}
		node.mu.Unlock()
		node = t
	}
	return node
}

func (m *MemDriveCacheManager) _cleanNode(node, parent *memDriveCacheNode, cleaned, total *int) {
	node.mu.Lock()
	defer node.mu.Unlock()
	for _, v := range utils.MapValues(node.children) {
		m._cleanNode(v, node, cleaned, total)
	}
	*total++
	if parent == nil {
		return
	}
	if len(node.children) == 0 &&
		(node.data == nil || (node.expiresAt != nil && node.expiresAt.Before(time.Now()))) {
		*cleaned++
		delete(parent.children, node.key)
	}
}

func (m *MemDriveCacheManager) clean() {
	cleaned, total := 0, 0
	m._cleanNode(m.cache, nil, &cleaned, &total)
	log.Printf("[MemDriveCacheManager] %d of %d cache items cleaned", cleaned, total)
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
	node, parent := m.mgr.findNode(m.addNs(path))
	if node == nil {
		return nil
	}
	node.mu.Lock()
	defer node.mu.Unlock()
	node.data = nil
	node.childrenNames = nil
	if descendants {
		node.children = make(map[string]*memDriveCacheNode)
		if parent != nil {
			delete(parent.children, node.key)
		}
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
	node, _ := m.mgr.findNode(m.addNs(path))
	if node == nil {
		return nil, nil
	}
	node.mu.RLock()
	defer node.mu.RUnlock()
	if node.childrenNames == nil {
		return nil, nil
	}
	data := make([]EntryCacheItem, 0, len(node.childrenNames))
	for _, key := range node.childrenNames {
		t, ok := node.children[key]
		if !ok || (t.expiresAt != nil && t.expiresAt.Before(time.Now())) || t.data == nil {
			return nil, nil
		}
		data = append(data, *t.data)
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
	node, _ := m.mgr.findNode(m.addNs(path))
	if node == nil {
		return nil, nil
	}
	node.mu.RLock()
	defer node.mu.RUnlock()
	if node.expiresAt != nil && node.expiresAt.Before(time.Now()) {
		return nil, nil
	}
	return node.data, nil
}

func (m *memDriveCache) PutChildren(parentPath string, entries []types.IEntry, ttl time.Duration) error {
	node := m.mgr.createNodeIfNotExists(m.addNs(parentPath))
	node.mu.Lock()
	defer node.mu.Unlock()
	childrenNames := make([]string, 0, len(entries))

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	for _, i := range entries {
		name := utils.PathBase(i.Path())
		childrenNames = append(childrenNames, name)
		cacheData := SerializeEntry(i)
		node.children[name] = &memDriveCacheNode{
			data:      &cacheData,
			key:       name,
			children:  make(map[string]*memDriveCacheNode),
			expiresAt: expiresAt,
		}
	}
	node.childrenNames = childrenNames
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
	node := m.mgr.createNodeIfNotExists(m.addNs(entry.Path()))
	node.mu.Lock()
	defer node.mu.Unlock()

	cacheData := SerializeEntry(entry)
	node.data = &cacheData

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}
	node.expiresAt = expiresAt
	return nil
}

func (m *memDriveCache) Dispose() error {
	return m.EvictAll()
}

type memDriveCacheNode struct {
	mu            sync.RWMutex
	key           string
	data          *EntryCacheItem
	childrenNames []string
	children      map[string]*memDriveCacheNode

	expiresAt *time.Time
}

var _ DriveCache = (*memDriveCache)(nil)
var _ types.IDisposable = (*MemDriveCacheManager)(nil)
