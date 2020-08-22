package drive

import (
	"go-drive/common"
	"go-drive/common/types"
	"io"
)

// PermissionWrapperDrive intercept the intercept the request
// based on the permission information in the database
type PermissionWrapperDrive struct {
	drive   types.IDrive
	session types.Session
}

func NewPermissionWrapperDrive(session types.Session, drive types.IDrive) *PermissionWrapperDrive {
	return &PermissionWrapperDrive{drive: drive, session: session}
}

func (p *PermissionWrapperDrive) Meta() types.IDriveMeta {
	return p.drive.Meta()
}

func (p *PermissionWrapperDrive) Get(path string) (types.IEntry, error) {
	return p.drive.Get(path)
}

func (p *PermissionWrapperDrive) Save(path string, reader io.Reader, progress types.OnProgress) (types.IEntry, error) {
	return p.drive.Save(path, reader, progress)
}

func (p *PermissionWrapperDrive) MakeDir(path string) (types.IEntry, error) {
	return p.drive.MakeDir(path)
}

func (p *PermissionWrapperDrive) Copy(from types.IEntry, to string, progress types.OnProgress) (types.IEntry, error) {
	return p.drive.Copy(from, to, progress)
}

func (p *PermissionWrapperDrive) Move(from string, to string) (types.IEntry, error) {
	return p.drive.Move(from, to)
}

func (p *PermissionWrapperDrive) List(path string) ([]types.IEntry, error) {
	return p.drive.List(path)
}

func (p *PermissionWrapperDrive) Delete(path string) error {
	return p.drive.Delete(path)
}

func (p *PermissionWrapperDrive) Upload(path string, size int64, overwrite bool) (*types.DriveUploadConfig, error) {
	return p.drive.Upload(path, size, overwrite)
}

type PermissionWrapperEntry struct {
	entry types.IEntry
}

func (p *PermissionWrapperEntry) Path() string {
	return p.entry.Path()
}

func (p *PermissionWrapperEntry) Name() string {
	return p.entry.Name()
}

func (p *PermissionWrapperEntry) Type() types.EntryType {
	return p.entry.Type()
}

func (p *PermissionWrapperEntry) Size() int64 {
	return p.entry.Size()
}

func (p *PermissionWrapperEntry) Meta() types.IEntryMeta {
	return p.entry.Meta()
}

func (p *PermissionWrapperEntry) CreatedAt() int64 {
	return p.entry.CreatedAt()
}

func (p *PermissionWrapperEntry) UpdatedAt() int64 {
	return p.entry.UpdatedAt()
}

func (p *PermissionWrapperEntry) GetReader() (io.ReadCloser, error) {
	if c, ok := p.entry.(types.IContent); ok {
		return c.GetReader()
	}
	return nil, common.NewUnsupportedError()
}

func (p *PermissionWrapperEntry) GetURL() (string, bool, error) {
	if c, ok := p.entry.(types.IContent); ok {
		return c.GetURL()
	}
	return "", false, common.NewUnsupportedError()
}
