package archive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	"go-drive/common/types"
	"io"
	"os"
	pathpkg "path"
	"strings"

	"github.com/mholt/archives"
)

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
