package drive_util

import (
	"errors"
	err "go-drive/common/errors"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/golang/groupcache/lru"
)

type ReaderGetter = func(int64, int64) (io.ReadCloser, error)

func NewCacheFillPool(maxCache int, dir string) (*CacheFilePool, error) {
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
	pool := &CacheFilePool{dir: dir, entries: lru.New(maxCache)}
	pool.entries.OnEvicted = pool.onCacheEvicted

	return pool, nil
}

type CacheFilePool struct {
	dir     string
	entries *lru.Cache
	mu      sync.Mutex
}

func (cfp *CacheFilePool) GetReader(key string, size int64, getReader ReaderGetter) (io.ReadSeekCloser, error) {
	// lru.Cache.Get mutates the internal list (MoveToFront), so it must be
	// called under the write lock, not a read lock.
	cfp.mu.Lock()
	cf, ok := cfp.entries.Get(key)
	cfp.mu.Unlock()
	if ok {
		return cf.(*cacheFile).Reader()
	}
	cfp.mu.Lock()
	defer cfp.mu.Unlock()

	// re-check after acquiring the write lock to avoid creating duplicated cacheFile
	if cf, ok = cfp.entries.Get(key); ok {
		return cf.(*cacheFile).Reader()
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
	cfp.entries.Add(key, cf)

	return cf.(*cacheFile).Reader()
}

func (cfp *CacheFilePool) onCacheEvicted(_ lru.Key, value any) {
	cf := value.(*cacheFile)

	cf.mu.Lock()
	defer cf.mu.Unlock()

	cf.evicted = true
	cf.removeFileIfIdleLocked()
}

type cacheFile struct {
	name      string
	size      int64
	getReader ReaderGetter

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

	mu sync.Mutex
}

func (cf *cacheFile) Reader() (*cacheFileReader, error) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	f, e := os.Open(cf.name)
	if e != nil {
		return nil, e
	}

	cfr := &cacheFileReader{
		f:           f,
		size:        cf.size,
		rl:          cf.rl,
		readRequest: cf.readRequest,
	}
	cfr.release = func() { cf.releaseReader(cfr) }

	cf.readers[cfr] = struct{}{}
	return cfr, nil
}

func (cf *cacheFile) Close() error {
	cf.mu.Lock()
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

func (cf *cacheFile) readRequest(start, readLen int64) error {
	if readLen < 0 {
		panic("invalid readLen")
	}
	if readLen == 0 {
		return nil
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

	rc, e := cf.getReader(offset, size)
	if e == nil {
		reader = rc
	} else {
		if !err.IsUnsupportedError(e) {
			_ = cf.Close()
			return e
		}
		if !cf.wl.tryExclusiveFeed(0, cf.size) {
			// the whole file is being downloaded
			return nil
		}
		rc, e = cf.getReader(-1, -1)
		if e != nil {
			_ = cf.Close()
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

func (cf *cacheFile) startWriter(reader io.ReadCloser, offset, _ int64) {
	go func() {
		defer func() { _ = reader.Close() }()
		writer, e := os.OpenFile(cf.name, os.O_WRONLY, 0600)
		if e != nil {
			_ = cf.Close()
			return
		}
		defer func() { _ = writer.Close() }()
		if offset > 0 {
			_, e = writer.Seek(offset, io.SeekStart)
			if e != nil {
				log.Printf("cache_file_pool seek error: %v", e)
				_ = cf.Close()
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
		for {
			nr, er := reader.Read(buf)
			if nr > 0 {
				nw, ew := writer.Write(buf[0:nr])
				if ew != nil {
					log.Printf("cache_file_pool write error: %v", ew)
					_ = cf.Close()
					return
				}
				cf.rl.feed(pos, int64(nw))
				pos += int64(nw)
			}
			if er != nil {
				if er != io.EOF {
					log.Printf("cache_file_pool read error: %v", er)
					_ = cf.Close()
					return
				}
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

	rl *rangeLock

	readRequest func(int64, int64) error
	release     func()

	mu sync.Mutex
}

func (cfr *cacheFileReader) Read(p []byte) (n int, err error) {
	cfr.mu.Lock()
	defer cfr.mu.Unlock()

	end := cfr.pos + int64(len(p))
	if end > cfr.size {
		end = cfr.size
	}
	readLen := end - cfr.pos

	if e := cfr.readRequest(cfr.pos, readLen); e != nil {
		return 0, e
	}
	if e := cfr.rl.acquire(cfr.pos, readLen); e != nil {
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
	e := cfr.f.Close()
	cfr.release()
	return e
}

func newRangeLock(max int64) *rangeLock {
	rl := &rangeLock{
		ranges: make([][]int64, 0),
		max:    max,
	}
	rl.cond = sync.NewCond(&rl.mu)
	return rl
}

type rangeLock struct {
	mu     sync.Mutex
	cond   *sync.Cond
	ranges [][]int64
	max    int64
	// canceled indicates this lock has been released/closed; waiting acquirers
	// will return os.ErrClosed.
	canceled bool
}

// acquire blocks until the [start, start+length) range is available, or the
// lock is canceled (returns os.ErrClosed).
func (rl *rangeLock) acquire(start, length int64) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for {
		if rl._satisfy(start, length) {
			return nil
		}
		if rl.canceled {
			return os.ErrClosed
		}
		rl.cond.Wait()
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
	rl.cond.Broadcast()
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
	rl.cond.Broadcast()
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
