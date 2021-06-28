package utils

import (
	"io"
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
		_ = cf.Close()
	}()

	content := "hello world"

	wait := make(chan struct{})

	go func() {
		defer func() {
			close(wait)
		}()

		n, e := cf.Seek(4, io.SeekStart)
		if e != nil {
			t.Error(e)
			return
		}
		t.Logf("seeked to: %d\n", n)

		buf := make([]byte, 11)
		start := time.Now().UnixNano()
		read, e := cf.Read(buf)
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
	}()

	_, e = cf.Write([]byte("1234"))
	if e != nil {
		t.Error(e)
		return
	}
	_, e = cf.Write([]byte(content))
	if e != nil {
		t.Error(e)
	}

	<-wait
}
