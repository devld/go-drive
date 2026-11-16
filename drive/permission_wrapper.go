package drive

import (
	"context"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
)

var _ types.IDrive = (*PermissionWrapperDrive)(nil)

// PermissionWrapperDrive intercept the request
// based on the permission information in the database.
// The permissions of a child path inherit from the parent path,
// but have priority over the permissions of the parent path.
// Permissions for users take precedence over permissions for user groups.
// REJECT takes precedence over ACCEPT
type PermissionWrapperDrive struct {
	drive types.IDrive
	pm    utils.PermMap
}

func NewPermissionWrapperDrive(drive types.IDrive, permissions utils.PermMap) *PermissionWrapperDrive {
	return &PermissionWrapperDrive{
		drive: drive,
		pm:    permissions,
	}
}

func (p *PermissionWrapperDrive) Meta(ctx context.Context) (types.DriveMeta, error) {
	return p.drive.Meta(ctx)
}

func (p *PermissionWrapperDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	permission, e := p.requirePermission(path, types.PermissionRead)
	if e != nil {
		return nil, e
	}
	entry, e := p.drive.Get(ctx, path)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{
		p:          p,
		IEntry:     entry,
		permission: permission,
	}, nil
}

func (p *PermissionWrapperDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	if _, e := p.PrepareWrite(ctx, path, driveutil.WriteModeForOverride(override)); e != nil {
		return nil, e
	}
	permission := p.pm.ResolvePath(path)

	entry, e := p.drive.Save(ctx, path, size, override, reader)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	if _, e := p.PrepareWrite(ctx, path, types.WriteOverwrite); e != nil {
		return nil, e
	}
	permission := p.pm.ResolvePath(path)

	entry, e := p.drive.MakeDir(ctx, path)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: permission}, nil
}

func (p *PermissionWrapperDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	if _, e := p.PrepareWrite(ctx, to, driveutil.WriteModeForOverride(override)); e != nil {
		return nil, e
	}
	toPermission, e := p.requirePathAndParentWritable(to)
	if e != nil {
		return nil, e
	}
	if e := p.requireDescendantPermission(from.Path(), types.PermissionRead); e != nil {
		return nil, e
	}
	entry, e := p.drive.Copy(ctx, from, to, override)
	if e != nil {
		return nil, e
	}
	return &permissionWrapperEntry{p: p, IEntry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	if _, e := p.PrepareWrite(ctx, to, driveutil.WriteModeForOverride(override)); e != nil {
		return nil, e
	}
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
			result = append(
				result,
				&permissionWrapperEntry{p: p, IEntry: e, permission: per},
			)
		}
	}
	return result, nil
}

func (p *PermissionWrapperDrive) Delete(ctx types.TaskCtx, path string) error {
	if _, e := p.requirePathAndParentWritable(path); e != nil {
		return e
	}
	if e := p.requireDescendantPermission(path, types.PermissionReadWrite); e != nil {
		return e
	}
	return p.drive.Delete(ctx, path)
}

func (p *PermissionWrapperDrive) Upload(ctx context.Context, path string, size int64,
	override bool, config types.SM) (*types.DriveUploadConfig, error) {
	if _, e := p.PrepareWrite(ctx, path, driveutil.WriteModeForOverride(override)); e != nil {
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
		return resolved, err.NewPermissionDeniedNotFoundError(i18n.T("error.permission_denied"))
	}
	return resolved, nil
}

func (p *PermissionWrapperDrive) requireDescendantPermission(path string, require types.Permission) error {
	pp := p.pm.ResolvePath(path)
	dp, defined := p.pm.ResolveDescendant(path)
	ok := pp&require == require && (!defined || dp&require == require)
	if !ok {
		return err.NewNotAllowedMessageError(i18n.T("api.permission_wrapper.no_subfolder_permission"))
	}
	return nil
}

var _ types.IEntryWrapper = (*permissionWrapperEntry)(nil)

type permissionWrapperEntry struct {
	types.IEntry
	p          *PermissionWrapperDrive
	permission types.Permission
}

func (p *permissionWrapperEntry) Meta() types.EntryMeta {
	meta := p.IEntry.Meta()
	meta.Readable = meta.Readable && p.permission.Readable()
	meta.Writable = meta.Writable && p.permission.Writable()
	return meta
}

func (p *permissionWrapperEntry) Drive() types.IDrive {
	return p.p
}

func (p *permissionWrapperEntry) GetIEntry() types.IEntry {
	return p.IEntry
}

// PrepareWrite authorizes the final virtual destination and every parent that
// execution would create. Storage routing and collision selection stay below it.
func (p *PermissionWrapperDrive) PrepareWrite(ctx context.Context, path string, mode types.WriteMode) (*types.WritePlan, error) {
	// Keep denied requests from probing raw storage existence through conflicts.
	if _, e := p.requirePermission(path, types.PermissionReadWrite); e != nil {
		return nil, e
	}
	plan, e := driveutil.PrepareWrite(ctx, p.drive, path, mode)
	if e != nil {
		return nil, e
	}
	if _, e := p.requirePermission(plan.Path, types.PermissionReadWrite); e != nil {
		return nil, e
	}
	if e := p.requireDescendantPermission(plan.Path, types.PermissionReadWrite); e != nil {
		return nil, e
	}
	for _, parent := range plan.Parents {
		if _, e := p.requirePermission(parent, types.PermissionReadWrite); e != nil {
			return nil, e
		}
	}
	return plan, nil
}
