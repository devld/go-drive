package thumbnail

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/artifact"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
)

const FolderType = "/"

const handlerName = "thumbnail"

type Maker struct {
	// handlers maps a lowercase file extension (or FolderType) to one handler.
	// A later config item for the same extension replaces the earlier one.
	handlers map[string]TypeHandler

	apiPath string

	validity    time.Duration
	concurrency int
}

var (
	_ artifact.Handler = (*Maker)(nil)
)

func init() {
	artifact.RegisterHandler(handlerName, NewMaker)
}

// NewMaker creates a thumbnail artifact processor. Persistence and task
// scheduling stay in the shared artifact service.
func NewMaker(ctx artifact.HandlerContext) (artifact.Handler, error) {
	handlers, e := createHandlers(ctx.Config.Thumbnail.Handlers)
	if e != nil {
		return nil, e
	}

	m := &Maker{
		handlers:    handlers,
		apiPath:     ctx.Config.APIPath,
		validity:    ctx.Config.Thumbnail.TTL,
		concurrency: ctx.Config.Thumbnail.Concurrent,
	}
	return m, nil
}

func createHandlers(items []common.ThumbnailHandlerItem) (map[string]TypeHandler, error) {
	hs := make(map[string]TypeHandler)
	for _, item := range items {
		factory, ok := typeHandlerFactories[item.Type]
		if !ok {
			availableTypes := make([]string, 0, len(typeHandlerFactories))
			for t := range typeHandlerFactories {
				availableTypes = append(availableTypes, t)
			}

			return nil, fmt.Errorf("unknown handler type: %s. Available types are %v",
				item.Type, availableTypes)
		}
		h, e := factory(item.Config)
		if e != nil {
			return nil, errors.New("failed to create thumbnail type handler " + item.Type + ": " + e.Error())
		}
		for ext := range strings.SplitSeq(item.FileTypes, ",") {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if ext == "" {
				continue
			}
			hs[ext] = h
		}
	}
	return hs, nil
}

func (m *Maker) Spec() artifact.Spec {
	return artifact.Spec{
		Caches:      []artifact.CacheSpec{{Policy: artifact.Policy{TTL: m.validity}}},
		Concurrency: m.concurrency,
		Config:      types.M{"extensions": m.supportedExtensions()},
	}
}

func (m *Maker) supportedExtensions() string {
	extensions := make([]string, 0, len(m.handlers))
	for ext := range m.handlers {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return strings.Join(extensions, ",")
}

func (m *Maker) Resolve(request artifact.Request) (artifact.ResolvedRequest, error) {
	if request.Args != "" {
		return artifact.ResolvedRequest{}, apierr.NewNotFoundMessageError("invalid thumbnail artifact request")
	}
	return artifact.ResolvedRequest{Fingerprint: "v1"}, nil
}

func (m *Maker) Produce(ctx types.TaskCtx, request artifact.Request, out artifact.Writer) error {
	entry := m.createThumbnailEntry(request.Source)
	handler, err := m.resolveHandler(entry)
	if err != nil {
		return thumbnailProduceError(err)
	}
	if err := out.WriteMeta(artifact.Meta{
		Name:     thumbnailArtifactName(entry.Name(), handler.MimeType()),
		MimeType: handler.MimeType(),
		ModTime:  time.UnixMilli(entry.ModTime()),
	}); err != nil {
		return thumbnailProduceError(err)
	}
	return thumbnailProduceError(m.createThumbnail(ctx, entry, handler, out))
}

func (m *Maker) createThumbnail(ctx types.TaskCtx, entry ThumbnailEntry, handler TypeHandler, dest io.Writer) error {
	logging.For("thumbn").Debugf("thumbnail artifact handler started path=%s mime=%s",
		logging.Sanitize(entry.Path()), handler.MimeType())
	handlerCtx := context.Context(ctx)
	if handler.Timeout() > 0 {
		var cancel context.CancelFunc
		handlerCtx, cancel = context.WithTimeout(handlerCtx, handler.Timeout())
		defer cancel()
	}
	return handler.CreateThumbnail(task.WithContext(handlerCtx, ctx), entry, dest)
}

func thumbnailProduceError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if !apierr.IsNotFoundError(err) {
		if public, ok := errors.AsType[apierr.Error](err); ok {
			err = apierr.NewNotFoundMessageError(public.Error())
		} else {
			err = errors.New("failed to generate thumbnail")
		}
	}
	return artifact.Cacheable(err)
}

func (m *Maker) createThumbnailEntry(entry types.IEntry) ThumbnailEntry {
	// IDispatcherEntry supplies GetRealPath for shell handlers and the
	// download URL used by remote thumbnail generators.
	dispatcherEntry, ok := driveutil.IEntryAs[types.IDispatcherEntry](entry)
	if !ok {
		panic("types.IDispatcherEntry required")
	}

	externalURL := m.apiPath

	if entry.Type().IsDir() {
		externalURL += "/list"
	} else {
		externalURL += "/download"
	}
	query := url.Values{"path": []string{entry.Path()}}

	ako, ok := entry.Meta().Props["accessKey"]
	ak := ""
	if ok {
		if t, ok := ako.(string); ok {
			ak = t
		}
	}
	if ak != "" {
		query.Set(common.SignatureQueryKey, ak)
	}
	externalURL += "?" + query.Encode()

	return &thumbnailEntry{IEntry: entry, IDispatcherEntry: dispatcherEntry, externalURL: externalURL}
}

func (m *Maker) resolveHandler(entry ThumbnailEntry) (TypeHandler, error) {
	if entry.Meta().HasThumbnail {
		if GetWrappedThumbnailEntry(entry) == nil {
			return nil, apierr.NewNotFoundMessageError("thumbnail handler not found")
		}
		return entryThumbnailTypeHandler, nil
	}

	fType := FolderType
	if !entry.Type().IsDir() {
		fType = utils.PathExt(entry.Path())
	}
	handler, ok := m.handlers[fType]
	if !ok {
		logging.For("thumbn").Debugf("thumbnail handler not found path=%s type=%s",
			logging.Sanitize(entry.Path()), fType)
		return nil, apierr.NewNotFoundMessageError("thumbnail handler not found")
	}
	return handler, nil
}

func thumbnailArtifactName(sourceName, mimeType string) string {
	base := utils.PathName(sourceName)
	if base == "" {
		base = "thumbnail"
	}
	ext := driveutil.ExtensionByMimeType(mimeType)
	if ext == "" {
		ext = ".thumb"
	}
	return base + ext
}
