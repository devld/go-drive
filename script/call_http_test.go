package script

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type capturedRequest struct {
	mu               sync.Mutex
	contentLength    int64
	transferEncoding []string
	body             string
}

func (c *capturedRequest) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contentLength = r.ContentLength
	c.transferEncoding = append([]string(nil), r.TransferEncoding...)
	c.body = string(body)
	w.WriteHeader(http.StatusNoContent)
}

func startCaptureServer(t *testing.T) (*httptest.Server, *capturedRequest) {
	t.Helper()
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	t.Cleanup(srv.Close)
	return srv, cap
}

func TestHTTPTempFileSetsContentLength(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = newTempFile();
		tmp.Write(newBytes("hello"));
		tmp.SeekTo(0, SEEK_START);
		var resp = http(newContext(), "PUT", url, { body: tmp });
		resp.Dispose();
		tmp.Close();
	`); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 5 {
		t.Fatalf("ContentLength = %d, want 5", cap.contentLength)
	}
	if len(cap.transferEncoding) != 0 {
		t.Fatalf("TransferEncoding = %v, want empty", cap.transferEncoding)
	}
	if cap.body != "hello" {
		t.Fatalf("body = %q, want hello", cap.body)
	}
}

func TestHTTPChunkedWhenTransferEncodingSet(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = newTempFile();
		tmp.Write(newBytes("hello"));
		tmp.SeekTo(0, SEEK_START);
		var resp = http(newContext(), "PUT", url, {
			headers: { "Transfer-Encoding": "chunked" },
			body: tmp
		});
		resp.Dispose();
		tmp.Close();
	`); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != -1 {
		t.Fatalf("ContentLength = %d, want -1", cap.contentLength)
	}
	if cap.body != "hello" {
		t.Fatalf("body = %q, want hello", cap.body)
	}
}

func TestHTTPWriteReaderThenUploadTempFile(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	sum := evalJSString(t, vm, `
(function() {
  var tmp = newTempFile();
  tmp.Write(newBytes("abc"));
  tmp.SeekTo(0, SEEK_START);
  var h = encUtils.newHash(HASH.MD5).WriteReader(tmp);
  var resp = http(newContext(), "PUT", url, { body: tmp });
  resp.Dispose();
  tmp.Close();
  return encUtils.toHex(h.Sum());
})()
`)
	want := md5.Sum([]byte("abc"))
	if sum != hex.EncodeToString(want[:]) {
		t.Fatalf("MD5 = %q, want %q", sum, hex.EncodeToString(want[:]))
	}
	if cap.contentLength != 3 {
		t.Fatalf("ContentLength = %d, want 3", cap.contentLength)
	}
	if cap.body != "abc" {
		t.Fatalf("body = %q, want abc", cap.body)
	}
}

func TestHTTPProgressReaderPreservesLength(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = newTempFile();
		tmp.Write(newBytes("xyz"));
		tmp.SeekTo(0, SEEK_START);
		var ctx = newTaskCtx(newContext());
		var resp = http(ctx, "PUT", url, { body: tmp.ProgressReader(ctx) });
		resp.Dispose();
		tmp.Close();
	`); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 3 {
		t.Fatalf("ContentLength = %d, want 3", cap.contentLength)
	}
	if cap.body != "xyz" {
		t.Fatalf("body = %q, want xyz", cap.body)
	}
}

func TestHTTPLimitReaderAutoLength(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = newTempFile();
		tmp.Write(newBytes("hello"));
		tmp.SeekTo(0, SEEK_START);
		var resp = http(newContext(), "PUT", url, { body: tmp.LimitReader(2) });
		resp.Dispose();
		tmp.Close();
	`); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 2 {
		t.Fatalf("ContentLength = %d, want 2", cap.contentLength)
	}
	if cap.body != "he" {
		t.Fatalf("body = %q, want he", cap.body)
	}
}

func TestHTTPContentLengthHeader(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = newTempFile();
		tmp.Write(newBytes("hello"));
		tmp.SeekTo(0, SEEK_START);
		var resp = http(newContext(), "PUT", url, {
			headers: { "Content-Length": "2" },
			body: tmp
		});
		resp.Dispose();
		tmp.Close();
	`); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 2 {
		t.Fatalf("ContentLength = %d, want 2", cap.contentLength)
	}
	if cap.body != "he" {
		t.Fatalf("body = %q, want he", cap.body)
	}
}
