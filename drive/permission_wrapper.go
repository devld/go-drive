package drive

import (
	"context"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"time"
)

const (
	accessKeyValidity = 12 * time.Hour
)

type EntrySigner interface {
	GetSignature(path string, notAfter time.Time) string
	CheckSignature(path string) bool
}

// PermissionWrapperDrive intercept the request
// based on the permission information in the database.
// The permissions of a child path inherit from the parent path,
// but have priority over the permissions of the parent path.
// Permissions for users take precedence over permissions for user groups.
// REJECT takes precedence over ACCEPT
type PermissionWrapperDrive struct {
	drive  types.IDrive
	pm     utils.PermMap
	signer EntrySigner
}

func NewPermissionWrapperDrive(drive types.IDrive, permissions utils.PermMap, signer EntrySigner) *PermissionWrapperDrive {
	return &PermissionWrapperDrive{
		drive:  drive,
		pm:     permissions,
		signer: signer,
	}
}

func (p *PermissionWrapperDrive) Meta(ctx context.Context) types.DriveMeta {
	return p.drive.Meta(ctx)
}

func (p *PermissionWrapperDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	canRead := false
	if p.signer != nil {
		canRead = p.signer.CheckSignature(path)
	}
	var permission = types.PermissionRead
	if !canRead {
		var e error
		permission, e = p.requirePermission(path, types.PermissionRead)
		if e != nil {
			return nil, e
		}
	}
	entry, e := p.drive.Get(ctx, path)
	if e != nil {
		return nil, e
	}
	ak := ""
	if p.signer != nil {
		ak = p.signer.GetSignature(path, time.Now().Add(accessKeyValidity))
	}
	return &permissionWrapperEntry{
		p:          p,
		IEntry:     entry,
		permission: permission,
		accessKey:  ak,
	}, nil
}

func (p *PermissionWrapperDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.Save(ctx, path, size, override, reader)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.MakeDir(ctx, path)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	toPermission, e := p.requirePathAndParentWritable(to)
	if e != nil {
		return nil, e
	}
	if e := p.requireDescendantPermission(to, types.PermissionReadWrite); e != nil {
		return nil, e
	}
	entry, e := p.drive.Copy(ctx, from, to, override)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	toPermission, e := p.requirePathAndParentWritable(to)
	if e != nil {
		return nil, e
	}
	if _, e := p.requirePathAndParentWritable(from.Path()); e != nil {
		return nil, e
	}
	if e := p.requireDescendantPermission(from.Path(), types.PermissionReadWrite); e != nil {
		return nil, e
	}
	if e := p.requireDescendantPermission(to, types.PermissionReadWrite); e != nil {
		return nil, e
	}
	entry, e := p.drive.Move(ctx, from, to, override)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	permission := p.pm.ResolvePath(path)
	// root path always can be read, but items cannot be read will be filtered
	if !utils.IsRootPath(path) && !permission.Readable() {
		return nil, err.NewNotFoundError()
	}
	entries, e := p.drive.List(ctx, path)
	if e != nil {
		return nil, e
	}

	result := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Meta().Readable {
			continue
		}
		per := p.pm.ResolvePath(e.Path())
		if per.Readable() {
			accessKey := ""
			if p.signer != nil {
				accessKey = p.signer.GetSignature(e.Path(), time.Now().Add(accessKeyValidity))
			}
			result = append(
				result,
				&permissionWrapperEntry{
					p:          p,
					IEntry:     e,
					permission: per,
					accessKey:  accessKey,
				},
			)
		}
	}
	return result, nil
}

func (p *PermissionWrapperDrive) Delete(ctx types.TaskCtx, path string) error {
	if _, e := p.requirePathAndParentWritable(path); e != nil {
		return e
	}
	return p.drive.Delete(ctx, path)
}

func (p *PermissionWrapperDrive) Upload(ctx context.Context, path string, size int64,
	override bool, config types.SM) (*types.DriveUploadConfig, error) {
	_, e := p.requirePermission(path, types.PermissionReadWrite)
	if e != nil {
		return nil, e
	}
	return p.drive.Upload(ctx, path, size, override, config)
}

func (p *PermissionWrapperDrive) requirePathAndParentWritable(path string) (types.Permission, error) {
	if !utils.IsRootPath(path) {
		perm, e := p.requirePermission(utils.PathParent(path), types.PermissionReadWrite)
		if e != nil {
			return perm, e
		}
	}
	return p.requirePermission(path, types.PermissionReadWrite)
}

func (p *PermissionWrapperDrive) requirePermission(path string, require types.Permission) (types.Permission, error) {
	resolved := p.pm.ResolvePath(path)
	if resolved&require != require {
		return resolved, err.NewNotFoundMessageError(i18n.T("error.permission_denied"))
	}
	return resolved, nil
}

func (p *PermissionWrapperDrive) requireDescendantPermission(path string, require types.Permission) error {
	pp := p.pm.ResolvePath(path)
	ok := pp&require == require
	if ok {
		permission := p.pm.ResolveDescendant(path)
		for _, p := range permission {
			if p&require != require {
				ok = false
				break
			}
		}
	}
	if !ok {
		return err.NewNotAllowedMessageError(i18n.T("api.permission_wrapper.no_subfolder_permission"))
	}
	return nil
}

type permissionWrapperEntry struct {
	types.IEntry
	p          *PermissionWrapperDrive
	permission types.Permission
	accessKey  string
}

func (p *permissionWrapperEntry) Meta() types.EntryMeta {
	meta := p.IEntry.Meta()
	meta.Readable = meta.Readable && p.permission.Readable()
	meta.Writable = meta.Writable && p.permission.Writable()
	if p.accessKey != "" {
		meta.Props = utils.CopyMap(meta.Props, nil)
		meta.Props["accessKey"] = p.accessKey
	}
	return meta
}

func (p *permissionWrapperEntry) Drive() types.IDrive {
	return p.p
}

func (p *permissionWrapperEntry) GetIEntry() types.IEntry {
	return p.IEntry
}
