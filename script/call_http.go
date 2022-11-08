package script

import (
	"bytes"
	"go-drive/common/utils"
	"io"
	"net/http"
)

var httpClient = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}}

// vm_http: (ctx Context, method, url string, headers types.SM, body interface{}) *httpResponse
func vm_http(vm *VM, args Values) interface{} {
	ctx := args.Get(0).Raw()
	method := args.Get(1).String()
	url := args.Get(2).String()
	headers := args.Get(3).SM()
	body := args.Get(4).Raw()
	var bodyReader io.Reader
	if body != nil {
		if vr := GetReader(body); vr != nil {
			bodyReader = vr
		} else if str, ok := body.(string); ok {
			bodyReader = bytes.NewReader([]byte((str)))
		} else if b := GetBytes(body); b != nil {
			bodyReader = bytes.NewReader(b)
		}
	}
	var req *http.Request
	var e error

	req, e = http.NewRequestWithContext(GetContext(ctx), method, url, bodyReader)
	if e != nil {
		vm.ThrowError(e)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, e := httpClient.Do(req)
	if e != nil {
		vm.ThrowError(e)
	}

	return newHttpResponse(vm, resp)
}

type HttpHeaders struct {
	vm *VM
	h  http.Header
}

func (h *HttpHeaders) Get(key string) string {
	return h.h.Get(key)
}

func (h *HttpHeaders) Values(key string) []string {
	return h.h.Values(key)
}

func (h *HttpHeaders) GetAll() map[string][]string {
	return h.h
}

func newHttpResponse(vm *VM, resp *http.Response) *httpResponse {
	return &httpResponse{
		vm:      vm,
		Status:  resp.StatusCode,
		Headers: &HttpHeaders{vm, resp.Header},
		Body:    NewReadCloser(vm, resp.Body),
	}
}

type httpResponse struct {
	vm      *VM
	Status  int
	Headers *HttpHeaders
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
