// Package archive provides bounded, read-only previews for supported archive
// formats. The previewer owns archive-specific source and extraction logic;
// generic artifact registration, task dispatch, and output persistence stay in
// the artifact package.
package archive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/artifact"
	"io"
	"io/fs"
	"mime"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mholt/archives"
)

const archiveArtifactVersion = 1

const (
	handlerName = "archive"

	argsIndex         = "index"
	argsContentPrefix = "content:"

	cacheIndex   = "index"
	cacheContent = "content"

	supportedExtensions = "zip,7z,rar"
)

var (
	msgArchiveTooLarge       = i18n.T("api.archive.too_large")
	msgArchiveTooManyEntries = i18n.T("api.archive.too_many_entries")
	msgArchiveMemberTooLarge = i18n.T("api.archive.member_too_large")
	msgInvalidArchive        = i18n.T("api.archive.invalid")
	msgUnsupportedArchive    = i18n.T("api.archive.unsupported")
	msgProcessFailed         = i18n.T("api.archive.process_failed")
	msgMemberNotFound        = i18n.T("api.archive.member_not_found")
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
	sources       *driveutil.CacheFilePool
	cache         artifact.Cache
}

var (
	_ artifact.Handler  = (*Previewer)(nil)
	_ types.IDisposable = (*Previewer)(nil)
)

func init() {
	artifact.RegisterHandler(handlerName, NewPreviewer)
}

// NewPreviewer creates the archive preview processor. Persistence, locking, and
// task scheduling stay in the shared artifact service. Local *os.File sources
// are used directly; other sources go through CacheFilePool.
func NewPreviewer(ctx artifact.HandlerContext) (artifact.Handler, error) {
	tempDir := ctx.Config.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	config := ctx.Config.Archive
	maxSize := utils.PositiveOr(config.MaxSize.DataSize(common.DefaultArchiveMaxSize), common.DefaultArchiveMaxSize)
	maxMemberSize := utils.PositiveOr(config.MaxMemberSize.DataSize(common.DefaultArchiveMaxMembers), common.DefaultArchiveMaxMembers)
	maxEntries := utils.PositiveOr(config.MaxEntries, common.DefaultArchiveMaxEntries)
	cacheItems := utils.PositiveOr(config.CacheItems, common.DefaultArchiveCacheItems)
	cacheSize := utils.PositiveOr(config.CacheSize.DataSize(common.DefaultArchiveCacheSize), common.DefaultArchiveCacheSize)
	indexTTL := utils.PositiveOr(config.IndexTTL, common.DefaultArchiveIndexTTL)
	contentTTL := utils.PositiveOr(config.ContentCacheTTL, common.DefaultArchiveContentCacheTTL)
	contentSize := utils.PositiveOr(config.ContentCacheSize.DataSize(common.DefaultArchiveContentCacheSize), common.DefaultArchiveContentCacheSize)

	sourceDir := filepath.Join(tempDir, "archive-sources")
	if e := os.MkdirAll(sourceDir, 0700); e != nil {
		return nil, e
	}
	sources, e := driveutil.NewCacheFilePool(driveutil.CacheFilePoolOptions{
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
		sources:       sources,
		cache:         ctx.Cache,
	}
	return s, nil
}

func (s *Previewer) Spec() artifact.Spec {
	return artifact.Spec{
		Caches: []artifact.CacheSpec{
			{Name: cacheIndex, Policy: artifact.Policy{TTL: s.indexTTL}},
			{Name: cacheContent, Policy: artifact.Policy{TTL: s.contentTTL, MaxBytes: s.contentSize}},
		},
		Config: types.M{
			"extensions": supportedExtensions,
			"maxSize":    s.maxSize,
		},
	}
}

func (s *Previewer) Dispose() error {
	if s == nil || s.sources == nil {
		return nil
	}
	err := s.sources.Dispose()
	s.sources = nil
	return err
}

func parseArchiveArgs(args string) (content bool, member string, err error) {
	switch {
	case args == argsIndex:
		return false, "", nil
	case strings.HasPrefix(args, argsContentPrefix):
		member, err = normalizeMember(strings.TrimPrefix(args, argsContentPrefix))
		if err != nil {
			return false, "", err
		}
		if member == "" {
			return false, "", notFound("archive content artifact args is empty")
		}
		return true, member, nil
	default:
		return false, "", notFound("invalid archive artifact args")
	}
}

func (s *Previewer) Resolve(request artifact.Request) (artifact.ResolvedRequest, error) {
	if request.Source == nil {
		return artifact.ResolvedRequest{}, notFound("archive artifact source is nil")
	}
	content, member, err := parseArchiveArgs(request.Args)
	if err != nil {
		return artifact.ResolvedRequest{}, err
	}
	if !content {
		return artifact.ResolvedRequest{
			Fingerprint: fmt.Sprintf("archive-index-v%d|max-size=%d|max-entries=%d",
				archiveArtifactVersion, s.maxSize, s.maxEntries),
			Cache: cacheIndex,
		}, nil
	}
	return artifact.ResolvedRequest{
		Key: member,
		Fingerprint: fmt.Sprintf("archive-content-v%d|max-size=%d|max-entries=%d|max-member=%d",
			archiveArtifactVersion, s.maxSize, s.maxEntries, s.maxMemberSize),
		Cache: cacheContent,
	}, nil
}

func (s *Previewer) Produce(ctx types.TaskCtx, request artifact.Request, out artifact.Writer) error {
	setArchiveProgressTotal(ctx, request.Source.Size())
	content, member, err := parseArchiveArgs(request.Args)
	if err != nil {
		return err
	}
	if !content {
		return archiveProduceError(s.produceIndex(ctx, request.Source, out))
	}
	return archiveProduceError(s.produceContent(ctx, request.Source, member, out))
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
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if !apierr.IsNotFoundError(err) {
		if public, ok := errors.AsType[apierr.Error](err); ok {
			err = notFound(public.Error())
		} else {
			err = errors.New(msgProcessFailed)
		}
	}
	found, ok := errors.AsType[apierr.NotFoundError](err)
	if !ok {
		return err
	}
	switch found.Error() {
	case msgArchiveTooLarge, msgArchiveTooManyEntries, msgArchiveMemberTooLarge,
		msgUnsupportedArchive, msgInvalidArchive:
		return artifact.Cacheable(err)
	default:
		return err
	}
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
func (s *Previewer) openMember(ctx types.TaskCtx, entry types.IEntry, member string) (*OpenedMember, error) {
	index, e := s.loadIndex(ctx, entry)
	if e != nil {
		return nil, e
	}
	item, ok := findEntry(index, member)
	if !ok {
		return nil, notFound(msgMemberNotFound)
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

type readerAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

func withArchiveProgress(ctx types.TaskCtx, reader io.ReadCloser) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{driveutil.ProgressReader(reader, ctx), reader}
}

func setArchiveProgressTotal(ctx types.TaskCtx, size int64) {
	if size < 0 {
		return
	}
	ctx.Total(size, true)
}

func setArchiveProgressComplete(ctx types.TaskCtx, size int64) {
	if size < 0 {
		return
	}
	ctx.Total(size, true)
	ctx.Progress(size, true)
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

func (s *Previewer) loadIndex(ctx types.TaskCtx, entry types.IEntry) ([]Entry, error) {
	if entries, ok := s.cachedIndex(entry); ok {
		return entries, nil
	}
	return s.buildIndex(ctx, entry)
}

func (s *Previewer) cachedIndex(entry types.IEntry) ([]Entry, bool) {
	if s.cache == nil {
		return nil, false
	}
	art, err := s.cache.Get(entry, argsIndex)
	if err != nil {
		return nil, false
	}
	defer func() { _ = art.Body.Close() }()
	var entries []Entry
	if err := json.NewDecoder(art.Body).Decode(&entries); err != nil {
		return nil, false
	}
	return entries, true
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

func (s *Previewer) buildIndex(ctx types.TaskCtx, entry types.IEntry) ([]Entry, error) {
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

func (s *Previewer) openSourceAndFormat(ctx types.TaskCtx, entry types.IEntry) (*source, archives.Extractor, error) {
	if entry.Type() != types.TypeFile {
		return nil, nil, notFound("archive entry is not a file")
	}
	size := entry.Size()
	if size < 0 {
		return nil, nil, notFound(msgUnsupportedArchive)
	}
	if size > s.maxSize {
		return nil, nil, notFound(msgArchiveTooLarge)
	}
	key := sourceCacheKey(entry)
	if !s.sources.Has(key) {
		if local, err := openLocalFile(ctx, entry); err != nil {
			return nil, nil, err
		} else if local != nil {
			format, e := s.detectFormat(entry.Name(), local, size)
			if e != nil {
				_ = local.Close()
				return nil, nil, e
			}
			return &source{reader: local, closer: local, size: size}, format, nil
		}
	}
	reader, e := s.sources.GetReader(ctx, key, size,
		func(reqCtx context.Context, start, length int64) (io.ReadCloser, error) {
			reader, e := driveutil.GetIContentReader(reqCtx, entry, start, length)
			if e != nil {
				return nil, e
			}
			return withArchiveProgress(ctx, reader), nil
		},
	)
	if e != nil {
		return nil, nil, e
	}
	ras, ok := reader.(readerAtSeeker)
	if !ok {
		_ = reader.Close()
		return nil, nil, errors.New("archive source cache is not seekable")
	}
	format, e := s.detectFormat(entry.Name(), ras, size)
	if e != nil {
		_ = reader.Close()
		return nil, nil, e
	}
	return &source{reader: ras, closer: reader, size: size}, format, nil
}

// openLocalFile returns a native *os.File when GetReader already provides one
// (local fs). URL-capable remotes skip this probe so the range cache can fetch
// them. A non-file reader is closed and the caller should use CacheFilePool.
func openLocalFile(ctx context.Context, entry types.IEntry) (*os.File, error) {
	if _, err := entry.GetURL(ctx); err == nil {
		return nil, nil
	}
	reader, err := entry.GetReader(ctx, -1, -1)
	if err != nil {
		return nil, err
	}
	file, ok := reader.(*os.File)
	if !ok {
		_ = reader.Close()
		return nil, nil
	}
	return file, nil
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

func sourceCacheKey(entry types.IEntry) string {
	realPath := entry.Path()
	if dispatcher, ok := driveutil.IEntryAs[types.IDispatcherEntry](entry); ok {
		realPath = dispatcher.GetRealPath()
	}
	return fmt.Sprintf("%s|%d|%d", realPath, entry.Size(), entry.ModTime())
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
	i, ok := slices.BinarySearchFunc(entries, name, func(item Entry, name string) int {
		return strings.Compare(item.Path, name)
	})
	if !ok {
		return Entry{}, false
	}
	return entries[i], true
}
