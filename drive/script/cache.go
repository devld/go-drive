package script

import (
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
	"time"
)

type scriptDriveCache struct {
	vm *s.VM
	c  drive_util.DriveCache
}

func (sc *scriptDriveCache) PutEntries(entries []scriptEntryStruct, ttl time.Duration) {
	if e := sc.c.PutEntries(utils.ArrayMap(entries, structToEntry), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
}

func (sc *scriptDriveCache) PutEntry(entry scriptEntryStruct, ttl time.Duration) {
	if e := sc.c.PutEntry(*structToEntry(&entry), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
}

func (sc *scriptDriveCache) PutChildren(parentPath string, entries []scriptEntryStruct, ttl time.Duration) {
	if e := sc.c.PutChildren(parentPath, utils.ArrayMap(entries, structToEntry), ttl); e != nil {
		sc.vm.ThrowError(e)
	}
}

func (sc *scriptDriveCache) Evict(path string, descendants bool) {
	if e := sc.c.Evict(path, descendants); e != nil {
		sc.vm.ThrowError(e)
	}
}

func (sc *scriptDriveCache) EvictAll() {
	if e := sc.c.EvictAll(); e != nil {
		sc.vm.ThrowError(e)
	}
}

func (sc *scriptDriveCache) GetEntry(path string) *drive_util.EntryCacheItem {
	r, e := sc.c.GetEntryRaw(path)
	if e != nil {
		sc.vm.ThrowError(r)
	}
	return r
}

func (sc *scriptDriveCache) GetChildren(path string) interface{} {
	// return interface{}, because we need to return 'nil slice'
	a, e := sc.c.GetChildrenRaw(path)
	if e != nil {
		sc.vm.ThrowError(e)
	}
	if a == nil {
		return nil
	}
	return a
}

func structToEntry(e *scriptEntryStruct) *types.IEntry {
	var a types.IEntry = &scriptDriveEntry{s: e}
	return &a
}
