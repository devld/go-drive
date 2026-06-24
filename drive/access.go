package drive

import (
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"sync"
)

const (
	anonymousRootPathKey = "anonymous.rootPath"
)

type Access struct {
	rootDrive *RootDrive

	perms         utils.PermMap
	permMux       *sync.RWMutex
	permissionDAO *storage.PathPermissionDAO

	options  *storage.OptionsDAO
	pathMeta *storage.PathMetaDAO

	ch  *registry.ComponentsHolder
	bus event.Bus
}

func NewAccess(ch *registry.ComponentsHolder,
	rootDrive *RootDrive, permissionDAO *storage.PathPermissionDAO,
	options *storage.OptionsDAO, pathMeta *storage.PathMetaDAO,
	bus event.Bus) (*Access, error) {

	da := &Access{
		rootDrive:     rootDrive,
		permMux:       &sync.RWMutex{},
		permissionDAO: permissionDAO,
		options:       options,
		pathMeta:      pathMeta,
		ch:            ch,
		bus:           bus,
	}
	if e := da.ReloadPerm(); e != nil {
		return nil, e
	}

	ch.Add(registry.KeyDriveAccess, da)
	return da, nil
}

func (da *Access) GetChroot(s types.Principal) (*Chroot, error) {
	if s.HasUserGroup(types.AdminUserGroup) {
		return nil, nil
	}
	rootPath := ""
	if s.IsAnonymous() {
		p, e := da.options.Get(anonymousRootPathKey)
		if e != nil {
			return nil, e
		}
		rootPath = p
	} else {
		rootPath = resolveUserRootPath(s.User)
	}
	if rootPath == "" {
		return nil, nil
	}
	return NewChroot(rootPath, nil), nil
}

func resolveUserRootPath(user types.User) string {
	if user.RootPath != "" {
		return user.RootPath
	}
	best := ""
	bestDepth := -1
	for _, g := range user.Groups {
		if g.RootPath == "" {
			continue
		}
		if d := utils.PathDepth(g.RootPath); bestDepth == -1 || d < bestDepth {
			best = g.RootPath
			bestDepth = d
		}
	}
	return best
}

func (da *Access) GetDrive(session types.Principal) (types.IDrive, error) {
	chroot, e := da.GetChroot(session)
	if e != nil {
		return nil, e
	}

	da.permMux.RLock()
	perms := da.perms
	da.permMux.RUnlock()

	var drive types.IDrive = NewPathMetaWrapper(
		NewPermissionWrapperDrive(da.GetRootDrive(&session), perms.Filter(session)),
		da.pathMeta, session,
	)
	if chroot != nil {
		drive = NewChrootWrapper(drive, chroot)
	}

	return drive, nil
}

func (da *Access) GetRootDrive(session *types.Principal) types.IDrive {
	return NewListenerWrapper(da.rootDrive.Get(), types.DriveListenerContext{
		Principal: session,
		Drive:     da.rootDrive.Get(),
	}, da.bus)
}

func (da *Access) GetPerms() utils.PermMap {
	da.permMux.RLock()
	defer da.permMux.RUnlock()
	return da.perms
}

func (da *Access) ReloadPerm() error {
	da.permMux.Lock()
	defer da.permMux.Unlock()
	all, e := da.permissionDAO.GetAll()
	if e != nil {
		return e
	}
	da.perms = utils.NewPermMap(all)
	return nil
}
