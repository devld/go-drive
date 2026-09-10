package script

import (
	"context"
	"encoding/json"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/task"
	"go-drive/drive/fs"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	err "go-drive/common/errors"
	"go-drive/common/types"
)

var (
	_ ConsoleStringer = jsObjDrive{}
	_ ConsoleStringer = jsObjEntry{}
	_ ConsoleStringer = jsObjBytes{}
	_ ConsoleStringer = jsObjReader{}
	_ ConsoleStringer = jsObjReadCloser{}
	_ ConsoleStringer = jsObjTempFile{}
	_ ConsoleStringer = (*httpHeadersJS)(nil)
	_ ConsoleStringer = (*httpResponseJS)(nil)
	_ ConsoleStringer = jsObjHttpFormData{}
	_ ConsoleStringer = jsObjHash{}
	_ ConsoleStringer = jsObjHmac{}
	_ json.Marshaler  = jsObjEntry{}
)

type inspectTestEntry struct {
	path    string
	name    string
	typ     types.EntryType
	size    int64
	modTime int64
	meta    types.EntryMeta
}

type totalChangingDrive struct {
	types.IDrive
}

func (d totalChangingDrive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	ctx.Total(99, true)
	ctx.Total(7, false)
	return d.IDrive.Save(ctx, path, size, override, reader)
}

func (e inspectTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, err.NewUnsupportedError()
}

func (e inspectTestEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return &types.ContentURL{URL: "https://example.com/file.txt", Proxy: true}, nil
}

func (e inspectTestEntry) Name() string          { return e.name }
func (e inspectTestEntry) Size() int64           { return e.size }
func (e inspectTestEntry) ModTime() int64        { return e.modTime }
func (e inspectTestEntry) Path() string          { return e.path }
func (e inspectTestEntry) Type() types.EntryType { return e.typ }
func (e inspectTestEntry) Meta() types.EntryMeta { return e.meta }
func (e inspectTestEntry) Drive() types.IDrive   { return nil }

func sampleInspectEntry() jsObjEntry {
	return jsObjEntry{e: inspectTestEntry{
		path:    "dir/file.txt",
		name:    "file.txt",
		typ:     types.TypeFile,
		size:    12,
		modTime: 1700000000000,
		meta:    types.EntryMeta{Readable: true, Writable: true, ThumbnailURL: "https://thumb"},
	}}
}

func TestEntryConsoleString(t *testing.T) {
	got := sampleInspectEntry().ConsoleString()
	want := `Entry { Path: "dir/file.txt", Type: "file", Name: "file.txt", Size: 12, ... }`
	if got != want {
		t.Fatalf("ConsoleString = %s, want %s", got, want)
	}
	if (jsObjEntry{}).ConsoleString() != "Entry {}" {
		t.Fatalf("nil Entry = %s", (jsObjEntry{}).ConsoleString())
	}
}

func TestEntryMarshalJSON(t *testing.T) {
	got, e := json.Marshal(sampleInspectEntry())
	if e != nil {
		t.Fatal(e)
	}
	want := `{"path":"dir/file.txt","name":"file.txt","type":"file","size":12,"modTime":1700000000000,"meta":{"readable":true,"writable":true,"thumbnailUrl":"https://thumb","selfThumbnail":false,"props":null}}`
	if string(got) != want {
		t.Fatalf("MarshalJSON = %s, want %s", got, want)
	}

	nilJSON, e := json.Marshal(jsObjEntry{})
	if e != nil {
		t.Fatal(e)
	}
	if string(nilJSON) != "null" {
		t.Fatalf("nil Entry JSON = %s, want null", nilJSON)
	}
}

func TestFormatConsoleArgInspectsGoHandles(t *testing.T) {
	root := newPoolTestVM(t)
	vm := root
	t.Cleanup(func() { _ = vm.Dispose() })
	entry := sampleInspectEntry()
	entry.ClassHost = NewClassHost(vm)
	mustDefineGlobal(t, vm, "entry", entry)
	mustDefineGlobal(t, vm, "entries", []jsObjEntry{entry})
	mustDefineGlobal(t, vm, "buf", []byte("hello"))
	mustDefineGlobal(t, vm, "headers", newHttpHeaders(vm, http.Header{"Content-Type": []string{"text/plain"}}))
	mustDefineGlobal(t, vm, "resp", newHttpResponse(vm, &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Length": []string{"5"}},
		Body:       http.NoBody,
	}))

	if got := evalConsoleArg(t, vm, `entry`); got != entry.ConsoleString() {
		t.Fatalf("entry log = %s, want %s", got, entry.ConsoleString())
	}

	stringified := evalConsoleArg(t, vm, `JSON.stringify(entry)`)
	if !strings.Contains(stringified, `"path":"dir/file.txt"`) || !strings.Contains(stringified, `"modTime":1700000000000`) || !strings.Contains(stringified, `"readable":true`) {
		t.Fatalf("JSON.stringify(entry) = %s", stringified)
	}

	wantList := "[ " + entry.ConsoleString() + " ]"
	if got := evalConsoleArg(t, vm, `entries`); got != wantList {
		t.Fatalf("entries log = %s, want %s", got, wantList)
	}

	if got := evalConsoleArg(t, vm, `[entry]`); got != wantList {
		t.Fatalf("js array log = %s, want %s", got, wantList)
	}

	if got := evalConsoleArg(t, vm, `buf`); got != "Bytes { Len: 5 }" {
		t.Fatalf("Bytes log = %s, want length only", got)
	}
	if got := evalConsoleArg(t, vm, `buf.toString()`); got != "hello" {
		t.Fatalf("Bytes.toString() = %s, want content", got)
	}

	if got := evalConsoleArg(t, vm, `headers`); got != "HttpHeaders { Len: 1, ... }" {
		t.Fatalf("headers log = %s", got)
	}

	wantResp := "HttpResponse { Status: 200, BodySize: 5, ... }"
	if got := evalConsoleArg(t, vm, `resp`); got != wantResp {
		t.Fatalf("response log = %s, want %s", got, wantResp)
	}

	if got := evalConsoleArg(t, vm, `new HttpFormData()`); got != "HttpFormData { Len: 0 }" {
		t.Fatalf("HttpFormData log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `new Hash("md5")`); got != "Hash { Size: 16, ... }" {
		t.Fatalf("Hash log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `entry.meta`); !jsonEqual(got, `{"readable":true,"writable":true,"thumbnailUrl":"https://thumb","selfThumbnail":false,"props":{}}`) {
		t.Fatalf("EntryMeta log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `entry.getUrl()`); !jsonEqual(got, `{"url":"https://example.com/file.txt","header":{},"proxy":true,"downloadFileName":""}`) {
		t.Fatalf("ContentURL log = %s", got)
	}
	got, e := vm.Run(context.Background(), `
		var meta = entry.meta;
		var url = entry.getUrl();
		meta.extra = true;
		url.extra = true;
		[
			meta instanceof Object && meta.constructor === Object,
			url instanceof Object && url.constructor === Object,
			typeof meta.extra,
			typeof url.extra
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|undefined|undefined" {
		t.Fatalf("Entry readonly values = %q", got.String())
	}

	tmp := evalConsoleArg(t, vm, `new TempFile()`)
	if tmp != "TempFile { Size: 0, ... }" {
		t.Fatalf("TempFile log = %s", tmp)
	}
}

func TestDriveAndEntryAreHostClasses(t *testing.T) {
	vm := newPoolTestVM(t)
	entry := sampleInspectEntry()
	entry.ClassHost = NewClassHost(vm)
	mustDefineGlobal(t, vm, "entry", entry)
	mustDefineGlobal(t, vm, "drive", notFoundDrive{})

	if _, e := vm.Run(context.Background(), `new Drive()`, ""); e == nil || !strings.Contains(e.Error(), "Drive cannot be constructed from JavaScript") {
		t.Fatalf("new Drive = %v", e)
	}
	if _, e := vm.Run(context.Background(), `new Entry()`, ""); e == nil || !strings.Contains(e.Error(), "Entry cannot be constructed from JavaScript") {
		t.Fatalf("new Entry = %v", e)
	}

	got, e := vm.Run(context.Background(), `
		function tag(v) { return Object.prototype.toString.call(v); }
		var pathDesc = Object.getOwnPropertyDescriptor(Entry.prototype, "path");
		[
			drive instanceof Drive,
			entry instanceof Entry,
			drive.get === Drive.prototype.get,
			typeof pathDesc.get === "function" && pathDesc.set === undefined,
			entry.path === "dir/file.txt",
			typeof entry.path === "string",
			!Object.prototype.hasOwnProperty.call(entry, "path"),
			tag(drive),
			tag(entry),
			typeof Drive,
			typeof Entry
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|true|true|true|true|true|true|[object Drive]|[object Entry]|function|function" {
		t.Fatalf("Drive/Entry class = %q", got.String())
	}
}

func TestDriveSaveReportsProgressOnce(t *testing.T) {
	root := t.TempDir()
	d, e := fs.NewDrive(context.Background(), types.SM{"path": root}, driveutil.DriveUtils{
		Config: common.Config{FreeFs: true},
	})
	if e != nil {
		t.Fatal(e)
	}
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "drive", totalChangingDrive{IDrive: d})
	ctx := task.NewTaskContext(context.Background())
	ctx.Total(5, true)
	progress := NewProgressReporter(ctx, true, false)
	mustDefineGlobal(t, vm, "progress", progress)
	if _, e := vm.Run(ctx, `
			const tmp = new TempFile();
			tmp.write(Bytes.fromString("hello"));
			tmp.seekTo(0, SEEK_START);
			drive.save("test.txt", 5, true, tmp, progress);
	`, ""); e != nil {
		t.Fatal(e)
	}
	if ctx.GetProgress() != 5 || ctx.GetTotal() != 5 {
		t.Fatalf("progress = %d/%d, want 5/5", ctx.GetProgress(), ctx.GetTotal())
	}
	data, e := os.ReadFile(filepath.Join(root, "test.txt"))
	if e != nil || string(data) != "hello" {
		t.Fatalf("saved content = %q, error = %v", data, e)
	}
}

func TestEntriesTreeEditableAndFlattenable(t *testing.T) {
	vm := newPoolTestVM(t)
	entry := sampleInspectEntry()
	entry.ClassHost = NewClassHost(vm)
	mustDefineGlobal(t, vm, "entry", entry)
	_, e := vm.Run(context.Background(), `
   const tree = buildEntriesTree(entry);
   tree.excluded = true;
   if (flattenEntriesTree(tree).length !== 0) throw new Error("excluded node retained");
   const first = {entry, children: []};
   const second = {...first};
   tree.children.push(first, second);
   tree.children.reverse();
   const filtered = flattenEntriesTree(tree);
   if (filtered.length !== 2 || filtered[0] !== second || filtered[1] !== first)
     throw new Error("edits or identity lost");
   tree.excluded = false;
   const pre = flattenEntriesTree(tree);
   const post = flattenEntriesTree(tree, true);
   if (pre[0] !== tree || post[2] !== tree) throw new Error("wrong traversal order");
   tree.children = [];
   if (flattenEntriesTree(tree).length !== 1) throw new Error("children replacement ignored");
   tree.children.push(tree);
   let rejected = false;
   try { flattenEntriesTree(tree); } catch (e) { rejected = e instanceof TypeError; }
   if (!rejected) throw new Error("cycle accepted");
 `, "tree-test.js")
	if e != nil {
		t.Fatal(e)
	}
}

func TestConvertEntryTreeNodeMaterializesChildren(t *testing.T) {
	vm := newPoolTestVM(t)
	rootEntry := sampleInspectEntry().e
	childEntry := inspectTestEntry{
		path: "dir/child.txt",
		name: "child.txt",
		typ:  types.TypeFile,
	}
	tree := convertEntryTreeNode(vm, driveutil.EntryTreeNode{
		Entry: rootEntry,
		Children: []driveutil.EntryTreeNode{
			{Entry: childEntry},
		},
	})
	mustDefineGlobal(t, vm, "tree", tree)
	got, e := vm.Run(context.Background(), `
		const child = tree.children[0];
		const plain = child instanceof Object && child.constructor === Object;
		child.excluded = true;
		[plain, child.entry.path, child.excluded].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "true|dir/child.txt|true" {
		t.Fatalf("child tree = %q", got.String())
	}
}
