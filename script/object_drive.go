package script

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go-drive/common/driveutil"
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
	case *Drive:
		if v != nil {
			return v.d
		}
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

func (d Drive) ConsoleString() string {
	return formatGoInspect("Drive", nil, true)
}

type Entry struct {
	e types.IEntry
}

func GetEntry(v any) types.IEntry {
	switch v := v.(type) {
	case Entry:
		return v.e
	case *Entry:
		if v != nil {
			return v.e
		}
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

func (e Entry) Meta() entryMeta {
	return entryMeta{e.e.Meta()}
}

func (e Entry) ModTime() int64 {
	return e.e.ModTime()
}

func (e Entry) GetURL(ctx any) *contentURL {
	vm := GetVM(ctx)
	r, er := e.e.GetURL(GetContext(ctx))
	if er != nil {
		throwForVM(vm, er)
	}
	if r == nil {
		return nil
	}
	return &contentURL{*r}
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
	entry := driveutil.UnwrapIEntry(e.e)
	return NewEntry(entry)
}

func (e Entry) Data() any {
	cacheableEntry := driveutil.GetIEntry(e.e, func(entry types.IEntry) bool {
		_, ok := entry.(driveutil.CacheableEntry)
		return ok
	})
	if cacheableEntry == nil {
		return nil
	}
	dat := cacheableEntry.(driveutil.CacheableEntry).EntryData()
	if dat == nil {
		return nil
	}
	return dat
}

func (e Entry) Drive() Drive {
	return NewDrive(e.e.Drive())
}

func (e Entry) ConsoleString() string {
	if e.e == nil {
		return "Entry {}"
	}
	return formatGoInspect("Entry", []string{
		"Path: " + strconv.Quote(e.e.Path()),
		"Type: " + strconv.Quote(string(e.e.Type())),
		"Name: " + strconv.Quote(e.e.Name()),
		fmt.Sprintf("Size: %d", e.e.Size()),
	}, true)
}

func (e Entry) MarshalJSON() ([]byte, error) {
	if e.e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(struct {
		Path    string          `json:"Path"`
		Name    string          `json:"Name"`
		Type    types.EntryType `json:"Type"`
		Size    int64           `json:"Size"`
		ModTime int64           `json:"ModTime"`
		Meta    types.EntryMeta `json:"Meta"`
	}{
		Path:    e.e.Path(),
		Name:    e.e.Name(),
		Type:    e.e.Type(),
		Size:    e.e.Size(),
		ModTime: e.e.ModTime(),
		Meta:    e.e.Meta(),
	})
}

type entryMeta struct {
	types.EntryMeta
}

func (m entryMeta) ConsoleString() string {
	return formatGoInspect("EntryMeta", []string{
		fmt.Sprintf("Readable: %t", m.Readable),
		fmt.Sprintf("Writable: %t", m.Writable),
	}, true)
}

type contentURL struct {
	types.ContentURL
}

func (u contentURL) ConsoleString() string {
	return formatGoInspect("ContentURL", []string{
		"URL: " + strconv.Quote(u.URL),
	}, true)
}
