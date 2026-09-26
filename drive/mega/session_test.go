package mega

import (
	"errors"
	"testing"

	"go-drive/common/types"

	megaapi "go-drive/drive/mega/internal/gomega"
)

func TestRestoreOrLoginReusesSessionUntilCredentialsChange(t *testing.T) {
	auth := &fakeAuth{}
	store := &memData{}

	if e := restoreOrLogin(auth, "user@example.com", "secret", "123456", store); e != nil {
		t.Fatalf("first login: %v", e)
	}
	if auth.logins != 1 || auth.lastMFA != "123456" || store.values[sessionSIDKey] == "" {
		t.Fatalf("login = %+v store = %#v", auth, store.values)
	}

	auth.logins = 0
	auth.lastMFA = ""
	if e := restoreOrLogin(auth, "user@example.com", "secret", "999999", store); e != nil {
		t.Fatalf("restore: %v", e)
	}
	if auth.keyLogins != 1 || auth.logins != 0 {
		t.Fatalf("restored login counted password login: %+v", auth)
	}

	if e := restoreOrLogin(auth, "user@example.com", "changed", "", store); e != nil {
		t.Fatalf("password change: %v", e)
	}
	if auth.logins != 1 || store.values[sessionPasswordTagKey] != passwordTag("user@example.com", "changed") {
		t.Fatalf("password change did not create a new session: %+v", auth)
	}
}

func TestRestoreOrLoginFallsBackWhenSessionExpires(t *testing.T) {
	auth := &fakeAuth{keysErr: megaapi.ESID}
	store := &memData{values: types.SM{
		sessionSIDKey:         "old",
		sessionMasterKeyKey:   "b2xk",
		sessionEmailKey:       "user@example.com",
		sessionPasswordTagKey: passwordTag("user@example.com", "secret"),
	}}
	if e := restoreOrLogin(auth, "user@example.com", "secret", "654321", store); e != nil {
		t.Fatalf("fallback: %v", e)
	}
	if auth.keyLogins != 1 || auth.logins != 1 || auth.lastMFA != "654321" {
		t.Fatalf("auth = %+v", auth)
	}
}

func TestProbeSessionAsksForCodeWithoutStoringIt(t *testing.T) {
	auth := &fakeAuth{loginErr: megaapi.EMFAREQUIRED}
	store := &memData{}
	cfg, e := probeSession(auth, "user@example.com", "secret", store)
	if e != nil {
		t.Fatalf("probe: %v", e)
	}
	if cfg.Configured || len(cfg.Form) != 1 || cfg.Form[0].Field != "mfa" || !cfg.Form[0].Required {
		t.Fatalf("config = %#v", cfg)
	}
	if len(store.values) != 0 {
		t.Fatalf("challenge stored data: %#v", store.values)
	}

	auth.loginErr = nil
	if e = loginWithCode(auth, "user@example.com", "secret", "123456", store); e != nil {
		t.Fatalf("code: %v", e)
	}
	if _, ok := store.values["mfa"]; ok || store.values[sessionSIDKey] == "" || auth.lastMFA != "123456" {
		t.Fatalf("store = %#v auth = %+v", store.values, auth)
	}
}

func TestProbeSessionCompletesWithoutCode(t *testing.T) {
	auth := &fakeAuth{}
	store := &memData{}
	cfg, e := probeSession(auth, "user@example.com", "secret", store)
	if e != nil {
		t.Fatalf("probe: %v", e)
	}
	if cfg == nil || !cfg.Configured || len(cfg.Form) != 0 || auth.lastMFA != "" || store.values[sessionSIDKey] == "" {
		t.Fatalf("config = %#v auth = %+v store = %#v", cfg, auth, store.values)
	}
}

func TestRestoreOrLoginMapsCredentialErrors(t *testing.T) {
	auth := &fakeAuth{loginErr: megaapi.EMFAREQUIRED}
	e := restoreOrLogin(auth, "user@example.com", "secret", "", &memData{})
	var unauthorized interface{ Code() int }
	if !errors.As(e, &unauthorized) || unauthorized.Code() != 401 {
		t.Fatalf("error = %v", e)
	}

	auth.loginErr = megaapi.EAGAIN
	if e = restoreOrLogin(auth, "user@example.com", "secret", "", &memData{}); !errors.Is(e, megaapi.EAGAIN) {
		t.Fatalf("temporary error = %v", e)
	}
}

type fakeAuth struct {
	sid       string
	key       []byte
	keysErr   error
	loginErr  error
	logins    int
	keyLogins int
	lastMFA   string
}

func (f *fakeAuth) LoginWithKeys(string, []byte) error {
	f.keyLogins++
	return f.keysErr
}

func (f *fakeAuth) MultiFactorLogin(_, _, multiFactor string) error {
	f.logins++
	f.lastMFA = multiFactor
	if f.loginErr != nil {
		return f.loginErr
	}
	f.sid = "sid-1"
	f.key = []byte("0123456789abcdef")
	return nil
}

func (f *fakeAuth) GetSessionID() string { return f.sid }

func (f *fakeAuth) GetMasterKey() []byte { return f.key }

type memData struct {
	values types.SM
}

func (m *memData) Save(values types.SM) error {
	m.values = values
	return nil
}

func (m *memData) SaveEncrypted(values types.SM) error {
	return m.Save(values)
}

func (m *memData) Load(key string, keys ...string) (types.SM, error) {
	names := append([]string{key}, keys...)
	loaded := types.SM{}
	for _, name := range names {
		if value, ok := m.values[name]; ok {
			loaded[name] = value
		}
	}
	return loaded, nil
}

func (m *memData) Clear() error {
	m.values = nil
	return nil
}
