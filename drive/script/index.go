package script

import (
	"context"
	"errors"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
	"io"
	"maps"
	"net/http"
	"sync"
	"time"
)

var _ types.IDrive = (*ScriptDrive)(nil)

type scriptDriveHas struct {
	meta, save, makeDir, copy, move, delete, upload bool
	getReader, getURL, getThumbnail, onInterval     bool
}

type ScriptDrive struct {
	name     string
	pool     *s.VMPool
	cache    driveutil.DriveCache
	cacheTTL time.Duration
	writable bool
	has      scriptDriveHas
	load     flightGroup

	// data is the place where the data of the script instance is stored
	data map[string]any
	mu   sync.RWMutex

	// oauth shares *driveutil.OAuthHolder across pooled VMs. Each VM still
	// gets its own JS wrapper.
	oauth oauthHolderShare

	intervals      []*driveInterval
	intervalCtx    context.Context
	intervalCancel context.CancelFunc
	intervalWG     sync.WaitGroup
}

func (sd *ScriptDrive) jsFunSetData(vm *s.VM, args s.Values) any {
	data := args.Get(0)
	keys := data.Keys()
	encodedValues := make(map[string]any, len(keys))
	for _, k := range keys {
		cloned, e := vm.FromJSValue(data.Get(k))
		if e != nil {
			vm.ThrowTypeError("shared state must be JSON serializable: " + e.Error())
		}
		encodedValues[k] = cloned
	}

	sd.mu.Lock()
	defer sd.mu.Unlock()
	maps.Copy(sd.data, encodedValues)
	return nil
}

func (sd *ScriptDrive) jsFunGetData(vm *s.VM, args s.Values) any {
	key := args.Get(0).String()
	sd.mu.RLock()
	value, ok := sd.data[key]
	sd.mu.RUnlock()
	if !ok {
		vm.ThrowError(errors.New(key + " not found"))
	}
	// FromJSValue JSON-decodes null as a Go nil interface. NativeFunction
	// treats a nil return as JavaScript undefined, so convert here to keep
	// JSON null as JS null.
	return vm.ToJSValue(value)
}

func (sd *ScriptDrive) call(ctx context.Context, vm *s.VM, fn string, args ...any) (*s.Value, error) {
	fn = "__drive_" + fn
	started := time.Now()
	value, e := vm.Call(ctx, fn, args...)
	if e != nil {
		if errors.Is(e, s.ErrFunctionUndefined) {
			return nil, err.NewUnsupportedError()
		}
		logging.For("scr-drv").Errorf("script call failed script=%s function=%s duration=%s: %s",
			logging.Sanitize(sd.name), fn, time.Since(started), s.FormatError(e))
		return nil, mapScriptDriveError(e)
	} else if elapsed := time.Since(started); elapsed >= time.Second {
		logging.For("scr-drv").Debugf("script call slow script=%s function=%s duration=%s",
			logging.Sanitize(sd.name), fn, elapsed)
	}
	return value, nil
}

func (sd *ScriptDrive) withVM(ctx context.Context, fn func(vm *s.VM) error) error {
	vm, e := sd.pool.Get(ctx)
	if e != nil {
		logging.For("scr-drv").Debugf("script VM unavailable script=%s: %v", logging.Sanitize(sd.name), e)
		return e
	}
	defer func() {
		if e := sd.pool.Return(context.Background(), vm); e != nil {
			logging.For("scr-drv").Warnf("script VM return failed script=%s: %v", logging.Sanitize(sd.name), e)
		}
	}()
	return vm.Do(ctx, func() error { return fn(vm) })
}

func mapScriptDriveError(e error) error {
	if e == nil {
		return nil
	}
	if _, ok := errors.AsType[err.Error](e); ok {
		return e
	}
	if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
		return e
	}
	return err.NewRemoteApiError(http.StatusInternalServerError, e.Error())
}

func invalidScriptResult(message string) error {
	return err.NewRemoteApiError(http.StatusInternalServerError, message)
}

func (sd *ScriptDrive) rootEntry() *scriptDriveEntry {
	return &scriptDriveEntry{
		d: sd,
		s: &scriptEntryStruct{
			Path:    "",
			IsDir:   true,
			Size:    -1,
			ModTime: -1,
			Meta:    types.EntryMeta{Readable: true, Writable: sd.writable},
		},
	}
}

func (sd *ScriptDrive) Meta(ctx context.Context) (types.DriveMeta, error) {
	if !sd.has.meta {
		return types.DriveMeta{Writable: sd.writable}, nil
	}
	var r types.DriveMeta
	e := sd.withVM(ctx, func(vm *s.VM) error {
		v, e := sd.call(ctx, vm, "meta")
		if e != nil {
			return e
		}
		if e := v.ParseInto(&r); e != nil {
			return mapScriptDriveError(e)
		}
		return nil
	})
	if e != nil {
		return types.DriveMeta{}, e
	}
	return r, nil
}

func (sd *ScriptDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return sd.rootEntry(), nil
	}
	if sd.cacheTTL > 0 {
		cached, e := sd.cache.GetEntry(path)
		if e != nil {
			return nil, e
		}
		if cached != nil {
			return cached, nil
		}
	}
	v, e := sd.load.do(ctx, "get:"+path, func() (any, error) {
		if sd.cacheTTL > 0 {
			cached, e := sd.cache.GetEntry(path)
			if e != nil {
				return nil, e
			}
			if cached != nil {
				return cached, nil
			}
		}
		var entry *scriptDriveEntry
		e := sd.withVM(ctx, func(vm *s.VM) error {
			v, e := sd.call(ctx, vm, "get", path)
			if e != nil {
				return e
			}
			entry, e = sd.valueToEntry(v)
			return e
		})
		if e != nil {
			return nil, e
		}
		if sd.cacheTTL > 0 {
			_ = sd.cache.PutEntry(entry, sd.cacheTTL)
		}
		return entry, nil
	})
	if e != nil {
		return nil, e
	}
	return v.(types.IEntry), nil
}

func (sd *ScriptDrive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	if !sd.has.save {
		return nil, err.NewUnsupportedError()
	}
	e := sd.withVM(ctx, func(vm *s.VM) error {
		ctx.Total(size, true)
		// Save borrows the stream; the caller retains ownership of closing it.
		borrowedReader := vm.NewInstance("Reader", reader)
		_, e := sd.call(ctx, vm, "save", path, size, override, borrowedReader, jsOnProgress(ctx))
		return e
	})
	if e != nil {
		return nil, e
	}
	sd.evictPathAndParent(path, false)
	return sd.Get(ctx, path)
}

func (sd *ScriptDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	if !sd.has.makeDir {
		return nil, err.NewUnsupportedError()
	}
	e := sd.withVM(ctx, func(vm *s.VM) error {
		_, e := sd.call(ctx, vm, "makeDir", path)
		return e
	})
	if e != nil {
		return nil, e
	}
	sd.evictPathAndParent(path, false)
	return sd.Get(ctx, path)
}

func (sd *ScriptDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	src, e := sd.ownedEntry(from)
	if e != nil {
		return nil, e
	}
	if !sd.has.copy {
		return nil, err.NewUnsupportedError()
	}
	e = sd.withVM(ctx, func(vm *s.VM) error {
		_, e := sd.call(ctx, vm, "copy", src, to, override, jsOnProgress(ctx))
		return e
	})
	if e != nil {
		return nil, e
	}
	sd.evictPathAndParent(to, true)
	return sd.Get(ctx, to)
}

func (sd *ScriptDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	src, e := sd.ownedEntry(from)
	if e != nil {
		return nil, e
	}
	if !sd.has.move {
		return nil, err.NewUnsupportedError()
	}
	e = sd.withVM(ctx, func(vm *s.VM) error {
		_, e := sd.call(ctx, vm, "move", src, to, override, jsOnProgress(ctx))
		return e
	})
	if e != nil {
		return nil, e
	}
	sd.evictPathAndParent(to, true)
	sd.evictPathAndParent(src.Path, true)
	return sd.Get(ctx, to)
}

func (sd *ScriptDrive) ownedEntry(from types.IEntry) (*scriptEntryStruct, error) {
	owned := driveutil.GetSelfEntry(sd, from)
	if owned == nil {
		return nil, err.NewUnsupportedError()
	}
	se, ok := owned.(*scriptDriveEntry)
	if !ok || se.s == nil {
		return nil, err.NewUnsupportedError()
	}
	return se.s, nil
}

func (sd *ScriptDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if sd.cacheTTL > 0 {
		cached, e := sd.cache.GetChildren(path)
		if e != nil {
			return nil, e
		}
		if cached != nil {
			return cached, nil
		}
	}
	v, e := sd.load.do(ctx, "list:"+path, func() (any, error) {
		if sd.cacheTTL > 0 {
			cached, e := sd.cache.GetChildren(path)
			if e != nil {
				return nil, e
			}
			if cached != nil {
				return cached, nil
			}
		}
		var entries []types.IEntry
		e := sd.withVM(ctx, func(vm *s.VM) error {
			v, e := sd.call(ctx, vm, "list", path)
			if e != nil {
				return e
			}
			arr := v.Array()
			if arr == nil {
				return invalidScriptResult("list must return an array")
			}
			entries = make([]types.IEntry, len(arr))
			for i, item := range arr {
				entry, e := sd.valueToEntry(item)
				if e != nil {
					return e
				}
				entries[i] = entry
			}
			return nil
		})
		if e != nil {
			return nil, e
		}
		if sd.cacheTTL > 0 {
			_ = sd.cache.PutChildren(path, entries, sd.cacheTTL)
		}
		return entries, nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]types.IEntry), nil
}

func (sd *ScriptDrive) Delete(ctx types.TaskCtx, path string) error {
	if !sd.has.delete {
		return err.NewUnsupportedError()
	}
	e := sd.withVM(ctx, func(vm *s.VM) error {
		_, e := sd.call(ctx, vm, "delete", path, jsOnProgress(ctx))
		return e
	})
	if e != nil {
		return e
	}
	sd.evictPathAndParent(path, true)
	return nil
}

func (sd *ScriptDrive) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	completed := config["action"] == "Completed"
	if completed {
		var result *types.DriveUploadConfig
		if sd.has.upload {
			e := sd.withVM(ctx, func(vm *s.VM) error {
				v, e := sd.call(ctx, vm, "upload", path, size, override, config)
				if e != nil {
					return e
				}
				result, e = parseUploadConfig(v)
				return e
			})
			if e != nil {
				return nil, e
			}
		}
		sd.evictPathAndParent(path, false)
		return result, nil
	}
	if !sd.has.upload {
		return types.UseLocalProvider(size), nil
	}
	var result *types.DriveUploadConfig
	e := sd.withVM(ctx, func(vm *s.VM) error {
		v, e := sd.call(ctx, vm, "upload", path, size, override, config)
		if e != nil {
			return e
		}
		result, e = parseUploadConfig(v)
		return e
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

func parseUploadConfig(v *s.Value) (*types.DriveUploadConfig, error) {
	if v == nil || v.IsNil() {
		return nil, nil
	}
	r := types.DriveUploadConfig{}
	if e := v.ParseInto(&r); e != nil {
		return nil, mapScriptDriveError(e)
	}
	return &r, nil
}

func (sd *ScriptDrive) valueToEntry(v *s.Value) (*scriptDriveEntry, error) {
	if v == nil || v.IsNil() || !v.IsObject() {
		return nil, invalidScriptResult("invalid entry value")
	}
	entry, e := valueToScriptEntryStruct(v)
	if e != nil {
		return nil, e
	}
	return &scriptDriveEntry{d: sd, s: entry}, nil
}

func (sd *ScriptDrive) Dispose() error {
	if sd.intervalCancel != nil {
		sd.intervalCancel()
		sd.intervalWG.Wait()
	}
	if sd.pool != nil {
		_ = sd.pool.Dispose()
	}
	return nil
}

func valueToScriptEntryStruct(v *s.Value) (*scriptEntryStruct, error) {
	pathV := v.Get("path")
	if pathV.IsNil() {
		return nil, invalidScriptResult("entry path is required")
	}
	isDirV := v.Get("isDir")
	if isDirV.IsNil() {
		return nil, invalidScriptResult("entry isDir is required")
	}

	meta := types.EntryMeta{Readable: true, Writable: true}
	metaV := v.Get("meta")
	if !metaV.IsNil() {
		if e := metaV.ParseInto(&meta); e != nil {
			return nil, mapScriptDriveError(e)
		}
	}

	return &scriptEntryStruct{
		Meta:    meta,
		IsDir:   isDirV.Bool(),
		Path:    pathV.String(),
		Size:    v.Get("size").Integer(),
		ModTime: v.Get("modTime").Integer(),
		Data:    v.Get("data").SM(),
	}, nil
}

type scriptEntryStruct struct {
	Path    string
	Size    int64
	ModTime int64
	Meta    types.EntryMeta
	IsDir   bool
	Data    types.SM
}

var _ types.IEntry = (*scriptDriveEntry)(nil)

type scriptDriveEntry struct {
	d *ScriptDrive
	s *scriptEntryStruct
}

// GetReader gets the reader of this entry
func (se *scriptDriveEntry) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	if !se.d.has.getReader {
		return nil, err.NewUnsupportedError()
	}
	var result io.ReadCloser
	e := se.d.withVM(ctx, func(vm *s.VM) error {
		v, e := se.d.call(ctx, vm, "getReader", se.s, start, size)
		if e != nil {
			return e
		}
		result = s.DetachReader(vm, v.Raw(), "")
		if result == nil {
			return invalidScriptResult("getReader must return a Reader")
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

func (se *scriptDriveEntry) GetURL(ctx context.Context) (*types.ContentURL, error) {
	if !se.d.has.getURL {
		return nil, err.NewUnsupportedError()
	}
	var r types.ContentURL
	e := se.d.withVM(ctx, func(vm *s.VM) error {
		v, e := se.d.call(ctx, vm, "getURL", se.s)
		if e != nil {
			return e
		}
		if e := v.ParseInto(&r); e != nil {
			return mapScriptDriveError(e)
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
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

func (se *scriptDriveEntry) Thumbnail(ctx context.Context) (types.IContentReader, error) {
	if se.d == nil || !se.d.has.getThumbnail || se.s == nil || !se.s.Meta.SelfThumbnail {
		return nil, err.NewUnsupportedError()
	}
	var result types.IContentReader
	e := se.d.withVM(ctx, func(vm *s.VM) error {
		v, e := se.d.call(ctx, vm, "getThumbnail", se.s)
		if e != nil {
			return e
		}
		if rc := s.DetachReader(vm, v.Raw(), ""); rc != nil {
			result = wrapContentReader(rc)
			return nil
		}
		r := types.ContentURL{}
		if e := v.ParseInto(&r); e != nil {
			return mapScriptDriveError(e)
		}
		result = driveutil.NewURLContentReader(r.URL, r.Header, r.Proxy)
		return nil
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}
