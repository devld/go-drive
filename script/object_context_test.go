package script

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-drive/common/task"
)

func TestGetContextAcceptsScriptWrappers(t *testing.T) {
	vm := newScriptTestVM(t)
	plain := NewContext(vm, context.Background())
	if GetContext(plain) == nil {
		t.Fatal("GetContext(Context) = nil")
	}
	if GetTaskCtx(plain) == nil {
		t.Fatal("GetTaskCtx(Context) = nil")
	}

	tc := NewTaskCtx(vm, task.DummyContext())
	if GetContext(tc) == nil {
		t.Fatal("GetContext(TaskCtx) = nil")
	}
	if GetTaskCtx(tc) == nil {
		t.Fatal("GetTaskCtx(TaskCtx) = nil")
	}
}

func TestNewContextWithTimeoutUsedByHTTP(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	vm := newScriptTestVM(t)
	vm.Set("url", srv.URL)
	done := make(chan error, 1)
	go func() {
		_, e := vm.Run(context.Background(), `
			var ctx = newContextWithTimeout(newContext(), ms(80));
			try {
				http(ctx, "GET", url);
			} finally {
				ctx.Cancel();
			}
		`)
		done <- e
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive request")
	}

	select {
	case e := <-done:
		if e == nil {
			t.Fatal("expected timeout error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("http() was not cancelled by timeout context")
	}
}
