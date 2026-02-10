package req

import (
	"io"
	"net/http"
)

type Response interface {
	Response() *http.Response
	Status() int
	Json(v any) error
	XML(v any) error
	Dispose() error
}

type RequestBody interface {
	ContentLength() int64
	ContentType() string
	Reader() io.Reader
}
