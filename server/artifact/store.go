// Package artifact stores complete preview outputs. It deliberately has no
// knowledge of drives or archive formats; processors own fingerprints and
// decide which failures are safe to cache.
package artifact

import (
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type cacheableError struct{ error }

func (c cacheableError) Unwrap() error { return c.error }

// Cacheable marks err so Service persists a failure record for this ResolvedRequest.
func Cacheable(err error) error {
	if err == nil {
		return nil
	}
	if cached, ok := errors.AsType[cacheableError](err); ok {
		return cached
	}
	return cacheableError{err}
}

func IsCacheable(err error) bool {
	var cached cacheableError
	return errors.As(err, &cached)
}

// Meta is the handler-supplied description of an artifact body.
type Meta struct {
	Name     string    `json:"name,omitempty"`
	MimeType string    `json:"mimeType,omitempty"`
	ModTime  time.Time `json:"modTime,omitzero"`
}

// Info is the stored description of a ready artifact. Service wraps it when a
// retrieval ref is part of the response. Ignore Info when the accompanying
// error is not nil.
type Info struct {
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Size     int64  `json:"size"`
}

// errCacheMiss is returned by Lookup when no usable record exists.
var errCacheMiss = errors.New("artifact cache miss")

type Artifact struct {
	Meta Meta
	Size int64
	// Body is the cached payload, reopened after generation finishes.
	// It is an io.ReadSeekCloser, the same shape http.ServeContent uses
	// for HEAD, GET, and Range.
	Body io.ReadSeekCloser
}

type blobMeta struct {
	Name     string    `json:"name,omitempty"`
	MimeType string    `json:"mimeType,omitempty"`
	ModTime  time.Time `json:"modTime,omitzero"`

	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"createdAt"`
	Failed      bool      `json:"failed,omitempty"`
}

func (m blobMeta) public() Meta {
	return Meta{Name: m.Name, MimeType: m.MimeType, ModTime: m.ModTime}
}

type typeCache = driveutil.BlobCache[blobMeta]

type Store struct {
	root string

	cacheMu sync.RWMutex
	caches  map[string]*typeCache

	cleanMu  map[string]*sync.Mutex
	policies map[string]Policy

	useMu     sync.Mutex
	using     map[string]map[string]int
	ephemeral map[string]map[string]bool
}

// Policy controls one bucket. TTL is how long a finished artifact stays after
// it is published; it must be positive. An open reader is kept either way.
// MaxBytes evicts the oldest artifacts after the byte budget is exceeded; zero
// disables that limit.
type Policy struct {
	TTL      time.Duration
	MaxBytes int64
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("artifact store directory is empty")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	store := &Store{
		root:      root,
		caches:    make(map[string]*typeCache),
		cleanMu:   make(map[string]*sync.Mutex),
		policies:  make(map[string]Policy),
		using:     make(map[string]map[string]int),
		ephemeral: make(map[string]map[string]bool),
	}
	return store, nil
}

func (s *Store) registerType(bucket string, policy Policy) error {
	cache, err := driveutil.NewBlobCache[blobMeta](filepath.Join(s.root, bucket))
	if err != nil {
		return err
	}
	if _, err := cache.CleanStartup(); err != nil {
		return fmt.Errorf("clean %s artifact cache: %w", bucket, err)
	}

	s.cacheMu.Lock()
	if _, exists := s.caches[bucket]; exists {
		s.cacheMu.Unlock()
		return fmt.Errorf("artifact bucket %q is already registered", bucket)
	}
	s.caches[bucket] = cache
	s.cleanMu[bucket] = &sync.Mutex{}
	s.policies[bucket] = policy
	s.useMu.Lock()
	s.using[bucket] = make(map[string]int)
	s.ephemeral[bucket] = make(map[string]bool)
	s.useMu.Unlock()
	s.cacheMu.Unlock()

	if _, err := s.Clean(bucket, policy.TTL, policy.MaxBytes); err != nil {
		return fmt.Errorf("clean %s artifact cache: %w", bucket, err)
	}
	return nil
}

func (s *Store) cache(bucket string) (*typeCache, error) {
	s.cacheMu.RLock()
	cache, ok := s.caches[bucket]
	s.cacheMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unregistered artifact bucket %q", bucket)
	}
	return cache, nil
}

func (s *Store) Lock(bucket string, key string) (func(), error) {
	cache, err := s.cache(bucket)
	if err != nil {
		return nil, err
	}
	return cache.Lock(key), nil
}

func (s *Store) Lookup(bucket string, key, fingerprint string, ttl time.Duration) (Info, error) {
	cache, err := s.cache(bucket)
	if err != nil {
		return Info{}, err
	}
	rec, size, err := cache.ReadMeta(key)
	if err != nil {
		if !apierr.IsNotFoundError(err) {
			_ = s.removeByKey(bucket, key, cache)
		}
		return Info{}, errCacheMiss
	}
	if rec.Fingerprint != fingerprint || rec.CreatedAt.IsZero() ||
		time.Since(rec.CreatedAt) > ttl {
		_ = s.removeByKey(bucket, key, cache)
		return Info{}, errCacheMiss
	}
	if rec.Failed {
		return Info{}, apierr.NewNotFoundError()
	}
	return Info{Name: rec.Name, MimeType: rec.MimeType, Size: size}, nil
}

// Create starts one artifact record. WriteMeta must be called before writing
// bytes. Close publishes it; Abort discards it. Callers normally hold Lock
// for the same bucket/key while processing. The key is stored as given; the
// blob cache hashes it for the file name.
func (s *Store) Create(bucket string, key, fingerprint string) (*storeWriter, error) {
	cache, err := s.cache(bucket)
	if err != nil {
		return nil, err
	}
	blob, err := cache.Create(key)
	if err != nil {
		return nil, err
	}
	s.pin(bucket, key)
	return &storeWriter{
		BlobWriter:  blob,
		store:       s,
		bucket:      bucket,
		key:         key,
		fingerprint: fingerprint,
	}, nil
}

type storeWriter struct {
	driveutil.BlobWriter[blobMeta]
	store       *Store
	bucket      string
	key         string
	fingerprint string
	meta        blobMeta
	released    bool
}

func (w *storeWriter) WriteMeta(meta Meta) error {
	w.meta = blobMeta{
		Name:        meta.Name,
		MimeType:    meta.MimeType,
		Fingerprint: w.fingerprint,
		CreatedAt:   time.Now(),
		ModTime:     meta.ModTime,
	}
	return w.BlobWriter.WriteMeta(w.meta)
}

func (w *storeWriter) Close() error {
	err := w.BlobWriter.Close()
	if err == nil && !w.released {
		if policy, ok := w.store.policy(w.bucket); ok && policy.MaxBytes > 0 {
			if w.Size() > policy.MaxBytes {
				w.store.setEphemeral(w.bucket, w.key)
			}
			_, _ = w.store.Clean(w.bucket, policy.TTL, policy.MaxBytes)
		}
	}
	w.release()
	return err
}

func (w *storeWriter) Abort() {
	w.BlobWriter.Abort()
	w.release()
}

func (w *storeWriter) info() Info {
	return Info{Name: w.meta.Name, MimeType: w.meta.MimeType, Size: w.Size()}
}

func (w *storeWriter) release() {
	if w.released {
		return
	}
	w.released = true
	w.store.unpin(w.bucket, w.key)
}

// Open returns a published artifact that is still inside the bucket TTL.
// Failed and expired records are not-found. Expired records are removed.
// Fingerprint matching stays with Lookup.
func (s *Store) Open(bucket string, key string) (*Artifact, error) {
	policy, ok := s.policy(bucket)
	if !ok {
		return nil, apierr.NewNotFoundMessageError("artifact not found")
	}
	cache, err := s.cache(bucket)
	if err != nil {
		return nil, apierr.NewNotFoundMessageError("artifact not found")
	}
	s.pin(bucket, key)
	rec, size, body, err := cache.Open(key)
	if err != nil {
		s.unpin(bucket, key)
		return nil, apierr.NewNotFoundMessageError("artifact not found")
	}
	if rec.CreatedAt.IsZero() || time.Since(rec.CreatedAt) > policy.TTL {
		_ = body.Close()
		s.unpin(bucket, key)
		_ = s.removeByKey(bucket, key, cache)
		return nil, apierr.NewNotFoundMessageError("artifact not found")
	}
	if rec.Failed {
		_ = body.Close()
		s.unpin(bucket, key)
		return nil, apierr.NewNotFoundError()
	}
	return &Artifact{Meta: rec.public(), Size: size, Body: &trackedBody{ReadSeekCloser: body, done: func() {
		s.unpin(bucket, key)
		if s.takeEphemeral(bucket, key) {
			_ = s.removeByKey(bucket, key, cache)
		}
	}}}, nil
}

func (s *Store) WriteFailure(bucket string, key, fingerprint string) error {
	cache, err := s.cache(bucket)
	if err != nil {
		return err
	}
	writer, err := cache.Create(key)
	if err != nil {
		return err
	}
	if err := writer.WriteMeta(blobMeta{
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
		Failed:      true,
	}); err != nil {
		writer.Abort()
		return err
	}
	return writer.Close()
}

func (s *Store) Clean(bucket string, ttl time.Duration, maxBytes int64) (int, error) {
	cache, err := s.cache(bucket)
	if err != nil {
		return 0, err
	}
	mutex := s.cleanMu[bucket]
	mutex.Lock()
	defer mutex.Unlock()

	type candidate struct {
		key       string
		createdAt time.Time
		size      int64
	}
	var drop []string
	var kept []candidate
	var total int64
	if err := cache.Visit(func(key string, meta blobMeta, size int64) error {
		expired := meta.CreatedAt.IsZero() || time.Since(meta.CreatedAt) > ttl
		if expired {
			drop = append(drop, key)
			return nil
		}
		if meta.Failed {
			return nil
		}
		kept = append(kept, candidate{key: key, createdAt: meta.CreatedAt, size: size})
		total += size
		return nil
	}); err != nil {
		return 0, err
	}
	removed := 0
	discard := func(key string) bool {
		if err := s.removeByKeyUnlocked(bucket, cache, key); err != nil {
			return false
		}
		removed++
		return true
	}
	for _, key := range drop {
		discard(key)
	}
	if maxBytes <= 0 || total <= maxBytes {
		return removed, nil
	}
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].createdAt.Before(kept[j].createdAt)
	})
	for _, candidate := range kept {
		if total <= maxBytes {
			break
		}
		if !discard(candidate.key) {
			continue
		}
		total -= candidate.size
	}
	return removed, nil
}

func (s *Store) policy(bucket string) (Policy, bool) {
	mutex := s.cleanMu[bucket]
	mutex.Lock()
	defer mutex.Unlock()
	policy, ok := s.policies[bucket]
	return policy, ok
}

func (s *Store) removeByKey(bucket string, key string, cache *typeCache) error {
	mutex := s.cleanMu[bucket]
	mutex.Lock()
	defer mutex.Unlock()
	return s.removeByKeyUnlocked(bucket, cache, key)
}

func (s *Store) removeByKeyUnlocked(bucket string, cache *typeCache, key string) error {
	if s.busy(bucket, key) {
		return errors.New("artifact is active")
	}
	if err := cache.Remove(key); err != nil {
		return err
	}
	s.clearEphemeral(bucket, key)
	return nil
}

func (s *Store) pin(bucket, key string) {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	s.using[bucket][key]++
}

func (s *Store) unpin(bucket, key string) {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	users := s.using[bucket]
	if users[key] <= 1 {
		delete(users, key)
		return
	}
	users[key]--
}

func (s *Store) busy(bucket, key string) bool {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	return s.using[bucket][key] > 0
}

func (s *Store) setEphemeral(bucket, key string) {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	s.ephemeral[bucket][key] = true
}

func (s *Store) takeEphemeral(bucket, key string) bool {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	marks := s.ephemeral[bucket]
	if !marks[key] {
		return false
	}
	delete(marks, key)
	return true
}

func (s *Store) clearEphemeral(bucket, key string) {
	s.useMu.Lock()
	defer s.useMu.Unlock()
	delete(s.ephemeral[bucket], key)
}

type trackedBody struct {
	io.ReadSeekCloser
	done func()
	once sync.Once
}

func (b *trackedBody) Close() error {
	var err error
	b.once.Do(func() {
		err = b.ReadSeekCloser.Close()
		b.done()
	})
	return err
}
