package driveutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"go-drive/common/types"

	"golang.org/x/oauth2"
)

type memOAuthData struct {
	data types.SM
}

type trackingRequestBody struct {
	closed bool
}

func (*trackingRequestBody) Read([]byte) (int, error) { return 0, io.EOF }

func (b *trackingRequestBody) Close() error {
	b.closed = true
	return nil
}

func (m *memOAuthData) Save(data types.SM) error {
	if m.data == nil {
		m.data = types.SM{}
	}
	for k, v := range data {
		if v == "" {
			delete(m.data, k)
			continue
		}
		m.data[k] = v
	}
	return nil
}

func (m *memOAuthData) Load(key string, keys ...string) (types.SM, error) {
	r := types.SM{}
	keys = append([]string{key}, keys...)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			r[k] = v
		}
	}
	return r, nil
}

func (m *memOAuthData) Clear() error {
	m.data = types.SM{}
	return nil
}

func TestOAuthTokenUsesValidCachedToken(t *testing.T) {
	ds := &memOAuthData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		DsKeyToken:        "access",
		DsKeyTokenType:    "Bearer",
		DsKeyRefreshToken: "refresh",
		DsKeyExpiresAt:    strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	oauthHolder, e := OAuthLoad(OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/unused"},
	}, OAuthCredentials{ClientID: "id", ClientSecret: "secret"}, ds)
	if e != nil {
		t.Fatal(e)
	}
	tok, e := oauthHolder.Token(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if tok.AccessToken != "access" {
		t.Fatalf("token = %q", tok.AccessToken)
	}
}

func TestOAuthTokenRefreshHonorsCallerContext(t *testing.T) {
	ds := &memOAuthData{}
	if e := ds.Save(types.SM{
		DsKeyToken:        "expired",
		DsKeyTokenType:    "Bearer",
		DsKeyRefreshToken: "refresh",
		DsKeyExpiresAt:    "1",
	}); e != nil {
		t.Fatal(e)
	}
	oauthHolder, e := OAuthLoad(OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/token"},
	}, OAuthCredentials{ClientID: "id", ClientSecret: "secret"}, ds)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = oauthHolder.Token(ctx)
	if e == nil {
		t.Fatal("expected refresh to fail")
	}
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("refresh error = %v, want context.Canceled", e)
	}
}

func TestOAuthTokenNilContextPanics(t *testing.T) {
	ds := &memOAuthData{}
	expiry := time.Now().Add(time.Hour).Unix()
	if e := ds.Save(types.SM{
		DsKeyToken:     "access",
		DsKeyTokenType: "Bearer",
		DsKeyExpiresAt: strconv.FormatInt(expiry, 10),
	}); e != nil {
		t.Fatal(e)
	}
	oauthHolder, e := OAuthLoad(OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/unused"},
	}, OAuthCredentials{ClientID: "id", ClientSecret: "secret"}, ds)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil context to panic")
		}
	}()
	_, _ = oauthHolder.Token(nil)
}

func TestOAuthClientClosesBodyWhenTokenRequestFails(t *testing.T) {
	ds := &memOAuthData{}
	if e := ds.Save(types.SM{
		DsKeyToken:        "expired",
		DsKeyTokenType:    "Bearer",
		DsKeyRefreshToken: "refresh",
		DsKeyExpiresAt:    "1",
	}); e != nil {
		t.Fatal(e)
	}
	oauthHolder, e := OAuthLoad(OAuthRequest{
		Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:1/token"},
	}, OAuthCredentials{ClientID: "id", ClientSecret: "secret"}, ds)
	if e != nil {
		t.Fatal(e)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	body := &trackingRequestBody{}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:1/data", body)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = oauthHolder.Client().Do(req)
	if !body.closed {
		t.Fatal("request body was not closed after token retrieval failed")
	}
}
