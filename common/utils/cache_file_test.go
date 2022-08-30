package utils

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestCacheFile(t *testing.T) {
	cf, e := NewCacheFile(1024, "", "cache-file-test")
	if e != nil {
		t.Error(e)
		return
	}
	defer func() {
		e := cf.Close()
		if e != nil {
			t.Error("failed to close file", e)
		}
	}()

	content := "hello world"

	wg := sync.WaitGroup{}

	testRead := func() {
		defer wg.Done()

		reader, e := cf.GetReader()
		if e != nil {
			t.Error(e)
			return
		}
		defer func() {
			_ = reader.Close()
		}()

		n, e := reader.Seek(4, io.SeekStart)
		if e != nil {
			t.Error(e)
			return
		}
		t.Logf("seeked to: %d\n", n)

		buf := make([]byte, 11)
		start := time.Now().UnixNano()
		read, e := reader.Read(buf)
		if e != nil {
			t.Error(e)
			return
		}
		t.Logf("read: %dms", (time.Now().UnixNano()-start)/int64(time.Millisecond))
		str := string(buf[0:read])
		t.Logf("%d bytes read: %s\n", read, str)

		if str != content {
			t.Error("unexpected content")
			return
		}
	}

	readersCount := 20
	wg.Add(readersCount)
	for i := 0; i < readersCount; i++ {
		go testRead()
	}

	_, e = cf.Write([]byte("1234"))
	if e != nil {
		t.Error(e)
		return
	}
	_, e = cf.Write([]byte(content))
	if e != nil {
		t.Error(e)
	}

	wg.Wait()

	if r := len(cf.readers); r != 0 {
		t.Errorf("%d opened files not released", r)
	}

}
