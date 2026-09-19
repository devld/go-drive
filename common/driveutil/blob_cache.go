package driveutil

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/utils"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	blobHeaderMagic      = "GDBLOB1"
	blobMaxHeaderJSON    = 1 << 20
	blobMetaLenOffset    = len(blobHeaderMagic)
	blobPayloadLenOffset = blobMetaLenOffset + 8
	blobPrefixSize       = blobPayloadLenOffset + 8
)

var errCorruptBlob = errors.New("corrupt blob cache record")

// BlobCache stores an opaque JSON metadata value plus an optional binary
// payload. Callers address entries by key; the on-disk layout is private.
type BlobCache[M any] struct {
	dir   string
	locks *utils.KeyLock
}

type BlobItem struct {
	Key  string
	Size int64
}

func NewBlobCache[M any](dir string) (*BlobCache[M], error) {
	if dir == "" {
		return nil, errors.New("blob cache directory is empty")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &BlobCache[M]{dir: dir, locks: utils.NewKeyLock(0)}, nil
}

func (c *BlobCache[M]) Lock(key string) func() {
	c.locks.Lock(key)
	return func() { c.locks.Unlock(key) }
}

func (c *BlobCache[M]) ReadMeta(key string) (M, int64, error) {
	var zero M
	rec, size, err := c.readRecord(key)
	if err != nil {
		return zero, 0, c.missErr(key, err)
	}
	return rec, size, nil
}

func (c *BlobCache[M]) Open(key string) (M, int64, io.ReadCloser, error) {
	var zero M
	path, err := c.pathForKey(key)
	if err != nil {
		return zero, 0, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return zero, 0, nil, c.missErr(key, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return zero, 0, nil, err
	}
	meta, offset, payloadSize, err := readBlobHeader[M](file)
	if err != nil {
		_ = file.Close()
		return zero, 0, nil, c.missErr(key, err)
	}
	if info.Size()-offset != payloadSize {
		_ = file.Close()
		return zero, 0, nil, c.missErr(key, errCorruptBlob)
	}
	return meta, payloadSize, &blobPayloadCloser{Reader: io.LimitReader(file, payloadSize), file: file}, nil
}

// Writer accepts metadata then bytes. WriteMeta must be called once before Write.
type Writer[M any] interface {
	WriteMeta(M) error
	io.Writer
}

// BlobWriter is a Writer that can commit or discard the in-progress blob.
type BlobWriter[M any] interface {
	Writer[M]
	io.Closer
	Abort()
	Size() int64
}

type blobWriter[M any] struct {
	path      string
	tmp       *os.File
	headerLen int
	size      int64
	done      bool
}

func (c *BlobCache[M]) Create(key string) (BlobWriter[M], error) {
	path, err := c.pathForKey(key)
	if err != nil {
		return nil, err
	}
	return &blobWriter[M]{path: path}, nil
}

var _ BlobWriter[struct{}] = (*blobWriter[struct{}])(nil)

func (w *blobWriter[M]) WriteMeta(meta M) error {
	if w.done {
		return errors.New("blob writer is closed")
	}
	if w.tmp != nil {
		return errors.New("blob meta already written")
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(w.path), ".blob-*.tmp")
	if err != nil {
		return err
	}
	header, err := encodeBlobHeader(meta, 0)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if _, err := tmp.Write(header); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	w.tmp = tmp
	w.headerLen = len(header)
	return nil
}

func (w *blobWriter[M]) Write(p []byte) (int, error) {
	if w.done {
		return 0, errors.New("blob writer is closed")
	}
	if w.tmp == nil {
		return 0, errors.New("WriteMeta required before writing blob body")
	}
	return w.tmp.Write(p)
}

func (w *blobWriter[M]) Close() error {
	if w.done {
		return nil
	}
	if w.tmp == nil {
		w.Abort()
		return errors.New("blob meta is required")
	}
	end, err := w.tmp.Seek(0, io.SeekCurrent)
	if err != nil {
		w.Abort()
		return err
	}
	size := end - int64(w.headerLen)
	if size < 0 {
		w.Abort()
		return errCorruptBlob
	}
	if _, err := w.tmp.Seek(int64(blobPayloadLenOffset), io.SeekStart); err != nil {
		w.Abort()
		return err
	}
	if err := binary.Write(w.tmp, binary.LittleEndian, uint64(size)); err != nil {
		w.Abort()
		return err
	}
	if err := w.tmp.Sync(); err != nil {
		w.Abort()
		return err
	}
	tmpName := w.tmp.Name()
	if err := w.tmp.Close(); err != nil {
		w.tmp = nil
		_ = os.Remove(tmpName)
		w.done = true
		return err
	}
	w.tmp = nil
	if err := os.Rename(tmpName, w.path); err != nil {
		_ = os.Remove(tmpName)
		w.done = true
		return err
	}
	w.size = size
	w.done = true
	return nil
}

func (w *blobWriter[M]) Size() int64 {
	return w.size
}

func (w *blobWriter[M]) Abort() {
	if w.done {
		return
	}
	w.done = true
	if w.tmp == nil {
		return
	}
	tmpName := w.tmp.Name()
	_ = w.tmp.Close()
	w.tmp = nil
	_ = os.Remove(tmpName)
}

func (c *BlobCache[M]) Remove(key string) error {
	path, err := c.pathForKey(key)
	if err != nil {
		return err
	}
	if err := removeBlobFile(path); err != nil {
		return err
	}
	_ = os.Remove(filepath.Dir(path))
	return nil
}

func (c *BlobCache[M]) Items() ([]BlobItem, error) {
	items := make([]BlobItem, 0)
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !isBlobFile(info.Name()) {
			return nil
		}
		_, payloadSize, err := c.readRecordPath(path)
		if err != nil {
			return nil
		}
		items = append(items, BlobItem{Key: info.Name(), Size: payloadSize})
		return nil
	})
	return items, err
}

func (c *BlobCache[M]) CleanStartup() (int, error) {
	return c.clean(func(path string, info os.FileInfo) bool {
		if info.IsDir() {
			return false
		}
		name := info.Name()
		if strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".tmp") || strings.HasPrefix(name, ".") {
			return true
		}
		if !isBlobFile(name) {
			return false
		}
		if info.Size() == 0 {
			return true
		}
		_, _, err := c.readRecordPath(path)
		return err != nil
	})
}

func (c *BlobCache[M]) readRecord(key string) (M, int64, error) {
	var zero M
	path, err := c.pathForKey(key)
	if err != nil {
		return zero, 0, err
	}
	return c.readRecordPath(path)
}

func (c *BlobCache[M]) readRecordPath(path string) (M, int64, error) {
	var zero M
	file, err := os.Open(path)
	if err != nil {
		return zero, 0, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return zero, 0, err
	}
	meta, offset, payloadSize, err := readBlobHeader[M](file)
	if err != nil {
		return zero, 0, err
	}
	if info.Size()-offset != payloadSize {
		return zero, 0, errCorruptBlob
	}
	return meta, payloadSize, nil
}

func (c *BlobCache[M]) clean(shouldRemove func(string, os.FileInfo) bool) (int, error) {
	count := 0
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == c.dir || info.IsDir() || !shouldRemove(path, info) {
			return nil
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		count++
		_ = os.Remove(filepath.Dir(path))
		return nil
	})
	return count, err
}

func (c *BlobCache[M]) missErr(key string, e error) error {
	if errors.Is(e, os.ErrNotExist) {
		return err.NewNotFoundError()
	}
	if errors.Is(e, errCorruptBlob) {
		_ = c.Remove(key)
		return err.NewNotFoundError()
	}
	return e
}

func (c *BlobCache[M]) pathForKey(key string) (string, error) {
	if err := validBlobKey(key); err != nil {
		return "", err
	}
	prefix := key
	if len(key) >= 2 {
		prefix = key[:2]
	}
	return filepath.Join(c.dir, prefix, key), nil
}

func encodeBlobHeader[M any](meta M, payloadSize int64) ([]byte, error) {
	if payloadSize < 0 {
		return nil, errors.New("blob payload size is negative")
	}
	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	jsonBytes = append(jsonBytes, '\n')
	if len(jsonBytes) > blobMaxHeaderJSON {
		return nil, errors.New("blob metadata is too large")
	}
	header := make([]byte, blobPrefixSize+len(jsonBytes))
	copy(header, blobHeaderMagic)
	binary.LittleEndian.PutUint64(header[blobMetaLenOffset:], uint64(len(jsonBytes)))
	binary.LittleEndian.PutUint64(header[blobPayloadLenOffset:], uint64(payloadSize))
	copy(header[blobPrefixSize:], jsonBytes)
	return header, nil
}

func readBlobHeader[M any](file *os.File) (M, int64, int64, error) {
	var zero M
	prefix := make([]byte, blobPrefixSize)
	if _, err := io.ReadFull(file, prefix); err != nil {
		return zero, 0, 0, errCorruptBlob
	}
	if !bytes.Equal(prefix[:len(blobHeaderMagic)], []byte(blobHeaderMagic)) {
		return zero, 0, 0, errCorruptBlob
	}
	metaLen := binary.LittleEndian.Uint64(prefix[blobMetaLenOffset:])
	payloadLen := binary.LittleEndian.Uint64(prefix[blobPayloadLenOffset:])
	if metaLen == 0 || metaLen > blobMaxHeaderJSON || payloadLen > math.MaxInt64 {
		return zero, 0, 0, errCorruptBlob
	}
	jsonBuf := make([]byte, metaLen)
	if _, err := io.ReadFull(file, jsonBuf); err != nil {
		return zero, 0, 0, errCorruptBlob
	}
	var meta M
	if err := json.Unmarshal(jsonBuf, &meta); err != nil {
		return zero, 0, 0, errCorruptBlob
	}
	offset := int64(blobPrefixSize) + int64(metaLen)
	return meta, offset, int64(payloadLen), nil
}

func validBlobKey(key string) error {
	if key == "" || key != filepath.Base(key) || key == "." || key == ".." {
		return errors.New("blob cache key is invalid")
	}
	return nil
}

func isBlobFile(name string) bool {
	return name != "" && !strings.HasPrefix(name, ".") &&
		!strings.HasSuffix(name, ".tmp") && !strings.HasSuffix(name, ".lock")
}

func removeBlobFile(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

type blobPayloadCloser struct {
	io.Reader
	file *os.File
}

func (p *blobPayloadCloser) Close() error {
	return p.file.Close()
}
