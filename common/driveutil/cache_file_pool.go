package driveutil

import (
	"context"
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/golang-lru/v2"
)

type ReaderGetter = func(int64, int64) (io.ReadCloser, error)
type ContextReaderGetter func(context.Context, int64, int64) (io.ReadCloser, error)

func NewCacheFillPool(maxCache int, dir string) (*CacheFilePool, error) {
	return NewCacheFillPoolWithOptions(CacheFilePoolOptions{
		MaxEntries: maxCache,
		Dir:        dir,
	})
}

// CacheFilePoolOptions controls both the number and the approximate total
// compressed size of cached sources. MaxBytes is zero for an unlimited byte
// budget, preserving the behavior of NewCacheFillPool's original API.
type CacheFilePoolOptions struct {
	MaxEntries   int
	MaxBytes     int64
	Dir          string
	CleanStartup bool
}

func NewCacheFillPoolWithOptions(options CacheFilePoolOptions) (*CacheFilePool, error) {
	maxCache := options.MaxEntries
	dir := options.Dir
	if dir == "" {
		dir = os.TempDir()
	}
	info, e := os.Stat(dir)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() {
		return nil, errors.New(dir + " is not a directory")
	}
	if options.CleanStartup {
		cleanupCacheFiles(dir)
	}
	pool := &CacheFilePool{dir: dir, maxBytes: options.MaxBytes}
	pool.entries, e = lru.NewWithEvict[string, *cacheFile](maxCache, pool.onCacheEvicted)
	if e != nil {
		return nil, e
	}

	return pool, nil
}

func cleanupCacheFiles(dir string) {
	entries, e := os.ReadDir(dir)
	if e != nil {
		return
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "cache-") || !entry.Type().IsRegular() {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

type CacheFilePool struct {
	dir      string
	entries  *lru.Cache[string, *cacheFile]
	maxBytes int64
	bytesMu  sync.Mutex
	bytes    int64
	mu       sync.Mutex
}

// Has reports whether a source currently has a live in-memory cache entry.
// It is intended for callers that can avoid an expensive capability probe on
// a cache hit; the result is only a hint because eviction may happen later.
func (cfp *CacheFilePool) Has(key string) bool {
	_, ok := cfp.entries.Peek(key)
	return ok
}

func (cfp *CacheFilePool) GetReader(key string, size int64, getReader ReaderGetter) (io.ReadSeekCloser, error) {
	return cfp.GetReaderContext(context.Background(), key, size,
		func(_ context.Context, start, length int64) (io.ReadCloser, error) {
			return getReader(start, length)
		},
	)
}

// GetReaderContext returns a seekable reader backed by the cache pool. The
// context belongs to this reader, rather than to the cache entry, so a cache
// hit never reuses the context from the request which first populated it.
func (cfp *CacheFilePool) GetReaderContext(ctx context.Context, key string, size int64, getReader ContextReaderGetter) (io.ReadSeekCloser, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cf, ok := cfp.entries.Get(key)
	if ok {
		return cf.Reader(ctx)
	}
	cfp.mu.Lock()
	defer cfp.mu.Unlock()

	// re-check after acquiring the write lock to avoid creating duplicated cacheFile
	if cf, ok = cfp.entries.Get(key); ok {
		return cf.Reader(ctx)
	}

	// Use a unique file per cacheFile instance instead of a deterministic name
	// derived from the key. Otherwise, when an entry is evicted while it still
	// has active readers, re-requesting the same key would create a new cacheFile
	// that opens the SAME file with O_TRUNC, truncating the file the old readers
	// are still reading from.
	file, e := os.CreateTemp(cfp.dir, "cache-")
	if e != nil {
		return nil, e
	}
	name := file.Name()
	_ = file.Close()

	newCf := &cacheFile{
		name:      name,
		size:      size,
		getReader: getReader,

		rl:      newRangeLock(size),
		readers: map[*cacheFileReader]struct{}{},

		wl:      newRangeLock(size),
		writers: make(map[io.WriteCloser]struct{}),
	}
	// evict removes this entry from the pool so that a later request for the
	// same key creates a fresh cacheFile instead of reusing a poisoned one
	// (e.g. after a download error). It is invoked from cacheFile.Close.
	newCf.evict = func() {
		cfp.mu.Lock()
		defer cfp.mu.Unlock()
		cfp.entries.Remove(key)
	}
	cf = newCf
	reader, e := cf.Reader(ctx)
	if e != nil {
		_ = os.Remove(name)
		return nil, e
	}
	cfp.entries.Add(key, cf)
	if cf.size > 0 {
		cfp.addBytes(cf.size)
	}
	cfp.trimBytes()

	return reader, nil
}

func (cfp *CacheFilePool) onCacheEvicted(_ string, cf *cacheFile) {
	if cf.size > 0 {
		cfp.addBytes(-cf.size)
	}
	cf.mu.Lock()
	defer cf.mu.Unlock()

	cf.evicted = true
	cf.removeFileIfIdleLocked()
}

func (cfp *CacheFilePool) addBytes(delta int64) {
	if delta == 0 {
		return
	}
	cfp.bytesMu.Lock()
	cfp.bytes += delta
	if cfp.bytes < 0 {
		cfp.bytes = 0
	}
	cfp.bytesMu.Unlock()
}

func (cfp *CacheFilePool) trimBytes() {
	if cfp.maxBytes <= 0 {
		return
	}
	for {
		cfp.bytesMu.Lock()
		over := cfp.bytes > cfp.maxBytes
		cfp.bytesMu.Unlock()
		if !over {
			return
		}
		if _, _, ok := cfp.entries.RemoveOldest(); !ok {
			return
		}
	}
}

// Dispose removes this pool's entries and marks active cache files for cleanup.
// Active readers keep their private backing file until they close.
func (cfp *CacheFilePool) Dispose() error {
	for _, key := range cfp.entries.Keys() {
		if cf, ok := cfp.entries.Peek(key); ok {
			cfp.entries.Remove(key)
			_ = cf.closeWithError(os.ErrClosed)
		}
	}
	return nil
}

type cacheFile struct {
	name      string
	size      int64
	getReader ContextReaderGetter

	rl      *rangeLock // for reading
	readers map[*cacheFileReader]struct{}

	wl      *rangeLock // for writing
	writers map[io.WriteCloser]struct{}

	// evicted indicates this cacheFile has been removed from the pool and its
	// backing file should be removed once there are no active readers/writers.
	evicted bool
	// removed indicates the backing file has already been removed.
	removed bool
	// evict removes this cacheFile's entry from the owning pool. Set by the
	// pool when the cacheFile is created.
	evict func()
	// err is the terminal source/cache error. Keeping it on the cache file
	// lets readers waiting for a fill observe the original error instead of the
	// less useful os.ErrClosed from the range lock.
	err error

	mu sync.Mutex
}

func (cf *cacheFile) Reader(ctx context.Context) (*cacheFileReader, error) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	if cf.err != nil {
		return nil, cf.err
	}
	f, e := os.Open(cf.name)
	if e != nil {
		return nil, e
	}

	cfr := &cacheFileReader{
		f:           f,
		size:        cf.size,
		rl:          cf.rl,
		readRequest: cf.readRequest,
		terminalErr: cf.terminalError,
		ctx:         ctx,
	}
	cfr.release = func() { cf.releaseReader(cfr) }

	cf.readers[cfr] = struct{}{}
	return cfr, nil
}

func (cf *cacheFile) Close() error {
	return cf.closeWithError(os.ErrClosed)
}

func (cf *cacheFile) closeWithError(cause error) error {
	cf.mu.Lock()
	if cf.err == nil {
		cf.err = cause
	}
	for w := range cf.writers {
		_ = w.Close()
	}

	// Cancel the range locks so any waiting readers wake up and return.
	// Readers remove themselves from cf.readers via their own Close().
	cf.rl.release()
	cf.wl.release()

	// Mark as evicted so the backing file is removed once idle. Close is only
	// called on error (download/IO failure), so this cacheFile must not be
	// reused for new reads.
	cf.evicted = true
	cf.removeFileIfIdleLocked()
	cf.mu.Unlock()

	// Drop the entry from the pool (outside cf.mu to keep the pool->cacheFile
	// lock ordering and avoid re-entering cf.mu via onCacheEvicted).
	if cf.evict != nil {
		cf.evict()
	}

	return nil
}

func (cf *cacheFile) terminalError() error {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return cf.err
}

// removeFileIfIdleLocked removes the backing file if the cacheFile has been
// evicted and there are no active readers/writers. cf.mu must be held.
func (cf *cacheFile) removeFileIfIdleLocked() {
	if cf.removed || !cf.evicted {
		return
	}
	if len(cf.readers) > 0 || len(cf.writers) > 0 {
		return
	}
	cf.removed = true
	_ = os.Remove(cf.name)
}

func (cf *cacheFile) readRequest(ctx context.Context, start, readLen int64) error {
	if readLen < 0 {
		panic("invalid readLen")
	}
	if readLen == 0 {
		return nil
	}
	if e := cf.terminalError(); e != nil {
		return e
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if cf.rl.satisfy(start, readLen) {
		return nil
	}
	end := start + readLen
	blockSize := int64(10 * 1024 * 1024) // 10M

	var offset, size int64

	if cf.size <= blockSize {
		offset = -1
		size = -1
		if !cf.wl.tryExclusiveFeed(0, cf.size) {
			// the whole file is being downloaded
			return nil
		}
	} else {
		offset = blockSize * (start / blockSize)
		blockEnd := int64(math.Min(
			float64(blockSize*int64(math.Ceil(float64(end)/float64(blockSize)))),
			float64(cf.size),
		))
		if cf.size-blockEnd < blockSize {
			blockEnd = cf.size
		}
		size = blockEnd - offset
		if !cf.wl.tryExclusiveFeed(offset, size) {
			// this part is being downloaded
			return nil
		}
	}

	var reader io.ReadCloser

	rc, e := cf.getReader(ctx, offset, size)
	if e == nil {
		reader = rc
	} else {
		if !err.IsUnsupportedError(e) {
			_ = cf.closeWithError(e)
			return e
		}
		if !cf.wl.tryExclusiveFeed(0, cf.size) {
			// the whole file is being downloaded
			return nil
		}
		rc, e = cf.getReader(ctx, -1, -1)
		if e != nil {
			_ = cf.closeWithError(e)
			return e
		}
		reader = rc
	}
	if offset < 0 {
		offset = 0
	}
	if size <= 0 {
		size = cf.size - offset
	}
	cf.startWriter(reader, offset, size)
	return nil
}

func (cf *cacheFile) startWriter(reader io.ReadCloser, offset, length int64) {
	go func() {
		defer func() { _ = reader.Close() }()
		writer, e := os.OpenFile(cf.name, os.O_WRONLY, 0600)
		if e != nil {
			_ = cf.closeWithError(e)
			return
		}
		defer func() { _ = writer.Close() }()
		if offset > 0 {
			_, e = writer.Seek(offset, io.SeekStart)
			if e != nil {
				logging.For("f-cache").Errorf("cache_file_pool seek error: %v", e)
				_ = cf.closeWithError(e)
				return
			}
		}
		cf.mu.Lock()
		cf.writers[writer] = struct{}{}
		cf.mu.Unlock()
		defer func() {
			cf.mu.Lock()
			delete(cf.writers, writer)
			cf.removeFileIfIdleLocked()
			cf.mu.Unlock()
		}()

		buf := make([]byte, 32*1024)
		pos := offset
		remaining := length
		for {
			readBuf := buf
			if remaining < int64(len(readBuf)) {
				readBuf = readBuf[:int(remaining)]
			}
			nr, er := reader.Read(readBuf)
			if nr > 0 {
				nw, ew := writer.Write(readBuf[:nr])
				if ew != nil {
					logging.For("f-cache").Errorf("cache_file_pool write error: %v", ew)
					_ = cf.closeWithError(ew)
					return
				}
				if nw != nr {
					logging.For("f-cache").Errorf("cache_file_pool short write: wrote %d of %d bytes", nw, nr)
					_ = cf.closeWithError(io.ErrShortWrite)
					return
				}
				cf.rl.feed(pos, int64(nw))
				pos += int64(nw)
				remaining -= int64(nw)
			}
			if remaining == 0 {
				break
			}
			if er != nil {
				if er != io.EOF {
					logging.For("f-cache").Errorf("cache_file_pool read error: %v", er)
					_ = cf.closeWithError(er)
					return
				}
				_ = cf.closeWithError(io.ErrUnexpectedEOF)
				break
			}
		}
	}()
}

func (cf *cacheFile) releaseReader(cfr *cacheFileReader) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	delete(cf.readers, cfr)
	cf.removeFileIfIdleLocked()
}

type cacheFileReader struct {
	f    *os.File
	pos  int64
	size int64
	ctx  context.Context

	rl *rangeLock

	readRequest func(context.Context, int64, int64) error
	terminalErr func() error
	release     func()

	mu   sync.Mutex
	once sync.Once
	err  error
}

func (cfr *cacheFileReader) Read(p []byte) (n int, err error) {
	cfr.mu.Lock()
	defer cfr.mu.Unlock()

	end := cfr.pos + int64(len(p))
	if end > cfr.size {
		end = cfr.size
	}
	readLen := end - cfr.pos

	if e := cfr.readRequest(cfr.ctx, cfr.pos, readLen); e != nil {
		return 0, e
	}
	if e := cfr.rl.acquireContext(cfr.ctx, cfr.pos, readLen); e != nil {
		if terminal := cfr.terminalErr(); terminal != nil {
			return 0, terminal
		}
		return 0, e
	}
	if _, e := cfr.f.Seek(cfr.pos, io.SeekStart); e != nil {
		return 0, e
	}
	n, err = cfr.f.Read(p)
	if err != nil {
		return
	}
	cfr.pos += int64(n)
	return
}

func (cfr *cacheFileReader) ReadAt(p []byte, off int64) (n int, err error) {
	cfr.mu.Lock()
	defer cfr.mu.Unlock()
	if off < 0 || off > cfr.size {
		return 0, os.ErrInvalid
	}
	if len(p) == 0 {
		return 0, nil
	}
	readLen := int64(len(p))
	if remaining := cfr.size - off; readLen > remaining {
		readLen = remaining
	}
	if e := cfr.readRequest(cfr.ctx, off, readLen); e != nil {
		return 0, e
	}
	if e := cfr.rl.acquireContext(cfr.ctx, off, readLen); e != nil {
		if terminal := cfr.terminalErr(); terminal != nil {
			return 0, terminal
		}
		return 0, e
	}
	n, err = cfr.f.ReadAt(p[:int(readLen)], off)
	if n < len(p) && err == nil {
		err = io.EOF
	}
	return n, err
}

func (cfr *cacheFileReader) Seek(offset int64, whence int) (int64, error) {
	cfr.mu.Lock()
	defer cfr.mu.Unlock()
	pos := cfr.pos
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos += offset
	case io.SeekEnd:
		pos = cfr.size + offset
	default:
		pos = -1
	}
	if pos < 0 || pos > cfr.size {
		return 0, os.ErrInvalid
	}
	cfr.pos = pos
	return cfr.pos, nil
}

func (cfr *cacheFileReader) Close() error {
	cfr.once.Do(func() {
		cfr.err = cfr.f.Close()
		cfr.release()
	})
	return cfr.err
}

func newRangeLock(max int64) *rangeLock {
	rl := &rangeLock{
		ranges: make([][]int64, 0),
		max:    max,
		notify: make(chan struct{}),
	}
	return rl
}

type rangeLock struct {
	mu     sync.Mutex
	notify chan struct{}
	ranges [][]int64
	max    int64
	// canceled indicates this lock has been released/closed; waiting acquirers
	// will return os.ErrClosed.
	canceled bool
}

// acquire blocks until the [start, start+length) range is available, or the
// lock is canceled (returns os.ErrClosed).
func (rl *rangeLock) acquire(start, length int64) error {
	return rl.acquireContext(context.Background(), start, length)
}

func (rl *rangeLock) acquireContext(ctx context.Context, start, length int64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		rl.mu.Lock()
		if rl._satisfy(start, length) {
			rl.mu.Unlock()
			return nil
		}
		if rl.canceled {
			rl.mu.Unlock()
			return os.ErrClosed
		}
		notify := rl.notify
		rl.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-notify:
		}
	}
}

func (rl *rangeLock) satisfy(start, length int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl._satisfy(start, length)
}

// tryExclusiveFeed marks [start, start+l) as being handled if it is not already
// satisfied. It returns true if the caller acquired the range (and is therefore
// responsible for filling it), false otherwise.
func (rl *rangeLock) tryExclusiveFeed(start, l int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl._satisfy(start, l) {
		return false
	}
	rl._feed(start, l)
	return true
}

func (rl *rangeLock) feed(start, l int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl._feed(start, l)
}

func (rl *rangeLock) release() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.canceled {
		return
	}
	rl.canceled = true
	rl.broadcastLocked()
}

func (rl *rangeLock) _satisfy(start, length int64) bool {
	if length < 0 {
		panic("invalid len")
	}
	if length == 0 {
		return true
	}
	end := start + length
	for _, ran := range rl.ranges {
		if start >= ran[0] && end <= ran[1] {
			return true
		}
	}
	return false
}

// _feed records a newly available range and wakes any waiting acquirers.
// rl.mu must be held.
func (rl *rangeLock) _feed(start, l int64) {
	rl.ranges = append(rl.ranges, []int64{start, start + l})
	rl._merge()
	rl.broadcastLocked()
}

func (rl *rangeLock) broadcastLocked() {
	close(rl.notify)
	rl.notify = make(chan struct{})
}

// _merge sorts and merges overlapping/adjacent ranges. rl.mu must be held.
func (rl *rangeLock) _merge() {
	if len(rl.ranges) <= 1 {
		return
	}
	sort.Slice(rl.ranges, func(i, j int) bool {
		return rl.ranges[i][0] < rl.ranges[j][0]
	})
	newRanges := make([][]int64, 0, len(rl.ranges))
	last := rl.ranges[0]
	for i := 1; i < len(rl.ranges); i++ {
		ran := rl.ranges[i]
		if ran[0] <= last[1] {
			// overlapping or adjacent: extend the current range, never shrink it
			if ran[1] > last[1] {
				last[1] = ran[1]
			}
		} else {
			newRanges = append(newRanges, last)
			last = ran
		}
	}
	newRanges = append(newRanges, last)
	rl.ranges = newRanges
}
