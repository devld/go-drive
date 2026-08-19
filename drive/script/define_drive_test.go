package script

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/task"
	"go-drive/common/types"
	s "go-drive/script"

	"golang.org/x/oauth2"
)

type memDriveData struct {
	data types.SM
}

func (m *memDriveData) Save(data types.SM) error {
	if m.data == nil {
		m.data = types.SM{}
	}
	for k, v := range data {
		if v == "" {
			delete(m.data, k)
			continue
		}
		m.data[k] = v
	}
	return nil
}

func (m *memDriveData) Load(key string, keys ...string) (types.SM, error) {
	r := types.SM{}
	keys = append([]string{key}, keys...)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			r[k] = v
		}
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

func TestDefineDriveStaticAndDynamicLifecycle(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
  configForm: [
    { Label: "Token", Field: "token", Type: "text", Required: true }
  ],
  initConfig: function (ctx, config, utils) {
    var data = utils.Data.Load("step");
    return {
      Configured: data.step === "done",
      Form: [{ Label: "Step", Field: "step", Type: "text", Required: true }],
      Value: data
    };
  },
  init: function (ctx, data, config, utils) {
    utils.Data.Save({ step: data.step, empty: data.empty });
  },
  createInstance: function (ctx, config, utils) {
    var data = utils.Data.Load("step");
    return { entryCacheTTL: config.token, writable: data.step === "done" };
  },
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
	formValue, e := vm.GetValue("__driveConfigForm")
	if e != nil {
		t.Fatal(e)
	}
	var form []types.FormItem
	formValue.ParseInto(&form)
	if len(form) != 1 || form[0].Field != "token" {
		t.Fatalf("static form = %#v", form)
	}

	if e := data.Save(types.SM{"step": "old", "empty": "old"}); e != nil {
		t.Fatal(e)
	}
	v, e := vm.Call(context.Background(), "__driveInitConfig", nil, types.SM{"token": "abc"}, utils)
	if e != nil {
		t.Fatal(e)
	}
	cfg := &driveutil.DriveInitConfig{}
	v.ParseInto(cfg)
	if cfg.Configured {
		t.Fatal("expected unconfigured before dynamic initialization")
	}
	if len(cfg.Form) != 1 || cfg.Form[0].Field != "step" {
		t.Fatalf("dynamic form = %#v", cfg.Form)
	}

	_, e = vm.Call(context.Background(), "__driveInit", nil,
		types.SM{"step": "done", "empty": ""}, types.SM{"token": "abc"}, utils)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := data.Load("step", "empty")
	if e != nil {
		t.Fatal(e)
	}
	if saved["step"] != "done" {
		t.Fatalf("saved step = %#v", saved)
	}
	if _, ok := saved["empty"]; ok {
		t.Fatalf("empty value was not cleared: %#v", saved)
	}

	v, e = vm.Call(context.Background(), "__driveCreate", s.NewContext(vm, context.Background()),
		types.SM{"token": "30m"}, utils)
	if e != nil {
		t.Fatal(e)
	}
	var created struct {
		Writable      bool
		EntryCacheTTL string
	}
	v.ParseInto(&created)
	if !created.Writable || created.EntryCacheTTL != "30m" {
		t.Fatalf("created = %#v", created)
	}
}

func TestDefineDriveRejectsReservedFormFields(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
    configForm: [{ Label: "Reserved", Field: "_reserved", Type: "text" }],
    createInstance: function () { return {}; }
  },
  {
    get: function () { return { Path: "x", IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`); e == nil {
		t.Fatal("expected reserved form field to be rejected")
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

	vm.Set("__setData", s.WrapVmCall(vm, d.setData))
	vm.Set("__getData", s.WrapVmCall(vm, d.getData))
	store := &memDriveData{data: data}
	utils := testDriveUtils(store)
	if cacheMgr != nil {
		utils.cache = &scriptDriveCache{d.cache}
	}
	createdVal, e := vm.Call(context.Background(), "__driveCreate", s.NewContext(vm, context.Background()), types.SM{}, utils)
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

func TestOAuthHolderTokenAcceptsTaskCtx(t *testing.T) {
	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:     "access",
		driveutil.DsKeyTokenType: "Bearer",
		driveutil.DsKeyExpiresAt: strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	holder := testDriveUtils(ds).OAuthLoad(driveutil.OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/unused"},
	}, driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"})

	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	vm.Set("holder", holder)
	vm.Set("plainCtx", s.NewContext(vm, context.Background()))
	vm.Set("taskCtx", s.NewTaskCtx(vm, task.DummyContext()))

	v, e := vm.Run(context.Background(), `holder.Token(plainCtx).AccessToken`)
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "access" {
		t.Fatalf("Context token = %q", v.String())
	}

	v, e = vm.Run(context.Background(), `holder.Token(taskCtx).AccessToken`)
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "access" {
		t.Fatalf("TaskCtx token = %q", v.String())
	}

	v, e = vm.Run(context.Background(), `
		var tctx = newContextWithTimeout(newContext(), ms(10000));
		var token = holder.Token(tctx).AccessToken;
		tctx.Cancel();
		token
	`)
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "access" {
		t.Fatalf("timeout Context token = %q", v.String())
	}
}

func TestOAuthInitAcceptsTaskCtx(t *testing.T) {
	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	vm.Set("utils", testDriveUtils(&memDriveData{}))
	vm.Set("taskCtx", s.NewTaskCtx(vm, task.DummyContext()))
	if _, e := vm.Run(context.Background(), `
		utils.OAuthInit(taskCtx, {}, { Endpoint: {} }, { ClientID: "id", ClientSecret: "secret" });
	`); e != nil {
		t.Fatal(e)
	}
}
