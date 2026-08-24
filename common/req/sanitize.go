package req

import (
	"errors"
	"fmt"
	"go-drive/common/logging"
	"net/url"
	"strings"
)

// SanitizeURL returns a URL suitable for diagnostic logs. Query parameters
// that commonly carry credentials, tokens, signatures, or OAuth values are
// retained by name but have their values replaced.
func SanitizeURL(rawURL string) string {
	u, e := url.Parse(rawURL)
	if e != nil {
		return logging.Sanitize(rawURL)
	}
	u.User = nil
	u.Fragment = ""
	query := u.Query()
	for key := range query {
		if isSensitiveQueryKey(key) {
			query[key] = []string{"REDACTED"}
		}
	}
	u.RawQuery = query.Encode()
	return logging.Sanitize(u.String())
}

// SanitizeError formats an HTTP error without exposing credentials embedded in
// a URL. In particular, net/http commonly wraps request failures in
// *url.Error, whose Error method includes the original URL.
func SanitizeError(e error) string {
	if e == nil {
		return ""
	}
	var urlError *url.Error
	if errors.As(e, &urlError) {
		return logging.Sanitize(fmt.Sprintf("%s %s: %s", urlError.Op,
			SanitizeURL(urlError.URL), SanitizeError(urlError.Err)))
	}
	return logging.Sanitize(e.Error())
}

func isSensitiveQueryKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", ""), "_", ""))
	for _, part := range []string{
		"token", "accesstoken", "refreshtoken", "idtoken", "authorization",
		"password", "passwd", "secret", "clientsecret", "apikey", "api_key",
		"signature", "sig", "oauth", "code", "state", "session", "ticket",
	} {
		if strings.Contains(key, strings.ReplaceAll(part, "_", "")) {
			return true
		}
	}
	return key == "key" || key == "k"
}
