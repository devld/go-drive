package common

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	"strconv"
	"time"
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
	_, ok := e.(UnsupportedError)
	return ok
}

func IsNotFoundError(e error) bool {
	_, ok := e.(NotFoundError)
	return ok
}

func NewNotFoundError(msg string) NotFoundError {
	return NotFoundError{msg}
}

func NewNotAllowedError() NotAllowedError {
	return NotAllowedError{"operation not allowed"}
}

func NewNotAllowedMessageError(msg string) NotAllowedError {
	return NotAllowedError{msg}
}

var unsupportedError = UnsupportedError{}

func NewUnsupportedError() UnsupportedError {
	return unsupportedError
}

func NewRemoteApiError(code int, msg string) RemoteApiError {
	return RemoteApiError{code, msg}
}

func PanicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

func RequireNotNil(v interface{}, msg string) {
	if v == nil {
		panic(msg)
	}
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

func CopyIContent(content IContent, w io.Writer, progress OnProgress) (int64, error) {
	// copy file from url
	url, _, e := content.GetURL()
	if e == nil {
		resp, e := http.Get(url)
		if e != nil {
			return -1, e
		}
		if resp.StatusCode != 200 {
			return -1, NewRemoteApiError(resp.StatusCode, "failed to copy file")
		}
		defer func() { _ = resp.Body.Close() }()
		return CopyWithProgress(w, resp.Body, progress)
	}
	// copy file from reader
	reader, e := content.GetReader()
	if e != nil {
		return -1, e
	}
	defer func() { _ = reader.Close() }()
	return CopyWithProgress(w, reader, progress)
}

func CopyIContentToTempFile(content IContent, progress OnProgress) (*os.File, error) {
	file, e := ioutil.TempFile("", "drive-copy")
	if e != nil {
		return nil, e
	}
	_, e = CopyIContent(content, file, progress)
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, e
	}
	_, e = file.Seek(0, 0)
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, e
	}
	return file, nil
}

func DownloadIContent(content IContent, w http.ResponseWriter, req *http.Request) error {
	url, proxy, e := content.GetURL()
	if e == nil {
		if proxy {
			dest, e := url2.Parse(url)
			if e != nil {
				return e
			}
			proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
				r.URL = dest
				r.Header.Set("Host", dest.Host)
				r.Header.Del("Referer")
			}}

			proxy.ServeHTTP(w, req)
			return nil
		} else {
			w.WriteHeader(302)
			w.Header().Set("Location", url)
		}
		return e
	}
	reader, e := content.GetReader()
	if e != nil {
		return e
	}
	defer func() { _ = reader.Close() }()
	readSeeker, ok := reader.(io.ReadSeeker)
	if ok {
		http.ServeContent(
			w, req, content.Name(),
			time.Unix(0, content.UpdatedAt()*int64(time.Millisecond)),
			readSeeker)
		return nil
	}

	w.Header().Set("Content-Length", strconv.FormatInt(content.Size(), 10))
	_, e = io.Copy(w, reader)
	return e
}
