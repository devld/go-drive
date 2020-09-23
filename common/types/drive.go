package types

import (
	"go-drive/common/task"
	"io"
)

const (
	TypeFile = "file"
	TypeDir  = "dir"
)

type DriveCreator = func(map[string]string) (IDrive, error)

type EntryType string

func (t EntryType) IsFile() bool {
	return t == TypeFile
}

func (t EntryType) IsDir() bool {
	return t == TypeDir
}

type EntryMeta struct {
	CanRead  bool
	CanWrite bool
	Props    map[string]interface{}
}

type IContent interface {
	Name() string
	Size() int64
	ModTime() int64

	GetReader() (io.ReadCloser, error)
	// GetURL gets the download url of the file.
	// if second parameter is `true`, this file will be downloaded by proxy
	GetURL() (string, bool, error)
}

type IEntry interface {
	Path() string
	Name() string
	Type() EntryType
	Size() int64
	Meta() EntryMeta
	ModTime() int64

	Drive() IDrive
}

type IEntryWrapper interface {
	GetIEntry() IEntry
}

type DriveMeta struct {
	CanWrite bool
	Props    map[string]interface{}
}

type IDrive interface {
	Meta() DriveMeta
	Get(path string) (IEntry, error)
	Save(path string, reader io.Reader, ctx task.Context) (IEntry, error)
	MakeDir(path string) (IEntry, error)
	Copy(from IEntry, to string, override bool, ctx task.Context) (IEntry, error)
	Move(from IEntry, to string, override bool, ctx task.Context) (IEntry, error)
	List(path string) ([]IEntry, error)
	Delete(path string, ctx task.Context) error

	// Upload returns the upload config of the path
	Upload(path string, size int64, override bool, config map[string]string) (*DriveUploadConfig, error)
}

const (
	LocalProvider      = "local"
	LocalChunkProvider = "localChunk"
	S3Provider         = "s3"
)

type DriveUploadConfig struct {
	Provider string
	Config   map[string]string
}
