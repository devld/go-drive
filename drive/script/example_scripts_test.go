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
