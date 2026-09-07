package script

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingDisposable struct {
	count *atomic.Int32
}

func TestVMPoolClearsRemovedIdleReferences(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 3, MaxIdle: 3, MinIdle: 3})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if tail := pool.idle[:cap(pool.idle)][2]; tail.vm != nil {
		t.Fatal("checkout retained the VM in the idle backing array")
	}
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
	// No background cleaner is running. Expire the first two members while
	// keeping the last member idle, exercising compaction as well as cleanup.
	expired := []*VM{pool.idle[0].vm, pool.idle[1].vm}
	pool.config.MinIdle = 1
	pool.config.IdleTime = time.Hour
	for i := range 2 {
		pool.idle[i].returnedAt = time.Now().Add(-2 * time.Hour)
	}
	pool.cleanExpired()
	if len(pool.idle) != 1 || pool.idle[0].vm != vm || pool.total != 1 || len(pool.members) != 1 {
		t.Fatal("expiration did not preserve the remaining idle member")
	}
	for _, item := range pool.idle[:cap(pool.idle)][1:] {
		if item.vm != nil {
			t.Fatal("expiration retained a VM in the idle backing array")
		}
	}
	for _, expiredVM := range expired {
		if !expiredVM.disposed {
			t.Fatal("expired VM was not disposed")
		}
	}
}

func TestVMPoolCreatesBaseVMBeforeInitializer(t *testing.T) {
	var initialized atomic.Int32
	pool, e := NewVMPool(context.Background(), func(ctx context.Context, vm *VM) error {
		if value, getErr := vm.GetValue("urlUtils"); getErr != nil || value.IsNil() {
			t.Fatalf("base VM APIs unavailable before initializer: value=%v err=%v", value, getErr)
		}
		sequence := initialized.Add(1)
		return vm.DefineGlobal("initializerSequence", sequence)
	}, &VMPoolConfig{MaxTotal: 4, MaxIdle: 4, MinIdle: 2})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	first, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	second, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if first == second {
		t.Fatal("pool returned the same Runtime twice")
	}
	if initialized.Load() != 2 {
		t.Fatalf("initializer calls = %d, want 2", initialized.Load())
	}
	if e := pool.Return(context.Background(), first); e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(context.Background(), second); e != nil {
		t.Fatal(e)
	}
}

func TestVMPoolConcurrentMembersAreIndependent(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 8, MaxIdle: 8})
	const workers = 8
	start := make(chan struct{})
	results := make(chan *VM, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			vm, e := pool.Get(context.Background())
			if e != nil {
				errs <- e
				return
			}
			results <- vm
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	seen := make(map[*VM]struct{}, workers)
	for vm := range results {
		if _, exists := seen[vm]; exists {
			t.Fatal("a Runtime was checked out concurrently more than once")
		}
		seen[vm] = struct{}{}
	}
	if len(seen) != workers {
		t.Fatalf("checked out %d Runtimes, want %d", len(seen), workers)
	}
	for vm := range seen {
		if e := pool.Return(context.Background(), vm); e != nil {
			t.Fatal(e)
		}
	}
}

func TestVMPoolRejectsDoubleReturn(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(context.Background(), vm); e == nil {
		t.Fatal("double return unexpectedly succeeded")
	}
}

func TestVMPoolDiscardsPromiseReturningVM(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), `new Promise(function () {})`, ""); e == nil {
		t.Fatal("pending Promise unexpectedly succeeded")
	}
	if e := pool.Return(context.Background(), vm); e != nil {
		t.Fatal(e)
	}
	replacement, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if replacement == vm {
		t.Fatal("non-reusable Runtime was returned to the idle pool")
	}
	if e := pool.Return(context.Background(), replacement); e != nil {
		t.Fatal(e)
	}
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

func mustDefineGlobal(t testing.TB, vm *VM, name string, value any) {
	t.Helper()
	if e := vm.DefineGlobal(name, value); e != nil {
		t.Fatal(e)
	}
}

func newTestPool(t *testing.T, config *VMPoolConfig) *VMPool {
	t.Helper()
	pool, e := NewVMPool(context.Background(), nil, config)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	return pool
}

func TestVMPoolWaitsForAvailableVM(t *testing.T) {
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})

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
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	var disposed atomic.Int32
	vm.PutDisposable(&countingDisposable{count: &disposed})
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
	pool := newTestPool(t, &VMPoolConfig{
		MaxTotal: 2,
		MaxIdle:  2,
		MinIdle:  1,
		IdleTime: 20 * time.Millisecond,
	})

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
	pool := newTestPool(t, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
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

func TestVMPoolFreezesGlobalAfterInitialize(t *testing.T) {
	pool, e := NewVMPool(context.Background(), func(_ context.Context, vm *VM) error {
		return vm.DefineGlobal("fromInit", 1)
	}, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1, MinIdle: 1})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = pool.Return(context.Background(), vm) }()
	got, e := vm.Run(context.Background(), `
		fromInit = 2;
		this.extraGlobal = true;
		[fromInit, typeof extraGlobal, Object.isFrozen(this)].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "1|undefined|true" {
		t.Fatalf("frozen global = %q", got.String())
	}
}

func TestVMPoolFreezeGlobalIgnoresThrowingObjectFreeze(t *testing.T) {
	pool, e := NewVMPool(context.Background(), func(_ context.Context, vm *VM) error {
		_, err := vm.Run(context.Background(), `
			Object.defineProperty(Object, "freeze", {
				get() { throw new Error("hijacked freeze"); },
				configurable: true,
			});
		`, "")
		return err
	}, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1, MinIdle: 1})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = pool.Dispose() })
	assertPoolGlobalFrozen(t, pool)
}

func assertPoolGlobalFrozen(t *testing.T, pool *VMPool) {
	t.Helper()
	vm, e := pool.Get(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = pool.Return(context.Background(), vm) }()
	got, e := vm.Run(context.Background(), `
		this.extraGlobal = true;
		[typeof extraGlobal, Object.isFrozen(this)].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	if got.String() != "undefined|true" {
		t.Fatalf("frozen global = %q", got.String())
	}
}

func TestVMPoolNewVMHonorsCanceledContextAfterInitialize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, e := NewVMPool(ctx, func(context.Context, *VM) error {
		cancel()
		return nil
	}, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1, MinIdle: 1})
	if e == nil {
		t.Fatal("expected canceled context to fail VM init")
	}
}

func TestVMPoolInitializerPanicReleasesCapacityAndResources(t *testing.T) {
	var calls, disposed atomic.Int32
	var failedVM *VM
	pool, e := NewVMPool(context.Background(), func(_ context.Context, vm *VM) error {
		if calls.Add(1) == 1 {
			failedVM = vm
			vm.PutDisposable(&countingDisposable{count: &disposed})
			panic("initializer failure")
		}
		return nil
	}, &VMPoolConfig{MaxTotal: 1, MaxIdle: 1})
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Dispose()
	func() {
		defer func() {
			if p := recover(); p != "initializer failure" {
				t.Errorf("panic=%v", p)
			}
		}()
		_, _ = pool.Get(context.Background())
	}()
	if disposed.Load() != 1 || failedVM.Reusable() {
		t.Fatal("failed VM was not disposed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	vm, e := pool.Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e := pool.Return(ctx, vm); e != nil {
		t.Fatal(e)
	}
}

func TestVMPoolPrewarmPanicDisposesAllMembers(t *testing.T) {
	var calls, disposed atomic.Int32
	func() {
		defer func() {
			if p := recover(); p != "prewarm failure" {
				t.Errorf("panic=%v", p)
			}
		}()
		_, _ = NewVMPool(context.Background(), func(_ context.Context, vm *VM) error {
			vm.PutDisposable(&countingDisposable{count: &disposed})
			if calls.Add(1) == 2 {
				panic("prewarm failure")
			}
			return nil
		}, &VMPoolConfig{MaxTotal: 2, MaxIdle: 2, MinIdle: 2})
	}()
	if disposed.Load() != 2 {
		t.Fatalf("disposed=%d, want 2", disposed.Load())
	}
}
