package common

import (
	"bytes"
	"encoding/json"
	"go-drive/common/types"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"sync"
)

const maxReadableBodySize int64 = 10 * 1024 * 1024 // 10MB

var defaultClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		// dont follow redirects
		return http.ErrUseLastResponse
	},
}

type HttpClient struct {
	c       *http.Client
	baseURL *url.URL
	before  func(*http.Request) error
	after   func(HttpResponse) error
}

func NewHttpClient(
	baseURL string,
	before func(*http.Request) error,
	after func(HttpResponse) error,
	client *http.Client) (*HttpClient, error) {

	var u *url.URL = nil
	if baseURL != "" {
		temp, e := url.Parse(baseURL)
		if e != nil {
			return nil, e
		}
		u = temp
	}
	return &HttpClient{
		c:       client,
		baseURL: u,
		before:  before,
		after:   after,
	}, nil
}

func (h *HttpClient) buildURL(requestUrl string) (string, error) {
	if h.baseURL == nil {
		return requestUrl, nil
	}
	ru, e := url.Parse(requestUrl)
	if e != nil {
		return "", e
	}
	p := path.Join(h.baseURL.Path, ru.Path)
	qs := h.baseURL.Query()
	for k, v := range ru.Query() {
		qs[k] = v
	}
	u := url.URL{
		Scheme:   h.baseURL.Scheme,
		Opaque:   h.baseURL.Opaque,
		User:     h.baseURL.User,
		Host:     h.baseURL.Host,
		Path:     p,
		RawQuery: qs.Encode(),
	}
	return u.String(), nil
}

func (h *HttpClient) newRequest(method string, requestUrl string, headers types.SM, body HttpRequestBody) (*http.Request, error) {
	requestUrl, e := h.buildURL(requestUrl)
	if e != nil {
		return nil, e
	}
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body.Reader()
	}
	req, e := http.NewRequest(method, requestUrl, bodyReader)
	if e != nil {
		return nil, e
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	if body != nil {
		req.Header.Set("Content-Type", body.ContentType())
		req.ContentLength = body.ContentLength()
	}
	if h.before != nil {
		if e := h.before(req); e != nil {
			return nil, e
		}
	}
	return req, nil
}

func (h *HttpClient) client() *http.Client {
	if h.c != nil {
		return h.c
	}
	return defaultClient
}

func (h *HttpClient) Request(method, requestUrl string, headers types.SM, body HttpRequestBody) (HttpResponse, error) {
	req, e := h.newRequest(method, requestUrl, headers, body)
	if e != nil {
		return nil, e
	}
	if IsDebugOn() {
		log.Printf("[HttpClient] %s %s %v", req.Method, req.URL.String(), req.Header)
	}
	r, e := h.client().Do(req)
	if e != nil {
		return nil, e
	}
	resp := &httpResp{r: r, mux: &sync.Mutex{}}
	if h.after != nil {
		e := h.after(resp)
		if e != nil {
			_ = resp.Dispose()
			return nil, e
		}
	}
	return resp, nil
}

func (h *HttpClient) Get(requestUrl string, headers types.SM) (HttpResponse, error) {
	return h.Request("GET", requestUrl, headers, nil)
}

func (h *HttpClient) Post(requestUrl string, headers types.SM, body HttpRequestBody) (HttpResponse, error) {
	return h.Request("POST", requestUrl, headers, body)
}

type HttpResponse interface {
	Response() *http.Response
	Status() int
	Json(v interface{}) error
	Dispose() error
}

type httpResp struct {
	r    *http.Response
	body []byte
	mux  *sync.Mutex
}

func (r *httpResp) Response() *http.Response {
	return r.r
}

func (r *httpResp) Status() int {
	return r.r.StatusCode
}

func (r *httpResp) getBody() ([]byte, error) {
	defer r.mux.Unlock()
	r.mux.Lock()
	if r.body == nil {
		defer func() { _ = r.r.Body.Close() }()
		dat, e := ioutil.ReadAll(io.LimitReader(r.r.Body, maxReadableBodySize))
		if e != nil {
			return nil, e
		}
		r.body = dat
	}
	return r.body, nil
}

func (r *httpResp) Json(v interface{}) error {
	defer func() { _ = r.r.Body.Close() }()
	dat, e := r.getBody()
	if e != nil {
		return e
	}
	return json.Unmarshal(dat, v)
}

func (r *httpResp) Dispose() error {
	if r.r.Body != nil {
		return r.r.Body.Close()

	}
	return nil
}

type HttpRequestBody interface {
	ContentLength() int64
	ContentType() string
	Reader() io.Reader
}

func NewURLEncodedBody(v types.SM) HttpRequestBody {
	q := make(url.Values)
	for k, v := range v {
		q.Set(k, v)
	}
	return &byteBody{b: []byte(q.Encode()), t: "application/x-www-form-urlencoded"}
}

func NewJsonBody(v interface{}) HttpRequestBody {
	b, _ := json.Marshal(v)
	if b == nil {
		b = make([]byte, 0)
	}
	return &byteBody{b: b, t: "application/json"}
}

type byteBody struct {
	b []byte
	t string
}

func (b *byteBody) ContentType() string {
	return b.t
}

func (b *byteBody) Reader() io.Reader {
	return bytes.NewReader(b.b)
}

func (b *byteBody) ContentLength() int64 {
	return int64(len(b.b))
}

func NewReadBody(r io.Reader, length int64) HttpRequestBody {
	if length < 0 {
		length = -1
	}
	return readerBody{r: r, s: length}
}

type readerBody struct {
	r io.Reader
	s int64
}

func (r readerBody) ContentType() string {
	return "application/octet-stream"
}

func (r readerBody) Reader() io.Reader {
	return r.r
}

func (r readerBody) ContentLength() int64 {
	return r.s
}
