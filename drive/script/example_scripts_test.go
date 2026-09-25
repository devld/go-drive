package script

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerSideExampleScriptsEvaluate(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "script-drives", "dropbox.js"),
		filepath.Join("..", "..", "script-drives", "qiniu.js"),
		filepath.Join("..", "..", "script-drives", "123pan.js"),
		filepath.Join("..", "..", "script-drives", "pcloud.js"),
		filepath.Join("..", "..", "script-drives", "yandex.js"),
		filepath.Join("..", "..", "docs", "script-drive-template.js"),
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			contents, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}

			vm := newDriveTestVM(t)
			if _, e = vm.Run(context.Background(), contents, filepath.Base(path)); e != nil {
				t.Fatalf("evaluate %s: %v", path, e)
			}
		})
	}
}

func TestAgentGuideCompleteExampleEvaluates(t *testing.T) {
	path := filepath.Join("..", "..", "script-drives", "AGENTS.md")
	contents, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}

	section := strings.Index(string(contents), "## 8. Minimal complete example")
	if section < 0 {
		t.Fatal("complete example section not found")
	}
	codeStart := strings.Index(string(contents[section:]), "```js\n")
	if codeStart < 0 {
		t.Fatal("complete example code block not found")
	}
	codeStart += section + len("```js\n")
	codeEnd := strings.Index(string(contents[codeStart:]), "\n```")
	if codeEnd < 0 {
		t.Fatal("complete example code block is not closed")
	}

	vm := newDriveTestVM(t)
	if _, e = vm.Run(context.Background(), contents[codeStart:codeStart+codeEnd], ""); e != nil {
		t.Fatalf("evaluate complete example: %v", e)
	}
}

func TestScriptDriveTemplateLifecycle(t *testing.T) {
	contents, e := os.ReadFile(filepath.Join("..", "..", "docs", "script-drive-template.js"))
	if e != nil {
		t.Fatal(e)
	}
	vm := newDriveTestVM(t)
	if _, e := vm.Run(context.Background(), contents, "template.js"); e != nil {
		t.Fatal(e)
	}
	// Stub OAuth so lifecycle argument validation never contacts the example endpoint.
	if _, e := vm.Run(context.Background(), `
		const config = {client_id: "id", client_secret: "secret", cache_ttl: "2h"};
		const holder = {};
		let initialized = false;
		function checkOAuth(request, credentials) {
			if (request.redirectUrl !== "https://local.test/callback" ||
				credentials.clientID !== "id" || credentials.clientSecret !== "secret") {
				throw new Error("incorrect OAuth arguments");
			}
		}
		const utils = {
			config: {oauthRedirectURI: "https://local.test/callback"},
			createCache() { return {}; },
			oauthInitConfig(request, credentials) {
				checkOAuth(request, credentials);
				return {config: {configured: true}};
			},
			oauthInit(data, request, credentials) {
				checkOAuth(request, credentials);
				if (data.code !== "code") throw new Error("incorrect init data");
				initialized = true;
			},
			oauthLoad(request, credentials) {
				checkOAuth(request, credentials);
				return holder;
			}
		};
		if (!__driveInitConfig(config, utils).configured) throw new Error("initConfig failed");
		__driveInit({code: "code"}, config, utils);
		if (!initialized) throw new Error("init failed");
		if (__driveCreate(config, utils).entryCacheTTL !== "2h") throw new Error("create failed");
		if (__drive_get("file.txt").path !== "file.txt") throw new Error("get path shifted");
	`, "template-lifecycle.js"); e != nil {
		t.Fatal(e)
	}
}

func TestQiniuListPagination(t *testing.T) {
	vm := newDriveTestVM(t)
	code, e := os.ReadFile("../../script-drives/qiniu.js")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), code, "qiniu.js"); e != nil {
		t.Fatal(e)
	}
	_, e = vm.Run(context.Background(), `
  let calls = 0;
  request = function(d, method, url) {
   const query = urlUtils.parse(url).searchParams;
   if (query.prefix[0] !== "dir/" || query.bucket[0] !== "test") throw new Error("lost query");
   calls++;
   if (calls === 1) {
    if (query.marker) throw new Error("unexpected initial marker");
    return {commonPrefixes:["dir/sub/"], items:[{key:"dir/a",fsize:1,putTime:0}], marker:"next+/=&"};
   }
   if (calls !== 2 || query.marker?.[0] !== "next+/=&") throw new Error("invalid next marker");
   return {items:[{key:"dir/b",fsize:2,putTime:0}], marker:""};
  };
  __driveCreate({bucket:"test",ak:"ak",sk:"sk",uploadURL:"https://upload.test",downloadBaseURL:"https://download.test"}, {createCache(){return {};}});
  const entries = __drive_list("dir");
  if (calls !== 2 || entries.map(e=>e.path).join(",") !== "dir/sub,dir/a,dir/b") throw new Error("incorrect merged pages");
 `, "pagination.js")
	if e != nil {
		t.Fatal(e)
	}
}

func Test123PanListPagination(t *testing.T) {
	vm := newDriveTestVM(t)
	code, e := os.ReadFile("../../script-drives/123pan.js")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), code, "123pan.js"); e != nil {
		t.Fatal(e)
	}
	mustDefineGlobal(t, vm, "panUtils", testDriveUtils(vm, &memDriveData{}))
	_, e = vm.Run(context.Background(), `
  let panCalls = 0;
  requestPan123 = function(d, method, route, params) {
    if (route !== "/api/v2/file/list" || params.parentFileId !== "0") {
      throw new Error("unexpected 123Pan list request");
    }
    panCalls++;
    if (panCalls === 1) {
      if (params.lastFileId !== 0) throw new Error("invalid initial 123Pan cursor");
      return {code: 0, data: {lastFileId: "9", fileList: [
        {filename: "one.txt", fileId: 11, type: 0, size: 1, trashed: 0}
      ]}};
    }
    if (panCalls !== 2 || params.lastFileId !== 9) {
      throw new Error("invalid next 123Pan cursor");
    }
    return {code: 0, data: {lastFileId: -1, fileList: [
      {filename: "two.txt", fileId: 12, type: 0, size: 2, trashed: 0},
      {filename: "deleted.txt", fileId: 13, type: 0, size: 3, trashed: 1}
    ]}};
  };
  __driveCreate({
    api_url: "https://open-api.test",
    access_token: "token",
    root_id: "0"
  }, panUtils);
  const entries = __drive_list("");
  if (panCalls !== 2 || entries.map(e => e.path).join(",") !== "one.txt,two.txt") {
    throw new Error("123Pan pagination or filtering failed");
  }
 `, "123pan-pagination")
	if e != nil {
		t.Fatal(e)
	}
}

func TestPCloudPathResolution(t *testing.T) {
	vm := newDriveTestVM(t)
	code, e := os.ReadFile("../../script-drives/pcloud.js")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), code, "pcloud.js"); e != nil {
		t.Fatal(e)
	}
	_, e = vm.Run(context.Background(), `
  requestPCloud = function(d, methodName, method, params) {
    if (methodName !== "listfolder" || method !== "GET") {
      throw new Error("unexpected pCloud request");
    }
    if (String(params.folderid) === "0") {
      return {result: 0, metadata: {contents: [
        {name: "photos", isfolder: 1, folderid: 21},
        {name: "root.txt", isfolder: 0, fileid: 22, size: 4}
      ]}};
    }
    if (String(params.folderid) === "21") {
      return {result: 0, metadata: {contents: [
        {name: "image.jpg", isfolder: 0, fileid: 23, size: 8}
      ]}};
    }
    throw new Error("unexpected pCloud folder");
  };
  __driveCreate({
    region: "us",
    access_token: "token",
    root_folder_id: "0"
  }, {createCache(){return {};}});
  const entries = __drive_list("photos");
  if (entries.length !== 1 || entries[0].path !== "photos/image.jpg" ||
      entries[0].data.id !== "23") {
    throw new Error("pCloud path resolution failed");
  }
  if (__drive_get("photos/image.jpg").size !== 8) {
    throw new Error("pCloud get failed");
  }
 `, "pcloud-paths")
	if e != nil {
		t.Fatal(e)
	}
}

func TestYandexListAndDownloadURL(t *testing.T) {
	vm := newDriveTestVM(t)
	code, e := os.ReadFile("../../script-drives/yandex.js")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), code, "yandex.js"); e != nil {
		t.Fatal(e)
	}
	_, e = vm.Run(context.Background(), `
  requestYandex = function(d, method, route, params) {
    if (method !== "GET") throw new Error("unexpected Yandex method");
    if (route === "/resources" && params.path === "/") {
      return {_embedded: {total: 1, items: [
        {name: "docs", type: "dir", path: "disk:/docs", modified: "2024-01-01T00:00:00Z"}
      ]}};
    }
    if (route === "/resources" && params.path === "/docs") {
      return {_embedded: {total: 1, items: [
        {name: "readme.md", type: "file", size: 5, path: "disk:/docs/readme.md"}
      ]}};
    }
    if (route === "/resources/download" && params.path === "/docs/readme.md") {
      return {href: "https://download.yandex.test/file"};
    }
    throw new Error("unexpected Yandex request");
  };
  const utils = {
    config: {oauthRedirectURI: "https://local.test/callback"},
    createCache(){return {};},
    oauthLoad(){return {token(){return {accessToken: "token"};}};}
  };
  __driveCreate({
    client_id: "id",
    client_secret: "secret",
    root_path: "/"
  }, utils);
  if (__drive_list("")[0].path !== "docs") throw new Error("Yandex list failed");
  if (__drive_getURL({path: "docs/readme.md"}).url !== "https://download.yandex.test/file") {
    throw new Error("Yandex download URL failed");
  }
 `, "yandex-paths")
	if e != nil {
		t.Fatal(e)
	}
}

func TestScriptDriveQueryBuildersUseURLUtils(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		filename string
		expr     string
		want     string
	}{
		{
			name:     "123pan",
			file:     "../../script-drives/123pan.js",
			filename: "123pan.js",
			expr: `appendPan123Query("https://example.test/api?existing=x#frag", {
				q: "hello world", marker: "next+/=&", ignored: null
			})`,
			want: "https://example.test/api?existing=x&marker=next%2B%2F%3D%26&q=hello+world#frag",
		},
		{
			name:     "pcloud",
			file:     "../../script-drives/pcloud.js",
			filename: "pcloud.js",
			expr: `pCloudURL(
				{apiURL: "https://api.pcloud.test", accessToken: "token"},
				"listfolder",
				{folderid: "0", name: "a b"}
			)`,
			want: "https://api.pcloud.test/listfolder?auth=token&folderid=0&name=a+b",
		},
		{
			name:     "yandex",
			file:     "../../script-drives/yandex.js",
			filename: "yandex.js",
			expr: `appendYandexQuery("https://api.example.test/resources", {
				path: "disk:/a b", overwrite: false, tag: ["one", "two"]
			})`,
			want: "https://api.example.test/resources?overwrite=false&path=disk%3A%2Fa+b&tag=one&tag=two",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vm := newDriveTestVM(t)
			code, e := os.ReadFile(test.file)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := vm.Run(context.Background(), code, test.filename); e != nil {
				t.Fatal(e)
			}
			value, e := vm.Run(context.Background(), test.expr, test.name+"-query.js")
			if e != nil {
				t.Fatal(e)
			}
			if got := value.String(); got != test.want {
				t.Fatalf("query URL = %q, want %q", got, test.want)
			}
		})
	}
}
