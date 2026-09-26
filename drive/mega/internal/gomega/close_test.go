package gomega

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCloseStopsEventPoll(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	api := New()
	api.baseurl = srv.URL
	api.SetClient(srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	api.pollCancel = cancel
	api.pollDone = make(chan struct{})
	go func() {
		defer close(api.pollDone)
		api.pollEvents(ctx)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("poll did not start")
	}

	done := make(chan struct{})
	go func() {
		api.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return")
	}
}
