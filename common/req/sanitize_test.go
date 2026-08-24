package req

import (
	"errors"
	"net/url"
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	got := SanitizeURL("https://user:password@example.com/path?token=secret&code=oauth-code&_k=signature&safe=value#fragment")
	want := "https://example.com/path?_k=REDACTED&code=REDACTED&safe=value&token=REDACTED"
	if got != want {
		t.Fatalf("SanitizeURL() = %q, want %q", got, want)
	}
}

func TestSanitizeError(t *testing.T) {
	err := &url.Error{Op: "Get", URL: "https://example.com/?access_token=secret", Err: errors.New("connection refused")}
	got := SanitizeError(err)
	want := "Get https://example.com/?access_token=REDACTED: connection refused"
	if got != want {
		t.Fatalf("SanitizeError() = %q, want %q", got, want)
	}
}
