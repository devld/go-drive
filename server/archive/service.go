// Package archive provides bounded, read-only previews for supported archive
// formats. It deliberately does not share the thumbnail handler registry:
// archive sources are sparse/seekable inputs, while thumbnails are complete
// output artifacts.
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
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
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

const archiveIndexVersion = 1

var (
	ErrUnsupportedFormat = errors.New("unsupported archive format")
	ErrInvalidArchive    = errors.New("invalid archive")
	ErrTooManyEntries    = errors.New("archive contains too many entries")
	ErrArchiveTooLarge   = errors.New("archive is too large")
	ErrMemberTooLarge    = errors.New("archive member is too large")
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

type Service struct {
	maxSize       int64
	maxMemberSize int64
	maxEntries    int
	indexTTL      time.Duration
	tempDir       string
	indexCache    *driveutil.ArtifactCache
	sources       *driveutil.CacheFilePool
	stopCleaner   func()

	indexLocksMu sync.Mutex
	indexLocks   map[string]*archiveIndexLock
}

type archiveIndexLock struct {
	sync.Mutex
	refs int
}

// NewService creates the archive preview service and its bounded source/index
// caches. Index files contain only metadata and a source fingerprint; partial
// source ranges are intentionally not persisted across restarts.
func NewService(config common.ArchiveConfig, tempDir string, ch *registry.ComponentsHolder) (*Service, error) {
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
	indexTTL := config.IndexTTL
	if indexTTL <= 0 {
		indexTTL = common.DefaultArchiveIndexTTL
	}
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	sourceDir := filepath.Join(tempDir, "archive-sources")
	indexDir := filepath.Join(tempDir, "archive-index")
	if e := os.MkdirAll(sourceDir, 0700); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(indexDir, 0700); e != nil {
		return nil, e
	}
	indexCache, e := driveutil.NewArtifactCache(indexDir)
	if e != nil {
		return nil, e
	}
	if cleaned, cleanErr := indexCache.CleanStartup(); cleanErr != nil {
		logging.For("archive").Warnf("archive index startup cleanup failed: %v", cleanErr)
	} else if cleaned > 0 {
		logging.For("archive").Debugf("archive index startup cleanup removed=%d", cleaned)
	}
	if cleaned, cleanErr := indexCache.CleanOlderThan(time.Now().Add(-indexTTL)); cleanErr != nil {
		logging.For("archive").Warnf("archive index expiration cleanup failed: %v", cleanErr)
	} else if cleaned > 0 {
		logging.For("archive").Debugf("archive index expiration cleanup removed=%d", cleaned)
	}
	sources, e := driveutil.NewCacheFillPoolWithOptions(driveutil.CacheFilePoolOptions{
		MaxEntries:   cacheItems,
		MaxBytes:     config.CacheSize.DataSize(common.DefaultArchiveCacheSize),
		Dir:          sourceDir,
		CleanStartup: true,
	})
	if e != nil {
		return nil, e
	}

	s := &Service{
		maxSize:       maxSize,
		maxMemberSize: maxMemberSize,
		maxEntries:    maxEntries,
		indexTTL:      indexTTL,
		tempDir:       tempDir,
		indexCache:    indexCache,
		sources:       sources,
		indexLocks:    make(map[string]*archiveIndexLock),
	}
	if ch != nil {
		ch.Add(registry.KeyArchive, s)
		s.stopCleaner = utils.TimeTick(s.cleanIndexCache, 12*time.Hour)
	}
	return s, nil
}

func (s *Service) Dispose() error {
	if s.stopCleaner != nil {
		s.stopCleaner()
	}
	if s.sources == nil {
		return nil
	}
	return s.sources.Dispose()
}

func (s *Service) cleanIndexCache() {
	cleaned, e := s.indexCache.CleanOlderThan(time.Now().Add(-s.indexTTL))
	if e != nil {
		logging.For("archive").Warnf("archive index expiration cleanup failed: %v", e)
	} else if cleaned > 0 {
		logging.For("archive").Debugf("archive index expiration cleanup removed=%d", cleaned)
	}
}

// List returns the immediate children of dir. An empty dir means archive root.
func (s *Service) List(ctx context.Context, entry types.IEntry, dir string) ([]Entry, error) {
	dir, e := normalizeDir(dir)
	if e != nil {
		return nil, e
	}
	index, e := s.getIndex(ctx, entry)
	if e != nil {
		return nil, e
	}

	if dir != "." && !containsDirectory(index, dir) {
		return nil, err.NewNotFoundMessageError("archive directory not found")
	}
	result := make([]Entry, 0)
	for _, item := range index {
		if pathpkg.Dir(item.Path) == dir {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type == types.TypeDir
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// Open opens one member as a streaming read closer. The member is never
// materialized as a second complete file.
func (s *Service) Open(ctx context.Context, entry types.IEntry, member string) (*OpenedMember, error) {
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
		return nil, err.NewNotFoundMessageError("archive member not found")
	}
	if item.Type == types.TypeDir {
		return nil, err.NewBadRequestError("archive member is a directory")
	}
	if item.Size > s.maxMemberSize {
		return nil, ErrMemberTooLarge
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
			return 0, ErrMemberTooLarge
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
		return n - int(r.read-r.limit), ErrMemberTooLarge
	}
	return n, e
}

func (s *Service) getIndex(ctx context.Context, entry types.IEntry) ([]Entry, error) {
	key, fingerprint := sourceFingerprint(entry)
	if result, ok := s.loadIndex(key, fingerprint); ok {
		return result, nil
	}
	lock := s.acquireIndexLock(key)
	defer s.releaseIndexLock(key, lock)
	if result, ok := s.loadIndex(key, fingerprint); ok {
		return result, nil
	}
	result, e := s.buildIndex(ctx, entry)
	if e != nil {
		if shouldReturnArchiveError(e) {
			return nil, e
		}
		return nil, invalidArchiveError(e)
	}
	if e := s.saveIndex(key, fingerprint, result); e != nil {
		// A cache write failure must not make an otherwise valid preview fail.
		return result, nil
	}
	return result, nil
}

func shouldReturnArchiveError(e error) bool {
	if errors.Is(e, ErrUnsupportedFormat) ||
		errors.Is(e, ErrArchiveTooLarge) ||
		errors.Is(e, ErrTooManyEntries) ||
		errors.Is(e, ErrInvalidArchive) ||
		errors.Is(e, context.Canceled) ||
		errors.Is(e, context.DeadlineExceeded) {
		return true
	}
	var publicError err.Error
	return errors.As(e, &publicError)
}

func invalidArchiveError(e error) error {
	if errors.Is(e, ErrInvalidArchive) {
		return e
	}
	return fmt.Errorf("%w: %v", ErrInvalidArchive, e)
}

func (s *Service) acquireIndexLock(key string) *archiveIndexLock {
	s.indexLocksMu.Lock()
	lock := s.indexLocks[key]
	if lock == nil {
		lock = &archiveIndexLock{}
		s.indexLocks[key] = lock
	}
	lock.refs++
	s.indexLocksMu.Unlock()
	lock.Lock()
	return lock
}

func (s *Service) releaseIndexLock(key string, lock *archiveIndexLock) {
	lock.Unlock()
	s.indexLocksMu.Lock()
	defer s.indexLocksMu.Unlock()
	lock.refs--
	if lock.refs == 0 && s.indexLocks[key] == lock {
		delete(s.indexLocks, key)
	}
}

func (s *Service) buildIndex(ctx context.Context, entry types.IEntry) ([]Entry, error) {
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
			return ErrTooManyEntries
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
		return nil, e
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func (s *Service) newArchiveFS(ctx context.Context, source *source, format archives.Extractor) *archives.ArchiveFS {
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
			return ErrTooManyEntries
		}
		return handle(ctx, item)
	})
}

func (s *Service) openSourceAndFormat(ctx context.Context, entry types.IEntry) (*source, archives.Extractor, error) {
	if entry.Type() != types.TypeFile {
		return nil, nil, err.NewBadRequestError("archive entry is not a file")
	}
	size := entry.Size()
	if size > s.maxSize {
		return nil, nil, ErrArchiveTooLarge
	}

	// A URL-capable remote entry can go straight to the range cache. Probe the
	// native reader only when the provider has no URL; local FS entries return an
	// *os.File here and can therefore bypass all copying.
	if size < 0 {
		initial, e := entry.GetReader(ctx, -1, -1)
		if e != nil {
			return nil, nil, e
		}
		file, copiedSize, copyErr := s.materialize(ctx, initial)
		_ = initial.Close()
		if copyErr != nil {
			return nil, nil, copyErr
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
			pending = &pendingReader{reader: initial}
		}
	}

	reader, e := s.sources.GetReaderContext(ctx, key, size,
		func(ctx context.Context, start, length int64) (io.ReadCloser, error) {
			if start < 0 && length < 0 && pending != nil {
				if reader := pending.Take(); reader != nil {
					return reader, nil
				}
			}
			return driveutil.GetIContentReader(ctx, entry, start, length)
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

func (s *Service) materialize(ctx context.Context, reader io.Reader) (*os.File, int64, error) {
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
				return nil, 0, ErrArchiveTooLarge
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

func (s *Service) detectFormat(name string, reader readerAtSeeker, size int64) (archives.Extractor, error) {
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
		return nil, ErrUnsupportedFormat
	}
	if format != nil && fmt.Sprintf("%T", format) != fmt.Sprintf("%T", detected) {
		return nil, ErrUnsupportedFormat
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

type persistedIndex struct {
	Version     int       `json:"version"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"createdAt"`
	Entries     []Entry   `json:"entries"`
}

func (s *Service) loadIndex(key, fingerprint string) ([]Entry, bool) {
	path := s.indexCache.ItemPath(key)
	file, exists, e := s.indexCache.OpenIfExists(path)
	if e != nil || !exists {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	var item persistedIndex
	if e := json.NewDecoder(file).Decode(&item); e != nil {
		_ = s.indexCache.Remove(path)
		return nil, false
	}
	if item.Version != archiveIndexVersion ||
		item.Fingerprint != fingerprint ||
		len(item.Entries) > s.maxEntries ||
		time.Since(item.CreatedAt) > s.indexTTL {
		_ = s.indexCache.Remove(path)
		return nil, false
	}
	return item.Entries, true
}

func (s *Service) saveIndex(key, fingerprint string, entries []Entry) error {
	item := persistedIndex{
		Version:     archiveIndexVersion,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
		Entries:     entries,
	}
	path := s.indexCache.ItemPath(key)
	return s.indexCache.Write(path, func(writer io.Writer) error {
		return json.NewEncoder(writer).Encode(item)
	})
}

func normalizeDir(name string) (string, error) {
	if name == "" || name == "." {
		return ".", nil
	}
	if strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '\\') {
		return "", err.NewBadRequestError("invalid archive path")
	}
	name = pathpkg.Clean(strings.TrimPrefix(name, "/"))
	if !validArchivePath(name) {
		return "", err.NewBadRequestError("invalid archive path")
	}
	return name, nil
}

func normalizeMember(name string) (string, error) {
	if strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '\\') {
		return "", err.NewBadRequestError("invalid archive path")
	}
	name = pathpkg.Clean(strings.TrimPrefix(name, "/"))
	if name == "." || !validArchivePath(name) {
		return "", err.NewBadRequestError("invalid archive path")
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

func containsDirectory(entries []Entry, name string) bool {
	item, ok := findEntry(entries, name)
	return ok && item.Type == types.TypeDir
}

func findEntry(entries []Entry, name string) (Entry, bool) {
	for _, item := range entries {
		if item.Path == name {
			return item, true
		}
	}
	return Entry{}, false
}
