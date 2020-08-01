package common

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

type IEntryMeta interface {
	CanWrite() bool
	Props() map[string]interface{}
}

type IEntry interface {
	Path() string
	Name() string
	Type() EntryType
	Size() int64
	Meta() IEntryMeta
	CreatedAt() int64
	UpdatedAt() int64

	GetReader() (io.ReadCloser, error)
	// GetURL gets the download url of the file.
	// if second parameter is `true`, this file will be downloaded by proxy
	GetURL() (string, bool, error)
}

type IDriveMeta interface {
	CanWrite() bool
	Props() map[string]interface{}
}

type IDrive interface {
	Meta() IDriveMeta
	Get(path string) (IEntry, error)
	Save(path string, reader io.Reader, progress OnProgress) (IEntry, error)
	MakeDir(path string) (IEntry, error)
	Copy(from IEntry, to string, progress OnProgress) (IEntry, error)
	Move(from string, to string) (IEntry, error)
	List(path string) ([]IEntry, error)
	Delete(path string) error

	// Upload returns the upload config of the path
	Upload(path string, overwrite bool) (*DriveUploadConfig, error)
}

type DriveUploadConfig struct {
	Provider string
	Config   interface{}
}

type OnProgress func(loaded int64)

type NotFoundError struct {
	msg string
}

func (d NotFoundError) Error() string {
	return d.msg
}

type NotAllowedError struct {
	msg string
}

func (d NotAllowedError) Error() string {
	return d.msg
}

type UnsupportedError struct {
}

func (n UnsupportedError) Error() string {
	return "not supported"
}

type RemoteApiError struct {
	code int
	msg  string
}

func (r RemoteApiError) Error() string {
	return "[" + string(r.code) + "] " + r.msg
}
