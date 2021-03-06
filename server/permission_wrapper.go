package server

import (
	"context"
	"fmt"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	accessKeyValidity = 12 * time.Hour
)

// PermissionWrapperDrive intercept the request
// based on the permission information in the database.
// The permissions of a child path inherit from the parent path,
// but have priority over the permissions of the parent path.
// Permissions for users take precedence over permissions for user groups.
// REJECT takes precedence over ACCEPT
type PermissionWrapperDrive struct {
	drive   types.IDrive
	request *http.Request
	pm      permMap
	signer  *utils.Signer
}

func NewPermissionWrapperDrive(
	request *http.Request, session types.Session, drive types.IDrive,
	permissions permMap, signer *utils.Signer) *PermissionWrapperDrive {

	return &PermissionWrapperDrive{
		drive:   drive,
		request: request,
		pm:      permissions.filter(makeSubjects(session)),
		signer:  signer,
	}
}

func (p *PermissionWrapperDrive) Meta(ctx context.Context) types.DriveMeta {
	return p.drive.Meta(ctx)
}

func (p *PermissionWrapperDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	canRead := false
	if p.signer != nil {
		canRead = checkSignature(p.signer, p.request, path)
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
		ak = signPathRequest(p.signer, p.request, path, time.Now().Add(accessKeyValidity))
	}
	return &permissionWrapperEntry{
		p:          p,
		entry:      entry,
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
	return &permissionWrapperEntry{p: p, entry: entry, permission: permission}, nil
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
	return &permissionWrapperEntry{p: p, entry: entry, permission: permission}, nil
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
	return &permissionWrapperEntry{p: p, entry: entry, permission: toPermission}, nil
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
	return &permissionWrapperEntry{p: p, entry: entry, permission: toPermission}, nil
}

func (p *PermissionWrapperDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	permission := p.pm.resolvePath(path)
	// root path always can be read, but items cannot be read will be filtered
	if !utils.IsRootPath(path) && !permission.CanRead() {
		return nil, err.NewNotFoundError()
	}
	entries, e := p.drive.List(ctx, path)
	if e != nil {
		return nil, e
	}

	result := make([]types.IEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Meta().CanRead {
			continue
		}
		per := p.pm.resolvePath(e.Path())
		if per.CanRead() {
			accessKey := ""
			if e.Type().IsFile() && p.signer != nil {
				accessKey = signPathRequest(p.signer, p.request, e.Path(), time.Now().Add(accessKeyValidity))
			}
			result = append(
				result,
				&permissionWrapperEntry{
					p:          p,
					entry:      e,
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
	resolved := p.pm.resolvePath(path)
	if resolved&require != require {
		return resolved, err.NewNotFoundMessageError(i18n.T("error.permission_denied"))
	}
	return resolved, nil
}

func (p *PermissionWrapperDrive) requireDescendantPermission(path string, require types.Permission) error {
	pp := p.pm.resolvePath(path)
	ok := pp&require == require
	if ok {
		permission := p.pm.resolveDescendant(path)
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

func makeSubjects(session types.Session) []string {
	subjects := make([]string, 0, 3)
	subjects = append(subjects, types.AnySubject) // Anonymous
	if !session.IsAnonymous() {
		subjects = append(subjects, types.UserSubject(session.User.Username))
		if session.User.Groups != nil {
			for _, g := range session.User.Groups {
				subjects = append(subjects, types.GroupSubject(g.Name))
			}
		}
	}
	return subjects
}

// permMap[subject][path]
type permMap map[string]map[string]*pathPermItem

func newPermMap(permissions []types.PathPermission) permMap {
	result := make(permMap)
	for _, p := range permissions {
		sp, ok := result[p.Subject]
		if !ok {
			sp = make(map[string]*pathPermItem)
			result[p.Subject] = sp
		}
		sp[*p.Path] = &pathPermItem{
			PathPermission: p,
			depth:          int8(utils.PathDepth(*p.Path)),
			descendant:     make([]*pathPermItem, 0),
		}
	}
	for _, pms := range result {
		added := make(map[string]bool)
		for p := range pms {
			paths := utils.PathParentTree(p)
			for _, pathPart := range paths {
				if _, ok := pms[pathPart]; !ok {
					added[pathPart] = true
				}
			}
		}
		for p := range added {
			// virtual path node helps finding path descendant
			pms[p] = &pathPermItem{
				PathPermission: types.PathPermission{Path: &p},
				depth:          -1,
				descendant:     make([]*pathPermItem, 0),
			}
		}
	}
	for _, sp := range result {
		for _, p := range sp {
			for _, c := range sp {
				if c.depth >= 0 && p.depth < c.depth && strings.HasPrefix(*c.Path, *p.Path) {
					p.descendant = append(p.descendant, c)
				}
			}
		}
	}
	return result
}

func (pm permMap) filter(subjects []string) permMap {
	result := make(permMap, len(subjects))
	for _, s := range subjects {
		if sp, ok := pm[s]; ok {
			result[s] = sp
		}
	}
	return result
}

// resolvePath resolves permission of the path
func (pm permMap) resolvePath(path string) types.Permission {
	paths := utils.PathParentTree(path)
	items := make([]*pathPermItem, 0)
	for _, p := range pm {
		for _, pathPart := range paths {
			if temp, ok := p[pathPart]; ok && temp.depth >= 0 {
				items = append(items, temp)
			}
		}
	}
	return resolveAcceptedPermissions(items)
}

// resolveDescendant resolves permissions of the path's descendant
func (pm permMap) resolveDescendant(path string) map[string]types.Permission {
	result := make(map[string]types.Permission)
	for _, p := range pm {
		if item, ok := p[path]; ok {
			for _, t := range item.descendant {
				result[*item.Path] = resolveAcceptedPermissions(t.descendant)
			}
		}
	}
	return result
}

type pathPermItem struct {
	types.PathPermission
	// if depth == -1, this node is a virtual node that holds descendant
	depth      int8
	descendant []*pathPermItem
}

func (p pathPermItem) String() string {
	return fmt.Sprintf("%s,%s,%d,%d (%v)", *p.Path, p.Subject, p.Permission, p.Policy, p.descendant)
}

func resolveAcceptedPermissions(items []*pathPermItem) types.Permission {
	sort.Slice(items, func(i, j int) bool { return pathPermissionLess(items[i], items[j]) })
	acceptedPermission := types.PermissionEmpty
	rejectedPermission := types.PermissionEmpty
	for _, item := range items {
		if item.IsAccept() {
			acceptedPermission |= item.Permission & ^rejectedPermission
		}
		if item.IsReject() {
			// acceptedPermission - ( item.Permission(reject) - acceptedPermission )
			acceptedPermission &= ^(item.Permission & (^acceptedPermission))
			rejectedPermission |= item.Permission
		}
	}
	return acceptedPermission
}

func pathPermissionLess(a, b *pathPermItem) bool {
	if a.depth != b.depth {
		return a.depth > b.depth
	}
	if a.IsForAnonymous() {
		if b.IsForAnonymous() {
			return a.Policy < b.Policy
		} else {
			return false
		}
	} else {
		if b.IsForAnonymous() {
			return true
		} else {
			if a.IsForUser() {
				if b.IsForUser() {
					return a.Policy < b.Policy
				} else {
					return true
				}
			} else {
				if b.IsForUser() {
					return false
				} else {
					return a.Policy < b.Policy
				}
			}
		}
	}
}

type permissionWrapperEntry struct {
	p          *PermissionWrapperDrive
	entry      types.IEntry
	permission types.Permission
	accessKey  string
}

func (p *permissionWrapperEntry) Path() string {
	return p.entry.Path()
}

func (p *permissionWrapperEntry) Type() types.EntryType {
	return p.entry.Type()
}

func (p *permissionWrapperEntry) Size() int64 {
	return p.entry.Size()
}

func (p *permissionWrapperEntry) Meta() types.EntryMeta {
	meta := p.entry.Meta()
	meta.CanRead = meta.CanRead && p.permission.CanRead()
	meta.CanWrite = meta.CanWrite && p.permission.CanWrite()
	if p.accessKey != "" {
		meta.Props = utils.CopyMap(meta.Props)
		meta.Props["access_key"] = p.accessKey
	}
	return meta
}

func (p *permissionWrapperEntry) ModTime() int64 {
	return p.entry.ModTime()
}

func (p *permissionWrapperEntry) Drive() types.IDrive {
	return p.p
}

func (p *permissionWrapperEntry) Name() string {
	return utils.PathBase(p.entry.Path())
}

func (p *permissionWrapperEntry) GetReader(ctx context.Context) (io.ReadCloser, error) {
	if c, ok := p.entry.(types.IContent); ok {
		return c.GetReader(ctx)
	}
	return nil, err.NewUnsupportedError()
}

func (p *permissionWrapperEntry) GetURL(ctx context.Context) (*types.ContentURL, error) {
	if c, ok := p.entry.(types.IContent); ok {
		return c.GetURL(ctx)
	}
	return &types.ContentURL{}, err.NewUnsupportedError()
}

func (p *permissionWrapperEntry) GetIEntry() types.IEntry {
	return p.entry
}
