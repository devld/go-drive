package server

import (
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/storage"
	"io"
	"net/http"
	"time"
)

// PermissionWrapperDrive intercept the request
// based on the permission information in the database.
// The permissions of a child path inherit from the parent path,
// but have priority over the permissions of the parent path.
// Permissions for users take precedence over permissions for user groups.
// REJECT takes precedence over ACCEPT
type PermissionWrapperDrive struct {
	drive             types.IDrive
	subjects          []string
	request           *http.Request
	permissionStorage *storage.PathPermissionStorage
	requestSigner     *common.Signer
}

func NewPermissionWrapperDrive(
	request *http.Request, session types.Session, drive types.IDrive,
	permissionStorage *storage.PathPermissionStorage,
	requestSigner *common.Signer) *PermissionWrapperDrive {

	subjects := make([]string, 0, 3)
	subjects = append(subjects, "") // Anonymous
	if !session.IsAnonymous() {
		subjects = append(subjects, "u:"+session.User.Username)
		if session.User.Groups != nil {
			for _, g := range session.User.Groups {
				subjects = append(subjects, "g:"+g.Name)
			}
		}
	}

	return &PermissionWrapperDrive{
		drive:             drive,
		subjects:          subjects,
		request:           request,
		permissionStorage: permissionStorage,
		requestSigner:     requestSigner,
	}
}

func (p *PermissionWrapperDrive) Meta() types.DriveMeta {
	return p.drive.Meta()
}

func (p *PermissionWrapperDrive) Get(path string) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionRead)
	if e != nil {
		e = p.requirePermissionBySignature(path)
	}
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.Get(path)
	if e != nil {
		return nil, e
	}
	return &PermissionWrapperEntry{
		entry:      entry,
		permission: permission,
		accessKey:  p.signPathAccess(entry.Path()),
	}, nil
}

func (p *PermissionWrapperDrive) Save(path string, reader io.Reader, progress types.OnProgress) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.Save(path, reader, progress)
	if e != nil {
		return nil, e
	}
	return &PermissionWrapperEntry{entry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) MakeDir(path string) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.MakeDir(path)
	if e != nil {
		return nil, e
	}
	return &PermissionWrapperEntry{entry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) Copy(from types.IEntry, to string, progress types.OnProgress) (types.IEntry, error) {
	_, e := p.requirePermission(from.Path(), types.PermissionRead)
	if e != nil {
		return nil, e
	}
	toPermission, e := p.requirePermission(to, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.Copy(from, to, progress)
	if e != nil {
		return nil, e
	}
	return &PermissionWrapperEntry{entry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) Move(from string, to string) (types.IEntry, error) {
	_, e := p.requirePermission(from, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	toPermission, e := p.requirePermission(to, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}

	entry, e := p.drive.Move(from, to)
	if e != nil {
		return nil, e
	}
	return &PermissionWrapperEntry{entry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) List(path string) ([]types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionRead)
	if e != nil {
		return nil, e
	}
	entries, e := p.drive.List(path)
	if e != nil {
		return nil, e
	}

	pMap, e := p.permissionStorage.ResolvePathChildrenPermission(p.subjects, path)
	if e != nil {
		return nil, e
	}
	result := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Meta().CanRead {
			continue
		}
		per := permission
		if temp, ok := pMap[e.Path()]; ok {
			per = temp
		}
		if per.CanRead() {
			result = append(
				result,
				&PermissionWrapperEntry{
					entry:      e,
					permission: per,
					accessKey:  p.signPathAccess(e.Path()),
				},
			)
		}
	}
	return result, nil
}

func (p *PermissionWrapperDrive) Delete(path string) error {
	_, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return e
	}
	return p.drive.Delete(path)
}

func (p *PermissionWrapperDrive) Upload(path string, size int64, overwrite bool) (*types.DriveUploadConfig, error) {
	_, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	return p.drive.Upload(path, size, overwrite)
}

func (p *PermissionWrapperDrive) requirePermissionBySignature(path string) error {
	signature := p.request.URL.Query().Get("k")
	if !p.requestSigner.Validate(p.getSignPayload(path), signature) {
		return common.NewNotFoundError("not found")
	}
	return nil
}

func (p *PermissionWrapperDrive) signPathAccess(path string) string {
	return p.requestSigner.Sign(p.getSignPayload(path), time.Now().Add(12*time.Hour))
}

func (p *PermissionWrapperDrive) getSignPayload(path string) string {
	return p.request.Host + "." + path + "." + common.GetRealIP(p.request)
}

func (p *PermissionWrapperDrive) requirePermission(path string, require types.Permission) (types.Permission, error) {
	resolved, e := p.permissionStorage.ResolvePathPermission(p.subjects, path)
	if e != nil {
		return types.PermissionEmpty, e
	}
	if resolved&require != require {
		return resolved, common.NewNotFoundError("not found")
	}
	return resolved, nil
}

type PermissionWrapperEntry struct {
	entry      types.IEntry
	permission types.Permission
	accessKey  string
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

func (p *PermissionWrapperEntry) Meta() types.EntryMeta {
	meta := p.entry.Meta()
	props := make(map[string]interface{})
	if meta.Props != nil {
		for k, v := range meta.Props {
			props[k] = v
		}
	}
	props["access_key"] = p.accessKey
	return types.EntryMeta{
		CanRead:  meta.CanRead && p.permission.CanRead(),
		CanWrite: meta.CanWrite && p.permission.CanWrite(),
		Props:    props,
	}
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
