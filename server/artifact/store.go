// Package artifact stores complete preview outputs. It deliberately has no
// knowledge of drives or archive formats; processors own fingerprints and
// decide which failures are safe to cache.
package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type ArtifactType string

const (
	SyncWait  = 30 * time.Second
	AsyncWait = 2 * time.Second
)

type ArtifactRequest struct {
	Source types.IEntry
	Type   ArtifactType
	Args   string
}

// Identity names one cache slot and the source version that may occupy it.
// Key is stable (path, archive member). Fingerprint changes when the source
// or generator settings change, so the same slot can be overwritten or missed.
type Identity struct {
	Key         string
	Fingerprint string
}

type cacheableError struct{ error }

func (c cacheableError) Unwrap() error { return c.error }

// Cacheable marks err so Service persists a failure record for this identity.
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

// Info is the public description of a ready artifact. Identity is derived from
// path, type, and args; the storage key is not exposed. Ignore Info when the
// accompanying error is not nil.
type Info struct {
	Type     ArtifactType `json:"type"`
	Name     string       `json:"name,omitempty"`
	MimeType string       `json:"mimeType,omitempty"`
	Size     int64        `json:"size"`
}

// errCacheMiss is returned by Lookup when no usable record exists.
var errCacheMiss = errors.New("artifact cache miss")

type Artifact struct {
	Meta Meta
	Size int64
	Body io.ReadCloser
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
	caches  map[ArtifactType]*typeCache

	activeMu    sync.Mutex
	active      map[string]int
	ephemeralMu sync.Mutex
	ephemeral   map[string]bool
	cleanMu     map[ArtifactType]*sync.Mutex
	policies    map[ArtifactType]Policy
}

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
		caches:    make(map[ArtifactType]*typeCache),
		active:    make(map[string]int),
		ephemeral: make(map[string]bool),
		cleanMu:   make(map[ArtifactType]*sync.Mutex),
		policies:  make(map[ArtifactType]Policy),
	}
	return store, nil
}

func (s *Store) registerType(typ ArtifactType, policy Policy) error {
	cache, err := driveutil.NewBlobCache[blobMeta](filepath.Join(s.root, string(typ)))
	if err != nil {
		return err
	}
	if _, err := cache.CleanStartup(); err != nil {
		return fmt.Errorf("clean %s artifact cache: %w", typ, err)
	}

	s.cacheMu.Lock()
	if _, exists := s.caches[typ]; exists {
		s.cacheMu.Unlock()
		return fmt.Errorf("artifact type %q is already registered", typ)
	}
	s.caches[typ] = cache
	s.cleanMu[typ] = &sync.Mutex{}
	s.policies[typ] = policy
	s.cacheMu.Unlock()

	if _, err := s.Clean(typ, policy.TTL, policy.MaxBytes); err != nil {
		return fmt.Errorf("clean %s artifact cache: %w", typ, err)
	}
	return nil
}

func (s *Store) cache(typ ArtifactType) (*typeCache, error) {
	s.cacheMu.RLock()
	cache, ok := s.caches[typ]
	s.cacheMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unregistered artifact type %q", typ)
	}
	return cache, nil
}

func (s *Store) Lock(typ ArtifactType, key string) (func(), error) {
	cache, err := s.cache(typ)
	if err != nil {
		return nil, err
	}
	return cache.Lock(s.artifactKey(typ, key)), nil
}

func (s *Store) Lookup(typ ArtifactType, key, fingerprint string, ttl time.Duration) (Info, error) {
	cache, err := s.cache(typ)
	if err != nil {
		return Info{}, err
	}
	cacheKey := s.artifactKey(typ, key)
	rec, size, err := cache.ReadMeta(cacheKey)
	if err != nil {
		if !apierr.IsNotFoundError(err) {
			_ = s.removeByKey(typ, cacheKey, cache)
		}
		return Info{}, errCacheMiss
	}
	if rec.Fingerprint != fingerprint || rec.CreatedAt.IsZero() ||
		(ttl > 0 && time.Since(rec.CreatedAt) > ttl) {
		_ = s.removeByKey(typ, cacheKey, cache)
		return Info{}, errCacheMiss
	}
	if rec.Failed {
		return Info{}, apierr.NewNotFoundError()
	}
	return Info{Type: typ, Name: rec.Name, MimeType: rec.MimeType, Size: size}, nil
}

// Create starts one artifact record. WriteMeta must be called before writing
// bytes. Close publishes it; Abort discards it. Callers normally hold Lock
// for the same type/key while processing.
func (s *Store) Create(typ ArtifactType, key, fingerprint string) (*storeWriter, error) {
	cache, err := s.cache(typ)
	if err != nil {
		return nil, err
	}
	cacheKey := s.artifactKey(typ, key)
	blob, err := cache.Create(cacheKey)
	if err != nil {
		return nil, err
	}
	s.markActive(cacheKey, 1)
	return &storeWriter{
		BlobWriter:  blob,
		store:       s,
		typ:         typ,
		key:         cacheKey,
		fingerprint: fingerprint,
	}, nil
}

type storeWriter struct {
	driveutil.BlobWriter[blobMeta]
	store       *Store
	typ         ArtifactType
	key         string
	fingerprint string
	meta        blobMeta
	released    bool
}

func (w *storeWriter) WriteMeta(meta Meta) error {
	return w.writeMeta(meta, time.Now())
}

func (w *storeWriter) writeMeta(meta Meta, createdAt time.Time) error {
	w.meta = blobMeta{
		Name:        meta.Name,
		MimeType:    meta.MimeType,
		Fingerprint: w.fingerprint,
		CreatedAt:   createdAt,
		ModTime:     meta.ModTime,
	}
	return w.BlobWriter.WriteMeta(w.meta)
}

func (w *storeWriter) Close() error {
	err := w.BlobWriter.Close()
	if err == nil && !w.released {
		if policy, ok := w.store.policy(w.typ); ok && policy.MaxBytes > 0 {
			if w.Size() > policy.MaxBytes {
				w.store.markEphemeral(w.key, true)
			}
			_, _ = w.store.Clean(w.typ, policy.TTL, policy.MaxBytes)
		}
	}
	w.release()
	return err
}

func (w *storeWriter) Abort() {
	w.BlobWriter.Abort()
	w.release()
}

func (w *storeWriter) Info() Info {
	return Info{Type: w.typ, Name: w.meta.Name, MimeType: w.meta.MimeType, Size: w.Size()}
}

func (w *storeWriter) release() {
	if w.released {
		return
	}
	w.released = true
	w.store.markActive(w.key, -1)
}

func (s *Store) Open(typ ArtifactType, key string) (*Artifact, error) {
	cache, err := s.cache(typ)
	if err != nil {
		return nil, err
	}
	cacheKey := s.artifactKey(typ, key)
	s.markActive(cacheKey, 1)
	ephemeral := s.isEphemeral(cacheKey)
	rec, size, body, err := cache.Open(cacheKey)
	if err != nil {
		s.markActive(cacheKey, -1)
		return nil, err
	}
	if rec.Failed {
		_ = body.Close()
		s.markActive(cacheKey, -1)
		return nil, apierr.NewNotFoundError()
	}
	return &Artifact{Meta: rec.public(), Size: size, Body: &trackedBody{ReadCloser: body, done: func() {
		s.markActive(cacheKey, -1)
		if ephemeral {
			s.clearEphemeral(cacheKey)
			_ = s.removeByKey(typ, cacheKey, cache)
		}
	}}}, nil
}

func (s *Store) WriteFailure(typ ArtifactType, key, fingerprint string) error {
	cache, err := s.cache(typ)
	if err != nil {
		return err
	}
	writer, err := cache.Create(s.artifactKey(typ, key))
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

func (s *Store) Clean(typ ArtifactType, ttl time.Duration, maxBytes int64) (int, error) {
	cache, err := s.cache(typ)
	if err != nil {
		return 0, err
	}
	mutex := s.cleanMu[typ]
	mutex.Lock()
	defer mutex.Unlock()

	items, err := cache.Items()
	if err != nil {
		return 0, err
	}
	type kept struct {
		key       string
		createdAt time.Time
		size      int64
	}
	keptItems := make([]kept, 0, len(items))
	var total int64
	removed := 0
	drop := func(key string) {
		if s.removeByKeyUnlocked(cache, key) == nil {
			removed++
		}
	}
	for _, cached := range items {
		rec, size, readErr := cache.ReadMeta(cached.Key)
		createdAt, dropNow := time.Time{}, true
		switch {
		case readErr != nil:
		case !rec.CreatedAt.IsZero():
			createdAt, dropNow = rec.CreatedAt, false
		}
		if !dropNow && ttl > 0 && time.Since(createdAt) > ttl {
			dropNow = true
		}
		if dropNow {
			drop(cached.Key)
			continue
		}
		if rec.Failed {
			continue
		}
		keptItems = append(keptItems, kept{key: cached.Key, createdAt: createdAt, size: size})
		total += size
	}
	if maxBytes <= 0 || total <= maxBytes {
		return removed, nil
	}
	sort.Slice(keptItems, func(i, j int) bool {
		return keptItems[i].createdAt.Before(keptItems[j].createdAt)
	})
	for _, candidate := range keptItems {
		if total <= maxBytes {
			break
		}
		if s.removeByKeyUnlocked(cache, candidate.key) != nil {
			continue
		}
		total -= candidate.size
		removed++
	}
	return removed, nil
}

func (s *Store) policy(typ ArtifactType) (Policy, bool) {
	mutex := s.cleanMu[typ]
	mutex.Lock()
	defer mutex.Unlock()
	policy, ok := s.policies[typ]
	return policy, ok
}

func (s *Store) removeByKey(typ ArtifactType, key string, cache *typeCache) error {
	mutex := s.cleanMu[typ]
	mutex.Lock()
	defer mutex.Unlock()
	return s.removeByKeyUnlocked(cache, key)
}

func (s *Store) removeByKeyUnlocked(cache *typeCache, key string) error {
	if s.isActive(key) {
		return errors.New("artifact is active")
	}
	s.clearEphemeral(key)
	return cache.Remove(key)
}

// artifactKey is the storage key for a type/source pair.
func (s *Store) artifactKey(typ ArtifactType, key string) string {
	logicalKey := string(typ) + "|" + key
	digest := sha256.Sum256([]byte(logicalKey))
	return hex.EncodeToString(digest[:])
}

func (s *Store) markActive(id string, delta int) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	s.active[id] += delta
	if s.active[id] <= 0 {
		delete(s.active, id)
	}
}

func (s *Store) isActive(id string) bool {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	return s.active[id] > 0
}

func (s *Store) markEphemeral(id string, value bool) {
	s.ephemeralMu.Lock()
	defer s.ephemeralMu.Unlock()
	if value {
		s.ephemeral[id] = true
	} else {
		delete(s.ephemeral, id)
	}
}

func (s *Store) clearEphemeral(id string) {
	s.markEphemeral(id, false)
}

func (s *Store) isEphemeral(id string) bool {
	s.ephemeralMu.Lock()
	defer s.ephemeralMu.Unlock()
	return s.ephemeral[id]
}

type trackedBody struct {
	io.ReadCloser
	done func()
	once sync.Once
}

func (b *trackedBody) Close() error {
	var err error
	b.once.Do(func() {
		err = b.ReadCloser.Close()
		b.done()
	})
	return err
}
