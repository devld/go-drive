package script

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/task"
	"go-drive/common/types"
	s "go-drive/script"
)

type memDriveData struct {
	data types.SM
}

func (m *memDriveData) Save(data types.SM) error {
	if m.data == nil {
		m.data = types.SM{}
	}
	for k, v := range data {
		m.data[k] = v
	}
	return nil
}

func (m *memDriveData) Load(keys ...string) (types.SM, error) {
	r := types.SM{}
	for _, k := range keys {
		r[k] = m.data[k]
	}
	return r, nil
}

func (m *memDriveData) Clear() error {
	m.data = types.SM{}
	return nil
}

func testDriveUtils(data *memDriveData) *scriptDriveUtils {
	return newScriptDriveUtils(driveutil.DriveUtils{
		Data: data,
		CreateCache: func(driveutil.EntryDeserialize) driveutil.DriveCache {
			return driveutil.DummyCache()
		},
		Config: common.Config{},
	})
}

func TestParseDurationFromJS(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	v, e := vm.Run(context.Background(), `parseDuration("2h")`)
	if e != nil {
		t.Fatal(e)
	}
	if time.Duration(v.Integer()) != 2*time.Hour {
		t.Fatalf("parseDuration(2h) = %d", v.Integer())
	}

	v, e = vm.Run(context.Background(), `parseDuration("")`)
	if e != nil {
		t.Fatal(e)
	}
	if v.Integer() != 0 {
		t.Fatalf("parseDuration empty = %d", v.Integer())
	}

	v, e = vm.Run(context.Background(), `parseDuration("nope")`)
	if e != nil {
		t.Fatal(e)
	}
	if time.Duration(v.Integer()) != -1 {
		t.Fatalf("parseDuration invalid = %d", v.Integer())
	}
}

func TestDefineDriveInitConfigRequiredFields(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
  configForm: [
    { Label: "Token", Field: "token", Type: "text", Required: true },
    entryCacheTTLFormItem("2h")
  ],
  createInstance: function (data) { return { entryCacheTTL: data.cache_ttl, token: data.token }; },
  },
  {
  get: function () { return { Path: "x", IsDir: false, Size: 1, ModTime: -1 }; },
  list: function () { return []; },
  getURL: function () { return { URL: "https://example.com" }; }
  }
);
`); e != nil {
		t.Fatal(e)
	}

	data := &memDriveData{}
	utils := testDriveUtils(data)
	v, e := vm.Call(context.Background(), "__driveInitConfig", nil, types.SM{}, utils)
	if e != nil {
		t.Fatal(e)
	}
	cfg := &driveutil.DriveInitConfig{}
	v.ParseInto(cfg)
	if cfg.Configured {
		t.Fatal("expected unconfigured without required token")
	}
	if len(cfg.Form) != 2 {
		t.Fatalf("form len = %d", len(cfg.Form))
	}

	_ = data.Save(types.SM{"token": "abc", "cache_ttl": "30m"})
	v, e = vm.Call(context.Background(), "__driveInitConfig", nil, types.SM{}, utils)
	if e != nil {
		t.Fatal(e)
	}
	cfg = &driveutil.DriveInitConfig{}
	v.ParseInto(cfg)
	if !cfg.Configured {
		t.Fatal("expected configured when required fields are saved")
	}
}

func TestDefineDriveCreateAndGetRoot(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () { return {}; }
  },
  {
    get: function (ctx, path) { throw new Error("get must not handle root"); },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, nil)
	entry, e := d.Get(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if !entry.Type().IsDir() || entry.Path() != "" {
		t.Fatalf("root entry = %#v", entry)
	}
	if !entry.Meta().Writable {
		t.Fatal("default root Writable should be true")
	}
}

func TestDefineDriveWritableFromCreateInstance(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () { return { writable: false }; }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, nil)
	meta, e := d.Meta(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if meta.Writable {
		t.Fatal("expected Writable false from createInstance")
	}
	root, e := d.Get(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if root.Meta().Writable {
		t.Fatal("root entry Writable should follow createInstance")
	}
}

func TestScriptDriveGetUsesCacheWithoutCallingJS(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return { entryCacheTTL: "1h", $hits: 0 };
    }
  },
  {
    get: function (ctx, path) {
      this.$hits = this.$hits + 1;
      return { Path: path, IsDir: false, Size: this.$hits, ModTime: 1, Data: { id: "x" } };
    },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, mgr)

	a, e := d.Get(context.Background(), "file.txt")
	if e != nil {
		t.Fatal(e)
	}
	if a.Size() != 1 {
		t.Fatalf("first get size = %d", a.Size())
	}
	b, e := d.Get(context.Background(), "file.txt")
	if e != nil {
		t.Fatal(e)
	}
	if b.Size() != 1 {
		t.Fatalf("cached get should not call JS again, size = %d", b.Size())
	}
}

func TestScriptDriveSaveEvictsAndReggets(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return { entryCacheTTL: "1h", $n: 0 };
    }
  },
  {
    get: function (ctx, path) {
      this.$n = this.$n + 1;
      return { Path: path, IsDir: false, Size: this.$n, ModTime: 1 };
    },
    list: function () { return []; },
    save: function () {},
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, mgr)

	first, e := d.Get(context.Background(), "a.txt")
	if e != nil {
		t.Fatal(e)
	}
	if first.Size() != 1 {
		t.Fatalf("size = %d", first.Size())
	}
	saved, e := d.Save(task.DummyContext(), "a.txt", 0, true, bytes.NewReader(nil))
	if e != nil {
		t.Fatal(e)
	}
	if saved.Size() != 2 {
		t.Fatalf("after save expected re-get, size = %d", saved.Size())
	}
}

func TestScriptDriveSaveInvalidatesParentList(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return { entryCacheTTL: "1h", $lists: 0 };
    }
  },
  {
    get: function (ctx, path) {
      return { Path: path, IsDir: false, Size: 1, ModTime: 1 };
    },
    list: function (ctx, path) {
      this.$lists = this.$lists + 1;
      return [{ Path: path ? path + "/old.txt" : "old.txt", IsDir: false, Size: this.$lists, ModTime: 1 }];
    },
    save: function () {},
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, mgr)

	listed, e := d.List(context.Background(), "dir")
	if e != nil {
		t.Fatal(e)
	}
	if len(listed) != 1 || listed[0].Size() != 1 {
		t.Fatalf("first list = %#v", listed)
	}
	if _, e := d.Save(task.DummyContext(), "dir/new.txt", 0, true, bytes.NewReader(nil)); e != nil {
		t.Fatal(e)
	}
	listed, e = d.List(context.Background(), "dir")
	if e != nil {
		t.Fatal(e)
	}
	if len(listed) != 1 || listed[0].Size() != 2 {
		t.Fatalf("list after save should miss cache, got size %d", listed[0].Size())
	}
}

func TestScriptDriveConcurrentGetCoalescesJS(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return { entryCacheTTL: "1h", $hits: 0 };
    }
  },
  {
    get: function (ctx, path) {
      this.$hits = this.$hits + 1;
      sleep(ms(40));
      return { Path: path, IsDir: false, Size: this.$hits, ModTime: 1 };
    },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, mgr)

	const n = 8
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	sizes := make([]int64, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			entry, e := d.Get(context.Background(), "file.txt")
			if e != nil {
				errCh <- e
				return
			}
			sizes[i] = entry.Size()
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
	for i, size := range sizes {
		if size != 1 {
			t.Fatalf("goroutine %d size = %d, want coalesced JS get", i, size)
		}
	}
}

func TestScriptDriveUploadDefaultsWithoutJS(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () { return {}; }
  },
  {
    get: function (ctx, path) { return { Path: path, IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`, nil, nil)
	cfg, e := d.Upload(context.Background(), "a.txt", 1, true, types.SM{})
	if e != nil {
		t.Fatal(e)
	}
	if cfg == nil || cfg.Provider != types.LocalProvider {
		t.Fatalf("upload config = %#v", cfg)
	}
}

func newTestScriptDrive(t *testing.T, js string, data types.SM, cacheMgr *driveutil.MemDriveCacheManager) *ScriptDrive {
	t.Helper()
	vm := baseVM.Fork()
	if _, e := vm.Run(context.Background(), js); e != nil {
		_ = vm.Dispose()
		t.Fatal(e)
	}
	d := &ScriptDrive{
		baseVM:   vm,
		data:     make(map[string]json.RawMessage),
		writable: true,
	}
	if cacheMgr != nil {
		d.cache = cacheMgr.GetCacheStore("test", d.deserializeEntry)
	} else {
		d.cache = driveutil.DummyCache()
	}
	t.Cleanup(func() { _ = d.Dispose() })

	vm.Set("setData", s.WrapVmCall(vm, d.setData))
	vm.Set("getData", s.WrapVmCall(vm, d.getData))
	store := &memDriveData{data: data}
	utils := testDriveUtils(store)
	if cacheMgr != nil {
		utils.cache = &scriptDriveCache{d.cache}
	}
	createdVal, e := vm.Call(context.Background(), "__driveCreate", nil, types.SM{}, utils)
	if e != nil {
		t.Fatal(e)
	}
	var created struct {
		Writable      bool
		EntryCacheTTL string
	}
	if createdVal != nil && !createdVal.IsNil() {
		createdVal.ParseInto(&created)
		d.writable = created.Writable
		ttl := types.SV(created.EntryCacheTTL).Duration(0)
		if ttl > 0 {
			d.cacheTTL = ttl
		}
	}
	d.inspectMethods(vm)
	vm.Set("selfDrive", s.NewDrive(d))
	d.pool = s.NewVMPool(vm, &s.VMPoolConfig{
		MaxTotal: 4, MaxIdle: 2, MinIdle: 0, IdleTime: time.Minute,
	})
	return d
}
