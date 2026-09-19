package utils

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestKeyLockSerializesAndReleasesEntries(t *testing.T) {
	locks := NewKeyLock(0)
	var running atomic.Int32
	var overlapped atomic.Bool
	var wg sync.WaitGroup

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locks.Lock("same")
			defer locks.Unlock("same")
			if running.Add(1) != 1 {
				overlapped.Store(true)
			}
			runtime.Gosched()
			running.Add(-1)
		}()
	}
	wg.Wait()

	if overlapped.Load() {
		t.Fatal("same key was used concurrently")
	}
	if got := len(locks.m); got != 0 {
		t.Fatalf("lock entries = %d, want 0 after release", got)
	}
}
