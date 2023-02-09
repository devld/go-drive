package script

import (
	"context"
	"errors"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
	"io"
	"sync"
)

func init() {
	drive_util.RegisterDynamicDrive("script", func(config common.Config) *drive_util.DriveFactoryConfig {
		scripts, _ := ListDriveScripts(config)
		scriptOptions := (utils.ArrayMap(scripts, func(t *DriveScript) types.FormItemOption {
			return types.FormItemOption{Value: t.Name, Name: t.DisplayName}
		}))

		return &drive_util.DriveFactoryConfig{
			DisplayName: t("name"),
			README:      t("readme"),
			ConfigForm: []types.FormItem{
				{Field: "script", Label: t("form.script.label"), Type: "select", Description: t("form.script.description"), Options: &scriptOptions, Required: true},
				{Field: "pool", Label: t("form.pool.label"), Type: "text", Description: t("form.pool.description")},
			},
			Factory: drive_util.DriveFactory{
				Create:     newScriptDrive,
				InitConfig: initConfig,
				Init:       init_,
			},
		}
	})
}

type ScriptDrive struct {
	baseVM *s.VM
	pool   *s.VMPool

	// data is the place where the data of the script instance is stored
	data map[string]*s.Value
	mu   sync.RWMutex
}

func (sd *ScriptDrive) setData(vm *s.VM, args s.Values) interface{} {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	data := args.Get(0)
	keys := data.Keys()
	for _, k := range keys {
		sd.data[k] = data.Get(k)
	}
	return nil
}

func (sd *ScriptDrive) getData(vm *s.VM, args s.Values) interface{} {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	key := args.Get(0).String()
	v, ok := sd.data[key]
	if !ok {
		vm.ThrowError(errors.New(key + " not found"))
	}
	return v.InternalValue()
}

func (sd *ScriptDrive) call(vm *s.VM, fn string, args ...interface{}) (*s.Value, error) {
	fn = "__drive_" + fn
	gotValue, e := vm.GetValue(fn)
	if e != nil {
		return nil, e
	}
	if gotValue.IsNil() {
		return nil, err.NewUnsupportedError()
	}
	return vm.Call(context.Background(), fn, args...)
}

func (sd *ScriptDrive) Meta(ctx context.Context) (types.DriveMeta, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return types.DriveMeta{}, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "meta", s.NewContext(vm, ctx))
	r := types.DriveMeta{}
	if e != nil {
		return r, nil
	}
	v.ParseInto(&r)
	return r, nil
}

func (sd *ScriptDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "get", s.NewContext(vm, ctx), path)
	if e != nil {
		return nil, e
	}
	return sd.valueToEntry(v), nil
}

func (sd *ScriptDrive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "save", s.NewTaskCtx(vm, ctx), path, size, override, s.NewReader(vm, reader))
	if e != nil {
		return nil, e
	}
	return sd.valueToEntry(v), nil
}

func (sd *ScriptDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "makeDir", s.NewContext(vm, ctx), path)
	if e != nil {
		return nil, e
	}
	return sd.valueToEntry(v), nil
}

func (sd *ScriptDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "copy", s.NewTaskCtx(vm, ctx), s.NewEntry(vm, from), to, override)
	if e != nil {
		return nil, e
	}
	return sd.valueToEntry(v), nil
}

func (sd *ScriptDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "move", s.NewTaskCtx(vm, ctx), s.NewEntry(vm, from), to, override)
	if e != nil {
		return nil, e
	}
	return sd.valueToEntry(v), nil
}

func (sd *ScriptDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "list", s.NewContext(vm, ctx), path)
	if e != nil {
		return nil, e
	}
	arr := v.Array()
	if arr == nil {
		panic("invalid value got from drive")
	}
	return utils.ArrayMap(arr, func(t **s.Value) types.IEntry { return sd.valueToEntry(*t) }), nil
}

func (sd *ScriptDrive) Delete(ctx types.TaskCtx, path string) error {
	vm, e := sd.pool.Get()
	if e != nil {
		return e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	_, e = sd.call(vm, "delete", s.NewTaskCtx(vm, ctx), path)
	return e
}

func (sd *ScriptDrive) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	vm, e := sd.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = sd.pool.Return(vm) }()
	v, e := sd.call(vm, "upload", s.NewContext(vm, ctx), path, size, override, config)
	if e != nil {
		return nil, e
	}
	r := types.DriveUploadConfig{}
	v.ParseInto(&r)
	return &r, nil
}

func (sd *ScriptDrive) valueToEntry(v *s.Value) *scriptDriveEntry {
	if v.IsNil() {
		panic(errors.New("nil entry value"))
	}
	return &scriptDriveEntry{
		d: sd,
		s: valueToScriptEntryStruct(v),
	}
}

func (sd *ScriptDrive) Dispose() error {
	_ = sd.baseVM.Dispose()
	_ = sd.pool.Dispose()
	return nil
}

func valueToScriptEntryStruct(v *s.Value) *scriptEntryStruct {
	meta := types.EntryMeta{Readable: true, Writable: true}

	metaV := v.Get("Meta")
	if !metaV.IsNil() {
		metaV.ParseInto(&meta)
	}

	return &scriptEntryStruct{
		Meta:    meta,
		IsDir:   v.Get("IsDir").Bool(),
		Path:    v.Get("Path").String(),
		Size:    v.Get("Size").Integer(),
		ModTime: v.Get("ModTime").Integer(),
		Data:    v.Get("Data").SM(),
	}
}

type scriptEntryStruct struct {
	Path    string
	Size    int64
	ModTime int64
	Meta    types.EntryMeta
	IsDir   bool
	Data    types.SM
}

type scriptDriveEntry struct {
	d *ScriptDrive
	s *scriptEntryStruct
}

// GetReader gets the reader of this entry
func (se *scriptDriveEntry) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	vm, e := se.d.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = se.d.pool.Return(vm) }()
	v, e := se.d.call(vm, "getReader", s.NewContext(vm, ctx), se.s, start, size)
	if e != nil {
		return nil, e
	}
	reader := s.GetReader(v.Raw())
	if reader == nil {
		panic("invalid returned value from getReader")
	}
	return wrapReader(reader), nil
}

func (se *scriptDriveEntry) GetURL(ctx context.Context) (*types.ContentURL, error) {
	vm, e := se.d.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = se.d.pool.Return(vm) }()
	v, e := se.d.call(vm, "getURL", s.NewContext(vm, ctx), se.s)
	if e != nil {
		return nil, e
	}
	r := types.ContentURL{}
	v.ParseInto(&r)
	return &r, nil
}

func (se *scriptDriveEntry) Name() string {
	return utils.PathBase(se.s.Path)
}

func (se *scriptDriveEntry) Size() int64 {
	return se.s.Size
}

func (se *scriptDriveEntry) ModTime() int64 {
	return se.s.ModTime
}

func (se *scriptDriveEntry) Path() string {
	return se.s.Path
}

func (se *scriptDriveEntry) Type() types.EntryType {
	if se.s.IsDir {
		return types.TypeDir
	} else {
		return types.TypeFile
	}
}

func (se *scriptDriveEntry) Meta() types.EntryMeta {
	return se.s.Meta
}

func (se *scriptDriveEntry) Drive() types.IDrive {
	return se.d
}

func (se *scriptDriveEntry) EntryData() types.SM {
	return se.s.Data
}

func (se *scriptDriveEntry) HasThumbnail() bool {
	vm, e := se.d.pool.Get()
	if e != nil {
		return false
	}
	defer func() { _ = se.d.pool.Return(vm) }()
	v, e := se.d.call(vm, "hasThumbnail", se.s)
	if e != nil {
		return false
	}
	return v.Bool()
}

func (se *scriptDriveEntry) Thumbnail(ctx context.Context) (types.IContentReader, error) {
	vm, e := se.d.pool.Get()
	if e != nil {
		return nil, e
	}
	defer func() { _ = se.d.pool.Return(vm) }()
	v, e := se.d.call(vm, "getThumbnail", s.NewContext(vm, ctx), se.s)
	if e != nil {
		return nil, e
	}
	if obj := s.GetReader(v.Raw()); obj != nil {
		// reader returned
		return wrapContentURL(obj), nil
	}
	r := types.ContentURL{}
	v.ParseInto(&r)
	return drive_util.NewURLContentReader(r.URL, r.Header, r.Proxy), nil
}
