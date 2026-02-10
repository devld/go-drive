package script

import (
	"bytes"
	gReq "go-drive/common/req"
	"go-drive/common/utils"
	"io"
	"mime/multipart"
	"net/http"
)

var httpClient = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}}

// vm_http: (ctx Context, method, url string, headers types.SM, body any) *httpResponse
func vm_http(vm *VM, args Values) any {
	ctx := args.Get(0).Raw()
	method := args.Get(1).String()
	url := args.Get(2).String()
	headers := args.Get(3).SM()
	body := args.Get(4).Raw()

	var bodyReader io.Reader
	var contentType string
	var errChan chan error

	if body != nil {
		if str, ok := body.(string); ok {
			bodyReader = bytes.NewReader([]byte((str)))
		} else if vr := GetReader(body); vr != nil {
			bodyReader = vr
		} else if b := GetBytes(body); b != nil {
			bodyReader = bytes.NewReader(b)
		} else if fd, ok := body.(*formData); ok {
			r, w := io.Pipe()
			errChan = make(chan error, 1)
			bodyReader = r
			contentType = fd.prepare(w)
			go func() {
				defer func() { _ = w.Close() }()
				errChan <- fd.write()
			}()
		}
	}
	var req *http.Request
	var e error

	req, e = http.NewRequestWithContext(GetContext(ctx), method, url, bodyReader)
	if e != nil {
		vm.ThrowError(e)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", gReq.DefaultUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, e := httpClient.Do(req)
	var wErr error
	if errChan != nil {
		wErr = <-errChan
	}
	if wErr != nil {
		vm.ThrowError(wErr)
	}
	if e != nil {
		vm.ThrowError(e)
	}

	return newHttpResponse(vm, resp)
}

// vm_newFormData: () *formData
func vm_newFormData(vm *VM, args Values) any {
	return &formData{vm, make([]formDataField, 0), nil}
}

type formData struct {
	vm   *VM
	data []formDataField
	mw   *multipart.Writer
}

type formDataField struct {
	field    string
	filename string
	data     any
}

func (fd *formData) AppendField(key string, v any) {
	var data []byte
	if str, ok := v.(string); ok {
		data = []byte((str))
	} else if b := GetBytes(v); b != nil {
		data = b
	} else {
		fd.vm.ThrowTypeError("AppendField: value must be string or Bytes")
	}
	fd.data = append(fd.data, formDataField{key, "", data})
}

func (fd *formData) AppendFile(key, filename string, reader any) {
	var r io.Reader
	if vr := GetReader(reader); vr != nil {
		r = vr
	} else if str, ok := reader.(string); ok {
		r = bytes.NewReader([]byte((str)))
	} else if b := GetBytes(reader); b != nil {
		r = bytes.NewReader(b)
	} else {
		fd.vm.ThrowTypeError("AppendFile: value must be string, Bytes or Reader")
	}
	fd.data = append(fd.data, formDataField{key, filename, r})
}

func (fd *formData) prepare(w io.Writer) string {
	if fd.mw != nil {
		panic("already prepared")
	}
	fd.mw = multipart.NewWriter(w)
	return fd.mw.FormDataContentType()
}

func (fd *formData) write() error {
	defer func() { _ = fd.mw.Close() }()

	var e error
	for _, item := range fd.data {
		if b, ok := item.data.([]byte); ok {
			if e != nil {
				continue
			}
			var fw io.Writer
			fw, e = fd.mw.CreateFormField(item.field)
			if e != nil {
				continue
			}
			_, e = fw.Write(b)
		} else if r, ok := item.data.(io.Reader); ok {
			if rc, ok := r.(io.ReadCloser); ok {
				defer func() {
					_ = rc.Close()
				}()
			}
			if e != nil {
				continue
			}
			var fw io.Writer
			fw, e = fd.mw.CreateFormFile(item.field, item.filename)
			if e != nil {
				continue
			}
			_, e = io.Copy(fw, r)
		}
	}
	return e
}

type httpHeaders struct {
	vm *VM
	h  http.Header
}

func (h *httpHeaders) Get(key string) string {
	return h.h.Get(key)
}

func (h *httpHeaders) Values(key string) []string {
	return h.h.Values(key)
}

func (h *httpHeaders) GetAll() map[string][]string {
	return h.h
}

func newHttpResponse(vm *VM, resp *http.Response) *httpResponse {
	return &httpResponse{
		vm:      vm,
		Status:  resp.StatusCode,
		Headers: &httpHeaders{vm, resp.Header},
		Body:    NewReadCloser(vm, resp.Body),
	}
}

type httpResponse struct {
	vm      *VM
	Status  int
	Headers *httpHeaders
	Body    ReadCloser
}

func (r *httpResponse) BodySize() int64 {
	return utils.ToInt64(r.Headers.Get("Content-Length"), -1)
}

func (r *httpResponse) Text() string {
	defer r.Dispose()
	return r.Body.ReadAsString()
}

func (r *httpResponse) Dispose() {
	GetReadCloser(r.Body).Close()
}
