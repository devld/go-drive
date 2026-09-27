package req

import (
	"go-drive/common/logging"
	"net/http"
	"time"
)

// WithDefaultRoundTripper returns a shallow copy of client whose transport
// sets the default User-Agent and then logs the request. base nil uses
// http.DefaultTransport. A RoundTripper already installed on client runs
// after the default User-Agent is written, so it can replace that value.
// A nil client uses http.DefaultClient.
func WithDefaultRoundTripper(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	copy := *client
	copy.Transport = defaultRoundTripper(copy.Transport)
	return &copy
}

func defaultRoundTripper(base http.RoundTripper) http.RoundTripper {
	return userAgentTransport{base: withLogging(base)}
}

func withLogging(base http.RoundTripper) http.RoundTripper {
	return loggingTransport{base: base, log: logging.For("http-c")}
}

type loggingTransport struct {
	base http.RoundTripper
	log  *logging.Logger
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

type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	req2 := req.Clone(req.Context())
	setDefaultUserAgent(req2)
	return base.RoundTrip(req2)
}

func setDefaultUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", DefaultUserAgent)
}
