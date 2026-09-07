package script

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go-drive/common/driveutil"
	"go-drive/common/types"
	"go-drive/common/utils"
)

var jsBuildEntriesTree = NativeFunction(func(vm *VM, args Values) any {
	entry := GetEntry(vm, args.Get(0).Raw(), "buildEntriesTree requires an Entry")
	byteProgress := args.Get(1).Bool()
	r, e := driveutil.BuildEntriesTree(runTaskCtx(vm.ExecutionContext()), entry, byteProgress)
	if e != nil {
		vm.ThrowError(e)
	}
	return convertEntryTreeNode(vm, r)
})

var jsFindEntries = NativeFunction(func(vm *VM, args Values) any {
	drive := GetDrive(vm, args.Get(0).Raw(), "findEntries requires a Drive")
	pattern := args.Get(1).String()
	byteProgress := args.Get(2).Bool()
	r, e := driveutil.FindEntries(runTaskCtx(vm.ExecutionContext()), drive, pattern, byteProgress)
	if e != nil {
		vm.ThrowError(e)
	}
	return utils.ArrayMap(r, func(t *types.IEntry) jsObjEntry { return newEntry(vm, *t) })
})

func newDrive(vm *VM, d types.IDrive) jsObjDrive {
	return jsObjDrive{ClassHost: NewClassHost(vm), d: d}
}

func newEntry(vm *VM, e types.IEntry) jsObjEntry {
	return jsObjEntry{ClassHost: NewClassHost(vm), e: e}
}

var (
	_ jsDrive = jsObjDrive{}
	_ jsEntry = jsObjEntry{}
)

var jsClassDrive = JSClass{
	Name:      "Drive",
	Handle:    jsObjDrive{},
	Construct: ConstructorHostOnly("Drive"),
	Accepts:   AcceptsNonNil[types.IDrive](),
	Wrap:      func(vm *VM, v any) any { return newDrive(vm, v.(types.IDrive)) },
	Methods: map[string]ClassMethod{
		"get": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.get").Get(args.Get(0).String())
		},
		"save": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.save").Save(
				args.Get(0).String(), args.Get(1).Integer(), args.Get(2).Bool(), args.Get(3),
			)
		},
		"makeDir": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.makeDir").MakeDir(args.Get(0).String())
		},
		"copy": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.copy").Copy(
				args.Get(0), args.Get(1).String(), args.Get(2).Bool(),
			)
		},
		"move": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.move").Move(
				args.Get(0), args.Get(1).String(), args.Get(2).Bool(),
			)
		},
		"list": func(vm *VM, this *Value, args Values) any {
			return This[jsObjDrive](vm, this, "Drive.list").List(args.Get(0).String())
		},
		"delete": func(vm *VM, this *Value, args Values) any {
			This[jsObjDrive](vm, this, "Drive.delete").Delete(args.Get(0).String())
			return nil
		},
	},
}

var jsClassEntry = JSClass{
	Name:      "Entry",
	Handle:    jsObjEntry{},
	Construct: ConstructorHostOnly("Entry"),
	Accepts:   AcceptsNonNil[types.IEntry](),
	Wrap:      func(vm *VM, v any) any { return newEntry(vm, v.(types.IEntry)) },
	Getters: map[string]ClassMethod{
		"path": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.path").Path()
		},
		"name": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.name").Name()
		},
		"type": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.type").Type()
		},
		"size": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.size").Size()
		},
		"meta": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.meta").Meta()
		},
		"modTime": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.modTime").ModTime()
		},
		"unwrap": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.unwrap").Unwrap()
		},
		"data": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.data").Data()
		},
		"drive": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.drive").Drive()
		},
	},
	Methods: map[string]ClassMethod{
		"getUrl": func(vm *VM, this *Value, args Values) any {
			return This[jsObjEntry](vm, this, "Entry.getUrl").GetURL()
		},
		"getReader": func(vm *VM, this *Value, args Values) any {
			return This[jsObjEntry](vm, this, "Entry.getReader").GetReader(
				args.Get(0).Integer(), args.Get(1).Integer(),
			)
		},
		"toJSON": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjEntry](vm, this, "Entry.toJSON").jsonShape()
		},
	},
}

type jsObjDrive struct {
	ClassHost
	d types.IDrive
}

func GetDrive(vm *VM, v any, required string) types.IDrive {
	if d, ok := HostAs[jsDrive](v); ok {
		if n := d.NativeDrive(); n != nil {
			return n
		}
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

type jsDrive interface {
	NativeDrive() types.IDrive
}

func (d jsObjDrive) NativeDrive() types.IDrive {
	return d.d
}

func (d jsObjDrive) Get(path string) jsObjEntry {
	entry, e := d.d.Get(d.VM().ExecutionContext(), path)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return newEntry(d.VM(), entry)
}

func (d jsObjDrive) Save(path string, size int64, override bool, reader *Value) jsObjEntry {
	ctx := d.VM().ExecutionContext()
	// The destination Drive owns progress reporting for Save.
	r := GetReader(d.VM(), reader, "Drive.save requires a Reader")
	entry, e := d.d.Save(runTaskCtx(ctx), path, size, override, r)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return newEntry(d.VM(), entry)
}

func (d jsObjDrive) MakeDir(path string) jsObjEntry {
	entry, e := d.d.MakeDir(d.VM().ExecutionContext(), path)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return newEntry(d.VM(), entry)
}

func (d jsObjDrive) Copy(from *Value, to string, override bool) jsObjEntry {
	entry, e := d.d.Copy(
		runTaskCtx(d.VM().ExecutionContext()),
		GetEntry(d.VM(), from, "Drive.copy requires an Entry"),
		to,
		override,
	)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return newEntry(d.VM(), entry)
}

func (d jsObjDrive) Move(from *Value, to string, override bool) jsObjEntry {
	entry, e := d.d.Move(
		runTaskCtx(d.VM().ExecutionContext()),
		GetEntry(d.VM(), from, "Drive.move requires an Entry"),
		to,
		override,
	)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return newEntry(d.VM(), entry)
}

func (d jsObjDrive) List(path string) []jsObjEntry {
	entries, e := d.d.List(d.VM().ExecutionContext(), path)
	if e != nil {
		d.VM().ThrowError(e)
	}
	return utils.ArrayMap(entries, func(t *types.IEntry) jsObjEntry { return newEntry(d.VM(), *t) })
}

func (d jsObjDrive) Delete(path string) {
	if e := d.d.Delete(runTaskCtx(d.VM().ExecutionContext()), path); e != nil {
		d.VM().ThrowError(e)
	}
}

func (d jsObjDrive) ConsoleString() string {
	return formatGoInspect("Drive", nil, true)
}

type jsObjEntry struct {
	ClassHost
	e types.IEntry
}

func GetEntry(vm *VM, v any, required string) types.IEntry {
	if e, ok := HostAs[jsEntry](v); ok {
		if n := e.NativeEntry(); n != nil {
			return n
		}
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

type jsEntry interface {
	NativeEntry() types.IEntry
}

func (e jsObjEntry) NativeEntry() types.IEntry {
	return e.e
}

func (e jsObjEntry) Path() string {
	return e.e.Path()
}

func (e jsObjEntry) Name() string {
	return e.e.Name()
}

func (e jsObjEntry) Type() types.EntryType {
	return e.e.Type()
}

func (e jsObjEntry) Size() int64 {
	return e.e.Size()
}

func (e jsObjEntry) Meta() any {
	meta := e.e.Meta()
	if e.VM() != nil {
		return e.VM().ToJSValue(&meta)
	}
	return meta
}

func (e jsObjEntry) ModTime() int64 {
	return e.e.ModTime()
}

func (e jsObjEntry) GetURL() any {
	r, er := e.e.GetURL(e.VM().ExecutionContext())
	if er != nil {
		e.VM().ThrowError(er)
	}
	if r == nil {
		return nil
	}
	return e.VM().ToJSValue(r)
}

func (e jsObjEntry) GetReader(start, size int64) *Value {
	r, err := e.e.GetReader(e.VM().ExecutionContext(), start, size)
	if err != nil {
		e.VM().ThrowError(err)
	}
	if r == nil {
		return nil
	}
	return e.VM().NewInstance("ReadCloser", r)
}

func (e jsObjEntry) Unwrap() jsObjEntry {
	return newEntry(e.VM(), driveutil.UnwrapIEntry(e.e))
}

func (e jsObjEntry) Data() any {
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
	if e.VM() != nil {
		return e.VM().ToJSValue(dat)
	}
	return dat
}

func (e jsObjEntry) Drive() any {
	d := e.e.Drive()
	if d == nil {
		return nil
	}
	return d
}

func (e jsObjEntry) ConsoleString() string {
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

func (e jsObjEntry) jsonShape() any {
	if e.e == nil {
		return nil
	}
	return struct {
		Path    string          `json:"path"`
		Name    string          `json:"name"`
		Type    types.EntryType `json:"type"`
		Size    int64           `json:"size"`
		ModTime int64           `json:"modTime"`
		Meta    struct {
			Readable      bool    `json:"readable"`
			Writable      bool    `json:"writable"`
			ThumbnailURL  string  `json:"thumbnailUrl"`
			SelfThumbnail bool    `json:"selfThumbnail"`
			Props         types.M `json:"props"`
		} `json:"meta"`
	}{
		Path:    e.e.Path(),
		Name:    e.e.Name(),
		Type:    e.e.Type(),
		Size:    e.e.Size(),
		ModTime: e.e.ModTime(),
		Meta: struct {
			Readable      bool    `json:"readable"`
			Writable      bool    `json:"writable"`
			ThumbnailURL  string  `json:"thumbnailUrl"`
			SelfThumbnail bool    `json:"selfThumbnail"`
			Props         types.M `json:"props"`
		}{
			Readable:      e.e.Meta().Readable,
			Writable:      e.e.Meta().Writable,
			ThumbnailURL:  e.e.Meta().ThumbnailURL,
			SelfThumbnail: e.e.Meta().SelfThumbnail,
			Props:         e.e.Meta().Props,
		},
	}
}

func (e jsObjEntry) MarshalJSON() ([]byte, error) {
	if e.e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.jsonShape())
}

// Trees are editable JavaScript data; Entry values remain host handles.
func convertEntryTreeNode(vm *VM, root driveutil.EntryTreeNode) *Value {
	return vm.ToPlainJSValue(entryTreeNodeData(vm, root))
}

func entryTreeNodeData(vm *VM, root driveutil.EntryTreeNode) map[string]any {
	children := make([]any, len(root.Children))
	for i, child := range root.Children {
		children[i] = entryTreeNodeData(vm, child)
	}
	return map[string]any{
		"entry":    vm.ToJSValue(newEntry(vm, root.Entry)),
		"children": children,
		"excluded": root.Excluded,
	}
}
