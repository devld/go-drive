package driveutil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArtifactCache stores complete immutable files. It is intentionally separate
// from CacheFilePool, whose files are mutable sparse source ranges.
type ArtifactCache struct {
	dir string
}

func NewArtifactCache(dir string) (*ArtifactCache, error) {
	if dir == "" {
		return nil, errors.New("artifact cache directory is empty")
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	info, e := os.Stat(dir)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() {
		return nil, errors.New(dir + " is not a directory")
	}
	return &ArtifactCache{dir: dir}, nil
}

// ItemPath derives a stable, filesystem-safe path from a logical cache key.
func (c *ArtifactCache) ItemPath(key string) string {
	digest := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(digest[:]))
}

// DigestPath addresses an already-derived cache key. It is useful when an
// existing cache format already defines its digest (for example thumbnails).
func (c *ArtifactCache) DigestPath(digest string) string {
	return filepath.Join(c.dir, digest)
}

func (c *ArtifactCache) OpenIfExists(path string) (*os.File, bool, error) {
	file, e := os.Open(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	return file, true, nil
}

// Write atomically replaces path after the callback has completely written the
// new artifact. The temporary file is always removed on failure.
func (c *ArtifactCache) Write(path string, write func(io.Writer) error) error {
	tmp, e := os.CreateTemp(c.dir, ".artifact-*.tmp")
	if e != nil {
		return e
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()
	if e = write(tmp); e != nil {
		return e
	}
	if e = tmp.Sync(); e != nil {
		return e
	}
	if e = tmp.Close(); e != nil {
		return e
	}
	return os.Rename(tmpName, path)
}

// TryLock atomically creates a lock file. The returned file should be closed
// by the owner once the lock has been converted into the final artifact.
func (c *ArtifactCache) TryLock(path string) (io.Closer, bool, error) {
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(e, os.ErrExist) {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	return file, true, nil
}

func (c *ArtifactCache) Remove(path string) error {
	e := os.Remove(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	return e
}

// CleanStartup removes incomplete cache artifacts. Empty files are retained by
// some callers as negative-cache markers during a process lifetime, but they
// are intentionally invalid after a restart.
func (c *ArtifactCache) CleanStartup() (int, error) {
	return c.clean(func(path string, info os.FileInfo) bool {
		return strings.HasSuffix(info.Name(), ".lock") ||
			strings.HasSuffix(info.Name(), ".tmp") ||
			info.Size() == 0
	})
}

func (c *ArtifactCache) CleanOlderThan(cutoff time.Time) (int, error) {
	return c.clean(func(_ string, info os.FileInfo) bool {
		return info.ModTime().Before(cutoff)
	})
}

func (c *ArtifactCache) clean(shouldRemove func(string, os.FileInfo) bool) (int, error) {
	count := 0
	e := filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !shouldRemove(path, info) {
			return nil
		}
		if e := os.Remove(path); e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		count++
		return nil
	})
	return count, e
}
