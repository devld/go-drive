package common

import (
	"io"
	"os"
)

func FileExists(path string) (bool, error) {
	_, e := os.Stat(path)
	if os.IsNotExist(e) {
		return false, nil
	}
	return e == nil, e
}

func IsDir(path string) (bool, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return false, e
	}
	return stat.IsDir(), nil
}

func IsNotSupportedError(e error) bool {
	_, ok := e.(NotSupportedError)
	return ok
}

func NewNotFoundError(msg string) NotFoundError {
	return NotFoundError{msg}
}

func NewNotAllowedError(msg string) NotAllowedError {
	return NotAllowedError{msg}
}

var notSupportedError = NotSupportedError{}

func NewNotSupportedError() NotSupportedError {
	return notSupportedError
}

func CopyWithProgress(dst io.Writer, src io.Reader, progress OnProgress) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		w, err := io.CopyBuffer(dst, src, buf)
		if err != nil {
			break
		}
		if w == 0 {
			break
		}
		written += w
		progress(written)
	}
	return
}
