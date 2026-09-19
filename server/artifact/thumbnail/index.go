package thumbnail

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/artifact"
	"io"
	"mime"
	"net/url"
	"sort"
	"strings"
	"time"
)

const FolderType = "/"

const ThumbnailArtifactType artifact.ArtifactType = "thumbnail"

type Maker struct {
	// handlers maps a lowercase file extension (or FolderType) to one handler.
	// A later config item for the same extension replaces the earlier one.
	handlers map[string]TypeHandler

	artifactStore *artifact.Store
	apiPath       string

	validity    time.Duration
	concurrency int
	stopCleaner func()
}

var (
	_ artifact.Processor = (*Maker)(nil)
	_ artifact.Handler   = (*Maker)(nil)
)

func init() {
	artifact.RegisterHandler("thumbnail", func(ctx artifact.HandlerContext) (artifact.Handler, error) {
		return NewMaker(ctx.Config, ctx.Store, ctx.Components)
	})
}

// NewMaker creates a thumbnail artifact processor. Persistence and task
// scheduling stay in the shared artifact service.
func NewMaker(config common.Config, artifactStore *artifact.Store, ch *registry.ComponentsHolder) (*Maker, error) {
	if artifactStore == nil {
		return nil, errors.New("thumbnail artifact store is required")
	}
	handlers, e := createHandlers(config.Thumbnail.Handlers)
	if e != nil {
		return nil, e
	}

	m := &Maker{
		handlers:      handlers,
		artifactStore: artifactStore,
		apiPath:       config.APIPath,
		validity:      config.Thumbnail.TTL,
		concurrency:   config.Thumbnail.Concurrent,
	}
	m.stopCleaner = utils.TimeTick(m.clean, 12*time.Hour)
	if ch != nil {
		ch.Add(registry.KeyThumbnail, m)
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
		for _, ext := range strings.Split(item.FileTypes, ",") {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if ext == "" {
				continue
			}
			hs[ext] = h
		}
	}
	return hs, nil
}

func (m *Maker) Registrations() []artifact.Registration {
	return []artifact.Registration{{
		Type:        ThumbnailArtifactType,
		Policy:      artifact.Policy{TTL: m.validity},
		TaskGroup:   "drive/thumbnail",
		Concurrency: m.concurrency,
	}}
}

func (m *Maker) Config() types.M {
	return types.M{"extensions": m.supportedExtensions()}
}

func (m *Maker) supportedExtensions() string {
	extensions := make([]string, 0, len(m.handlers))
	for ext := range m.handlers {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return strings.Join(extensions, ",")
}

func (m *Maker) Identity(request artifact.ArtifactRequest) (artifact.Identity, error) {
	if request.Type != ThumbnailArtifactType || request.Source == nil || request.Args != "" {
		return artifact.Identity{}, apierr.NewNotFoundMessageError("invalid thumbnail artifact request")
	}
	entry, err := m.createThumbnailEntry(request.Source)
	if err != nil {
		return artifact.Identity{}, normalizeThumbnailError(err)
	}
	return artifact.Identity{Key: m.artifactKey(entry), Fingerprint: thumbnailFingerprint(entry)}, nil
}

func (m *Maker) Produce(ctx types.TaskCtx, request artifact.ArtifactRequest, out artifact.Writer) error {
	entry, err := m.createThumbnailEntry(request.Source)
	if err != nil {
		return thumbnailProduceError(err)
	}
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
	err = normalizeThumbnailError(err)
	if cacheableThumbnailFailure(err) {
		return artifact.Cacheable(err)
	}
	return err
}

func cacheableThumbnailFailure(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func normalizeThumbnailError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if apierr.IsNotFoundError(err) {
		return err
	}
	if public, ok := errors.AsType[apierr.Error](err); ok {
		return apierr.NewNotFoundMessageError(public.Error())
	}
	return errors.New("failed to generate thumbnail")
}

func (m *Maker) artifactKey(entry ThumbnailEntry) string {
	return entry.GetRealPath()
}

func (m *Maker) createThumbnailEntry(entry types.IEntry) (ThumbnailEntry, error) {
	// we need to use the absolute path of this entry to generate thumbnail cache key
	// so we get the wrapped IDispatcherEntry here
	dispatcherEntry := driveutil.GetIEntry(entry, func(e types.IEntry) bool {
		_, ok := e.(types.IDispatcherEntry)
		return ok
	})
	if dispatcherEntry == nil {
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

	return &thumbnailEntry{
		IEntry:           entry,
		IDispatcherEntry: dispatcherEntry.(types.IDispatcherEntry),
		externalURL:      externalURL,
	}, nil
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

func thumbnailFingerprint(entry ThumbnailEntry) string {
	return fmt.Sprintf("v1|real=%s|path=%s|type=%v|size=%d|mod=%d",
		entry.GetRealPath(), entry.Path(), entry.Type(), entry.Size(), entry.ModTime())
}

func (m *Maker) clean() {
	if cleaned, cleanErr := m.artifactStore.Clean(ThumbnailArtifactType, m.validity, 0); cleanErr != nil {
		logging.For("thumbn").Errorf("error when cleaning thumbnail artifacts: %v", cleanErr)
	} else if cleaned > 0 {
		logging.For("thumbn").Debugf("%d expired thumbnail artifacts cleaned", cleaned)
	}
}

func (m *Maker) Dispose() error {
	if m.stopCleaner != nil {
		m.stopCleaner()
	}
	return nil
}

func thumbnailArtifactName(sourceName, mimeType string) string {
	base := utils.PathName(sourceName)
	if base == "" {
		base = "thumbnail"
	}
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(exts) == 0 {
		return base + ".thumb"
	}
	return base + exts[0]
}
