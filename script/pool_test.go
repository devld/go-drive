package script

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingDisposable struct {
	count *atomic.Int32
}

func (d countingDisposable) Dispose() {
	d.count.Add(1)
}

func newPoolTestVM(t *testing.T) *VM {
	t.Helper()
	vm, e := NewVM()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = vm.Dispose() })
	return vm
}

func TestVMPoolWaitsForAvailableVM(t *testing.T) {
	pool := NewVMPool(newPoolTestVM(t), &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	t.Cleanup(func() { _ = pool.Dispose() })

	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, e := pool.Get(ctx); e != context.DeadlineExceeded {
		t.Fatalf("expected deadline exceeded, got %v", e)
	}
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
}

func TestVMPoolReturnDisposesRequestResources(t *testing.T) {
	pool := NewVMPool(newPoolTestVM(t), &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	t.Cleanup(func() { _ = pool.Dispose() })
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	var disposed atomic.Int32
	vm.PutDisposable(countingDisposable{count: &disposed})
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
	if disposed.Load() != 1 {
		t.Fatalf("disposable called %d times", disposed.Load())
	}
	reused, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if reused != vm {
		t.Fatal("expected idle VM to be reused")
	}
	if e := pool.Return(context.Background(), reused); e != nil {
		t.Fatal(e)
	}
}

func TestVMPoolEvictsIdleVMsAboveMinimum(t *testing.T) {
	pool := NewVMPool(newPoolTestVM(t), &VMPoolConfig{
		MaxTotal: 2,
		MaxIdle:  2,
		MinIdle:  1,
		IdleTime: 20 * time.Millisecond,
	})
	t.Cleanup(func() { _ = pool.Dispose() })

	first, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	second, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(context.Background(), first); e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(context.Background(), second); e != nil {
		t.Fatal(e)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		pool.mu.Lock()
		idle, total := len(pool.idle), pool.total
		pool.mu.Unlock()
		if idle == 1 && total == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("idle VM above MinIdle was not evicted")
}

func TestVMPoolDisposeWakesWaiters(t *testing.T) {
	pool := NewVMPool(newPoolTestVM(t), &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	result := make(chan error, 1)
	go func() {
		_, e := pool.Get(context.Background())
		result <- e
	}()
	if e := pool.Dispose(); e != nil {
		t.Fatal(e)
	}
	if e := <-result; e != ErrVMPoolClosed {
		t.Fatalf("expected closed error, got %v", e)
	}
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
}
