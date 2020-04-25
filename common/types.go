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
	CanRead() bool
	CanWrite() bool
	Props() map[string]interface{}
}

type IEntry interface {
	Name() string
	Type() EntryType
	Size() int64
	Meta() IEntryMeta
	CreatedAt() int64
	UpdatedAt() int64
}

type IDownloadable interface {
	// GetURL gets the download url of the file.
	// if second parameter is `true`, this file will be downloaded by proxy
	GetURL() (string, bool, error)
}

type IReadable interface {
	GetReader() (io.ReadCloser, error)
}

type IWriteable interface {
	Write(reader io.Reader, progress OnProgress) error
}

type IDriveMeta interface {
	CanWrite() bool
	DirectlyUpload() bool
	Props() map[string]interface{}
}

type IDrive interface {
	Meta() IDriveMeta
	Get(path string) (IEntry, error)
	Touch(path string) (IEntry, error)
	MakeDir(path string) (IEntry, error)
	Copy(from IEntry, to string, progress OnProgress) (IEntry, error)
	Move(from string, to string) (IEntry, error)
	List(path string) ([]IEntry, error)
	Delete(path string) error
}

type IDriveUpload interface {
	Upload(path string) (interface{}, error)
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

type NotSupportedError struct {
}

func (n NotSupportedError) Error() string {
	return "not supported"
}
