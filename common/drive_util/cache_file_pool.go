package drive_util

import (
	"crypto/md5"
	"errors"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/utils"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
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
	mu      sync.RWMutex
}

func (cfp *CacheFilePool) GetReader(key string, size int64, getReader ReaderGetter) (io.ReadSeekCloser, error) {
	cfp.mu.RLock()
	cf, ok := cfp.entries.Get(key)
	cfp.mu.RUnlock()
	if ok {
		return cf.(*cacheFile).Reader()
	}
	cfp.mu.Lock()
	defer cfp.mu.Unlock()

	name := filepath.Join(cfp.dir, fmt.Sprintf("%x", md5.Sum([]byte(key))))
	file, e := os.OpenFile(name, os.O_CREATE|os.O_TRUNC, 0600)
	if e != nil {
		return nil, e
	}
	_ = file.Close()

	cf = &cacheFile{
		name:      name,
		size:      size,
		getReader: getReader,

		rl:      newRangeLock(size),
		readers: map[*cacheFileReader]struct{}{},

		wl:      newRangeLock(size),
		writers: make(map[io.WriteCloser]struct{}),
	}
	cfp.entries.Add(key, cf)

	return cf.(*cacheFile).Reader()
}

func (cfp *CacheFilePool) onCacheEvicted(key_ lru.Key, value interface{}) {
	cf := value.(*cacheFile)

	cf.mu.Lock()
	defer cf.mu.Unlock()

	if len(cf.readers) > 0 || len(cf.writers) > 0 {
		return
	}

	_ = os.Remove(cf.name)
}

type cacheFile struct {
	name      string
	size      int64
	getReader ReaderGetter

	rl      *rangeLock // for reading
	readers map[*cacheFileReader]struct{}

	wl      *rangeLock // for writing
	writers map[io.WriteCloser]struct{}

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
	defer cf.mu.Unlock()

	for w := range cf.writers {
		_ = w.Close()
	}

	readers := utils.MapKeys(cf.readers)
	for _, r := range readers {
		cf.releaseReader(r)
	}

	cf.rl.release(false, true)
	cf.wl.release(false, true)

	return nil
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
			cf.mu.Unlock()
		}()

		buf := make([]byte, 32*1024)
		pos := offset
		for {
			nr, er := reader.Read(buf)
			if nr > 0 {
				nw, ew := writer.Write(buf[0:nr])
				if ew != nil {
					log.Printf("cache_file_pool write error: %v", er)
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
		pos += cfr.size + offset
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
	return &rangeLock{
		mu:     sync.RWMutex{},
		notify: make(chan bool),
		ranges: make([][]int64, 0),
		max:    max,
	}
}

type rangeLock struct {
	mu     sync.RWMutex
	notify chan bool
	ranges [][]int64
	max    int64
	done   bool

	waiting   int32
	waitingMu sync.Mutex
}

func (rl *rangeLock) acquire(start, len int64) error {
	for {
		if rl.satisfy(start, len) {
			return nil
		}
		rl.waitingMu.Lock()
		rl.waiting += 1
		rl.waitingMu.Unlock()
		ok, readOk := <-rl.notify
		if readOk {
			if !ok {
				return os.ErrClosed
			}
		}
	}
}

func (rl *rangeLock) satisfy(start, len int64) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl._satisfy(start, len)
}

func (rl *rangeLock) tryExclusiveFeed(start, l int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl._satisfy(start, l) {
		return false
	}
	rl._feed(start, l, false)
	return true
}

func (rl *rangeLock) feed(start, l int64) {
	rl._feed(start, l, true)
}

func (rl *rangeLock) release(v, lock bool) {
	if lock {
		rl.mu.Lock()
		defer rl.mu.Unlock()
	}
	if rl.done {
		return
	}
	rl.waitingMu.Lock()
	defer rl.waitingMu.Unlock()
	for rl.waiting > 0 {
		select {
		case rl.notify <- v:
		default:
		}
		rl.waiting--
	}
}

func (rl *rangeLock) _satisfy(start, len int64) bool {
	if len < 0 {
		panic("invalid len")
	}
	if len == 0 {
		return true
	}
	end := start + len
	for _, ran := range rl.ranges {
		if start >= ran[0] && end <= ran[1] {
			return true
		}
	}
	return false
}

func (rl *rangeLock) _feed(start, l int64, lock bool) {
	if lock {
		rl.mu.Lock()
	}
	rl.ranges = append(rl.ranges, []int64{start, start + l})
	rl._merge()
	if len(rl.ranges) == 1 && rl.ranges[0][0] == 0 && rl.ranges[0][1] == rl.max {
		close(rl.notify)
		rl.done = true
	}
	if lock {
		rl.mu.Unlock()
	}

	rl.release(true, lock)
}

func (rl *rangeLock) _merge() {
	if len(rl.ranges) == 0 {
		return
	}
	sort.Slice(rl.ranges, func(i, j int) bool {
		return rl.ranges[i][0] < rl.ranges[j][0]
	})
	newRanges := make([][]int64, 0, len(rl.ranges))
	lastRange := rl.ranges[0]
	for i := 1; i < len(rl.ranges); i++ {
		ran := rl.ranges[i]
		if lastRange[1] >= ran[0] {
			lastRange[1] = ran[1]
			newRanges = append(newRanges, lastRange)
		} else {
			newRanges = append(newRanges, lastRange)
			newRanges = append(newRanges, ran)
			lastRange = ran
		}
	}
	if len(newRanges) == 0 {
		newRanges = append(newRanges, lastRange)
	}
	rl.ranges = newRanges
}
