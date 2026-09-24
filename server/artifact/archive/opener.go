package archive

import (
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	"go-drive/common/types"
	"io"
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
	dfs, e := driveutil.NewDriveFS(ctx, entry.Drive(), "", s.sources)
	if e != nil {
		return nil, nil, e
	}
	file, e := dfs.OpenEntry(ctx, entry)
	if e != nil {
		return nil, nil, e
	}
	ras, ok := file.(readerAtSeeker)
	if !ok {
		_ = file.Close()
		return nil, nil, errors.New("archive source is not seekable")
	}
	format, e := s.detectFormat(entry.Name(), ras, size)
	if e != nil {
		_ = file.Close()
		return nil, nil, e
	}
	return &source{reader: ras, closer: file, size: size}, format, nil
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
