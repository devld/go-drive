package req

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-drive/common/types"
)

func TestDefaultRoundTripperReplacesExistingUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := WithDefaultRoundTripper(nil)
	req, e := http.NewRequest(http.MethodGet, server.URL, nil)
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("User-Agent", "aws-sdk-go-v2/1.41.5")
	resp, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	_ = resp.Body.Close()
	if got != DefaultUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "aws-sdk-go-v2/1.41.5" {
		t.Fatalf("original request User-Agent = %q", got)
	}
}

func TestClientHeadersOverrideDefaultUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client, e := NewClient("", nil, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	resp, e := client.Get(context.Background(), server.URL, nil)
	if e != nil {
		t.Fatal(e)
	}
	_ = resp.Dispose()
	if got != DefaultUserAgent {
		t.Fatalf("default User-Agent = %q", got)
	}

	resp, e = client.Get(context.Background(), server.URL, types.SM{"User-Agent": "rclone/v1.68.0"})
	if e != nil {
		t.Fatal(e)
	}
	_ = resp.Dispose()
	if got != "rclone/v1.68.0" {
		t.Fatalf("configured User-Agent = %q", got)
	}
}

func TestDefaultHTTPClientAllowsCallerUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client, e := NewClient("", nil, nil, DefaultHTTPClient)
	if e != nil {
		t.Fatal(e)
	}
	resp, e := client.Get(context.Background(), server.URL, types.SM{"User-Agent": "rclone/v1.68.0"})
	if e != nil {
		t.Fatal(e)
	}
	_ = resp.Dispose()
	if got != "rclone/v1.68.0" {
		t.Fatalf("configured User-Agent = %q", got)
	}

	direct, e := http.NewRequest(http.MethodGet, server.URL, nil)
	if e != nil {
		t.Fatal(e)
	}
	direct.Header.Set("User-Agent", "aws-sdk-go-v2/1.41.5")
	raw, e := DefaultHTTPClient.Do(direct)
	if e != nil {
		t.Fatal(e)
	}
	_ = raw.Body.Close()
	if got != DefaultUserAgent {
		t.Fatalf("DefaultHTTPClient User-Agent = %q", got)
	}
}
