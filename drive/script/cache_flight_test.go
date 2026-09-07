package script

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFlightGroupPanicReleasesWaitersAndKey(t *testing.T) {
	var group flightGroup
	var call *flightCall
	func() {
		defer func() {
			if p := recover(); p != "load panic" {
				t.Errorf("leader panic=%v", p)
			}
		}()
		_, _ = group.do(context.Background(), "key", func() (any, error) { call = group.m["key"]; panic("load panic") })
	}()
	if len(group.m) != 0 {
		t.Fatal("panic left a pending call")
	}
	select {
	case <-call.done:
	case <-time.After(time.Second):
		t.Fatal("pending waiters were not released")
	}
	if call.panicValue != "load panic" {
		t.Fatalf("waiter panic=%v", call.panicValue)
	}
	v, e := group.do(context.Background(), "key", func() (any, error) { return "retry", nil })
	if e != nil || v != "retry" {
		t.Fatalf("retry=%v, %v", v, e)
	}
}

func TestFlightGroupWaiterDeadlineDoesNotCancelLoader(t *testing.T) {
	var group flightGroup
	started := make(chan struct{})
	release := make(chan struct{})
	leaderDone := make(chan struct{})
	var leaderValue any
	var leaderErr error
	go func() {
		defer close(leaderDone)
		leaderValue, leaderErr = group.do(context.Background(), "key", func() (any, error) {
			close(started)
			<-release
			return "loaded", nil
		})
	}()
	defer func() { close(release); <-leaderDone }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	waiterDone := make(chan error, 1)
	go func() {
		_, e := group.do(ctx, "key", func() (any, error) {
			return nil, errors.New("waiter unexpectedly became loader")
		})
		waiterDone <- e
	}()
	select {
	case e := <-waiterDone:
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("waiter error=%v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("waiter ignored deadline")
	}
	group.mu.Lock()
	_, pending := group.m["key"]
	group.mu.Unlock()
	if !pending {
		t.Fatal("canceling waiter removed active loader")
	}
	// Check the loader's result after the deferred release and join.
	t.Cleanup(func() {
		if leaderErr != nil || leaderValue != "loaded" {
			t.Errorf("leader=%v, %v", leaderValue, leaderErr)
		}
	})
}

func TestFlightGroupCanceledContextDoesNotLoad(t *testing.T) {
	var group flightGroup
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := group.do(ctx, "key", func() (any, error) {
		t.Fatal("started loader with canceled context")
		return nil, nil
	})
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("error=%v", e)
	}
}
