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

type RootDrive struct {
	vm *VM
	d  types.IRootDrive
}

func (r RootDrive) Get() Drive {
	return NewDrive(r.vm, r.d.Get())
}

func (r RootDrive) ReloadDrive(ctx Context, ignoreFailure bool) {
	if e := r.d.ReloadDrive(GetContext(ctx), ignoreFailure); e != nil {
		r.vm.ThrowError(e)
	}
}

func (r RootDrive) ReloadMounts() {
	if e := r.d.ReloadMounts(); e != nil {
		r.vm.ThrowError(e)
	}
}

type Drive struct {
	vm *VM
	d  types.IDrive
}

func (d Drive) Get(ctx Context, path string) Entry {
	entry, e := d.d.Get(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Save(ctx TaskCtx, path string, size int64, override bool, reader Reader) Entry {
	entry, e := d.d.Save(GetTaskCtx(ctx), path, size, override, GetReader(reader))
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) MakeDir(ctx Context, path string) Entry {
	entry, e := d.d.MakeDir(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Copy(ctx TaskCtx, from Entry, to string, override bool) Entry {
	entry, e := d.d.Copy(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) Move(ctx TaskCtx, from Entry, to string, override bool) Entry {
	entry, e := d.d.Move(GetTaskCtx(ctx), GetEntry(from), to, override)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return NewEntry(d.vm, entry)
}

func (d Drive) List(ctx Context, path string) []Entry {
	entries, e := d.d.List(GetContext(ctx), path)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return utils.MapArray(entries, func(t *types.IEntry) *Entry {
		r := NewEntry(d.vm, *t)
		return &r
	})
}

func (d Drive) Delete(ctx TaskCtx, path string) {
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

func (e Entry) GetURL(ctx Context) *types.ContentURL {
	r, er := e.e.GetURL(GetContext(ctx))
	if er != nil {
		e.vm.ThrowError(er)
	}
	return r
}

func (e Entry) GetReader(ctx Context) ReadCloser {
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
