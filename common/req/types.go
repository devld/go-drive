package req

import (
	"io"
	"net/http"
)

type Response interface {
	Response() *http.Response
	Status() int
	Json(v interface{}) error
	XML(v interface{}) error
	Dispose() error
}

type RequestBody interface {
	ContentLength() int64
	ContentType() string
	Reader() io.Reader
}
