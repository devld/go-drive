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
	rf, e := os.Open(wf.Name())
	if e != nil {
		_ = wf.Close()
		_ = os.Remove(wf.Name())
		return nil, e
	}
	return &CacheFile{
		size: size,
		l:    newRangeLock(),
		wf:   wf,
		rf:   rf,
		pos:  0,
		mu:   sync.Mutex{},
	}, nil
}

type CacheFile struct {
	size int64
	wf   *os.File
	rf   *os.File
	l    *rangeLock
	pos  int64
	mu   sync.Mutex

	wPos int64
	wmu  sync.Mutex
}

func (c *CacheFile) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	end := c.pos + int64(len(p))
	if end > c.size {
		end = c.size
	}
	if e := c.l.acquire(c.pos, end-c.pos); e != nil {
		return 0, e
	}
	if _, e := c.rf.Seek(c.pos, io.SeekStart); e != nil {
		return 0, e
	}
	n, err = c.rf.Read(p)
	if err != nil {
		return
	}
	c.pos += int64(n)
	return
}

func (c *CacheFile) Write(p []byte) (n int, err error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	n, err = c.wf.Write(p)
	if err != nil {
		return
	}
	c.l.feed(c.wPos, int64(n))
	c.wPos += int64(n)
	return
}

func (c *CacheFile) Seek(offset int64, whence int) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	pos := c.pos
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos += offset
	case io.SeekEnd:
		pos += c.size + offset
	default:
		pos = -1
	}
	if pos < 0 || pos > c.size {
		return 0, os.ErrInvalid
	}
	c.pos = pos
	return c.pos, nil
}

func (c *CacheFile) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.l.release(false)
	_ = c.wf.Close()
	_ = c.rf.Close()
	_ = os.Remove(c.wf.Name())
	return nil
}

func newRangeLock() *rangeLock {
	return &rangeLock{
		mu:     sync.RWMutex{},
		notify: make(chan bool),
		ranges: make([][]int64, 0),
	}
}

type rangeLock struct {
	mu     sync.RWMutex
	notify chan bool
	ranges [][]int64
}

func (rl *rangeLock) acquire(start, len int64) error {
	for {
		if rl.satisfy(start, len) {
			return nil
		}
		if !(<-rl.notify) {
			return os.ErrClosed
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

func (rl *rangeLock) feed(start, len int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.ranges = append(rl.ranges, []int64{start, start + len})
	rl._merge()
	rl.release(true)
}

func (rl *rangeLock) release(v bool) {
	select {
	case rl.notify <- v:
	default:
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
