package script

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
)

func testConfig(t *testing.T) common.Config {
	t.Helper()
	return common.Config{
		DataDir:           t.TempDir(),
		DrivesDir:         "script-drives",
		DriveUploadersDir: "drive-uploaders",
	}
}

func TestRegisterAllScriptDrivesRegistersExpandedFactory(t *testing.T) {
	config := testConfig(t)
	drivesDir, e := config.GetDir(config.DrivesDir, true)
	if e != nil {
		t.Fatal(e)
	}
	content := `// @name Example Cloud
// @version 1.0.0
// @description Example description.

defineDrive(
  {
    configForm: [{ Label: "Token", Field: "token", Type: "password" }],
    createInstance: function () { return {}; }
  },
  {
    get: function () { return { Path: "x", IsDir: false, Size: 1, ModTime: -1 }; },
    list: function () { return []; },
    getURL: function () { return { URL: "https://example.com" }; }
  }
);
`
	if e := os.WriteFile(filepath.Join(drivesDir, "example.js"), []byte(content), 0644); e != nil {
		t.Fatal(e)
	}
	driveRegistry := driveutil.NewDriveRegistry(registry.NewComponentHolder())
	t.Cleanup(func() { _ = driveRegistry.ReplaceDriveGroup("script/", nil) })

	if e := RegisterAllScriptDrives(context.Background(), config, driveRegistry); e != nil {
		t.Fatalf("RegisterAllScriptDrives() error = %v", e)
	}
	factory := driveRegistry.GetDrive("script/example")
	if factory == nil {
		t.Fatal("expanded Script Drive factory was not registered")
	}
	if driveRegistry.GetDrive("script") != nil {
		t.Fatal("generic Script Drive factory is still registered")
	}
	if factory.DisplayName != "Example Cloud" {
		t.Fatalf("display name = %q", factory.DisplayName)
	}
	if factory.README != "Example description." || len(factory.ConfigForm) != 2 ||
		factory.ConfigForm[0].Field != "token" || factory.ConfigForm[1].Field != poolConfigField {
		t.Fatalf("expanded config form = %#v", factory.ConfigForm)
	}
}

func TestParseDriveScriptMeta(t *testing.T) {
	meta, ok, e := parseDriveScriptMeta([]byte("// @name Example Cloud\n// @version 1.2.3\n// @uploader custom-uploader.js\n// @description Example description.\n//\n// More details.\n\nfunction example() {}\n"), "example.js")
	if e != nil {
		t.Fatalf("parseDriveScriptMeta() error = %v", e)
	}
	if !ok {
		t.Fatal("parseDriveScriptMeta() ok = false, want true")
	}

	if meta.Name != "example" || meta.DisplayName != "Example Cloud" ||
		meta.Version != "1.2.3" || meta.Uploader != "custom-uploader.js" ||
		meta.Description != "Example description.\n\nMore details." {
		t.Fatalf("parseDriveScriptMeta() = %#v", meta)
	}
}

func TestParseDriveScriptMetaIgnoresCommentsBeforeDescription(t *testing.T) {
	meta, ok, e := parseDriveScriptMeta([]byte("// leftover first-line title\n// @name Example Cloud\n// leftover description\n// @version 1.0.0\n// @description Official description.\n"), "example.js")
	if e != nil {
		t.Fatalf("parseDriveScriptMeta() error = %v", e)
	}
	if !ok {
		t.Fatal("parseDriveScriptMeta() ok = false, want true")
	}
	if meta.DisplayName != "Example Cloud" || meta.Description != "Official description." {
		t.Fatalf("parseDriveScriptMeta() = %#v", meta)
	}
}

func TestParseDriveScriptMetaIgnoresScriptsWithoutName(t *testing.T) {
	meta, ok, e := parseDriveScriptMeta([]byte("// Legacy Drive\n// Legacy description\n\nfunction legacy() {}\n"), "legacy.js")
	if e != nil {
		t.Fatalf("parseDriveScriptMeta() error = %v", e)
	}
	if ok {
		t.Fatal("parseDriveScriptMeta() ok = true, want false")
	}
	if meta.Name != "legacy" {
		t.Fatalf("parseDriveScriptMeta() name = %q, want legacy", meta.Name)
	}
}

func TestParseDriveScriptMetaIgnoresScriptsWithoutVersion(t *testing.T) {
	_, ok, e := parseDriveScriptMeta([]byte("// @name Example Cloud\n// @description No version\n"), "example.js")
	if e != nil {
		t.Fatalf("parseDriveScriptMeta() error = %v", e)
	}
	if ok {
		t.Fatal("parseDriveScriptMeta() ok = true, want false")
	}
}

func TestSyncDriveScriptsFromRepositoryReadsMetadataAndUploader(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "example.js", DownloadURL: server.URL + "/example.js"},
			{Name: "custom-uploader.js", DownloadURL: server.URL + "/custom-uploader.js"},
			{Name: "AGENTS.md", DownloadURL: server.URL + "/AGENTS.md"},
			{Name: "other-uploader.js", DownloadURL: server.URL + "/other-uploader.js"},
		})
	})
	mux.HandleFunc("/example.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name Example Cloud\n// @version 2.0.0\n// @uploader custom-uploader.js\n// @description Remote description\n\nfunction example() {}\n"))
	})
	mux.HandleFunc("/custom-uploader.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("function uploader() {}\n"))
	})
	mux.HandleFunc("/AGENTS.md", func(w http.ResponseWriter, _ *http.Request) {
		t.Error("downloaded non-js repository file")
	})
	mux.HandleFunc("/other-uploader.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("function unused() {}\n"))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	config := testConfig(t)
	result, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), config, server.URL+"/repo")
	if e != nil {
		t.Fatalf("SyncDriveScriptsFromRepository() error = %v", e)
	}
	if !result.Ready || len(result.Scripts) != 1 {
		t.Fatalf("repository = %#v", result)
	}
	got := result.Scripts[0]
	if got.Name != "example" || got.DisplayName != "Example Cloud" ||
		got.Description != "Remote description" || got.Version != "2.0.0" ||
		got.DriveUploaderURL == "" {
		t.Fatalf("metadata = %#v", got)
	}

	_, filesDir, _, e := repositoryCachePaths(config, false)
	if e != nil {
		t.Fatal(e)
	}
	if filepath.Base(filesDir) != ".repo" {
		t.Fatalf("available scripts dir = %q, want .repo", filesDir)
	}
	if _, e := os.Stat(filepath.Join(filesDir, "example.js")); e != nil {
		t.Fatalf("cached drive missing: %v", e)
	}
	if _, e := os.Stat(filepath.Join(filesDir, "custom-uploader.js")); e != nil {
		t.Fatalf("cached uploader missing: %v", e)
	}
	if _, e := os.Stat(filepath.Join(filesDir, "other-uploader.js")); !os.IsNotExist(e) {
		t.Fatal("unreferenced uploader was cached")
	}
}

func TestSyncDriveScriptsFromRepositoryContinuesWhenOneScriptFails(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "broken.js", DownloadURL: server.URL + "/broken.js"},
			{Name: "ok.js", DownloadURL: server.URL + "/ok.js"},
		})
	})
	mux.HandleFunc("/broken.js", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/ok.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name OK\n// @version 1.0.0\n"))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	result, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), testConfig(t), server.URL+"/repo")
	if e != nil {
		t.Fatalf("SyncDriveScriptsFromRepository() error = %v", e)
	}
	if len(result.Scripts) != 1 || result.Scripts[0].Name != "ok" {
		t.Fatalf("scripts = %#v", result.Scripts)
	}
}

func TestListDriveScriptsReadsVersion(t *testing.T) {
	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "example.js"), []byte("// @name Example\n// @version 3.0.0\n// @description Description\n\nfunction example() {}\n"), 0644); e != nil {
		t.Fatal(e)
	}

	items, e := ListDriveScripts(config)
	if e != nil {
		t.Fatalf("ListDriveScripts() error = %v", e)
	}
	if len(items) != 1 || items[0].Version != "3.0.0" {
		t.Fatalf("installed metadata = %#v", items)
	}
}

func TestGetDriveScriptConfigForm(t *testing.T) {
	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	const source = `// @name Example
// @version 1.0.0
// @description Example description

defineDrive({
  configForm: [{ Label: "Token", Field: "token", Type: "password", Required: true }],
  createInstance: function () { return {}; }
}, {
  get: function () { return { Path: "x", IsDir: false, Size: 1, ModTime: -1 }; },
  list: function () { return []; },
  getURL: function () { return { URL: "https://example.com" }; }
});
`
	if e := os.WriteFile(filepath.Join(drivesDir, "example.js"), []byte(source), 0644); e != nil {
		t.Fatal(e)
	}

	form, e := GetDriveScriptConfigForm(context.Background(), config, "example")
	if e != nil {
		t.Fatalf("GetDriveScriptConfigForm() error = %v", e)
	}
	if len(form) != 1 || form[0].Field != "token" || form[0].Type != "password" {
		t.Fatalf("form = %#v", form)
	}

	if e := validateScriptForm([]types.FormItem{{Field: "_reserved"}}); e == nil {
		t.Fatal("expected reserved field validation error")
	}
}

func TestReadDriveScriptFileStaysWithinScriptsRoot(t *testing.T) {
	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "inside.js"), []byte("inside"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(config.DataDir, "outside.js"), []byte("outside"), 0644); e != nil {
		t.Fatal(e)
	}

	content, e := readDriveScriptFile("inside.js", config)
	if e != nil || string(content) != "inside" {
		t.Fatalf("read inside.js = %q, %v", content, e)
	}
	if _, e := readDriveScriptFile("../outside.js", config); e == nil {
		t.Fatal("expected parent traversal to be rejected")
	}
	if _, e := readDriveScriptFile(filepath.Join(config.DataDir, "outside.js"), config); e == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestGetSaveUninstallDriveScriptRoundTrip(t *testing.T) {
	config := testConfig(t)
	content := DriveScriptContent{Drive: "drive-body", Uploader: "uploader-body"}
	if e := SaveDriveScript(config, "example", content); e != nil {
		t.Fatalf("SaveDriveScript() error = %v", e)
	}

	got, e := GetDriveScript(config, "example")
	if e != nil {
		t.Fatalf("GetDriveScript() error = %v", e)
	}
	if got != content {
		t.Fatalf("GetDriveScript() = %#v, want %#v", got, content)
	}

	if e := UninstallDriveScript(config, "example"); e != nil {
		t.Fatalf("UninstallDriveScript() error = %v", e)
	}
	if _, e := GetDriveScript(config, "example"); e == nil {
		t.Fatal("GetDriveScript() after uninstall error = nil, want error")
	}
}

func TestDriveScriptManagementRejectsPathTraversal(t *testing.T) {
	config := testConfig(t)
	outside := filepath.Join(config.DataDir, "outside.js")
	if e := os.WriteFile(outside, []byte("secret"), 0644); e != nil {
		t.Fatal(e)
	}

	names := []string{
		"../outside",
		"..\\outside",
		"/tmp/outside",
		filepath.Join(config.DataDir, "outside"),
		"..",
		"",
	}
	for _, name := range names {
		if _, e := GetDriveScript(config, name); e == nil {
			t.Fatalf("GetDriveScript(%q) error = nil, want error", name)
		}
		if e := SaveDriveScript(config, name, DriveScriptContent{Drive: "pwned"}); e == nil {
			t.Fatalf("SaveDriveScript(%q) error = nil, want error", name)
		}
		if e := UninstallDriveScript(config, name); e == nil {
			t.Fatalf("UninstallDriveScript(%q) error = nil, want error", name)
		}
		if e := InstallDriveScript(config, name); e == nil {
			t.Fatalf("InstallDriveScript(%q) error = nil, want error", name)
		}
	}

	got, e := os.ReadFile(outside)
	if e != nil || string(got) != "secret" {
		t.Fatalf("outside.js = %q, %v; want unmodified", got, e)
	}
}

func TestInstallDriveScriptCopiesCachedFiles(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "example.js", DownloadURL: server.URL + "/drive.js"},
			{Name: "example-uploader.js", DownloadURL: server.URL + "/uploader.js"},
		})
	})
	mux.HandleFunc("/drive.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name New Drive\n// @version 2.0.0\n// @uploader example-uploader.js\n"))
	})
	mux.HandleFunc("/uploader.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("new uploader"))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	config := testConfig(t)
	if _, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), config, server.URL+"/repo"); e != nil {
		t.Fatal(e)
	}
	if e := InstallDriveScript(config, "example"); e != nil {
		t.Fatalf("InstallDriveScript() error = %v", e)
	}

	content, e := os.ReadFile(filepath.Join(config.DataDir, config.DrivesDir, "example.js"))
	if e != nil {
		t.Fatal(e)
	}
	if string(content) != "// @name New Drive\n// @version 2.0.0\n// @uploader example-uploader.js\n" {
		t.Fatalf("installed drive = %q", content)
	}
	content, e = os.ReadFile(filepath.Join(config.DataDir, config.DriveUploadersDir, "example.js"))
	if e != nil {
		t.Fatal(e)
	}
	if string(content) != "new uploader" {
		t.Fatalf("installed uploader = %q", content)
	}
}

func TestInstallDriveScriptRemovesStaleUploader(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "example.js", DownloadURL: server.URL + "/drive.js"},
		})
	})
	mux.HandleFunc("/drive.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name Drive without uploader\n// @version 1.0.0\n"))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	config := testConfig(t)
	uploadersDir := filepath.Join(config.DataDir, config.DriveUploadersDir)
	if e := os.MkdirAll(uploadersDir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(uploadersDir, "example.js"), []byte("old uploader"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), config, server.URL+"/repo"); e != nil {
		t.Fatal(e)
	}
	if e := InstallDriveScript(config, "example"); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(uploadersDir, "example.js")); !os.IsNotExist(e) {
		t.Fatal("stale uploader was not removed")
	}
}

func TestInstallDriveScriptRejectsOversizedScriptWithoutReplacingFile(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "example.js", DownloadURL: server.URL + "/drive.js"},
		})
	})
	mux.HandleFunc("/drive.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), int(maxScriptSize)+1))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	oldDrive := []byte("old drive")
	if e := os.WriteFile(filepath.Join(drivesDir, "example.js"), oldDrive, 0644); e != nil {
		t.Fatal(e)
	}

	result, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), config, server.URL+"/repo")
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Scripts) != 0 {
		t.Fatalf("oversized script was cached: %#v", result.Scripts)
	}
	if e := InstallDriveScript(config, "example"); e == nil {
		t.Fatal("InstallDriveScript() error = nil, want error")
	}
	content, e := os.ReadFile(filepath.Join(drivesDir, "example.js"))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(content, oldDrive) {
		t.Fatalf("drive changed after oversized update: %q", content)
	}
}

func TestListDriveScriptsIgnoresRepositoryCache(t *testing.T) {
	config := testConfig(t)
	if e := os.MkdirAll(filepath.Join(config.DataDir, config.DrivesDir, ".repo"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(config.DataDir, config.DrivesDir, ".repo", "example.js"), []byte("// Cached\n"), 0644); e != nil {
		t.Fatal(e)
	}

	items, e := ListDriveScripts(config)
	if e != nil {
		t.Fatal(e)
	}
	if len(items) != 0 {
		t.Fatalf("listed cached scripts as installed: %#v", items)
	}
}

func TestListDriveScriptsIgnoresScriptsWithoutNameOrVersion(t *testing.T) {
	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "legacy.js"), []byte("// Legacy Drive\n// no directives\n"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "nameless.js"), []byte("// @version 1.0.0\n"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "unversioned.js"), []byte("// @name Unversioned\n"), 0644); e != nil {
		t.Fatal(e)
	}

	items, e := ListDriveScripts(config)
	if e != nil {
		t.Fatal(e)
	}
	if len(items) != 0 {
		t.Fatalf("listed invalid scripts: %#v", items)
	}
}

func TestListAllDriveScriptsMergesInstalledAndAvailable(t *testing.T) {
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/repo", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]driveRepositoryListResp{
			{Name: "example.js", DownloadURL: server.URL + "/example.js"},
			{Name: "extra.js", DownloadURL: server.URL + "/extra.js"},
		})
	})
	mux.HandleFunc("/example.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name Example Cloud\n// @version 2.0.0\n"))
	})
	mux.HandleFunc("/extra.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("// @name Extra Drive\n// @version 1.0.0\n"))
	})
	server = httptest.NewServer(mux)
	defer server.Close()

	config := testConfig(t)
	drivesDir := filepath.Join(config.DataDir, config.DrivesDir)
	if e := os.MkdirAll(drivesDir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "example.js"), []byte("// @name Example Cloud\n// @version 1.0.0\n"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(drivesDir, "local.js"), []byte("// @name Local Drive\n// @version 3.0.0\n"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := SyncDriveScriptsFromRepository(task.NewTaskContext(context.Background()), config, server.URL+"/repo"); e != nil {
		t.Fatal(e)
	}

	result, e := ListAllDriveScripts(config)
	if e != nil {
		t.Fatalf("ListAllDriveScripts() error = %v", e)
	}
	if !result.Ready || len(result.Scripts) != 3 {
		t.Fatalf("scripts = %#v", result)
	}

	byName := make(map[string]DriveScriptListItem, len(result.Scripts))
	for _, item := range result.Scripts {
		byName[item.Name] = item
	}

	example := byName["example"]
	if example.Version != "2.0.0" || example.Installed == nil || example.Installed.Version != "1.0.0" || !example.UpdateAvailable {
		t.Fatalf("example = %#v", example)
	}
	local := byName["local"]
	if local.Installed == nil || local.Version != "3.0.0" || local.UpdateAvailable || local.DriveURL != "" {
		t.Fatalf("local = %#v", local)
	}
	extra := byName["extra"]
	if extra.Installed != nil || extra.Version != "1.0.0" || extra.UpdateAvailable {
		t.Fatalf("extra = %#v", extra)
	}
}
