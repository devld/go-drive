package driveutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// byteReaderGetter returns a ReaderGetter serving the given data. It handles
// the "whole file" request (start == -1) used for files smaller than the block
// size, as well as ranged requests.
func byteReaderGetter(data []byte) ReaderGetter {
	return func(_ context.Context, start, size int64) (io.ReadCloser, error) {
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
// fed, then returns without a lost wakeup.
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
	pool, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 8, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := bytes.Repeat([]byte("hello world "), 1000)

	r, e := pool.GetReader(context.Background(), "k1", int64(len(data)), byteReaderGetter(data))
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
	pool, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 8, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("0123456789abcdefghij")

	r, e := pool.GetReader(context.Background(), "k1", int64(len(data)), byteReaderGetter(data))
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

func TestCacheFilePool_ReadAtDoesNotChangePosition(t *testing.T) {
	dir := t.TempDir()
	p, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 2, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("0123456789abcdefghij")
	r, e := p.GetReader(context.Background(), "read-at", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r.Close() }()

	readerAt, ok := r.(io.ReaderAt)
	if !ok {
		t.Fatal("cache reader does not implement io.ReaderAt")
	}
	part := make([]byte, 4)
	if _, e := readerAt.ReadAt(part, 6); e != nil {
		t.Fatal(e)
	}
	if string(part) != "6789" {
		t.Fatalf("ReadAt() = %q", part)
	}
	all, e := io.ReadAll(r)
	if e != nil {
		t.Fatal(e)
	}
	if string(all) != string(data) {
		t.Fatalf("Read after ReadAt() = %q", all)
	}
}

func TestCacheFilePool_ReturnsSourceErrorToWaitingReaders(t *testing.T) {
	dir := t.TempDir()
	p, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 2, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	want := errors.New("source failed")
	getter := func(context.Context, int64, int64) (io.ReadCloser, error) {
		return nil, want
	}
	r1, e := p.GetReader(context.Background(), "error", 4, getter)
	if e != nil {
		t.Fatal(e)
	}
	r2, e := p.GetReader(context.Background(), "error", 4, getter)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r1.Close(); _ = r2.Close() }()

	results := make(chan error, 2)
	for _, r := range []io.Reader{r1, r2} {
		go func(r io.Reader) {
			_, readErr := io.ReadFull(r, make([]byte, 4))
			results <- readErr
		}(r)
	}
	for range 2 {
		if got := <-results; !errors.Is(got, want) {
			t.Fatalf("read error = %v, want %v", got, want)
		}
	}
}

func TestCacheFilePool_CancelDoesNotInterruptOtherReaders(t *testing.T) {
	dir := t.TempDir()
	pool, err := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 8, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("cached-file-a"), 256)

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	getter := func(ctx context.Context, start, size int64) (io.ReadCloser, error) {
		startOnce.Do(func() { close(started) })
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return byteReaderGetter(data)(ctx, start, size)
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	r1, err := pool.GetReader(ctx1, "a", int64(len(data)), getter)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := pool.GetReader(context.Background(), "a", int64(len(data)), getter)
	if err != nil {
		t.Fatal(err)
	}

	firstErr := make(chan error, 1)
	go func() {
		_, readErr := io.ReadAll(r1)
		_ = r1.Close()
		firstErr <- readErr
	}()
	<-started
	secondErr := make(chan error, 1)
	go func() {
		got, readErr := io.ReadAll(r2)
		_ = r2.Close()
		if readErr != nil {
			secondErr <- readErr
			return
		}
		if !bytes.Equal(got, data) {
			secondErr <- errors.New("unexpected payload")
			return
		}
		secondErr <- nil
	}()
	cancel1()
	close(release)
	if err := <-firstErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("first reader error = %v, want context.Canceled", err)
	}
	if err := <-secondErr; err != nil {
		t.Fatal(err)
	}
}

func TestCacheFilePool_LastReaderKeepsInFlightFill(t *testing.T) {
	dir := t.TempDir()
	pool, err := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 8, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("cached-file-a"), 32)

	started := make(chan struct{})
	release := make(chan struct{})
	var fetches atomic.Int32
	getter := func(ctx context.Context, start, size int64) (io.ReadCloser, error) {
		fetches.Add(1)
		inner, err := byteReaderGetter(data)(ctx, start, size)
		if err != nil {
			return nil, err
		}
		return &releaseReader{Reader: inner, ctx: ctx, release: release, started: started}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	r, err := pool.GetReader(ctx, "a", int64(len(data)), getter)
	if err != nil {
		t.Fatal(err)
	}
	readErr := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(r)
		_ = r.Close()
		readErr <- err
	}()
	<-started
	cancel()
	if err := <-readErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("reader error = %v, want context.Canceled", err)
	}
	select {
	case <-time.After(50 * time.Millisecond):
	case <-release:
	}
	if !pool.Has("a") {
		t.Fatal("cache entry was removed after the last reader closed")
	}
	if n := countCacheFiles(t, dir); n != 1 {
		t.Fatalf("cache files = %d, want 1", n)
	}

	close(release)
	r2, err := pool.GetReader(context.Background(), "a", int64(len(data)), getter)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r2.Close() }()
	got, err := io.ReadAll(r2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("reused cache payload mismatch")
	}
	if got := fetches.Load(); got != 1 {
		t.Fatalf("fetches = %d, want 1", got)
	}
}

// releaseReader blocks the first source read until release is closed, or until
// the fill context is cancelled.
type releaseReader struct {
	io.Reader
	ctx     context.Context
	release <-chan struct{}
	started chan struct{}
	once    sync.Once
}

func (r *releaseReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	select {
	case <-r.release:
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}
	return r.Reader.Read(p)
}

func (r *releaseReader) Close() error { return nil }

func TestCacheFilePool_ConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 8, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := bytes.Repeat([]byte("abcdefgh"), 4096)

	const n = 30
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			r, e := pool.GetReader(context.Background(), "shared", int64(len(data)), byteReaderGetter(data))
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

func TestCacheFilePool_MaxBytesEvictsOldest(t *testing.T) {
	dir := t.TempDir()
	data := []byte("cached content")
	p, err := NewCacheFilePool(CacheFilePoolOptions{
		MaxEntries: 4,
		MaxBytes:   int64(len(data)),
		Dir:        dir,
	})
	if err != nil {
		t.Fatal(err)
	}

	r1, err := p.GetReader(context.Background(), "first", int64(len(data)), byteReaderGetter(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(r1); err != nil {
		t.Fatal(err)
	}
	_ = r1.Close()

	r2, err := p.GetReader(context.Background(), "second", int64(len(data)), byteReaderGetter(data))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r2.Close() }()
	if _, err := io.ReadAll(r2); err != nil {
		t.Fatal(err)
	}
	eventually(t, time.Second, func() bool { return countCacheFiles(t, dir) == 1 })
}

// TestCacheFilePool_EvictionRemovesFile verifies the backing file is removed
// once an idle entry is evicted from the pool.
func TestCacheFilePool_EvictionRemovesFile(t *testing.T) {
	dir := t.TempDir()
	pool, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 1, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("some cached content")

	r1, e := pool.GetReader(context.Background(), "k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, e := io.ReadAll(r1); e != nil {
		t.Fatal(e)
	}
	_ = r1.Close()

	// adding a second entry evicts k1 (capacity 1)
	r2, e := pool.GetReader(context.Background(), "k2", int64(len(data)), byteReaderGetter(data))
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
	pool, e := NewCacheFilePool(CacheFilePoolOptions{MaxEntries: 1, Dir: dir})
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("some cached content")

	// active reader on k1 (not read, not closed)
	r1, e := pool.GetReader(context.Background(), "k1", int64(len(data)), byteReaderGetter(data))
	if e != nil {
		t.Fatal(e)
	}

	// evict k1 by adding k2
	r2, e := pool.GetReader(context.Background(), "k2", int64(len(data)), byteReaderGetter(data))
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

func TestDownloadMap(t *testing.T) {
	block := int64(cacheBlockSize)
	if got := downloadMap(0, nil); got != "" {
		t.Fatalf("empty file: got %q", got)
	}
	if got := downloadMap(block, [][]int64{{0, block}}); got != "#" {
		t.Fatalf("one full block: got %q", got)
	}
	if got := downloadMap(block*3, [][]int64{{block, block * 2}}); got != "_#_" {
		t.Fatalf("middle block: got %q", got)
	}
	if got := downloadMap(block*2, [][]int64{{0, block / 2}}); got != "__" {
		t.Fatalf("partial block: got %q", got)
	}
	if got := downloadMap(block*21, [][]int64{{0, block * 21}}); got != strings.Repeat("#", cacheMapCells) {
		t.Fatalf("scaled full file: got %q", got)
	}
	half := strings.Repeat("#", cacheMapCells/2) + strings.Repeat("_", cacheMapCells/2)
	if got := downloadMap(block*40, [][]int64{{0, block * 20}}); got != half {
		t.Fatalf("scaled half file: got %q", got)
	}
	// 40 blocks scale to 20 cells, so the first cell is two blocks. One downloaded block is half of that cell.
	partial := "=" + strings.Repeat("_", cacheMapCells-1)
	if got := downloadMap(block*40, [][]int64{{0, block}}); got != partial {
		t.Fatalf("scaled partial cell: got %q", got)
	}
	sparse := "." + strings.Repeat("_", cacheMapCells-1)
	if got := downloadMap(block*40, [][]int64{{0, 1}}); got != sparse {
		t.Fatalf("scaled sparse cell: got %q", got)
	}
}
