package server

import (
	"context"
	"errors"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type notifyingRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
	once    sync.Once
}

func (r *notifyingRecorder) Flush() {
	r.ResponseRecorder.Flush()
	r.once.Do(func() { close(r.flushed) })
}

func TestExecuteTaskStreamingWritesWhileActive(t *testing.T) {
	runner := task.NewTaskRunner(common.Config{MaxConcurrentTask: 1})
	t.Cleanup(func() { _ = runner.Dispose() })

	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/jobs/test", nil)
	SetMessageSource(c, responseTestMessages{})
	err := ExecuteTaskStreaming(c, runner, func(_ types.TaskCtx, stream io.Writer) (any, error) {
		if _, err := stream.Write([]byte("ready\n")); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Body.String(); got != "ready\n" {
		t.Fatalf("response body = %q", got)
	}
	if !response.Flushed {
		t.Fatal("streaming response was not flushed")
	}
	if response.Header().Get(common.ResponseHeaderKey) == "" {
		t.Fatal("streaming task header is missing")
	}
}

func TestExecuteTaskStreamingClosesWriterOnRequestCancel(t *testing.T) {
	runner := task.NewTaskRunner(common.Config{MaxConcurrentTask: 1})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		if err := runner.Dispose(); err != nil {
			t.Errorf("dispose task runner: %v", err)
		}
	})

	response := &notifyingRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{})}
	c, _ := gin.CreateTestContext(response)
	requestCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest(http.MethodGet, "/jobs/test", nil).WithContext(requestCtx)
	SetMessageSource(c, responseTestMessages{})

	started := make(chan struct{})
	lateWrite := make(chan error, 1)
	returned := make(chan error, 1)
	go func() {
		returned <- ExecuteTaskStreaming(c, runner, func(_ types.TaskCtx, stream io.Writer) (any, error) {
			if _, err := stream.Write([]byte("before\n")); err != nil {
				return nil, err
			}
			close(started)
			<-release
			_, err := stream.Write([]byte("after\n"))
			lateWrite <- err
			return nil, nil
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("streaming task did not start")
	}
	select {
	case <-response.flushed:
	case <-time.After(time.Second):
		t.Fatal("streaming response was not flushed")
	}
	cancel()
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("request handler did not return after cancellation")
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-lateWrite:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("late write error = %v, want closed pipe", err)
		}
	case <-time.After(time.Second):
		t.Fatal("task did not attempt the late write")
	}
	if got := response.Body.String(); got != "before\n" {
		t.Fatalf("response body = %q, want only data written before cancellation", got)
	}
}
