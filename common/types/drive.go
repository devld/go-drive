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

// EntryMeta is the metadata of entry
type EntryMeta struct {
	// Readable indicates is this entry can be read
	Readable bool
	// Writable indicates is this entry can be written
	Writable bool
	// Thumbnail is the thumbnail image url if available.
	// For internal generated thumbnails, this is always empty.
	// There is an API for internal generated thumbnails.
	Thumbnail string
	// Props is some drive-specified properties of this entry
	Props M
}

type ContentURL struct {
	// URL is the download url of the entry content
	URL string
	// Header is the extra HTTP headers to the request to URL
	Header SM
	// If Proxy is true, URL will always be proxied by server,
	// otherwise just a http.StatusFound redirection will be sent to client.
	Proxy bool
}

type IContentReader interface {
	// GetReader gets the reader of this entry
	GetReader(context.Context) (io.ReadCloser, error)
	// GetURL gets the download url of this entry.
	// This API is optional, returns err.NewUnsupportedError if not supported.
	GetURL(context.Context) (*ContentURL, error)
}

// IContent is the extension of IEntry for file
type IContent interface {
	IContentReader

	// Name is filename of this entry
	Name() string
	Size() int64
	ModTime() int64
}

// IEntry is the abstraction of file and directory
type IEntry interface {
	Path() string
	Type() EntryType
	Size() int64
	Meta() EntryMeta
	ModTime() int64

	// Drive should return the drive of this entry
	Drive() IDrive
}

// IEntryWrapper is for some 'wrapper drive' such as dispatcher.go and permission_wrapper.go
type IEntryWrapper interface {
	// GetIEntry returns the real IEntry of this wrapper IEntry
	GetIEntry() IEntry
}

// IDispatcherEntry is for dispatcher.go to get the dispatched drive
type IDispatcherEntry interface {
	// GetDispatchedDrive returns the dispatched drive
	GetDispatchedDrive() (string, IDrive)
	GetRealPath() string
}

// DriveMeta is the metadata of drive
type DriveMeta struct {
	// Writable indicates is this drive writable
	Writable bool
	// Props is some drive-specified properties of this drive
	Props M
}

// IDrive is the main API. It's the abstraction of drives.
// The path argument in all public API always is valid path that not starts with slash,
// and the root path is always empty string. Implementations can consider the path to be absolute.
//
// All API returned error should created from err package, or the HTTP status will be 500.
//
// All API with TaskCtx argument should report operation progress to TaskCtx if possible.
type IDrive interface {
	// Meta returns the metadata of this drive
	Meta(ctx context.Context) DriveMeta
	// Get gets the entry. If the path is not exist, Get should returns err.NewNotFoundError
	Get(ctx context.Context, path string) (IEntry, error)
	// Save reads from reader and "save" the file.
	// If override is false and the path already exists, Save should failed with err.NewNotAllowedError.
	Save(ctx TaskCtx, path string, size int64, override bool, reader io.Reader) (IEntry, error)
	// MakeDir makes new directory at path.
	MakeDir(ctx context.Context, path string) (IEntry, error)
	// Copy copies 'from' to 'to'.
	// If the underlying drive does not support copying, Copy can return err.NewUnsupportedError,
	// then dispatcher drive(see dispatcher.go) will do the copying job.
	//
	// If override is false, the entry with the same path encountered during the copy process should be skipped.
	Copy(ctx TaskCtx, from IEntry, to string, override bool) (IEntry, error)
	// Move moves 'from' to 'to'.
	//
	// If override is false, the entry with the same path encountered during the copy process should be skipped.
	Move(ctx TaskCtx, from IEntry, to string, override bool) (IEntry, error)
	// List gets all children of the path.
	List(ctx context.Context, path string) ([]IEntry, error)
	// Delete deletes path and it's all descendants.
	Delete(ctx TaskCtx, path string) error

	// Upload returns the upload config of the path
	Upload(ctx context.Context, path string, size int64, override bool, config SM) (*DriveUploadConfig, error)
}

const (
	// LocalProvider is for smaller files. It's upload file directly
	LocalProvider = "local"
	// LocalChunkProvider is for files larger than LocalProviderChunkSize. It split files into some pieces to upload.
	LocalChunkProvider = "localChunk"
	// S3Provider is for S3 protocol uploading
	S3Provider = "s3"
	// OneDriveProvider is for OneDrive uploading API
	OneDriveProvider = "onedrive"
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

// DriveUploadConfig is the upload configuration of the path
type DriveUploadConfig struct {
	// Provider is the upload provider.
	// Available providers are LocalProvider, LocalChunkProvider, S3Provider, OneDriveProvider
	Provider string
	// Config is the provider-specified configuration for this uploading
	Config SM
}
