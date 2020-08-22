package server

import (
	"go-drive/common"
	"os"
	"testing"
	"time"
)

const (
	validity = 2 * time.Second
	cleanup  = 1 * time.Second
)

var ts *MemTokenStore

func TestMain(m *testing.M) {
	ts = NewMemTokenStore(validity, true, cleanup)

	code := m.Run()

	ts.Dispose()

	os.Exit(code)
}

func TestMemTokenStore_Create(t *testing.T) {
	t0, e := ts.Create("hello world")
	if e != nil {
		t.Error(e)
	}
	t1, e := ts.Validate(t0.Token)
	if e != nil {
		t.Error(e)
		return
	}
	t2, ok := t1.(Token)
	if !ok {
		t.Error("token store returned invalid token")
		return
	}
	if t2.Value != "hello world" {
		t.Error("the value of token store returned token is not same")
		return
	}
}

func TestMemTokenStore_Update(t *testing.T) {
	t0, e := ts.Create("hello world")
	if e != nil {
		t.Error(e)
	}
	_, _ = ts.Update(t0.Token, "Hello World")
	t1, e := ts.Validate(t0.Token)
	if e != nil {
		t.Error(e)
		return
	}
	t2, ok := t1.(Token)
	if !ok {
		t.Error("token store returned invalid token")
		return
	}
	if t2.Value != "Hello World" {
		t.Error("the value of token store returned token is not same after update")
		return
	}
}

func TestMemTokenStore_Revoke(t *testing.T) {
	t0, e := ts.Create("hello world")
	if e != nil {
		t.Error(e)
	}
	_ = ts.Revoke(t0.Token)

	_, e = ts.Validate(t0.Token)
	if _, ok := e.(common.UnauthorizedError); !ok {
		t.Error("expect InvalidTokenError")
		return
	}
}

func TestMemTokenStore_AutoRefresh(t *testing.T) {

	t0, e := ts.Create("hello world")
	if e != nil {
		t.Error(e)
	}

	// after 1s
	<-time.After(1 * time.Second)
	// refresh token
	_, _ = ts.Validate(t0.Token)

	// 2s, not expired because of refreshing
	<-time.After(1 * time.Second)
	_, e = ts.Validate(t0.Token)
	if e != nil {
		t.Error("expect token refreshed")
		return
	}
	// expired
	<-time.After(validity)

	_, e = ts.Validate(t0.Token)
	if _, ok := e.(common.UnauthorizedError); !ok {
		t.Error("expect InvalidTokenError")
		return
	}

	// cleaned
	<-time.After(cleanup)

	_, ok := ts.store.Get(t0.Token)
	if ok {
		t.Error("expect expired token gets cleaned")
		return
	}

}
