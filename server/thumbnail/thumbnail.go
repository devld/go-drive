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
	// CreateThumbnail creates thumbnail for entry, and writes to dest.
	// Returns err.UnsupportedError if this TypeHandler is unable to create thumbnail for this entry
	CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error
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
	HasThumbnail() bool
	// Thumbnail returns err.UnsupportedError if this entry is not supported
	Thumbnail(context.Context) (types.IContentReader, error)
}

func GetWrappedThumbnailEntry(entry types.IEntry) IEntryThumbnail {
	e := drive_util.GetIEntry(entry, func(e types.IEntry) bool {
		_, ok := e.(IEntryThumbnail)
		return ok
	})
	if e == nil {
		return nil
	}
	th := e.(IEntryThumbnail)
	if !th.HasThumbnail() {
		return nil
	}
	return th
}

var entrySelfThumbnailTypeHandler = &entrySelfThumbnailHandler{}

type entrySelfThumbnailHandler struct {
}

func (est *entrySelfThumbnailHandler) CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error {
	te := GetWrappedThumbnailEntry(entry)
	if te == nil {
		return errors.New("cannot generate thumbnail")
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

type ThumbnailEntry interface {
	types.IEntry
	types.IDispatcherEntry
	GetExternalURL() string
}

var _ types.IEntryWrapper = (*thumbnailEntry)(nil)

type thumbnailEntry struct {
	types.IEntry
	types.IDispatcherEntry
	externalURL string
}

func (te *thumbnailEntry) GetExternalURL() string {
	return te.externalURL
}

func (te *thumbnailEntry) GetIEntry() types.IEntry {
	return te.IEntry
}
