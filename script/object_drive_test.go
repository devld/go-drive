package script

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	err "go-drive/common/errors"
	"go-drive/common/types"
)

var (
	_ ConsoleStringer = Drive{}
	_ ConsoleStringer = Entry{}
	_ ConsoleStringer = entryMeta{}
	_ ConsoleStringer = contentURL{}
	_ ConsoleStringer = Bytes{}
	_ ConsoleStringer = Reader{}
	_ ConsoleStringer = ReadCloser{}
	_ ConsoleStringer = TempFile{}
	_ ConsoleStringer = httpHeaders{}
	_ ConsoleStringer = httpResponse{}
	_ ConsoleStringer = formData{}
	_ ConsoleStringer = Hasher{}
	_ ConsoleStringer = Context{}
	_ ConsoleStringer = TaskCtx{}
	_ ConsoleStringer = contextWithTimeout{}
	_ ConsoleStringer = (*locker)(nil)
	_ ConsoleStringer = entryTreeNode{}
	_ json.Marshaler  = Entry{}
)

type inspectTestEntry struct {
	path    string
	name    string
	typ     types.EntryType
	size    int64
	modTime int64
	meta    types.EntryMeta
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

func sampleInspectEntry() Entry {
	return NewEntry(inspectTestEntry{
		path:    "dir/file.txt",
		name:    "file.txt",
		typ:     types.TypeFile,
		size:    12,
		modTime: 1700000000000,
		meta:    types.EntryMeta{Readable: true, Writable: true, ThumbnailURL: "https://thumb"},
	})
}

func TestEntryConsoleString(t *testing.T) {
	got := sampleInspectEntry().ConsoleString()
	want := `Entry { Path: "dir/file.txt", Type: "file", Name: "file.txt", Size: 12, ... }`
	if got != want {
		t.Fatalf("ConsoleString = %s, want %s", got, want)
	}
	if (Entry{}).ConsoleString() != "Entry {}" {
		t.Fatalf("nil Entry = %s", (Entry{}).ConsoleString())
	}
}

func TestEntryMarshalJSON(t *testing.T) {
	got, e := json.Marshal(sampleInspectEntry())
	if e != nil {
		t.Fatal(e)
	}
	want := `{"Path":"dir/file.txt","Name":"file.txt","Type":"file","Size":12,"ModTime":1700000000000,"Meta":{"Readable":true,"Writable":true,"ThumbnailURL":"https://thumb","SelfThumbnail":false,"Props":null}}`
	if string(got) != want {
		t.Fatalf("MarshalJSON = %s, want %s", got, want)
	}

	nilJSON, e := json.Marshal(Entry{})
	if e != nil {
		t.Fatal(e)
	}
	if string(nilJSON) != "null" {
		t.Fatalf("nil Entry JSON = %s, want null", nilJSON)
	}
}

func TestFormatConsoleArgInspectsGoHandles(t *testing.T) {
	root := newPoolTestVM(t)
	vm := root.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	entry := sampleInspectEntry()
	vm.Set("entry", entry)
	vm.Set("entries", []Entry{entry})
	vm.Set("buf", NewBytes(vm, "hello"))
	vm.Set("headers", &httpHeaders{vm: vm, h: http.Header{"Content-Type": []string{"text/plain"}}})
	vm.Set("resp", &httpResponse{
		vm:      vm,
		Status:  200,
		Headers: &httpHeaders{vm: vm, h: http.Header{"Content-Length": []string{"5"}}},
	})

	if got := evalConsoleArg(t, vm, `entry`); got != entry.ConsoleString() {
		t.Fatalf("entry log = %s, want %s", got, entry.ConsoleString())
	}

	stringified := evalConsoleArg(t, vm, `JSON.stringify(entry)`)
	if !strings.Contains(stringified, `"Path":"dir/file.txt"`) || !strings.Contains(stringified, `"ModTime":1700000000000`) || !strings.Contains(stringified, `"Readable":true`) {
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
	if got := evalConsoleArg(t, vm, `buf.String()`); got != "hello" {
		t.Fatalf("Bytes.String() = %s, want content", got)
	}

	if got := evalConsoleArg(t, vm, `headers`); got != "HttpHeaders { Len: 1, ... }" {
		t.Fatalf("headers log = %s", got)
	}

	wantResp := "HttpResponse { Status: 200, BodySize: 5, ... }"
	if got := evalConsoleArg(t, vm, `resp`); got != wantResp {
		t.Fatalf("response log = %s, want %s", got, wantResp)
	}

	if got := evalConsoleArg(t, vm, `newContext()`); got != "Context { ... }" {
		t.Fatalf("Context log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `newTaskCtx(newContext())`); got != "TaskCtx { ... }" {
		t.Fatalf("TaskCtx log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `newLocker()`); got != "Locker { ... }" {
		t.Fatalf("Locker log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `newFormData()`); got != "HttpFormData { Len: 0 }" {
		t.Fatalf("HttpFormData log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `encUtils.newHash(HASH.MD5)`); got != "Hasher { Size: 16, ... }" {
		t.Fatalf("Hasher log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `entry.Meta()`); got != "EntryMeta { Readable: true, Writable: true, ... }" {
		t.Fatalf("EntryMeta log = %s", got)
	}
	if got := evalConsoleArg(t, vm, `entry.GetURL(newContext())`); got != `ContentURL { URL: "https://example.com/file.txt", ... }` {
		t.Fatalf("ContentURL log = %s", got)
	}

	tmp := evalConsoleArg(t, vm, `newTempFile()`)
	if tmp != "TempFile { Size: 0, ... }" {
		t.Fatalf("TempFile log = %s", tmp)
	}
	if got := evalConsoleArg(t, vm, `(function() { var c = newContextWithTimeout(newContext(), 1); c.Cancel(); return c; })()`); got != "ContextWithTimeout { ... }" {
		t.Fatalf("ContextWithTimeout log = %s", got)
	}
}
