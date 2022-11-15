package script

import (
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"go-drive/common/utils"
)

func NewDrive(vm *VM, d types.IDrive) Drive {
	return Drive{vm, d}
}

func NewEntry(vm *VM, e types.IEntry) Entry {
	return Entry{vm, e}
}

type Drive struct {
	vm *VM
	d  types.IDrive
}

func GetDrive(v interface{}) types.IDrive {
	switch v := v.(type) {
	case Drive:
		return v.d
	}
	return nil
}

func (d Drive) Get(ctx interface{}, path string) Entry {
	entry, e := d.d.Get(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Save(ctx interface{}, path string, size int64, override bool, reader interface{}) Entry {
	entry, e := d.d.Save(GetTaskCtx(ctx), path, size, override, GetReader(reader))
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) MakeDir(ctx interface{}, path string) Entry {
	entry, e := d.d.MakeDir(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Copy(ctx interface{}, from interface{}, to string, override bool) Entry {
	entry, e := d.d.Copy(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Move(ctx interface{}, from interface{}, to string, override bool) Entry {
	entry, e := d.d.Move(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) List(ctx interface{}, path string) []Entry {
	entries, e := d.d.List(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return utils.MapArray(entries, func(t *types.IEntry) *Entry {
		r := NewEntry(d.vm, *t)
		return &r
	})
}

func (d Drive) Delete(ctx interface{}, path string) {
	if e := d.d.Delete(GetTaskCtx(ctx), path); e != nil {
		d.vm.ThrowError(e)
	}
}

type Entry struct {
	vm *VM
	e  types.IEntry
}

func GetEntry(v interface{}) types.IEntry {
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

func (e Entry) GetURL(ctx interface{}) *types.ContentURL {
	r, er := e.e.GetURL(GetContext(ctx))
	if er != nil {
		e.vm.ThrowError(er)
	}
	return r
}

func (e Entry) GetReader(ctx interface{}) ReadCloser {
	r, err := e.e.GetReader(GetContext(ctx))
	if err != nil {
		e.vm.ThrowError(err)
	}
	return NewReadCloser(e.vm, r)
}

func (e Entry) Unwrap() Entry {
	entry := drive_util.UnwrapIEntry(e.e)
	return NewEntry(e.vm, entry)
}

func (e Entry) Data() interface{} {
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
	return NewDrive(e.vm, e.e.Drive())
}
