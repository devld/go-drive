package script

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go-drive/common/task"
)

type capturedRequest struct {
	mu               sync.Mutex
	method           string
	contentLength    int64
	transferEncoding []string
	body             string
}

func (c *capturedRequest) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.method = r.Method
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

func TestHTTPDefaultsToGET(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `http(url).dispose()`, ""); e != nil {
		t.Fatal(e)
	}
	if cap.method != http.MethodGet {
		t.Fatalf("method = %q, want GET", cap.method)
	}
}

func TestHTTPRejectsEmptyMethod(t *testing.T) {
	vm := newScriptTestVM(t)
	if _, e := vm.Run(context.Background(), `http("http://127.0.0.1", { method: "" })`, ""); e == nil || !strings.Contains(e.Error(), "http method") {
		t.Fatalf("empty method = %v", e)
	}
}

func TestHTTPTempFileSetsContentLength(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = new TempFile();
		tmp.write(Bytes.fromString("hello"));
		tmp.seekTo(0, SEEK_START);
		var resp = http(url, { method: "PUT", body: tmp });
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
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
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = new TempFile();
		tmp.write(Bytes.fromString("hello"));
		tmp.seekTo(0, SEEK_START);
		var resp = http(url, { method: "PUT",
			headers: { "Transfer-Encoding": "chunked" },
			body: tmp
		});
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
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
	mustDefineGlobal(t, vm, "url", srv.URL)
	sum := evalJSString(t, vm, `
(function() {
  var tmp = new TempFile();
  tmp.write(Bytes.fromString("abc"));
  tmp.seekTo(0, SEEK_START);
	var h = new Hash("md5").writeFrom(tmp);
	var resp = http(url, { method: "PUT", body: tmp });
	resp.dispose();
	tmp.close();
	return h.sum().toString("hex");
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
	mustDefineGlobal(t, vm, "url", srv.URL)
	tc := task.NewTaskContext(context.Background())
	progress := NewProgressReporter(tc, true, false)
	mustDefineGlobal(t, vm, "progress", progress)
	if _, e := vm.Run(context.Background(), `
			var tmp = new TempFile();
			tmp.write(Bytes.fromString("xyz"));
			tmp.seekTo(0, SEEK_START);
			var resp = http(url, { method: "PUT", body: tmp.withProgress(progress) });
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 3 {
		t.Fatalf("ContentLength = %d, want 3", cap.contentLength)
	}
	if cap.body != "xyz" {
		t.Fatalf("body = %q, want xyz", cap.body)
	}
	if tc.GetProgress() != 3 {
		t.Fatalf("progress = %d, want 3", tc.GetProgress())
	}
}

func TestHTTPLimitReaderAutoLength(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = new TempFile();
		tmp.write(Bytes.fromString("hello"));
		tmp.seekTo(0, SEEK_START);
		var resp = http(url, { method: "PUT", body: tmp.limitReader(2) });
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
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
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `
		var tmp = new TempFile();
		tmp.write(Bytes.fromString("hello"));
		tmp.seekTo(0, SEEK_START);
		var resp = http(url, { method: "PUT",
			headers: { "Content-Length": "2" },
			body: tmp
		});
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if cap.contentLength != 2 {
		t.Fatalf("ContentLength = %d, want 2", cap.contentLength)
	}
	if cap.body != "he" {
		t.Fatalf("body = %q, want he", cap.body)
	}
}

func TestHTTPResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"goja","items":[1,true]}`)
	}))
	t.Cleanup(server.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", server.URL)
	got := evalJSString(t, vm, `
(function() {
  var resp = http(url);
  var data = resp.json();
  return data.name + "|" + data.items[0] + "|" + data.items[1];
})()
`)
	if got != "goja|1|true" {
		t.Fatalf("HttpResponse.json() = %q, want goja|1|true", got)
	}
}

func TestHTTPResponseJSONEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", server.URL)
	got := evalJSString(t, vm, `
(function() {
  var resp = http(url);
		return String(resp.json());
})()
`)
	if got != "null" {
		t.Fatalf("HttpResponse.json() empty body = %q, want null", got)
	}
}

func TestHTTPResponseJSONRejectsInvalidBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not-json")
	}))
	t.Cleanup(server.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", server.URL)
	if _, e := vm.Run(context.Background(), `http(url).json()`, ""); e == nil {
		t.Fatal("HttpResponse.json() accepted invalid JSON")
	}
}

func TestHTTPTimeoutCancelsHungRequest(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	start := time.Now()
	_, e := vm.Run(context.Background(), `
		http(url, { timeout: "80ms" });
	`, "")
	if e == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("http timeout took %s", time.Since(start))
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive request")
	}
}

func TestHTTPTimeoutZeroKeepsParentDeadline(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, e := vm.Run(ctx, `http(url, { timeout: 0 });`, "")
		done <- e
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive request")
	}

	select {
	case e := <-done:
		if e == nil {
			t.Fatal("expected parent context timeout")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("http() was not cancelled by parent context")
	}
}

func TestHTTPRejectsNegativeTimeout(t *testing.T) {
	vm := newScriptTestVM(t)
	if _, e := vm.Run(context.Background(), `http("http://127.0.0.1", { timeout: ms(-1) })`, ""); e == nil || !strings.Contains(e.Error(), "http timeout") {
		t.Fatalf("negative timeout = %v", e)
	}
}

func TestHTTPUploadReportsTaskProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = io.Copy(io.Discard, r.Body)
	}))
	t.Cleanup(srv.Close)

	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	mustDefineGlobal(t, vm, "payload", bytes.Repeat([]byte("x"), 2<<20))
	tc := task.NewTaskContext(context.Background())
	if _, e := vm.Run(tc, `
			var plain = new TempFile();
			plain.write(payload);
			plain.seekTo(0, SEEK_START);
			var plainResp = http(url, { method: "PUT", body: plain });
			plainResp.dispose();
			plain.close();
		`, ""); e != nil {
		t.Fatal(e)
	}
	if tc.GetProgress() != 0 {
		t.Fatalf("unwrapped HTTP progress = %d, want 0", tc.GetProgress())
	}
	progress := NewProgressReporter(tc, true, false)
	mustDefineGlobal(t, vm, "progress", progress)
	if _, e := vm.Run(tc, `
			var tmp = new TempFile();
			tmp.write(payload);
			tmp.seekTo(0, SEEK_START);
			var resp = http(url, { method: "PUT", body: tmp.withProgress(progress) });
		resp.dispose();
		tmp.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if tc.GetProgress() < 1 {
		t.Fatalf("upload progress = %d, want at least 1", tc.GetProgress())
	}
}

func TestHTTPFormDataReportsOnlyWrappedReader(t *testing.T) {
	srv, _ := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	tc := task.NewTaskContext(context.Background())
	progress := NewProgressReporter(tc, true, false)
	mustDefineGlobal(t, vm, "progress", progress)

	if _, e := vm.Run(tc, `
		const source = new TempFile();
		source.write(Bytes.fromString("abc"));
		source.seekTo(0, SEEK_START);
		const fd = new HttpFormData();
		fd.appendFile("file", "a.txt", source.withProgress(progress));
		const resp = http(url, {method: "POST", body: fd});
		resp.dispose();
		source.close();
	`, ""); e != nil {
		t.Fatal(e)
	}
	if tc.GetProgress() != 3 || tc.GetTotal() != 0 {
		t.Fatalf("multipart progress = %d/%d, want 3/0", tc.GetProgress(), tc.GetTotal())
	}
}

func TestHTTPFormDataDoesNotCloseTempFile(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	got := evalJSString(t, vm, `
(function() {
  var src = new TempFile();
  src.write(Bytes.fromString("abc"));
  src.seekTo(0, SEEK_START);
  var fd = new HttpFormData();
  fd.appendFile("f", "a.txt", src);
  var resp = http(url, { method: "POST", body: fd });
  resp.dispose();
  src.seekTo(0, SEEK_START);
  var again = src.readAsString();
  src.close();
  return again;
})()
`)
	if got != "abc" {
		t.Fatalf("TempFile after appendFile = %q, want abc", got)
	}
	if !strings.Contains(cap.body, "abc") {
		t.Fatalf("multipart body missing abc: %q", cap.body)
	}
}

func TestHTTPTimeoutKeepsResponseBodyAlive(t *testing.T) {
	for _, tc := range []struct {
		name, options, read string
	}{
		{"text-default", `{}`, `.text()`},
		{"json-timeout", `{timeout: "2s"}`, `.json().message`},
		{"detached-body", `{timeout: "2s"}`, `.body`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				select {
				case <-time.After(30 * time.Millisecond):
					_, _ = io.WriteString(w, `{"message":"payload"}`)
				case <-r.Context().Done():
				}
			}))
			defer srv.Close()
			vm := newScriptTestVM(t)
			mustDefineGlobal(t, vm, "url", srv.URL)
			value, e := vm.Run(context.Background(), `http(url, `+tc.options+`)`+tc.read, "")
			if e != nil {
				t.Fatal(e)
			}
			want := `{"message":"payload"}`
			var got string
			if tc.read == `.body` {
				raw := value.Raw()
				body := GetReadCloser(vm, raw, "")
				if body == nil {
					t.Fatal("missing response body")
				}
				vm.RemoveDisposable(raw)
				if e := vm.Dispose(); e != nil {
					t.Fatal(e)
				}
				defer body.Close()
				data, e := io.ReadAll(body)
				if e != nil {
					t.Fatal(e)
				}
				got = string(data)
			} else {
				got = value.String()
				if tc.read == `.json().message` {
					want = "payload"
				}
			}
			if got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestHTTPTimeoutAppliesWhileReadingBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	_, e := vm.Run(context.Background(), `http(url, {timeout: "50ms"}).text()`, "")
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("body read error = %v, want deadline exceeded", e)
	}
}

func TestHTTPDisposeCancelsRequest(t *testing.T) {
	cancelled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(cancelled)
	}))
	defer srv.Close()
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	if _, e := vm.Run(context.Background(), `http(url).dispose()`, ""); e != nil {
		t.Fatal(e)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("disposing response did not cancel request")
	}
}

func TestHTTPMultipartInvalidRequestReturns(t *testing.T) {
	for _, options := range []string{`http(":invalid", {method: "POST", body: fd})`, `http("http://localhost", {method: "invalid method", body: fd})`} {
		t.Run(options, func(t *testing.T) {
			done := make(chan error, 1)
			go func() {
				vm, e := NewVM()
				if e != nil {
					done <- e
					return
				}
				defer vm.Dispose()
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, e = vm.Run(ctx, `const fd = new HttpFormData(); fd.appendField("a", "b"); `+options, "")
				done <- e
			}()
			select {
			case e := <-done:
				if e == nil {
					t.Fatal("expected invalid request error")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("multipart writer did not exit")
			}
		})
	}
}

func TestHTTPPartiallyConsumedNestedLimit(t *testing.T) {
	srv, captured := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	_, e := vm.Run(context.Background(), `
  const f = new TempFile();
  f.write(Bytes.fromString("0123456789"));
  f.seekTo(0, SEEK_START);
  const r = f.limitReader(4).limitReader(8);
  r.read(new Bytes(2));
  http(url, {method:"PUT", body:r}).dispose();
 `, "")
	if e != nil {
		t.Fatal(e)
	}
	if captured.contentLength != 2 || captured.body != "23" {
		t.Fatalf("length=%d body=%q", captured.contentLength, captured.body)
	}
}

func TestProgressWrapperPreservesNestedRemainingLength(t *testing.T) {
	vm := newScriptTestVM(t)
	_, e := vm.Run(context.Background(), `
  const f = new TempFile(); f.write(Bytes.fromString("0123456789")); f.seekTo(0, SEEK_START);
  var limited = f.limitReader(4).limitReader(8);
 `, "")
	if e != nil {
		t.Fatal(e)
	}
	value, _ := vm.GetValue("limited")
	original := GetReader(vm, value.Raw(), "")
	wrapped := progressReportingReader{r: original}
	if _, e := io.ReadFull(wrapped, make([]byte, 2)); e != nil {
		t.Fatal(e)
	}
	if n := readerKnownLength(wrapped); n != 2 {
		t.Fatalf("remaining=%d", n)
	}
	if _, e := io.ReadAll(wrapped); e != nil {
		t.Fatal(e)
	}
	if n := readerKnownLength(wrapped); n != 0 {
		t.Fatalf("exhausted remaining=%d", n)
	}
}

func TestHTTPMultipartPreservesEarlyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = io.WriteString(w, "too large")
	}))
	defer srv.Close()
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	v, e := vm.Run(context.Background(), `
  const fd=new HttpFormData();
  fd.appendFile("file","large.bin",Bytes.fromString("a".repeat(16*1024*1024)));
  const response=http(url,{method:"POST",body:fd});
  response.status+":"+response.text();
 `, "")
	if e != nil {
		t.Fatal(e)
	}
	if v.String() != "413:too large" {
		t.Fatalf("response=%q", v.String())
	}
}

type multipartFailingReader struct{ err error }

func (r multipartFailingReader) Read([]byte) (int, error) { return 0, r.err }

func TestHTTPMultipartPreservesSourceErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	for _, want := range []error{io.ErrClosedPipe, errors.New("source read failed")} {
		t.Run(want.Error(), func(t *testing.T) {
			vm := newScriptTestVM(t)
			mustDefineGlobal(t, vm, "url", srv.URL)
			mustDefineGlobal(t, vm, "source", multipartFailingReader{want})
			_, e := vm.Run(context.Background(), `
    const fd=new HttpFormData();fd.appendFile("file","file.bin",source);
    http(url,{method:"POST",body:fd}).dispose();
   `, "")
			if !errors.Is(e, want) {
				t.Fatalf("error=%v, want %v", e, want)
			}
		})
	}
}

func TestHTTPFormDataCanBeReusedAfterFailedRequest(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	got := evalJSString(t, vm, `
(function() {
  var fd = new HttpFormData();
  fd.appendField("a", "b");
  fd.appendFile("f", "a.txt", Bytes.fromString("xyz"));
  try { http(":invalid", { method: "POST", body: fd }); } catch (e) {}
  http(url, { method: "POST", body: fd }).dispose();
  try { http(":invalid", { method: "POST", body: fd }); } catch (e) {}
  http(url, { method: "POST", body: fd }).dispose();
  return "ok";
})()
`)
	if got != "ok" {
		t.Fatalf("reuse = %q", got)
	}
	if !strings.Contains(cap.body, "b") || !strings.Contains(cap.body, "xyz") {
		t.Fatalf("multipart body missing field: %q", cap.body)
	}
}

func TestHTTPFormDataWithReaderCannotBeReused(t *testing.T) {
	srv, cap := startCaptureServer(t)
	vm := newScriptTestVM(t)
	mustDefineGlobal(t, vm, "url", srv.URL)
	for _, first := range []string{
		`try { http(":invalid", { method: "POST", body: fd }); } catch (e) {}`,
		`http(url, { method: "POST", body: fd }).dispose();`,
	} {
		got := evalJSString(t, vm, `
(function() {
  var src = new TempFile();
  src.write(Bytes.fromString("abc"));
  src.seekTo(0, SEEK_START);
  var fd = new HttpFormData();
  fd.appendFile("f", "a.txt", src);
  `+first+`
  try {
    http(url, { method: "POST", body: fd });
    src.close();
    return "missing";
  } catch (e) {
    src.close();
    return e instanceof TypeError ? "ok" : String(e);
  }
})()
`)
		if got != "ok" {
			t.Fatalf("reuse with Reader after %q = %q", first, got)
		}
	}
	if !strings.Contains(cap.body, "abc") {
		t.Fatalf("first successful send missing file: %q", cap.body)
	}
}
