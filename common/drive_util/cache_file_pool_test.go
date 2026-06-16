package drive_util

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// byteReaderGetter returns a ReaderGetter serving the given data. It handles
// the "whole file" request (start == -1) used for files smaller than the block
// size, as well as ranged requests.
func byteReaderGetter(data []byte) ReaderGetter {
	return func(start, size int64) (io.ReadCloser, error) {
		s := start
		if s < 0 {
			s = 0
		}
		end := int64(len(data))
		if size > 0 && s+size < end {
			end = s + size
		}
		return io.NopCloser(bytes.NewReader(data[s:end])), nil
	}
}

func countCacheFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for _, en := range entries {
		if strings.HasPrefix(en.Name(), "cache-") {
			n++
		}
	}
	return n
}

func eventually(t *testing.T, d time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", d)
}

func satisfyState(rl *rangeLock, start, length int64) bool {
	return rl.satisfy(start, length)
}

// TestRangeLock_MergeGapAndContained covers the previously broken _merge:
// it must not produce duplicate ranges, must merge adjacent ranges, and must
// never shrink a range when a contained range is fed.
func TestRangeLock_MergeGapAndContained(t *testing.T) {
	rl := newRangeLock(100)
	rl.feed(0, 10)  // [0,10]
	rl.feed(50, 10) // [50,60]
	rl.feed(10, 10) // [10,20] -> merges with [0,10] into [0,20]

	if !satisfyState(rl, 0, 20) {
		t.Errorf("expected [0,20] satisfied, ranges=%v", rl.ranges)
	}
	if !satisfyState(rl, 50, 10) {
		t.Errorf("expected [50,60] satisfied, ranges=%v", rl.ranges)
	}
	if satisfyState(rl, 20, 10) {
		t.Errorf("expected [20,30] NOT satisfied (gap), ranges=%v", rl.ranges)
	}
	if len(rl.ranges) != 2 {
		t.Errorf("expected exactly 2 merged ranges, got %v", rl.ranges)
	}

	// a fully contained range must not shrink the existing one
	rl2 := newRangeLock(100)
	rl2.feed(0, 100)
	rl2.feed(10, 10) // contained in [0,100]
	if !satisfyState(rl2, 0, 100) {
		t.Errorf("expected [0,100] still satisfied after contained feed, ranges=%v", rl2.ranges)
	}
	if len(rl2.ranges) != 1 {
		t.Errorf("expected exactly 1 range, got %v", rl2.ranges)
	}
}

// TestRangeLock_AcquireWaitsForFeed verifies acquire blocks until the range is
// fed, then returns. This exercises the sync.Cond based wakeup.
func TestRangeLock_AcquireWaitsForFeed(t *testing.T) {
	rl := newRangeLock(100)
	done := make(chan error, 1)
	go func() { done <- rl.acquire(0, 50) }()

	select {
	case <-done:
		t.Fatal("acquire returned before the range was fed")
	case <-time.After(50 * time.Millisecond):
	}

	rl.feed(0, 50)

	select {
	case e := <-done:
		if e != nil {
			t.Fatalf("unexpected error: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("acquire did not return after feed (lost wakeup?)")
	}
}

// TestRangeLock_ReleaseCancelsAcquire verifies a waiting acquire is woken with
// os.ErrClosed when the lock is released/canceled.
func TestRangeLock_ReleaseCancelsAcquire(t *testing.T) {
	rl := newRangeLock(100)
	done := make(chan error, 1)
	go func() { done <- rl.acquire(0, 50) }()

	time.Sleep(20 * time.Millisecond)
	rl.release()

	select {
	case e := <-done:
		if e != os.ErrClosed {
			t.Fatalf("expected os.ErrClosed, got %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("acquire did not return after release")
	}
}

func TestCacheFilePool_ReadFull(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFillPool(8, dir)
	if e != nil {
		t.Fatal(e)
	}
	data := bytes.Repeat([]byte("hello world "), 1000)

	r, e := pool.GetReader("k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(r)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %d bytes, want %d", len(got), len(data))
	}
	_ = r.Close()
}

func TestCacheFilePool_SeekRead(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFillPool(8, dir)
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("0123456789abcdefghij")

	r, e := pool.GetReader("k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r.Close() }()

	if _, e := r.Seek(6, io.SeekStart); e != nil {
		t.Fatal(e)
	}
	buf := make([]byte, 4)
	if _, e := io.ReadFull(r, buf); e != nil {
		t.Fatal(e)
	}
	if string(buf) != "6789" {
		t.Fatalf("expected 6789, got %q", string(buf))
	}
}

func TestCacheFilePool_ConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFillPool(8, dir)
	if e != nil {
		t.Fatal(e)
	}
	data := bytes.Repeat([]byte("abcdefgh"), 4096)

	const n = 30
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			r, e := pool.GetReader("shared", int64(len(data)), byteReaderGetter(data))
			if e != nil {
				errs <- e
				return
			}
			defer func() { _ = r.Close() }()
			got, e := io.ReadAll(r)
			if e != nil {
				errs <- e
				return
			}
			if !bytes.Equal(got, data) {
				errs <- io.ErrUnexpectedEOF
				return
			}
			errs <- nil
		}()
	}
	for i := 0; i < n; i++ {
		if e := <-errs; e != nil {
			t.Fatalf("reader %d failed: %v", i, e)
		}
	}
}

// TestCacheFilePool_EvictionRemovesFile verifies the backing file is removed
// once an idle entry is evicted from the pool.
func TestCacheFilePool_EvictionRemovesFile(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFillPool(1, dir)
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("some cached content")

	r1, e := pool.GetReader("k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, e := io.ReadAll(r1); e != nil {
		t.Fatal(e)
	}
	_ = r1.Close()

	// adding a second entry evicts k1 (capacity 1)
	r2, e := pool.GetReader("k2", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r2.Close() }()

	eventually(t, 2*time.Second, func() bool { return countCacheFiles(t, dir) == 1 })
}

// TestCacheFilePool_EvictionKeepsFileWhileActive verifies that evicting an
// entry that still has an active reader does NOT remove its file (avoiding
// truncating/leaking), and that the file is removed once the reader closes.
func TestCacheFilePool_EvictionKeepsFileWhileActive(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFillPool(1, dir)
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("some cached content")

	// active reader on k1 (not read, not closed)
	r1, e := pool.GetReader("k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}

	// evict k1 by adding k2
	r2, e := pool.GetReader("k2", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r2.Close() }()

	if n := countCacheFiles(t, dir); n != 2 {
		t.Fatalf("expected 2 files while k1 reader is active, got %d", n)
	}

	_ = r1.Close()
	eventually(t, 2*time.Second, func() bool { return countCacheFiles(t, dir) == 1 })
}
