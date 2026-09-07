package script

import (
	"context"
	"errors"
	"sync"

	"go-drive/common/driveutil"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
)

type scriptDriveCache struct {
	vm *s.VM
	c  driveutil.DriveCache
}

func newScriptDriveCache(vm *s.VM, c driveutil.DriveCache) *scriptDriveCache {
	return &scriptDriveCache{vm: vm, c: c}
}

func (sc *scriptDriveCache) PutEntries(_ *s.VM, args s.Values) any {
	ttl := s.GetDuration(sc.vm, args.Get(1), "cache ttl requires a Duration or duration string")
	if e := sc.c.PutEntries(utils.ArrayMap(sc.entriesFromValue(args.Get(0)), structToEntry), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
	return nil
}

func (sc *scriptDriveCache) PutEntry(_ *s.VM, args s.Values) any {
	ttl := s.GetDuration(sc.vm, args.Get(1), "cache ttl requires a Duration or duration string")
	if e := sc.c.PutEntry(structToEntry(sc.entryFromValue(args.Get(0))), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
	return nil
}

func (sc *scriptDriveCache) PutChildren(_ *s.VM, args s.Values) any {
	ttl := s.GetDuration(sc.vm, args.Get(2), "cache ttl requires a Duration or duration string")
	if e := sc.c.PutChildren(args.Get(0).String(), utils.ArrayMap(sc.entriesFromValue(args.Get(1)), structToEntry), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
	return nil
}

func (sc *scriptDriveCache) Evict(_ *s.VM, args s.Values) any {
	if e := sc.c.Evict(args.Get(0).String(), args.Get(1).Bool()); e != nil {
		sc.vm.ThrowError(e)
	}
	return nil
}

func (sc *scriptDriveCache) EvictAll(_ *s.VM, _ s.Values) any {
	if e := sc.c.EvictAll(); e != nil {
		sc.vm.ThrowError(e)
	}
	return nil
}

func (sc *scriptDriveCache) GetEntry(_ *s.VM, args s.Values) any {
	r, e := sc.c.GetEntryRaw(args.Get(0).String())
	if e != nil {
		sc.vm.ThrowError(e)
	}
	if r == nil {
		return sc.vm.ToJSValue(nil)
	}
	return cacheItemJS(*r)
}

func (sc *scriptDriveCache) GetChildren(_ *s.VM, args s.Values) any {
	a, e := sc.c.GetChildrenRaw(args.Get(0).String())
	if e != nil {
		sc.vm.ThrowError(e)
	}
	if a == nil {
		return sc.vm.ToJSValue(nil)
	}
	items := make([]any, len(a))
	for i := range a {
		items[i] = cacheItemJS(a[i])
	}
	return items
}

func (sc *scriptDriveCache) entryFromValue(v *s.Value) *scriptEntryStruct {
	entry, e := valueToScriptEntryStruct(v)
	if e != nil {
		sc.vm.ThrowError(e)
	}
	return entry
}

func (sc *scriptDriveCache) entriesFromValue(v *s.Value) []scriptEntryStruct {
	arr := v.Array()
	if arr == nil {
		sc.vm.ThrowTypeError("cache entries requires an array")
	}
	out := make([]scriptEntryStruct, len(arr))
	for i, item := range arr {
		out[i] = *sc.entryFromValue(item)
	}
	return out
}

// cacheItemJS is the script-facing cache record. EntryCacheItem json tags are
// the compact store format, not the Drive script API.
func cacheItemJS(item driveutil.EntryCacheItem) map[string]any {
	return map[string]any{
		"modTime": item.ModTime,
		"size":    item.Size,
		"path":    item.Path,
		"type":    item.Type,
		"data":    item.Data,
		"meta":    item.Meta,
	}
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
	meta := types.EntryMeta{Readable: true, Writable: writable}
	if item.Meta != nil {
		meta = *item.Meta
		meta.Writable = writable
	}
	return &scriptEntryStruct{
		Path:    item.Path,
		Size:    item.Size,
		ModTime: item.ModTime,
		IsDir:   item.Type.IsDir(),
		Data:    item.Data,
		Meta:    meta,
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
	done       chan struct{}
	val        any
	err        error
	panicValue any
}

var errScriptLoadAborted = errors.New("script cache load aborted")

// flightGroup coalesces concurrent cache fills for the same key.
type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flightCall
}

func (g *flightGroup) do(ctx context.Context, key string, fn func() (any, error)) (any, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*flightCall)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.done:
		}
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		if c.panicValue != nil {
			panic(c.panicValue)
		}
		return c.val, c.err
	}
	c := &flightCall{done: make(chan struct{}), err: errScriptLoadAborted}
	g.m[key] = c
	g.mu.Unlock()

	defer func() {
		// Release coalesced callers on panic or Goexit as well as normal return.
		// A panic remains a panic for both the loader and its waiters.
		c.panicValue = recover()
		g.mu.Lock()
		delete(g.m, key)
		g.mu.Unlock()
		close(c.done)
		if c.panicValue != nil {
			panic(c.panicValue)
		}
	}()
	c.val, c.err = fn()
	return c.val, c.err
}
