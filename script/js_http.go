package script

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	gReq "go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const defaultHTTPTimeout = 30 * time.Second

var jsHTTP = NativeFunction(func(vm *VM, args Values) any {
	ctx := vm.ExecutionContext()
	url := args.Get(0).String()
	opt := args.Get(1)

	method := http.MethodGet
	var headers types.SM
	var body any
	timeout := defaultHTTPTimeout
	if !opt.IsNil() {
		if m := opt.Get("method"); m != nil && !m.IsNil() {
			method = m.String()
			if method == "" {
				vm.ThrowTypeError("http method must not be empty")
			}
		}
		if h := opt.Get("headers"); h != nil && !h.IsNil() {
			headers = h.SM()
		}
		if b := opt.Get("body"); b != nil && !b.IsNil() {
			body = b.Raw()
		}
		if t := opt.Get("timeout"); t != nil && !t.IsNil() {
			timeout = GetDuration(vm, t, "http timeout requires a Duration or duration string")
			if timeout < 0 {
				vm.ThrowTypeError("http timeout must be >= 0")
			}
		}
	}

	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer func() {
			// On success the response body takes ownership of cancellation.
			if cancel != nil {
				cancel()
			}
		}()
	}

	var reqBody gReq.RequestBody
	var errChan chan error
	var multipartReader *io.PipeReader

	if body != nil {
		if str, ok := body.(string); ok {
			b := []byte(str)
			reqBody = newHTTPRequestBody(bytes.NewReader(b), int64(len(b)), "")
		} else if vr := GetReader(vm, body, ""); vr != nil {
			n, e := readerBodyLength(headers, vr)
			if e != nil {
				vm.ThrowError(e)
			}
			if n >= 0 {
				vr = io.LimitReader(vr, n)
			}
			vr = wrapReaderProgress(vm.ExecutionContext(), vr)
			reqBody = newHTTPRequestBody(vr, n, "")
		} else if b := GetBytes(vm, body, ""); b != nil {
			reqBody = newHTTPRequestBody(bytes.NewReader(b), int64(len(b)), "")
		} else if fd, ok := HostAs[*jsObjHttpFormData](body); ok {
			fd.consumeIfReaders()
			r, w := io.Pipe()
			multipartReader = r
			defer r.Close()
			errChan = make(chan error, 1)
			mw := multipart.NewWriter(w)
			reqBody = newHTTPRequestBody(r, -1, mw.FormDataContentType())
			go func() {
				defer func() { _ = w.Close() }()
				errChan <- fd.writeTo(mw)
			}()
		}
	}

	resp, e := httpClient.Request(reqCtx, method, url, headers, reqBody)
	var wErr error
	if errChan != nil {
		// Request construction can fail before net/http owns the body.
		// Also unblock the writer if the server responds without consuming it.
		_ = multipartReader.CloseWithError(e)
		wErr = <-errChan
	}
	if e != nil {
		vm.ThrowError(e)
	}
	// A valid early response may stop the pipe writer. Preserve that response,
	// but never hide an error from reading the source file (even ErrClosedPipe).
	_, sourceFailed := errors.AsType[*multipartSourceError](wErr)
	if wErr != nil && (sourceFailed || !errors.Is(wErr, io.ErrClosedPipe)) {
		_ = resp.Dispose()
		vm.ThrowError(wErr)
	}

	response := resp.Response()
	if cancel != nil {
		response.Body = &httpCancelBody{ReadCloser: response.Body, cancel: cancel}
		cancel = nil
	}
	return newHttpResponse(vm, response)
})

// Keep the request deadline alive until the body is closed, including when a
// Script Drive detaches the body from its VM and returns it to a Go caller.
type httpCancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *httpCancelBody) Close() error {
	defer b.cancel()
	return b.ReadCloser.Close()
}

var jsClassHttpFormData = JSClass{
	Name:   "HttpFormData",
	Handle: jsObjHttpFormData{},
	Construct: func(vm *VM, _ Values) any {
		return &jsObjHttpFormData{ClassHost: NewClassHost(vm), data: make([]formDataField, 0)}
	},
	Methods: map[string]ClassMethod{
		"appendField": func(vm *VM, this *Value, args Values) any {
			This[*jsObjHttpFormData](vm, this, "HttpFormData.appendField").AppendField(args.Get(0).String(), args.Get(1).Raw())
			return nil
		},
		"appendFile": func(vm *VM, this *Value, args Values) any {
			This[*jsObjHttpFormData](vm, this, "HttpFormData.appendFile").AppendFile(args.Get(0).String(), args.Get(1).String(), args.Get(2).Raw())
			return nil
		},
	},
}

var httpClient, _ = gReq.NewClient("", nil, nil, nil)

type httpRequestBody struct {
	r   io.Reader
	n   int64
	typ string
}

func (b *httpRequestBody) ContentType() string  { return b.typ }
func (b *httpRequestBody) ContentLength() int64 { return b.n }
func (b *httpRequestBody) Reader() io.Reader    { return b.r }

func newHTTPRequestBody(r io.Reader, n int64, typ string) gReq.RequestBody {
	if n < 0 {
		n = -1
	}
	return &httpRequestBody{r: r, n: n, typ: typ}
}

func headerValue(headers types.SM, name string) (string, bool) {
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

func readerBodyLength(headers types.SM, r io.Reader) (int64, error) {
	if te, ok := headerValue(headers, "Transfer-Encoding"); ok && strings.Contains(strings.ToLower(te), "chunked") {
		return -1, nil
	}
	if raw, ok := headerValue(headers, "Content-Length"); ok && raw != "" {
		n, e := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if e != nil || n < 0 {
			return 0, fmt.Errorf("invalid Content-Length")
		}
		return n, nil
	}
	return readerKnownLength(r), nil
}

type jsObjHttpFormData struct {
	ClassHost
	data []formDataField
	used bool
}

type formDataField struct {
	field    string
	filename string
	file     bool
	data     any
}

func (fd *jsObjHttpFormData) AppendField(key string, v any) {
	var data []byte
	if str, ok := v.(string); ok {
		data = []byte((str))
	} else if b := GetBytes(fd.vm, v, ""); b != nil {
		data = b
	} else {
		fd.vm.ThrowTypeError("AppendField: value must be string or Bytes")
	}
	fd.data = append(fd.data, formDataField{field: key, data: data})
}

func (fd *jsObjHttpFormData) AppendFile(key, filename string, reader any) {
	if vr := GetReader(fd.vm, reader, ""); vr != nil {
		r := wrapReaderProgress(fd.vm.ExecutionContext(), vr)
		fd.data = append(fd.data, formDataField{field: key, filename: filename, file: true, data: r})
		return
	}
	var data []byte
	if str, ok := reader.(string); ok {
		data = []byte(str)
	} else if b := GetBytes(fd.vm, reader, ""); b != nil {
		data = b
	} else {
		fd.vm.ThrowTypeError("AppendFile: value must be string, Bytes or Reader")
	}
	fd.data = append(fd.data, formDataField{field: key, filename: filename, file: true, data: data})
}

func (fd *jsObjHttpFormData) hasReader() bool {
	for _, item := range fd.data {
		if _, ok := item.data.(io.Reader); ok {
			return true
		}
	}
	return false
}

func (fd *jsObjHttpFormData) consumeIfReaders() {
	if !fd.hasReader() {
		return
	}
	if fd.used {
		fd.vm.ThrowTypeError("HttpFormData with a Reader cannot be reused")
	}
	fd.used = true
}

func (fd jsObjHttpFormData) ConsoleString() string {
	n := len(fd.data)
	return formatGoInspect("HttpFormData", []string{fmt.Sprintf("Len: %d", n)}, n > 0)
}

func (fd *jsObjHttpFormData) writeTo(mw *multipart.Writer) error {
	defer func() { _ = mw.Close() }()

	var e error
	for _, item := range fd.data {
		if e != nil {
			continue
		}
		var fw io.Writer
		if item.file {
			fw, e = mw.CreateFormFile(item.field, item.filename)
		} else {
			fw, e = mw.CreateFormField(item.field)
		}
		if e != nil {
			continue
		}
		if b, ok := item.data.([]byte); ok {
			_, e = fw.Write(b)
		} else if r, ok := item.data.(io.Reader); ok {
			_, e = io.Copy(fw, multipartSourceReader{r})
		}
	}
	return e
}

type multipartSourceError struct{ error }

func (e *multipartSourceError) Unwrap() error { return e.error }

type multipartSourceReader struct{ io.Reader }

func (r multipartSourceReader) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	if e != nil && e != io.EOF {
		e = &multipartSourceError{e}
	}
	return n, e
}

type httpHeadersJS struct {
	vm *VM
	h  http.Header
}

func newHttpHeaders(vm *VM, h http.Header) *httpHeadersJS {
	return &httpHeadersJS{vm: vm, h: h}
}

func (h *httpHeadersJS) Get(_ *VM, args Values) any {
	return h.h.Get(args.Get(0).String())
}

func (h *httpHeadersJS) Values(_ *VM, args Values) any {
	return h.h.Values(args.Get(0).String())
}

func (h *httpHeadersJS) GetAll(_ *VM, _ Values) any {
	return map[string][]string(h.h)
}

func (h *httpHeadersJS) ConsoleString() string {
	n := 0
	if h.h != nil {
		n = len(h.h)
	}
	return formatGoInspect("HttpHeaders", []string{fmt.Sprintf("Len: %d", n)}, n > 0)
}

type httpResponseJS struct {
	vm      *VM
	Status  int
	Headers *httpHeadersJS
	Body    *Value
	body    io.ReadCloser
}

func newHttpResponse(vm *VM, resp *http.Response) *httpResponseJS {
	return &httpResponseJS{
		vm:      vm,
		Status:  resp.StatusCode,
		Headers: newHttpHeaders(vm, resp.Header),
		Body:    vm.NewInstance("ReadCloser", resp.Body),
		body:    resp.Body,
	}
}

func (r *httpResponseJS) BodySize(_ *VM, _ Values) any {
	return r.bodySize()
}

func (r *httpResponseJS) Text(_ *VM, _ Values) any {
	return string(r.readBody())
}

func (r *httpResponseJS) JSON(_ *VM, _ Values) any {
	bytes := r.readBody()
	if len(bytes) == 0 {
		return goja.Null()
	}
	var value any
	if e := json.Unmarshal(bytes, &value); e != nil {
		r.vm.ThrowTypeError(fmt.Sprintf("invalid JSON: %v", e))
	}
	return r.vm.ToJSValue(value)
}

func (r *httpResponseJS) Dispose(_ *VM, _ Values) any {
	r.dispose()
	return nil
}

func (r *httpResponseJS) bodySize() int64 {
	if r.Headers == nil || r.Headers.h == nil {
		return -1
	}
	return utils.ToInt64(r.Headers.h.Get("Content-Length"), -1)
}

func (r *httpResponseJS) readBody() []byte {
	defer r.dispose()
	bytes, e := io.ReadAll(r.body)
	if e != nil {
		r.vm.ThrowError(e)
	}
	return bytes
}

func (r *httpResponseJS) dispose() {
	if r.Body != nil {
		if c, ok := HostAs[jsCloser](r.Body); ok {
			c.Close()
			return
		}
	}
	if r.body != nil {
		_ = r.body.Close()
	}
}

func (r *httpResponseJS) ConsoleString() string {
	return formatGoInspect("HttpResponse", []string{
		fmt.Sprintf("Status: %d", r.Status),
		fmt.Sprintf("BodySize: %d", r.bodySize()),
	}, true)
}
