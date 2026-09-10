package script

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
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

func testDriveUtilEnv(data *memDriveData) driveutil.DriveUtils {
	return driveutil.DriveUtils{
		Data: data,
		CreateCache: func(driveutil.EntryDeserialize) driveutil.DriveCache {
			return driveutil.DummyCache()
		},
		Config: common.Config{},
	}
}

func testDriveUtils(vm *s.VM, data *memDriveData) *scriptDriveUtils {
	return newScriptDriveUtils(vm, testDriveUtilEnv(data), nil, nil)
}

func newDriveTestVM(t *testing.T) *s.VM {
	t.Helper()
	vm, e := s.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	if e = vm.WithBridge(map[string]any{
		"version": "",
		"name":    "test",
	}, func() error {
		_, e := vm.Run(context.Background(), helperProgram, "helper.js")
		return e
	}); e != nil {
		_ = vm.Dispose()
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })
	return vm
}

func mustDefineGlobal(t *testing.T, vm *s.VM, name string, value any) {
	t.Helper()
	if e := vm.DefineGlobal(name, value); e != nil {
		t.Fatal(e)
	}
}

func TestSleepAndTimeoutAcceptDurationString(t *testing.T) {
	vm := newDriveTestVM(t)

	start := time.Now()
	if _, e := vm.Run(context.Background(), `sleep("20ms")`, ""); e != nil {
		t.Fatal(e)
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("sleep(\"20ms\") returned too quickly")
	}

	if _, e := vm.Run(context.Background(), `sleep("nope")`, ""); e == nil {
		t.Fatal("expected sleep(\"nope\") to fail")
	}
}

func TestHelperExportsAreFrozen(t *testing.T) {
	vm := newDriveTestVM(t)
	got, e := vm.Run(context.Background(), `
		var original = defineDrive;
		defineDrive = function () {};
		useLocalProvider.extra = true;
		[
			defineDrive === original,
			typeof useLocalProvider.extra,
			Object.isFrozen(defineDrive),
			Object.isFrozen(useLocalProvider),
			Object.isFrozen(useCustomProvider),
			Object.isFrozen(entryCacheTTLFormItem)
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|undefined|true|true|true|true" {
		t.Fatalf("helper exports = %q", got.String())
	}
}

func TestScriptGenericErrorIsRemoteApiError(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function () { throw new Error("boom"); },
    list: function () { return "not-an-array"; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	_, e := d.Get(context.Background(), "x")
	var remote err.RemoteApiError
	if !errors.As(e, &remote) || remote.Status() != http.StatusInternalServerError {
		t.Fatalf("generic throw = %#v (%v)", e, e)
	}
	_, e = d.List(context.Background(), "")
	if !errors.As(e, &remote) || remote.Status() != http.StatusInternalServerError {
		t.Fatalf("invalid list = %#v (%v)", e, e)
	}
}

func TestScriptTypedErrorIsPreserved(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function () { throw new NotFoundError("missing"); },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	_, e := d.Get(context.Background(), "x")
	if !err.IsNotFoundError(e) || e.Error() != "missing" {
		t.Fatalf("typed throw = %v", e)
	}
}

func TestDefineDriveStaticAndDynamicLifecycle(t *testing.T) {
	vm := newDriveTestVM(t)
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
  configForm: [
    { label: "Token", field: "token", type: "text", required: true }
  ],
  initConfig: function (config, utils) {
    var data = utils.data.load("step");
    return {
      configured: data.step === "done",
      form: [{ label: "Step", field: "step", type: "text", required: true }],
      value: data
    };
  },
  init: function (data, config, utils) {
    utils.data.save({ step: data.step, empty: data.empty });
  },
  createInstance: function (config, utils) {
    var data = utils.data.load("step");
    return { entryCacheTTL: config.token, writable: data.step === "done" };
  },
  },
  {
  get: function () { return { path: "x", isDir: false, size: 1, modTime: -1 }; },
  list: function () { return []; },
  getURL: function () { return { url: "https://example.com" }; }
  }
);
`, ""); e != nil {
		t.Fatal(e)
	}

	data := &memDriveData{}
	utils := testDriveUtils(vm, data)
	formValue, e := vm.GetValue("__driveConfigForm")
	if e != nil {
		t.Fatal(e)
	}
	var form []types.FormItem
	if e := formValue.ParseInto(&form); e != nil {
		t.Fatal(e)
	}
	if len(form) != 1 || form[0].Field != "token" {
		t.Fatalf("static form = %#v", form)
	}

	if e := data.Save(types.SM{"step": "old", "empty": "old"}); e != nil {
		t.Fatal(e)
	}
	v, e := vm.Call(context.Background(), "__driveInitConfig", types.SM{"token": "abc"}, utils)
	if e != nil {
		t.Fatal(e)
	}
	cfg := &driveutil.DriveInitConfig{}
	if e := v.ParseInto(cfg); e != nil {
		t.Fatal(e)
	}
	if cfg.Configured {
		t.Fatal("expected unconfigured before dynamic initialization")
	}
	if len(cfg.Form) != 1 || cfg.Form[0].Field != "step" {
		t.Fatalf("dynamic form = %#v", cfg.Form)
	}

	_, e = vm.Call(context.Background(), "__driveInit",
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

	v, e = vm.Call(context.Background(), "__driveCreate",
		types.SM{"token": "30m"}, utils)
	if e != nil {
		t.Fatal(e)
	}
	var created struct {
		Writable      bool
		EntryCacheTTL string
	}
	if e := v.ParseInto(&created); e != nil {
		t.Fatal(e)
	}
	if !created.Writable || created.EntryCacheTTL != "30m" {
		t.Fatalf("created = %#v", created)
	}
}

func TestDefineDriveRejectsReservedFormFields(t *testing.T) {
	vm := newDriveTestVM(t)
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
    configForm: [{ label: "Reserved", field: "_reserved", type: "text" }],
    createInstance: function () { return {}; }
  },
  {
    get: function () { return { path: "x", isDir: false, size: 1, modTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, ""); e == nil {
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
    get: function (path) { throw new Error("get must not handle root"); },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
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
    get: function (path) { return { path: path, isDir: false, size: 1, modTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
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
    get: function (path) {
      this.$hits = this.$hits + 1;
      return { path: path, isDir: false, size: this.$hits, modTime: 1, data: { id: "x" } };
    },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
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

func TestScriptDriveCacheGetIsDetachedSnapshot(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d := newTestScriptDrive(t, `
defineDrive(
  {
    createInstance: function () {
      return { entryCacheTTL: "1h" };
    }
  },
  {
    get: function (path) {
      if (path === "peek") {
        var hit = this.cache.getEntry("file.txt");
        var kids = this.cache.getChildren("");
        hit.data.id = "mutated";
        kids[0].data.id = "mutated-child";
        hit.path = "changed";
        return {
          path: path,
          isDir: false,
          size: 1,
          modTime: 1,
          data: {
            id: hit.data.id,
            child: kids[0].data.id,
            path: hit.path,
            type: String(hit.type),
            plain: String(hit.constructor === Object && kids.constructor === Array)
          }
        };
      }
      return { path: path, isDir: false, size: 1, modTime: 1, data: { id: "orig" } };
    },
    list: function () {
      return [{ path: "file.txt", isDir: false, size: 1, modTime: 1, data: { id: "orig" } }];
    },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, mgr)

	if _, e := d.Get(context.Background(), "file.txt"); e != nil {
		t.Fatal(e)
	}
	if _, e := d.List(context.Background(), ""); e != nil {
		t.Fatal(e)
	}
	peek, e := d.Get(context.Background(), "peek")
	if e != nil {
		t.Fatal(e)
	}
	data := peek.(driveutil.CacheableEntry).EntryData()
	if data["id"] != "orig" || data["child"] != "orig" {
		t.Fatalf("peek readonly data = %#v", data)
	}
	if data["path"] != "file.txt" || data["type"] != "file" || data["plain"] != "true" {
		t.Fatalf("cache item fields = %#v", data)
	}
	cached, e := d.Get(context.Background(), "file.txt")
	if e != nil {
		t.Fatal(e)
	}
	if cached.(driveutil.CacheableEntry).EntryData()["id"] != "orig" {
		t.Fatalf("cache mutated: %#v", cached.(driveutil.CacheableEntry).EntryData())
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
    get: function (path) {
      this.$n = this.$n + 1;
      return { path: path, isDir: false, size: this.$n, modTime: 1 };
    },
    list: function () { return []; },
    save: function () {},
    getURL: function () { return { url: "https://example.com" }; }
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

func TestScriptDriveSavePreservesCallerReader(t *testing.T) {
	for _, maxIdle := range []int{0, 1} {
		t.Run(fmt.Sprintf("MaxIdle=%d", maxIdle), func(t *testing.T) {
			d := newTestScriptDrivePool(t, `
defineDrive({ createInstance() { return {}; } }, {
  get(path) { return { path, isDir: false, size: 3 }; },
  list() { return []; },
  getURL() { return { url: "https://example.com" }; },
  save(path, size, override, reader) {
    if (reader.readAsString() !== "abc") throw new Error("incorrect content");
    if (path === "fail") throw new Error("upload failed");
  }
});
`, &s.VMPoolConfig{MaxTotal: 1, MaxIdle: maxIdle})
			vm, e := s.NewVM()
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = vm.Dispose() })
			mustDefineGlobal(t, vm, "target", d)
			tc := task.NewTaskContext(context.Background())
			progress := s.NewProgressReporter(tc, true, false)
			mustDefineGlobal(t, vm, "progress", progress)
			_, e = vm.Run(context.Background(), `
const file = new TempFile();
try {
  file.write(Bytes.fromString("abc"));
  for (const path of ["first", "fail", "second"]) {
    file.seekTo(0, SEEK_START);
    let failed = false;
    try {
	  target.save(path, 3, true, file, progress);
    } catch (e) {
      if (path !== "fail" || !e.message.includes("upload failed")) throw e;
      failed = true;
    }
    if (failed !== (path === "fail")) throw new Error("unexpected upload result");
    file.seekTo(0, SEEK_START);
    if (file.readAsString() !== "abc") throw new Error("caller reader changed");
  }
} finally {
  file.close();
}
`, "caller.js")
			if e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestScriptDriveProgressReporterAddsLoadedAndKeepsSaveTotal(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) { return { path: path, isDir: false, size: 1, modTime: 1 }; },
    list: function () { return []; },
	    save: function (path, size, override, reader, progress) {
	      progress.addLoaded(3);
	      progress.addLoaded(5);
    },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	tc := task.NewTaskContext(context.Background())
	if _, e := d.Save(tc, "a.txt", 7, true, bytes.NewReader(nil)); e != nil {
		t.Fatal(e)
	}
	if tc.GetProgress() != 8 {
		t.Fatalf("progress = %d, want 8", tc.GetProgress())
	}
	if tc.GetTotal() != 7 {
		t.Fatalf("total = %d, want 7", tc.GetTotal())
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
    get: function (path) {
      return { path: path, isDir: false, size: 1, modTime: 1 };
    },
    list: function (path) {
      this.$lists = this.$lists + 1;
      return [{ path: path ? path + "/old.txt" : "old.txt", isDir: false, size: this.$lists, modTime: 1 }];
    },
    save: function () {},
    getURL: function () { return { url: "https://example.com" }; }
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
    get: function (path) {
      this.$hits = this.$hits + 1;
      sleep(ms(40));
      return { path: path, isDir: false, size: this.$hits, modTime: 1 };
    },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
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
    get: function (path) { return { path: path, isDir: false, size: 1, modTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
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

func TestScriptDriveGetRejectsEntryWithoutIsDir(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) { return { path: path }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	_, e := d.Get(context.Background(), "file")
	if e == nil {
		t.Fatal("expected invalid entry error")
	}
	if !strings.Contains(e.Error(), "isDir") {
		t.Fatalf("error = %v", e)
	}
}

func TestScriptDriveGetConvertsGettersBeforeVMReturn(t *testing.T) {
	js := `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return {
        get path() { return path + "/" + Bytes.fromString("abc").length; },
        get isDir() { return false; },
        get size() { return 1; },
        get modTime() { return 1; }
      };
    },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`
	d := newTestScriptDrivePool(t, js, &s.VMPoolConfig{
		MaxTotal: 1, MaxIdle: 0, MinIdle: 0, IdleTime: 0,
	})
	entry, e := d.Get(context.Background(), "a/b")
	if e != nil {
		t.Fatal(e)
	}
	if entry.Path() != "a/b/3" || entry.Size() != 1 {
		t.Fatalf("entry = %#v", entry)
	}

	pooled := newTestScriptDrivePool(t, js, &s.VMPoolConfig{
		MaxTotal: 1, MaxIdle: 1, MinIdle: 0, IdleTime: time.Minute,
	})
	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for i := range 32 {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			path := fmt.Sprintf("x/%d", i)
			got, e := pooled.Get(context.Background(), path)
			if e != nil {
				errCh <- e
				return
			}
			want := path + "/3"
			if got.Path() != want {
				errCh <- fmt.Errorf("path = %q want %q", got.Path(), want)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
}

func TestScriptDriveGetGetterThrowIsError(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return {
        get path() { throw new Error("bad path"); },
        isDir: false,
        size: 1,
        modTime: 1
      };
    },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Get panicked: %v", r)
		}
	}()
	_, e := d.Get(context.Background(), "file")
	if e == nil {
		t.Fatal("expected getter throw to be an error")
	}
}

func TestScriptDriveGetGetterHonorsContextDeadline(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return {
        path: path,
        isDir: false,
        get size() { sleep("100ms"); return 1; },
        modTime: 1
      };
    },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, e := d.Get(ctx, "file")
	elapsed := time.Since(started)
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v after %s", e, elapsed)
	}
	if elapsed >= 80*time.Millisecond {
		t.Fatalf("getter sleep was not interrupted: %s", elapsed)
	}
}

func TestDefineDriveRejectsAsyncLifecycle(t *testing.T) {
	mustFail := func(name, js string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			vm := newDriveTestVM(t)
			if _, e := vm.Run(context.Background(), js, ""); e != nil {
				t.Fatal(e)
			}
			utils := testDriveUtils(vm, &memDriveData{})
			_, e := vm.Call(context.Background(), "__driveCreate", types.SM{}, utils)
			if e == nil || !strings.Contains(e.Error(), "Promise") {
				t.Fatalf("%s error = %v", name, e)
			}
		})
	}
	mustFail("createInstance", `
defineDrive(
  { createInstance: async function () { throw new Error("async create"); } },
  {
    get: function () { return { path: "x", isDir: false, size: 1, modTime: 1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`)
	mustFail("validateConfig", `
defineDrive(
  {
    validateConfig: async function () { throw new Error("async validate"); },
    createInstance: function () { return {}; }
  },
  {
    get: function () { return { path: "x", isDir: false, size: 1, modTime: 1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`)

	vm := newDriveTestVM(t)
	if _, e := vm.Run(context.Background(), `
defineDrive(
  {
    init: async function () { throw new Error("async init"); },
    createInstance: function () { return {}; }
  },
  {
    get: function () { return { path: "x", isDir: false, size: 1, modTime: 1 }; },
    list: function () { return []; },
    getURL: function () { return { url: "https://example.com" }; }
  }
);
`, ""); e != nil {
		t.Fatal(e)
	}
	_, e := vm.Call(context.Background(), "__driveInit", types.SM{}, types.SM{}, testDriveUtils(vm, &memDriveData{}))
	if e == nil || !strings.Contains(e.Error(), "Promise") {
		t.Fatalf("init error = %v", e)
	}
}

func newTestScriptDrive(t *testing.T, js string, data types.SM, cacheMgr *driveutil.MemDriveCacheManager) *ScriptDrive {
	t.Helper()
	d, e := buildTestScriptDrive(js, data, cacheMgr, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = d.Dispose() })
	return d
}

func newTestScriptDrivePool(t *testing.T, js string, pool *s.VMPoolConfig) *ScriptDrive {
	t.Helper()
	d, e := buildTestScriptDrive(js, nil, nil, pool)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = d.Dispose() })
	return d
}

func buildTestScriptDrive(js string, data types.SM, cacheMgr *driveutil.MemDriveCacheManager, pool *s.VMPoolConfig) (*ScriptDrive, error) {
	d := &ScriptDrive{data: make(map[string]any), writable: true}
	program, e := s.Compile("test-drive.js", js)
	if e != nil {
		return d, e
	}
	compiled := &compiledDriveScript{program: program, name: "test-drive"}
	if cacheMgr != nil {
		d.cache = cacheMgr.GetCacheStore("test", d.deserializeEntry)
	} else {
		d.cache = driveutil.DummyCache()
	}
	store := &memDriveData{data: data}
	var initializeOnce sync.Once
	var initializeErr error
	initializer := func(ctx context.Context, vm *s.VM) error {
		if e := initializeDriveScriptVM(ctx, vm, compiled, d); e != nil {
			return e
		}
		var cache *scriptDriveCache
		if cacheMgr != nil {
			cache = newScriptDriveCache(vm, d.cache)
		}
		utils := newScriptDriveUtils(vm, testDriveUtilEnv(store), &d.oauth, cache)
		return vm.Do(ctx, func() error {
			createdVal, e := vm.Call(ctx, "__driveCreate", types.SM{}, utils)
			if e != nil {
				return e
			}
			initializeOnce.Do(func() {
				initializeErr = d.applyCreated(createdVal)
				if initializeErr == nil {
					d.inspectMethods(vm)
				}
			})
			if initializeErr != nil {
				return initializeErr
			}
			return vm.DefineGlobal("selfDrive", d)
		})
	}
	if pool == nil {
		pool = &s.VMPoolConfig{MaxTotal: 4, MaxIdle: 2, MinIdle: 0, IdleTime: time.Minute}
	}
	d.pool, e = s.NewVMPool(context.Background(), initializer, pool)
	if e != nil {
		return d, e
	}
	vm, e := d.pool.Get(context.Background())
	if e != nil {
		_ = d.Dispose()
		return nil, e
	}
	if e := d.pool.Return(context.Background(), vm); e != nil {
		_ = d.Dispose()
		return nil, e
	}
	if e := d.startIntervals(); e != nil {
		_ = d.Dispose()
		return nil, e
	}
	return d, nil
}

func newTestScriptDriveExpectError(t *testing.T, js string) error {
	t.Helper()
	d, e := buildTestScriptDrive(js, nil, nil, nil)
	if d != nil {
		_ = d.Dispose()
	}
	return e
}

func TestOAuthHolderToken(t *testing.T) {
	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:     "access",
		driveutil.DsKeyTokenType: "Bearer",
		driveutil.DsKeyExpiresAt: strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	vm := newDriveTestVM(t)
	mustDefineGlobal(t, vm, "utils", testDriveUtils(vm, ds))
	mustDefineGlobal(t, vm, "req", driveutil.OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/unused"},
	})
	mustDefineGlobal(t, vm, "cred", driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"})
	v, e := vm.Run(context.Background(), `utils.oauthLoad(req, cred).token().accessToken`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "access" {
		t.Fatalf("token = %q", v.String())
	}
}

func TestOAuthHolderRefreshFromJS(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"new-%d","token_type":"Bearer","expires_in":3600}`, id)
	}))
	t.Cleanup(srv.Close)

	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:        "access",
		driveutil.DsKeyTokenType:    "Bearer",
		driveutil.DsKeyRefreshToken: "refresh",
		driveutil.DsKeyExpiresAt:    strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	vm := newDriveTestVM(t)
	mustDefineGlobal(t, vm, "utils", testDriveUtils(vm, ds))
	mustDefineGlobal(t, vm, "req", driveutil.OAuthRequest{Endpoint: oauth2.Endpoint{TokenURL: srv.URL}})
	mustDefineGlobal(t, vm, "cred", driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"})
	if _, e := vm.Run(context.Background(), `holder = utils.oauthLoad(req, cred)`, ""); e != nil {
		t.Fatal(e)
	}

	v, e := vm.Run(context.Background(), `holder.refresh().accessToken`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "new-1" {
		t.Fatalf("Refresh = %q", v.String())
	}
	v, e = vm.Run(context.Background(), `holder.token().accessToken`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "new-1" {
		t.Fatalf("Token after Refresh = %q", v.String())
	}
}

func TestOAuthLoadSharesHolderAcrossVMs(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"new-%d","token_type":"Bearer","expires_in":3600,"refresh_token":"rotated-%d"}`, id, id)
	}))
	t.Cleanup(srv.Close)

	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:        "access",
		driveutil.DsKeyTokenType:    "Bearer",
		driveutil.DsKeyRefreshToken: "refresh",
		driveutil.DsKeyExpiresAt:    strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}

	share := &oauthHolderShare{}
	req := driveutil.OAuthRequest{Endpoint: oauth2.Endpoint{TokenURL: srv.URL}}
	cred := driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"}

	first := newDriveTestVM(t)
	firstUtils := testDriveUtils(first, ds)
	firstUtils.oauth = share
	mustDefineGlobal(t, first, "utils", firstUtils)
	mustDefineGlobal(t, first, "req", req)
	mustDefineGlobal(t, first, "cred", cred)
	if _, e := first.Run(context.Background(), `holder = utils.oauthLoad(req, cred)`, ""); e != nil {
		t.Fatal(e)
	}

	second := newDriveTestVM(t)
	secondUtils := testDriveUtils(second, ds)
	secondUtils.oauth = share
	mustDefineGlobal(t, second, "utils", secondUtils)
	mustDefineGlobal(t, second, "req", req)
	mustDefineGlobal(t, second, "cred", cred)
	if _, e := second.Run(context.Background(), `holder = utils.oauthLoad(req, cred)`, ""); e != nil {
		t.Fatal(e)
	}

	v, e := first.Run(context.Background(), `holder.refresh().accessToken`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "new-1" {
		t.Fatalf("first VM refresh = %q", v.String())
	}

	v, e = second.Run(context.Background(), `holder.token().accessToken`, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "new-1" {
		t.Fatalf("second VM token after first refresh = %q", v.String())
	}
	if got := n.Load(); got != 1 {
		t.Fatalf("token endpoint calls = %d, want 1", got)
	}
}

func TestOAuthLoadDoesNotShareHolderAcrossAuthStyle(t *testing.T) {
	var headerAuth, paramsAuth atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if _, _, ok := r.BasicAuth(); ok {
			headerAuth.Store(true)
		}
		if r.Form.Get("client_id") != "" {
			paramsAuth.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"new","token_type":"Bearer","expires_in":3600}`)
	}))
	t.Cleanup(srv.Close)

	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:        "access",
		driveutil.DsKeyTokenType:    "Bearer",
		driveutil.DsKeyRefreshToken: "refresh",
		driveutil.DsKeyExpiresAt:    strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}

	share := &oauthHolderShare{}
	cred := driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"}
	headerReq := driveutil.OAuthRequest{Endpoint: oauth2.Endpoint{
		TokenURL: srv.URL, AuthStyle: oauth2.AuthStyleInHeader,
	}}
	paramsReq := driveutil.OAuthRequest{Endpoint: oauth2.Endpoint{
		TokenURL: srv.URL, AuthStyle: oauth2.AuthStyleInParams,
	}}

	vm := newDriveTestVM(t)
	u := testDriveUtils(vm, ds)
	u.oauth = share
	mustDefineGlobal(t, vm, "utils", u)
	mustDefineGlobal(t, vm, "headerReq", headerReq)
	mustDefineGlobal(t, vm, "paramsReq", paramsReq)
	mustDefineGlobal(t, vm, "cred", cred)
	if _, e := vm.Run(context.Background(), `
		headerHolder = utils.oauthLoad(headerReq, cred);
		paramsHolder = utils.oauthLoad(paramsReq, cred);
	`, ""); e != nil {
		t.Fatal(e)
	}

	if _, e := vm.Run(context.Background(), `headerHolder.refresh(); paramsHolder.refresh()`, ""); e != nil {
		t.Fatal(e)
	}
	if !headerAuth.Load() {
		t.Fatal("InHeader refresh did not send Basic credentials")
	}
	if !paramsAuth.Load() {
		t.Fatal("InParams refresh reused InHeader holder")
	}
}

func TestOAuthInitEmptyData(t *testing.T) {
	vm := newDriveTestVM(t)
	mustDefineGlobal(t, vm, "utils", testDriveUtils(vm, &memDriveData{}))
	if _, e := vm.Run(context.Background(), `
		utils.oauthInit({}, { endpoint: {} }, { clientID: "id", clientSecret: "secret" });
	`, ""); e != nil {
		t.Fatal(e)
	}
}

func TestOAuthInitConfigResultHolderIsCallable(t *testing.T) {
	ds := &memDriveData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		driveutil.DsKeyToken:     "access",
		driveutil.DsKeyTokenType: "Bearer",
		driveutil.DsKeyExpiresAt: strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	vm := newDriveTestVM(t)
	mustDefineGlobal(t, vm, "utils", testDriveUtils(vm, ds))
	mustDefineGlobal(t, vm, "req", driveutil.OAuthRequest{
		Endpoint:    oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		RedirectURL: "https://app/cb",
		Text:        "Connect",
	})
	mustDefineGlobal(t, vm, "cred", driveutil.OAuthCredentials{ClientID: "id", ClientSecret: "secret"})
	if _, e := vm.Run(context.Background(), `result = utils.oauthInitConfig(req, cred)`, ""); e != nil {
		t.Fatal(e)
	}

	got, e := vm.Run(context.Background(), `
		result.config.configured = false;
		result.config.oauth.principal = "nope";
		var tok = result.oauthHolder.token();
		tok.accessToken = "stolen";
		[
			tok.accessToken,
			String(result.config.configured),
			String(result.config.oauth.principal || ""),
			typeof result.oauthHolder.token,
			typeof tok.expiry.getTime
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "access|true||function|function" {
		t.Fatalf("oauthInitConfig result = %q", got.String())
	}

	copied, e := vm.Run(context.Background(), `({
		configured: true,
		oauth: {
			url: result.config.oauth.url,
			text: result.config.oauth.text,
			principal: "user"
		}
	})`, "")
	if e != nil {
		t.Fatal(e)
	}
	parsed := &driveutil.DriveInitConfig{}
	if e := copied.ParseInto(parsed); e != nil {
		t.Fatal(e)
	}
	if !parsed.Configured || parsed.OAuth == nil || parsed.OAuth.Principal != "user" || parsed.OAuth.URL == "" {
		t.Fatalf("copied config = %#v", parsed)
	}
}

func TestOAuthInitConfigValueReadableFromJS(t *testing.T) {
	vm := newDriveTestVM(t)
	oauth := &driveutil.OAuthConfig{URL: "https://example.com/auth", Text: "Connect"}
	cfg := &driveutil.DriveInitConfig{
		Configured: true,
		OAuth:      oauth,
		Value:      types.SM{"drive_id": "abc"},
	}
	mustDefineGlobal(t, vm, "cfg", cfg)
	got, e := vm.Run(context.Background(), `
		cfg.value.drive_id = cfg.value.drive_id + "-x";
		cfg.configured = false;
		cfg.oauth.principal = "user";
		[cfg.value.drive_id, String(cfg.configured), cfg.oauth.principal].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "abc|true|" {
		t.Fatalf("oauth init config = %q", got.String())
	}
	if cfg.Value["drive_id"] != "abc" {
		t.Fatalf("Go value mutated: %#v", cfg.Value)
	}
	if !cfg.Configured {
		t.Fatal("Go configured mutated")
	}
	if cfg.OAuth.Principal != "" {
		t.Fatalf("Go oauth mutated: %#v", cfg.OAuth)
	}

	cfgVal, e := vm.GetValue("cfg")
	if e != nil {
		t.Fatal(e)
	}
	parsed := &driveutil.DriveInitConfig{}
	if e := cfgVal.ParseInto(parsed); e != nil {
		t.Fatal(e)
	}
	if !parsed.Configured {
		t.Fatal("ParseInto configured")
	}
	if parsed.OAuth == nil || parsed.OAuth.Principal != "" || parsed.OAuth.URL != oauth.URL {
		t.Fatalf("ParseInto oauth = %#v", parsed.OAuth)
	}
	if parsed.Value["drive_id"] != "abc" {
		t.Fatalf("ParseInto value = %#v", parsed.Value)
	}
}

func TestScriptDriveGetReaderSurvivesVMReturn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "payload")
	}))
	t.Cleanup(srv.Close)

	d := newTestScriptDrive(t, fmt.Sprintf(`
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return {
        path: path,
        isDir: false,
        size: 7,
        modTime: -1,
        meta: { readable: true, selfThumbnail: true }
      };
    },
    list: function () { return []; },
    getReader: function () {
      return http(%q).body;
    },
    getThumbnail: function () {
      return http(%q).body;
    }
  }
);
`, srv.URL, srv.URL), nil, nil)

	entry, e := d.Get(context.Background(), "f")
	if e != nil {
		t.Fatal(e)
	}
	rc, e := entry.GetReader(context.Background(), -1, -1)
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(rc)
	if e != nil || string(got) != "payload" {
		t.Fatalf("getReader after VM return = %q %v", got, e)
	}
	if e := rc.Close(); e != nil {
		t.Fatal(e)
	}

	thumb, ok := entry.(interface {
		Thumbnail(context.Context) (types.IContentReader, error)
	})
	if !ok {
		t.Fatal("entry does not implement Thumbnail")
	}
	content, e := thumb.Thumbnail(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	trc, e := content.GetReader(context.Background(), -1, -1)
	if e != nil {
		t.Fatal(e)
	}
	got, e = io.ReadAll(trc)
	if e != nil || string(got) != "payload" {
		t.Fatalf("getThumbnail after VM return = %q %v", got, e)
	}
	if e := trc.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestScriptDriveGetReaderTempFileSurvivesVMReturn(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return { path: path, isDir: false, size: 5, modTime: -1, meta: { readable: true } };
    },
    list: function () { return []; },
    getReader: function () {
      var tmp = new TempFile();
      tmp.write(Bytes.fromString("hello"));
      tmp.seekTo(0, 0);
      return tmp;
    }
  }
);
`, nil, nil)
	entry, e := d.Get(context.Background(), "f")
	if e != nil {
		t.Fatal(e)
	}
	rc, e := entry.GetReader(context.Background(), -1, -1)
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(rc)
	if e != nil || string(got) != "hello" {
		t.Fatalf("TempFile getReader after VM return = %q %v", got, e)
	}
	if e := rc.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestScriptDriveGetURLReadsEntryData(t *testing.T) {
	d := newTestScriptDrive(t, `
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      return {
        path: path,
        isDir: false,
        size: 1,
        modTime: 1,
        data: { id: "file-1" },
        meta: { readable: true, writable: true, props: { kind: "doc" } }
      };
    },
    list: function () { return []; },
    getURL: function (entry) {
      return { url: "https://example.com/" + entry.data.id + "/" + entry.meta.props.kind };
    }
  }
);
`, nil, nil)
	entry, e := d.Get(context.Background(), "a.txt")
	if e != nil {
		t.Fatal(e)
	}
	u, e := entry.GetURL(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if u == nil || u.URL != "https://example.com/file-1/doc" {
		t.Fatalf("getURL = %#v", u)
	}
}

func TestScriptDriveCachePutEntryDetachesNestedProxyAcrossVMs(t *testing.T) {
	mgr := driveutil.NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	d, e := buildTestScriptDrive(`
defineDrive(
  { createInstance: function () { return {}; } },
  {
    get: function (path) {
      var hit = this.cache.getEntry(path);
      if (hit) {
        return { path: hit.path, isDir: false, size: hit.size, modTime: hit.modTime, meta: hit.meta };
      }
      var entry = {
        path: path,
        isDir: false,
        size: 1,
        modTime: 1,
        meta: { readable: true, writable: true, props: { nested: new Proxy({ kind: "doc" }, {}) } }
      };
      this.cache.putEntry(entry, "1h");
      this.cache.putEntries([entry], "1h");
      this.cache.putChildren("", [entry], "1h");
      return entry;
    },
    list: function () { return []; },
    getURL: function (entry) {
      return { url: "https://example.com/" + entry.meta.props.nested.kind };
    }
  }
);
`, nil, mgr, &s.VMPoolConfig{MaxTotal: 1, MaxIdle: 0, MinIdle: 0, IdleTime: 0})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = d.Dispose() })

	if _, e := d.Get(context.Background(), "a.txt"); e != nil {
		t.Fatal(e)
	}
	entry, e := d.Get(context.Background(), "a.txt")
	if e != nil {
		t.Fatal(e)
	}
	u, e := entry.GetURL(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if u == nil || u.URL != "https://example.com/doc" {
		t.Fatalf("getURL = %#v", u)
	}
}

func TestScriptDriveLimitedReadersSurvivePoolReturn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "abcdef") }))
	defer srv.Close()
	for _, maxIdle := range []int{0, 1} {
		for _, source := range []string{
			`const f=new TempFile(); f.write(Bytes.fromString("abcdef"));f.seekTo(0,SEEK_START);`,
			fmt.Sprintf(`const f=http(%q).body;`, srv.URL),
		} {
			t.Run(fmt.Sprintf("idle=%d/%s", maxIdle, source), func(t *testing.T) {
				d := newTestScriptDrivePool(t, `
     function reader(){`+source+`const r=f.limitReader(4).limitReader(8);r.read(new Bytes(1));return r;}
     defineDrive({createInstance(){return {}}},{
      get(path){return {path,isDir:false,meta:{readable:true,selfThumbnail:true}}},
      list(){return []},getReader:reader,getThumbnail:reader
     });
    `, &s.VMPoolConfig{MaxTotal: 1, MaxIdle: maxIdle})
				entry, e := d.Get(context.Background(), "x")
				if e != nil {
					t.Fatal(e)
				}
				r, e := entry.GetReader(context.Background(), -1, -1)
				if e != nil {
					t.Fatal(e)
				}
				check := func(r io.ReadCloser) {
					t.Helper()
					defer r.Close()
					b, e := io.ReadAll(r)
					if e != nil || string(b) != "bcd" {
						t.Fatalf("read=%q, %v", b, e)
					}
				}
				check(r)
				thumb, e := entry.(*scriptDriveEntry).Thumbnail(context.Background())
				if e != nil {
					t.Fatal(e)
				}
				r, e = thumb.GetReader(context.Background(), -1, -1)
				if e != nil {
					t.Fatal(e)
				}
				check(r)
			})
		}
	}
}
