package thumbnail

import (
	"context"
	"errors"
	"go-drive/common/drive_util"
	"go-drive/common/task"
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

// IEntryThumbnail is the extension of IEntry.
// Entries implement this interface to supports generating thumbnail by themself.
// The wrapper IEntry must NOT implementing this interface.
type IEntryThumbnail interface {
	Thumbnail(context.Context) (types.IContentReader, error)
}

func NewThumbnailURLReader(url string, headers types.SM) types.IContentReader {
	return &thumbnailContentReaderImpl{url: url, headers: headers}
}

type thumbnailContentReaderImpl struct {
	url     string
	headers types.SM
}

func (t *thumbnailContentReaderImpl) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return drive_util.GetURL(ctx, t.url, t.headers)
}

func (t *thumbnailContentReaderImpl) GetURL(context.Context) (*types.ContentURL, error) {
	return &types.ContentURL{
		URL:    t.url,
		Header: t.headers,
		Proxy:  true,
	}, nil
}

func GetWrappedThumbnailEntry(entry types.IEntry) IEntryThumbnail {
	e := drive_util.GetIEntry(entry, func(e types.IEntry) bool {
		_, ok := e.(IEntryThumbnail)
		return ok
	})
	if e == nil {
		return nil
	}
	return e.(IEntryThumbnail)
}

var entrySelfThumbnailTypeHandler = &entrySelfThumbnailHandler{}

type entrySelfThumbnailHandler struct {
}

func (est *entrySelfThumbnailHandler) CreateThumbnail(ctx context.Context, entry types.IEntry, dest io.Writer) error {
	te := GetWrappedThumbnailEntry(entry)
	if te == nil {
		return errors.New("entry is not ThumbnailEntry")
	}
	tr, e := te.Thumbnail(ctx)
	if e != nil {
		return e
	}
	return drive_util.CopyIContent(task.NewContextWrapper(ctx), tr, dest)
}

func (est *entrySelfThumbnailHandler) MimeType() string {
	return ""
}

func (est *entrySelfThumbnailHandler) Timeout() time.Duration {
	return 30 * time.Second
}
