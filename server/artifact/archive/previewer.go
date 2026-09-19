// Package archive provides bounded, read-only previews for supported archive
// formats. The previewer owns archive-specific source and extraction logic;
// generic artifact registration, task dispatch, and output persistence stay in
// the artifact package.
package archive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"io/fs"
	"mime"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mholt/archives"
)

const archiveArtifactVersion = 1

const (
	IndexArtifactType   artifact.ArtifactType = "archive-index"
	ContentArtifactType artifact.ArtifactType = "archive-content"
	supportedExtensions                       = "zip,7z,rar"

	msgArchiveTooLarge       = "archive exceeds the configured size limit"
	msgArchiveTooManyEntries = "archive contains too many entries"
	msgArchiveMemberTooLarge = "archive member exceeds the configured size limit"
	msgInvalidArchive        = "invalid archive"
	msgUnsupportedArchive    = "unsupported archive format"
)

// Entry is the safe, format-independent metadata exposed by the preview API.
type Entry struct {
	Path     string          `json:"path"`
	Name     string          `json:"name"`
	Type     types.EntryType `json:"type"`
	Size     int64           `json:"size"`
	ModTime  int64           `json:"modTime"`
	MimeType string          `json:"mimeType,omitempty"`
}

// OpenedMember owns both the decompressed member and the source archive. The
// caller must close it after the response has finished streaming.
type OpenedMember struct {
	Entry
	Reader io.ReadCloser
}

type Previewer struct {
	maxSize       int64
	maxMemberSize int64
	maxEntries    int
	indexTTL      time.Duration
	contentTTL    time.Duration
	contentSize   int64
	tempDir       string
	sources       *driveutil.CacheFilePool
	artifactStore *artifact.Store
	runtime       *artifact.Service
	stopCleaner   func()
}

var (
	_ artifact.Processor = (*Previewer)(nil)
	_ artifact.Handler   = (*Previewer)(nil)
)

func init() {
	artifact.RegisterHandler("archive", func(ctx artifact.HandlerContext) (artifact.Handler, error) {
		tempDir := ctx.Config.TempDir
		if tempDir == "" {
			tempDir = os.TempDir()
		}
		return NewPreviewer(ctx.Config.Archive, tempDir, ctx.Store, ctx.Service, ctx.Components)
	})
}

// NewPreviewer creates the archive preview processor. Persistence, locking, and
// task scheduling stay in the shared artifact service; only remote source
// ranges use the separate sparse CacheFilePool.
func NewPreviewer(config common.ArchiveConfig, tempDir string,
	artifactStore *artifact.Store, runtime *artifact.Service, ch *registry.ComponentsHolder) (*Previewer, error) {
	if artifactStore == nil {
		return nil, errors.New("archive artifact store is required")
	}
	maxSize := config.MaxSize.DataSize(common.DefaultArchiveMaxSize)
	maxMemberSize := config.MaxMemberSize.DataSize(common.DefaultArchiveMaxMembers)
	maxEntries := config.MaxEntries
	cacheItems := config.CacheItems
	if maxSize <= 0 {
		maxSize = common.DefaultArchiveMaxSize
	}
	if maxMemberSize <= 0 {
		maxMemberSize = common.DefaultArchiveMaxMembers
	}
	if maxEntries <= 0 {
		maxEntries = common.DefaultArchiveMaxEntries
	}
	if cacheItems <= 0 {
		cacheItems = common.DefaultArchiveCacheItems
	}
	cacheSize := config.CacheSize.DataSize(common.DefaultArchiveCacheSize)
	if cacheSize <= 0 {
		cacheSize = common.DefaultArchiveCacheSize
	}
	indexTTL := config.IndexTTL
	if indexTTL <= 0 {
		indexTTL = common.DefaultArchiveIndexTTL
	}
	contentTTL := config.ContentCacheTTL
	if contentTTL <= 0 {
		contentTTL = common.DefaultArchiveContentCacheTTL
	}
	contentSize := config.ContentCacheSize.DataSize(common.DefaultArchiveContentCacheSize)
	if contentSize <= 0 {
		contentSize = common.DefaultArchiveContentCacheSize
	}
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	sourceDir := filepath.Join(tempDir, "archive-sources")
	if e := os.MkdirAll(sourceDir, 0700); e != nil {
		return nil, e
	}
	sources, e := driveutil.NewCacheFillPoolWithOptions(driveutil.CacheFilePoolOptions{
		MaxEntries:   cacheItems,
		MaxBytes:     cacheSize,
		Dir:          sourceDir,
		CleanStartup: true,
	})
	if e != nil {
		return nil, e
	}

	s := &Previewer{
		maxSize:       maxSize,
		maxMemberSize: maxMemberSize,
		maxEntries:    maxEntries,
		indexTTL:      indexTTL,
		contentTTL:    contentTTL,
		contentSize:   contentSize,
		tempDir:       tempDir,
		sources:       sources,
		artifactStore: artifactStore,
		runtime:       runtime,
	}
	if ch != nil {
		ch.Add(registry.KeyArchive, s)
		s.stopCleaner = utils.TimeTick(s.cleanIndexCache, 12*time.Hour)
	}
	return s, nil
}

func (s *Previewer) Registrations() []artifact.Registration {
	return []artifact.Registration{
		{
			Type:      IndexArtifactType,
			Policy:    artifact.Policy{TTL: s.indexTTL},
			TaskGroup: "drive/archive",
		},
		{
			Type: ContentArtifactType,
			Policy: artifact.Policy{
				TTL:      s.contentTTL,
				MaxBytes: s.contentSize,
			},
			TaskGroup: "drive/archive",
		},
	}
}

func (s *Previewer) Config() types.M {
	return types.M{
		"extensions": supportedExtensions,
		"maxSize":    s.maxSize,
	}
}

func (s *Previewer) Dispose() error {
	if s.stopCleaner != nil {
		s.stopCleaner()
	}
	if s.sources == nil {
		return nil
	}
	return s.sources.Dispose()
}

func (s *Previewer) cleanIndexCache() {
	if cleaned, cleanErr := s.artifactStore.Clean(IndexArtifactType, s.indexTTL, 0); cleanErr != nil {
		logging.For("archive").Warnf("archive index artifact cleanup failed: %v", cleanErr)
	} else if cleaned > 0 {
		logging.For("archive").Debugf("archive index artifacts cleaned=%d", cleaned)
	}
	if cleaned, cleanErr := s.artifactStore.Clean(ContentArtifactType, s.contentTTL, s.contentSize); cleanErr != nil {
		logging.For("archive").Warnf("archive content artifact cleanup failed: %v", cleanErr)
	} else if cleaned > 0 {
		logging.For("archive").Debugf("archive content artifacts cleaned=%d", cleaned)
	}
}

func (s *Previewer) Identity(request artifact.ArtifactRequest) (artifact.Identity, error) {
	if request.Source == nil {
		return artifact.Identity{}, notFound("archive artifact source is nil")
	}
	key, source := sourceFingerprint(request.Source)
	switch request.Type {
	case IndexArtifactType:
		if request.Args != "" {
			return artifact.Identity{}, notFound("archive index artifact args must be empty")
		}
		return artifact.Identity{Key: key, Fingerprint: s.indexArtifactFingerprint(source)}, nil
	case ContentArtifactType:
		if request.Args == "" {
			return artifact.Identity{}, notFound("archive content artifact args is empty")
		}
		member, err := normalizeMember(request.Args)
		if err != nil {
			return artifact.Identity{}, err
		}
		return artifact.Identity{
			Key:         key + "|" + member,
			Fingerprint: s.contentArtifactFingerprint(source, member),
		}, nil
	default:
		return artifact.Identity{}, notFound("invalid archive artifact type")
	}
}

func (s *Previewer) Produce(ctx types.TaskCtx, request artifact.ArtifactRequest, out artifact.Writer) error {
	setArchiveProgressTotal(ctx, request.Source.Size())
	switch request.Type {
	case IndexArtifactType:
		return archiveProduceError(s.produceIndex(ctx, request.Source, out))
	case ContentArtifactType:
		return archiveProduceError(s.produceContent(ctx, request.Source, request.Args, out))
	default:
		return notFound("invalid archive artifact type")
	}
}

func (s *Previewer) produceIndex(ctx types.TaskCtx, entry types.IEntry, out artifact.Writer) error {
	entries, err := s.buildIndex(ctx, entry)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if err := out.WriteMeta(artifact.Meta{
		Name:     entry.Name() + ".index.json",
		MimeType: "application/json",
		ModTime:  time.UnixMilli(entry.ModTime()),
	}); err != nil {
		return err
	}
	if _, err := out.Write(payload); err != nil {
		return err
	}
	setArchiveProgressComplete(ctx, entry.Size())
	return nil
}

func (s *Previewer) produceContent(ctx types.TaskCtx, entry types.IEntry, member string, out artifact.Writer) error {
	member, err := normalizeMember(member)
	if err != nil {
		return err
	}
	opened, err := s.openMember(ctx, entry, member)
	if err != nil {
		return err
	}
	defer func() { _ = opened.Reader.Close() }()
	if err := out.WriteMeta(artifact.Meta{
		Name:     opened.Name,
		MimeType: opened.MimeType,
		ModTime:  time.UnixMilli(opened.ModTime),
	}); err != nil {
		return err
	}
	_, err = io.Copy(out, &contextReader{ctx: ctx, reader: opened.Reader})
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logging.For("archive").Debugf("archive content artifact write failed member=%s: %v",
				logging.Sanitize(member), err)
		}
		return err
	}
	setArchiveProgressComplete(ctx, entry.Size())
	return nil
}

func archiveProduceError(err error) error {
	err = normalizeArchiveError(err)
	if cacheableArchiveFailure(err) {
		return artifact.Cacheable(err)
	}
	return err
}

func (s *Previewer) indexArtifactFingerprint(source string) string {
	return fmt.Sprintf("archive-index-v%d|source=%s|max-size=%d|max-entries=%d",
		archiveArtifactVersion, source, s.maxSize, s.maxEntries)
}

func (s *Previewer) contentArtifactFingerprint(source, member string) string {
	return fmt.Sprintf("archive-content-v%d|source=%s|member=%s|max-size=%d|max-entries=%d|max-member=%d",
		archiveArtifactVersion, source, member, s.maxSize, s.maxEntries, s.maxMemberSize)
}

func cacheableArchiveFailure(err error) bool {
	found, ok := errors.AsType[apierr.NotFoundError](err)
	if !ok {
		return false
	}
	switch found.Error() {
	case msgArchiveTooLarge, msgArchiveTooManyEntries, msgArchiveMemberTooLarge,
		msgUnsupportedArchive, msgInvalidArchive:
		return true
	default:
		return false
	}
}

func normalizeArchiveError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if apierr.IsNotFoundError(err) {
		return err
	}
	if public, ok := errors.AsType[apierr.Error](err); ok {
		return notFound(public.Error())
	}
	return errors.New("failed to process archive")
}

func notFound(msg string) error {
	return apierr.NewNotFoundMessageError(msg)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// openMember opens one archive member so Produce can copy it into the shared
// complete-artifact cache.
func (s *Previewer) openMember(ctx context.Context, entry types.IEntry, member string) (*OpenedMember, error) {
	member, e := normalizeMember(member)
	if e != nil {
		return nil, e
	}
	index, e := s.getIndex(ctx, entry)
	if e != nil {
		return nil, e
	}
	item, ok := findEntry(index, member)
	if !ok {
		return nil, notFound("archive member not found")
	}
	if item.Type == types.TypeDir {
		return nil, notFound("archive member is a directory")
	}
	if item.Size > s.maxMemberSize {
		return nil, notFound(msgArchiveMemberTooLarge)
	}

	source, format, e := s.openSourceAndFormat(ctx, entry)
	if e != nil {
		return nil, e
	}
	archiveFS := s.newArchiveFS(ctx, source, format)
	file, e := archiveFS.Open(member)
	if e != nil {
		_ = source.Close()
		return nil, invalidArchiveError(e)
	}

	readCloser := &memberReadCloser{reader: file, member: file, source: source}
	// Do not rely solely on the size advertised by the archive header: a
	// malformed or hostile archive may expand beyond it.
	readCloser.reader = &boundedReader{
		reader: readCloser.reader,
		limit:  s.maxMemberSize,
	}
	return &OpenedMember{Entry: item, Reader: readCloser}, nil
}

type source struct {
	reader readerAtSeeker
	closer io.Closer
	size   int64
}

func (s *source) Close() error {
	if s == nil || s.closer == nil {
		return nil
	}
	return s.closer.Close()
}

type pendingReader struct {
	mu     sync.Mutex
	reader io.ReadCloser
}

func (r *pendingReader) Take() io.ReadCloser {
	r.mu.Lock()
	defer r.mu.Unlock()
	reader := r.reader
	r.reader = nil
	return reader
}

func (r *pendingReader) Close() error {
	reader := r.Take()
	if reader == nil {
		return nil
	}
	return reader.Close()
}

type combinedCloser struct {
	closers []io.Closer
	once    sync.Once
	err     error
}

func (c *combinedCloser) Close() error {
	c.once.Do(func() {
		for _, closer := range c.closers {
			if e := closer.Close(); c.err == nil {
				c.err = e
			}
		}
	})
	return c.err
}

type readerAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type archiveProgressReader struct {
	reader io.ReadCloser
	ctx    types.TaskCtx
}

func (r *archiveProgressReader) Read(p []byte) (int, error) {
	n, e := r.reader.Read(p)
	if n > 0 {
		r.ctx.Progress(int64(n), false)
	}
	return n, e
}

func (r *archiveProgressReader) Close() error {
	return r.reader.Close()
}

func withArchiveProgress(ctx context.Context, reader io.ReadCloser) io.ReadCloser {
	taskCtx, ok := ctx.(types.TaskCtx)
	if !ok {
		return reader
	}
	return &archiveProgressReader{reader: reader, ctx: taskCtx}
}

func setArchiveProgressTotal(ctx context.Context, size int64) {
	if size < 0 {
		return
	}
	if taskCtx, ok := ctx.(types.TaskCtx); ok {
		taskCtx.Total(size, true)
	}
}

func setArchiveProgressComplete(ctx context.Context, size int64) {
	if size < 0 {
		return
	}
	if taskCtx, ok := ctx.(types.TaskCtx); ok {
		taskCtx.Total(size, true)
		taskCtx.Progress(size, true)
	}
}

type memberReadCloser struct {
	reader io.Reader
	member io.Closer
	source *source
	once   sync.Once
	err    error
}

func (r *memberReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *memberReadCloser) Close() error {
	r.once.Do(func() {
		r.err = r.member.Close()
		if e := r.source.Close(); r.err == nil {
			r.err = e
		}
	})
	return r.err
}

type boundedReader struct {
	reader io.Reader
	limit  int64
	read   int64
}

func (r *boundedReader) Read(p []byte) (int, error) {
	if r.read >= r.limit {
		var one [1]byte
		n, e := r.reader.Read(one[:])
		if n > 0 {
			return 0, notFound(msgArchiveMemberTooLarge)
		}
		return 0, e
	}
	max := r.limit - r.read + 1
	if int64(len(p)) > max {
		p = p[:int(max)]
	}
	n, e := r.reader.Read(p)
	r.read += int64(n)
	if r.read > r.limit {
		return n - int(r.read-r.limit), notFound(msgArchiveMemberTooLarge)
	}
	return n, e
}

func (s *Previewer) getIndex(ctx context.Context, entry types.IEntry) ([]Entry, error) {
	if s.runtime == nil {
		return nil, errors.New("archive artifact service is required")
	}
	taskCtx, ok := ctx.(types.TaskCtx)
	if !ok {
		taskCtx = task.NewTaskContext(ctx)
	}
	if _, err := s.runtime.Generate(taskCtx, artifact.ArtifactRequest{
		Source: entry,
		Type:   IndexArtifactType,
	}); err != nil {
		return nil, err
	}
	opened, err := s.runtime.Open(artifact.ArtifactRequest{
		Source: entry,
		Type:   IndexArtifactType,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = opened.Body.Close() }()
	var result []Entry
	if err := json.NewDecoder(opened.Body).Decode(&result); err != nil {
		return nil, invalidArchiveError(err)
	}
	return result, nil
}

func invalidArchiveError(e error) error {
	if e == nil || errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
		return e
	}
	if apierr.IsNotFoundError(e) {
		return e
	}
	return notFound(msgInvalidArchive)
}

func (s *Previewer) buildIndex(ctx context.Context, entry types.IEntry) ([]Entry, error) {
	source, format, e := s.openSourceAndFormat(ctx, entry)
	if e != nil {
		return nil, e
	}
	defer func() { _ = source.Close() }()
	archiveFS := s.newArchiveFS(ctx, source, format)

	result := make([]Entry, 0)
	seen := make(map[string]struct{})
	e = fs.WalkDir(archiveFS, ".", func(name string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if !validArchivePath(name) {
			return fmt.Errorf("unsafe archive path %q", name)
		}
		if _, ok := seen[name]; ok {
			return nil
		}
		if len(result) >= s.maxEntries {
			return notFound(msgArchiveTooManyEntries)
		}
		info, e := item.Info()
		if e != nil {
			return e
		}
		entryType := types.TypeFile
		if info.IsDir() {
			entryType = types.TypeDir
		}
		member := Entry{
			Path:    name,
			Name:    pathpkg.Base(name),
			Type:    entryType,
			Size:    info.Size(),
			ModTime: utils.Millisecond(info.ModTime()),
		}
		if entryType == types.TypeFile {
			member.MimeType = mime.TypeByExtension(strings.ToLower(pathpkg.Ext(name)))
		}
		seen[name] = struct{}{}
		result = append(result, member)
		return nil
	})
	if e != nil {
		return nil, invalidArchiveError(e)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func (s *Previewer) newArchiveFS(ctx context.Context, source *source, format archives.Extractor) *archives.ArchiveFS {
	return &archives.ArchiveFS{
		Stream:  io.NewSectionReader(source.reader, 0, source.size),
		Format:  boundedExtractor{inner: format, max: s.maxEntries},
		Context: ctx,
	}
}

// boundedExtractor applies the entry limit before ArchiveFS builds its own
// complete directory index. Checking only the fs.WalkDir callback would be too
// late: ArchiveFS.ReadDir indexes all format entries before returning.
type boundedExtractor struct {
	inner archives.Extractor
	max   int
}

func (e boundedExtractor) Extract(ctx context.Context, source io.Reader, handle archives.FileHandler) error {
	count := 0
	return e.inner.Extract(ctx, source, func(ctx context.Context, item archives.FileInfo) error {
		count++
		if count > e.max {
			return notFound(msgArchiveTooManyEntries)
		}
		return handle(ctx, item)
	})
}

func (s *Previewer) openSourceAndFormat(ctx context.Context, entry types.IEntry) (*source, archives.Extractor, error) {
	if entry.Type() != types.TypeFile {
		return nil, nil, notFound("archive entry is not a file")
	}
	size := entry.Size()
	if size > s.maxSize {
		return nil, nil, notFound(msgArchiveTooLarge)
	}

	// A URL-capable remote entry can go straight to the range cache. Probe the
	// native reader only when the provider has no URL; local FS entries return an
	// *os.File here and can therefore bypass all copying.
	if size < 0 {
		initial, e := entry.GetReader(ctx, -1, -1)
		if e != nil {
			return nil, nil, e
		}
		initial = withArchiveProgress(ctx, initial)
		file, copiedSize, copyErr := s.materialize(ctx, initial)
		_ = initial.Close()
		if copyErr != nil {
			return nil, nil, copyErr
		}
		setArchiveProgressTotal(ctx, copiedSize)
		if taskCtx, ok := ctx.(types.TaskCtx); ok {
			taskCtx.Progress(copiedSize, true)
		}
		format, detectErr := s.detectFormat(entry.Name(), file, copiedSize)
		if detectErr != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
			return nil, nil, detectErr
		}
		return &source{reader: file, closer: &tempSource{File: file}}, format, nil
	}
	key, _ := sourceFingerprint(entry)
	var pending *pendingReader
	if !s.sources.Has(key) {
		if _, urlErr := entry.GetURL(ctx); urlErr != nil {
			initial, e := entry.GetReader(ctx, -1, -1)
			if e != nil {
				return nil, nil, e
			}
			if ras, ok := initial.(readerAtSeeker); ok {
				format, e := s.detectFormat(entry.Name(), ras, size)
				if e != nil {
					_ = initial.Close()
					return nil, nil, e
				}
				return &source{reader: ras, closer: initial, size: size}, format, nil
			}
			pending = &pendingReader{reader: withArchiveProgress(ctx, initial)}
		}
	}

	reader, e := s.sources.GetReader(ctx, key, size,
		func(ctx context.Context, start, length int64) (io.ReadCloser, error) {
			if start < 0 && length < 0 && pending != nil {
				if reader := pending.Take(); reader != nil {
					return reader, nil
				}
			}
			reader, e := driveutil.GetIContentReader(ctx, entry, start, length)
			if e != nil {
				return nil, e
			}
			return withArchiveProgress(ctx, reader), nil
		},
	)
	if e != nil {
		if pending != nil {
			_ = pending.Close()
		}
		return nil, nil, e
	}
	closer := io.Closer(reader)
	if pending != nil {
		closer = &combinedCloser{closers: []io.Closer{reader, pending}}
	}
	ras, ok := reader.(readerAtSeeker)
	if !ok {
		_ = closer.Close()
		return nil, nil, errors.New("archive source cache is not seekable")
	}
	format, e := s.detectFormat(entry.Name(), ras, size)
	if e != nil {
		_ = closer.Close()
		return nil, nil, e
	}
	return &source{reader: ras, closer: closer, size: size}, format, nil
}

type tempSource struct {
	*os.File
	once sync.Once
}

func (t *tempSource) Close() error {
	var e error
	t.once.Do(func() {
		e = t.File.Close()
		if removeErr := os.Remove(t.File.Name()); e == nil {
			e = removeErr
		}
	})
	return e
}

func (s *Previewer) materialize(ctx context.Context, reader io.Reader) (*os.File, int64, error) {
	file, e := os.CreateTemp(s.tempDir, "archive-source-")
	if e != nil {
		return nil, 0, e
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}
	var total int64
	buf := make([]byte, 32*1024)
	for {
		if e := ctx.Err(); e != nil {
			cleanup()
			return nil, 0, e
		}
		n, readErr := reader.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > s.maxSize {
				cleanup()
				return nil, 0, notFound(msgArchiveTooLarge)
			}
			if _, e := file.Write(buf[:n]); e != nil {
				cleanup()
				return nil, 0, e
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				cleanup()
				return nil, 0, readErr
			}
			break
		}
	}
	if _, e := file.Seek(0, io.SeekStart); e != nil {
		cleanup()
		return nil, 0, e
	}
	return file, total, nil
}

func (s *Previewer) detectFormat(name string, reader readerAtSeeker, size int64) (archives.Extractor, error) {
	var format archives.Extractor
	switch strings.ToLower(pathpkg.Ext(name)) {
	case ".zip":
		format = archives.Zip{}
	case ".7z":
		format = archives.SevenZip{}
	case ".rar":
		format = archives.Rar{}
	}

	var detected archives.Extractor
	switch {
	case hasPrefix(reader, []byte("PK\x03\x04"), size) ||
		hasPrefix(reader, []byte("PK\x05\x06"), size) ||
		hasPrefix(reader, []byte("PK\x07\x08"), size):
		detected = archives.Zip{}
	case hasPrefix(reader, []byte("7z\xBC\xAF\x27\x1C"), size):
		detected = archives.SevenZip{}
	case hasPrefix(reader, []byte("Rar!\x1A\x07\x00"), size) ||
		hasPrefix(reader, []byte("Rar!\x1A\x07\x01\x00"), size):
		detected = archives.Rar{}
	default:
		return nil, notFound(msgUnsupportedArchive)
	}
	if format != nil && fmt.Sprintf("%T", format) != fmt.Sprintf("%T", detected) {
		return nil, notFound(msgUnsupportedArchive)
	}
	return detected, nil
}

func hasPrefix(reader readerAtSeeker, prefix []byte, size int64) bool {
	if size >= 0 && size < int64(len(prefix)) {
		return false
	}
	buf := make([]byte, len(prefix))
	n, e := reader.ReadAt(buf, 0)
	return e == nil && n == len(prefix) && string(buf) == string(prefix)
}

func sourceFingerprint(entry types.IEntry) (string, string) {
	base := driveutil.UnwrapIEntry(entry)
	realPath := entry.Path()
	if dispatcher := driveutil.GetIEntry(entry, func(item types.IEntry) bool {
		_, ok := item.(types.IDispatcherEntry)
		return ok
	}); dispatcher != nil {
		realPath = dispatcher.(types.IDispatcherEntry).GetRealPath()
	}
	fingerprint := fmt.Sprintf("v1|drive=%T|real=%s|path=%s|size=%d|mod=%d",
		base.Drive(), realPath, entry.Path(), entry.Size(), entry.ModTime())
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:]), fingerprint
}

func normalizeMember(name string) (string, error) {
	if strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '\\') {
		return "", notFound("invalid archive path")
	}
	name = pathpkg.Clean(strings.TrimPrefix(name, "/"))
	if name == "." || !validArchivePath(name) {
		return "", notFound("invalid archive path")
	}
	return name, nil
}

func validArchivePath(name string) bool {
	return name != "." && name != "" &&
		!pathpkg.IsAbs(name) && name != ".." &&
		!strings.HasPrefix(name, "../") &&
		!strings.ContainsRune(name, '\x00') &&
		!strings.ContainsRune(name, '\\')
}

func findEntry(entries []Entry, name string) (Entry, bool) {
	for _, item := range entries {
		if item.Path == name {
			return item, true
		}
	}
	return Entry{}, false
}
