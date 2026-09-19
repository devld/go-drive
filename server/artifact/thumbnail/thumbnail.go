package thumbnail

import (
	"context"
	"errors"
	"go-drive/common/driveutil"
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
	// CreateThumbnail writes a thumbnail for entry.
	CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error
	// MimeType returns the mime-type of this TypeHandler can generate
	MimeType() string
	// Timeout returns the timeout when generating thumbnail,
	// it won't time out when negative value returned
	Timeout() time.Duration
}

// IEntryThumbnail is the extension of IEntry.
// Entries implement this interface to produce a thumbnail when Meta.HasThumbnail
// is true. Wrapper IEntry types must NOT implement this interface.
type IEntryThumbnail interface {
	// Thumbnail returns common/errors.NewUnsupportedError if this entry is not supported.
	Thumbnail(context.Context) (types.IContentReader, error)
}

func GetWrappedThumbnailEntry(entry types.IEntry) IEntryThumbnail {
	t, ok := driveutil.IEntryAs[IEntryThumbnail](entry)
	if !ok {
		return nil
	}
	e, ok := t.(types.IEntry)
	if !ok || !e.Meta().HasThumbnail {
		return nil
	}
	return t
}

var entryThumbnailTypeHandler = &entryThumbnailHandler{}

type entryThumbnailHandler struct{}

func (est *entryThumbnailHandler) CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error {
	te := GetWrappedThumbnailEntry(entry)
	if te == nil {
		return errors.New("cannot generate thumbnail")
	}
	tr, e := te.Thumbnail(ctx)
	if e != nil {
		return e
	}
	return driveutil.CopyIContent(task.NewContextWrapper(ctx), tr, dest)
}

func (est *entryThumbnailHandler) MimeType() string {
	return ""
}

func (est *entryThumbnailHandler) Timeout() time.Duration {
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
