package utils

import "io"

func NewLazyReader(get func() (io.ReadCloser, error)) io.ReadCloser {
	return &lazyReader{get: get}
}

type lazyReader struct {
	r   io.ReadCloser
	get func() (io.ReadCloser, error)
}

func (l *lazyReader) Read(p []byte) (n int, err error) {
	if l.r == nil {
		reader, e := l.get()
		l.get = nil
		if e != nil {
			return 0, e
		}
		l.r = reader
	}
	return l.r.Read(p)
}

func (l *lazyReader) Close() error {
	if l.r != nil {
		return l.r.Close()
	}
	return nil
}
