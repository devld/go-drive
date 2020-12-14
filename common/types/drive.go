package types

import (
	"context"
	"io"
)

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

	GetReader(context.Context) (io.ReadCloser, error)
	GetURL(context.Context) (*ContentURL, error)
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
	Meta(ctx context.Context) DriveMeta
	Get(ctx context.Context, path string) (IEntry, error)
	Save(ctx TaskCtx, path string, size int64, override bool, reader io.Reader) (IEntry, error)
	MakeDir(ctx context.Context, path string) (IEntry, error)
	Copy(ctx TaskCtx, from IEntry, to string, override bool) (IEntry, error)
	Move(ctx TaskCtx, from IEntry, to string, override bool) (IEntry, error)
	List(ctx context.Context, path string) ([]IEntry, error)
	Delete(ctx TaskCtx, path string) error

	// Upload returns the upload config of the path
	Upload(ctx context.Context, path string, size int64, override bool, config SM) (*DriveUploadConfig, error)
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
