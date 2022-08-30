package utils

import (
	"io"
	"os"
	"sort"
	"sync"
)

// NewCacheFile creates a temporary file that can be linearly writing and random reading simultaneously
func NewCacheFile(size int64, tempDir, pattern string) (*CacheFile, error) {
	wf, e := os.CreateTemp(tempDir, pattern)
	if e != nil {
		return nil, e
	}
	e = wf.Truncate(size)
	if e != nil {
		_ = wf.Close()
		_ = os.Remove(wf.Name())
		return nil, e
	}
	return &CacheFile{
		size:    size,
		l:       newRangeLock(size),
		wf:      wf,
		readers: make(map[*cacheFileReader]struct{}, 1),
	}, nil
}

type CacheFile struct {
	size int64
	wf   *os.File
	l    *rangeLock

	wPos int64
	mu   sync.Mutex

	readers map[*cacheFileReader]struct{}
	rmu     sync.Mutex
}

func (c *CacheFile) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, err = c.wf.Write(p)
	if err != nil {
		return
	}
	c.l.feed(c.wPos, int64(n))
	c.wPos += int64(n)
	return
}

func (c *CacheFile) GetReader() (io.ReadSeekCloser, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	rf, e := os.Open(c.wf.Name())
	if e != nil {
		return nil, e
	}
	cfr := &cacheFileReader{
		cf: c,
		rf: rf,
	}
	c.readers[cfr] = struct{}{}
	cfr.release = func() {
		c.rmu.Lock()
		defer c.rmu.Unlock()
		delete(c.readers, cfr)
	}
	return cfr, nil
}

func (c *CacheFile) Close() error {
	c.rmu.Lock()
	readers := make([]*cacheFileReader, 0, len(c.readers))
	for r := range c.readers {
		readers = append(readers, r)
	}
	c.rmu.Unlock()

	c.l.release(false)
	_ = c.wf.Close()

	for _, r := range readers {
		_ = r.Close()
	}

	return os.Remove(c.wf.Name())
}

type cacheFileReader struct {
	cf      *CacheFile
	release func()

	rf  *os.File
	pos int64
	mu  sync.Mutex
}

func (cfr *cacheFileReader) Read(p []byte) (n int, err error) {
	cfr.mu.Lock()
	defer cfr.mu.Unlock()
	end := cfr.pos + int64(len(p))
	if end > cfr.cf.size {
		end = cfr.cf.size
	}
	if e := cfr.cf.l.acquire(cfr.pos, end-cfr.pos); e != nil {
		return 0, e
	}
	if _, e := cfr.rf.Seek(cfr.pos, io.SeekStart); e != nil {
		return 0, e
	}
	n, err = cfr.rf.Read(p)
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
		pos += cfr.cf.size + offset
	default:
		pos = -1
	}
	if pos < 0 || pos > cfr.cf.size {
		return 0, os.ErrInvalid
	}
	cfr.pos = pos
	return cfr.pos, nil
}

func (cfr *cacheFileReader) Close() error {
	cfr.release()
	return cfr.rf.Close()
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

func (rl *rangeLock) feed(start, l int64) {
	rl.mu.Lock()
	rl.ranges = append(rl.ranges, []int64{start, start + l})
	rl._merge()
	if len(rl.ranges) == 1 && rl.ranges[0][0] == 0 && rl.ranges[0][1] == rl.max {
		close(rl.notify)
		rl.done = true
	}
	rl.mu.Unlock()

	rl.release(true)
}

func (rl *rangeLock) release(v bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
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
