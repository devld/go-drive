package thumbnail

import (
	"context"
	"go-drive/common/types"
	"io"
	"time"
)

var typeHandlerFactories = make(map[string]TypeHandlerFactory, 0)

func RegisterTypeHandler(name string, thf TypeHandlerFactory) {
	if _, ok := typeHandlerFactories[name]; ok {
		panic("TypeHandlerFactory " + name + " already registered")
	}
	typeHandlerFactories[name] = thf
}

type TypeHandlerFactory = func(config types.SM) (TypeHandler, error)

type TypeHandler interface {
	// CreateThumbnail creates thumbnail for entry, and writes to dest
	CreateThumbnail(ctx context.Context, entry types.IEntry, dest io.Writer) error
	// MimeType returns the mime-type of this TypeHandler can generate
	MimeType() string
	// Timeout returns the timeout when generating thumbnail,
	// it won't time out when negative value returned
	Timeout() time.Duration
}

type Thumbnail interface {
	io.ReadSeeker
	io.Closer
	ModTime() time.Time
	Size() int64
	MimeType() string
}
