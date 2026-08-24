package req

import (
	"go-drive/common/logging"
	"net/http"
	"time"
)

var defaultLoggingClient = NewLoggingClient(defaultClient)

func wrapClient(client *http.Client) *http.Client {
	if client == nil {
		return nil
	}
	return NewLoggingClient(client)
}

// NewLoggingClient returns a shallow copy of client whose transport emits a
// safe request/response trace.
func NewLoggingClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	copy := *client
	copy.Transport = loggingTransport{base: copy.Transport, log: logging.For("http-c")}
	return &copy
}

type loggingTransport struct {
	base http.RoundTripper
	log  *logging.Logger
}

// HTTPClient is the small interface used by generated SDKs for an HTTP
// client. NewLoggingHTTPClient adapts those clients without requiring them to
// use net/http.Client directly.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func NewLoggingHTTPClient(client HTTPClient) HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return loggingHTTPClient{base: client, log: logging.For("http-c")}
}

type loggingHTTPClient struct {
	base HTTPClient
	log  *logging.Logger
}

func (c loggingHTTPClient) Do(r *http.Request) (*http.Response, error) {
	url := SanitizeURL(r.URL.String())
	started := time.Now()
	c.log.Debugf("request %s %s", r.Method, url)
	resp, e := c.base.Do(r)
	if e != nil {
		c.log.Debugf("response %s %s error=%s duration=%s", r.Method, url, SanitizeError(e), time.Since(started))
		return nil, e
	}
	c.log.Debugf("response %s %s status=%d duration=%s", r.Method, url, resp.StatusCode, time.Since(started))
	return resp, nil
}

func (t loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	url := SanitizeURL(r.URL.String())
	started := time.Now()
	t.log.Debugf("request %s %s", r.Method, url)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, e := base.RoundTrip(r)
	if e != nil {
		t.log.Debugf("response %s %s error=%s duration=%s", r.Method, url, SanitizeError(e), time.Since(started))
		return nil, e
	}
	t.log.Debugf("response %s %s status=%d duration=%s", r.Method, url, resp.StatusCode, time.Since(started))
	return resp, nil
}
