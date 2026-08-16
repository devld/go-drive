package script

import (
	"go-drive/common/driveutil"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
	"sync"
	"time"
)

type scriptDriveCache struct {
	c driveutil.DriveCache
}

func (sc *scriptDriveCache) PutEntries(entries []scriptEntryStruct, ttl int64) {
	if e := sc.c.PutEntries(utils.ArrayMap(entries, structToEntry), time.Duration(ttl)); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (sc *scriptDriveCache) PutEntry(entry scriptEntryStruct, ttl int64) {
	if e := sc.c.PutEntry(structToEntry(&entry), time.Duration(ttl)); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (sc *scriptDriveCache) PutChildren(parentPath string, entries []scriptEntryStruct, ttl int64) {
	if e := sc.c.PutChildren(parentPath, utils.ArrayMap(entries, structToEntry), time.Duration(ttl)); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (sc *scriptDriveCache) Evict(path string, descendants bool) {
	if e := sc.c.Evict(path, descendants); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (sc *scriptDriveCache) EvictAll() {
	if e := sc.c.EvictAll(); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (sc *scriptDriveCache) GetEntry(path string) *driveutil.EntryCacheItem {
	r, e := sc.c.GetEntryRaw(path)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	return r
}

func (sc *scriptDriveCache) GetChildren(path string) any {
	// return any, because we need to return 'nil slice'
	a, e := sc.c.GetChildrenRaw(path)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	if a == nil {
		return nil
	}
	return a
}

func structToEntry(e *scriptEntryStruct) types.IEntry {
	return &scriptDriveEntry{s: e}
}

func (sd *ScriptDrive) deserializeEntry(item driveutil.EntryCacheItem) (types.IEntry, error) {
	return &scriptDriveEntry{
		d: sd,
		s: cacheItemToStruct(item, sd.writable),
	}, nil
}

func cacheItemToStruct(item driveutil.EntryCacheItem, writable bool) *scriptEntryStruct {
	return &scriptEntryStruct{
		Path:    item.Path,
		Size:    item.Size,
		ModTime: item.ModTime,
		IsDir:   item.Type.IsDir(),
		Data:    item.Data,
		Meta:    types.EntryMeta{Readable: true, Writable: writable},
	}
}

// evictPathAndParent drops the path and its parent. Evict(parent, false)
// clears both the parent's GetEntry cache and its children listing, which
// is required when Save/MakeDir/Upload create a new child.
func (sd *ScriptDrive) evictPathAndParent(path string, descendants bool) {
	if sd.cache == nil {
		return
	}
	_ = sd.cache.Evict(path, descendants)
	_ = sd.cache.Evict(utils.PathParent(path), false)
}

type flightCall struct {
	wg  sync.WaitGroup
	val any
	err error
}

// flightGroup coalesces concurrent cache fills for the same key.
type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flightCall
}

func (g *flightGroup) do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*flightCall)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &flightCall{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	c.wg.Done()
	return c.val, c.err
}
