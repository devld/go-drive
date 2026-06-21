package script

import (
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"go-drive/common/utils"
)

func NewDrive(d types.IDrive) Drive {
	return Drive{d}
}

func NewEntry(e types.IEntry) Entry {
	return Entry{e}
}

type Drive struct {
	d types.IDrive
}

func GetDrive(v any) types.IDrive {
	switch v := v.(type) {
	case Drive:
		return v.d
	}
	return nil
}

func (d Drive) Get(ctx any, path string) Entry {
	vm := GetVM(ctx)
	entry, e := d.d.Get(GetContext(ctx), path)
	if e != nil {
		throwForVM(vm, e)
	}
	return NewEntry(entry)
}

func (d Drive) Save(ctx any, path string, size int64, override bool, reader any) Entry {
	vm := GetVM(ctx)
	entry, e := d.d.Save(GetTaskCtx(ctx), path, size, override, GetReader(reader))
	if e != nil {
		throwForVM(vm, e)
	}
	return NewEntry(entry)
}

func (d Drive) MakeDir(ctx any, path string) Entry {
	vm := GetVM(ctx)
	entry, e := d.d.MakeDir(GetContext(ctx), path)
	if e != nil {
		throwForVM(vm, e)
	}
	return NewEntry(entry)
}

func (d Drive) Copy(ctx any, from any, to string, override bool) Entry {
	vm := GetVM(ctx)
	entry, e := d.d.Copy(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		throwForVM(vm, e)
	}
	return NewEntry(entry)
}

func (d Drive) Move(ctx any, from any, to string, override bool) Entry {
	vm := GetVM(ctx)
	entry, e := d.d.Move(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		throwForVM(vm, e)
	}
	return NewEntry(entry)
}

func (d Drive) List(ctx any, path string) []Entry {
	vm := GetVM(ctx)
	entries, e := d.d.List(GetContext(ctx), path)
	if e != nil {
		throwForVM(vm, e)
	}
	return utils.ArrayMap(entries, func(t *types.IEntry) Entry { return NewEntry(*t) })
}

func (d Drive) Delete(ctx any, path string) {
	vm := GetVM(ctx)
	if e := d.d.Delete(GetTaskCtx(ctx), path); e != nil {
		throwForVM(vm, e)
	}
}

type Entry struct {
	e types.IEntry
}

func GetEntry(v any) types.IEntry {
	switch v := v.(type) {
	case Entry:
		return v.e
	}
	return nil
}

func (e Entry) Path() string {
	return e.e.Path()
}

func (e Entry) Name() string {
	return e.e.Name()
}

func (e Entry) Type() types.EntryType {
	return e.e.Type()
}

func (e Entry) Size() int64 {
	return e.e.Size()
}

func (e Entry) Meta() types.EntryMeta {
	return e.e.Meta()
}

func (e Entry) ModTime() int64 {
	return e.e.ModTime()
}

func (e Entry) GetURL(ctx any) *types.ContentURL {
	vm := GetVM(ctx)
	r, er := e.e.GetURL(GetContext(ctx))
	if er != nil {
		throwForVM(vm, er)
	}
	return r
}

func (e Entry) GetReader(ctx any, start, size int64) ReadCloser {
	vm := GetVM(ctx)
	r, err := e.e.GetReader(GetContext(ctx), start, size)
	if err != nil {
		throwForVM(vm, err)
	}
	return NewReadCloser(vm, r)
}

func (e Entry) Unwrap() Entry {
	entry := drive_util.UnwrapIEntry(e.e)
	return NewEntry(entry)
}

func (e Entry) Data() any {
	cacheableEntry := drive_util.GetIEntry(e.e, func(entry types.IEntry) bool {
		_, ok := entry.(drive_util.CacheableEntry)
		return ok
	})
	if cacheableEntry == nil {
		return nil
	}
	dat := cacheableEntry.(drive_util.CacheableEntry).EntryData()
	if dat == nil {
		return nil
	}
	return dat
}

func (e Entry) Drive() Drive {
	return NewDrive(e.e.Drive())
}
