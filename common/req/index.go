package req

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sync"
)

const maxReadableBodySize int64 = 10 * 1024 * 1024 // 10MB
var absoluteURLPattern = regexp.MustCompile("(?i)^https?://")

var defaultClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		// dont follow redirects
		return http.ErrUseLastResponse
	},
}

type Client struct {
	c       *http.Client
	baseURL *url.URL
	before  func(*http.Request) error
	after   func(Response) error
}

func NewClient(
	baseURL string,
	before func(*http.Request) error,
	after func(Response) error,
	client *http.Client) (*Client, error) {

	var u *url.URL = nil
	if baseURL != "" {
		temp, e := url.Parse(baseURL)
		if e != nil {
			return nil, e
		}
		u = temp
	}
	return &Client{
		c:       client,
		baseURL: u,
		before:  before,
		after:   after,
	}, nil
}

func (h *Client) BuildURL(requestUrl string) (string, error) {
	if absoluteURLPattern.MatchString(requestUrl) {
		return requestUrl, nil
	}
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

func (h *Client) newRequest(method string, requestUrl string, headers types.SM,
	body RequestBody, ctx context.Context) (*http.Request, error) {
	requestUrl, e := h.BuildURL(requestUrl)
	if e != nil {
		return nil, e
	}
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body.Reader()
	}
	var req *http.Request
	if ctx != nil {
		req, e = http.NewRequestWithContext(ctx, method, requestUrl, bodyReader)
	} else {
		req, e = http.NewRequest(method, requestUrl, bodyReader)
	}
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

func (h *Client) client() *http.Client {
	if h.c != nil {
		return h.c
	}
	return defaultClient
}

func (h *Client) request(req *http.Request) (Response, error) {
	if utils.IsDebugOn() {
		log.Printf("[HttpClient  req] %s %s", req.Method, req.URL.String())
	}
	r, e := h.client().Do(req)
	if utils.IsDebugOn() {
		var v interface{} = nil
		if e == nil {
			v = r.StatusCode
		} else {
			v = e
		}
		log.Printf("[HttpClient resp] %s %s %v", req.Method, req.URL.String(), v)
	}
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

func (h *Client) Request(ctx context.Context, method, requestUrl string,
	headers types.SM, body RequestBody) (Response, error) {
	req, e := h.newRequest(method, requestUrl, headers, body, ctx)
	if e != nil {
		return nil, e
	}
	return h.request(req)
}

func (h *Client) Get(ctx context.Context, requestUrl string, headers types.SM) (Response, error) {
	return h.Request(ctx, "GET", requestUrl, headers, nil)
}

func (h *Client) Post(ctx context.Context, requestUrl string,
	headers types.SM, body RequestBody) (Response, error) {
	return h.Request(ctx, "POST", requestUrl, headers, body)
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

func (r *httpResp) XML(v interface{}) error {
	defer func() { _ = r.r.Body.Close() }()
	dat, e := r.getBody()
	if e != nil {
		return e
	}
	return xml.Unmarshal(dat, v)
}

func (r *httpResp) Dispose() error {
	if r.r.Body != nil {
		return r.r.Body.Close()

	}
	return nil
}

func NewURLEncodedBody(v types.SM) RequestBody {
	q := make(url.Values)
	for k, v := range v {
		q.Set(k, v)
	}
	return &byteBody{b: []byte(q.Encode()), t: "application/x-www-form-urlencoded"}
}

func NewJsonBody(v interface{}) RequestBody {
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

func NewReaderBody(r io.Reader, length int64) RequestBody {
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
