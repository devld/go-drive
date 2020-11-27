package types

import "io"

const (
	TypeFile = "file"
	TypeDir  = "dir"
)

type EntryType string

func (t EntryType) IsFile() bool {
	return t == TypeFile
}

func (t EntryType) IsDir() bool {
	return t == TypeDir
}

type EntryMeta struct {
	CanRead   bool
	CanWrite  bool
	Thumbnail string
	Props     M
}

type ContentURL struct {
	URL    string
	Header SM
	Proxy  bool
}

type IContent interface {
	Name() string
	Size() int64
	ModTime() int64

	GetReader() (io.ReadCloser, error)
	GetURL() (*ContentURL, error)
}

type IEntry interface {
	Path() string
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
	Props    M
}

type IDrive interface {
	Meta() DriveMeta
	Get(path string) (IEntry, error)
	Save(path string, size int64, override bool, reader io.Reader, ctx TaskCtx) (IEntry, error)
	MakeDir(path string) (IEntry, error)
	Copy(from IEntry, to string, override bool, ctx TaskCtx) (IEntry, error)
	Move(from IEntry, to string, override bool, ctx TaskCtx) (IEntry, error)
	List(path string) ([]IEntry, error)
	Delete(path string, ctx TaskCtx) error

	// Upload returns the upload config of the path
	Upload(path string, size int64, override bool, config SM) (*DriveUploadConfig, error)
}

const (
	LocalProvider      = "local"
	LocalChunkProvider = "localChunk"
	S3Provider         = "s3"
	OneDriveProvider   = "onedrive"
)

const (
	LocalProviderChunkSize = 5 * 1024 * 1024
)

func UseLocalProvider(size int64) *DriveUploadConfig {
	if size <= LocalProviderChunkSize {
		return &DriveUploadConfig{Provider: LocalProvider}
	}
	return &DriveUploadConfig{Provider: LocalChunkProvider}
}

type DriveUploadConfig struct {
	Provider string
	Config   SM
}
