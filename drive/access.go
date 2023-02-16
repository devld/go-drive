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

	options *storage.OptionsDAO

	ch  *registry.ComponentsHolder
	bus event.Bus
}

func NewAccess(ch *registry.ComponentsHolder,
	rootDrive *RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	options *storage.OptionsDAO, bus event.Bus) (*Access, error) {

	da := &Access{
		rootDrive:     rootDrive,
		permMux:       &sync.RWMutex{},
		permissionDAO: permissionDAO,
		options:       options,
		ch:            ch,
		bus:           bus,
	}
	if e := da.ReloadPerm(); e != nil {
		return nil, e
	}

	ch.Add("driveAccess", da)
	return da, nil
}

func (da *Access) GetChroot(s types.Session) (*Chroot, error) {
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
		rootPath = s.User.RootPath
	}
	if rootPath == "" {
		return nil, nil
	}
	return NewChroot(rootPath, nil), nil
}

func (da *Access) GetDrive(session types.Session) (types.IDrive, error) {
	chroot, e := da.GetChroot(session)
	if e != nil {
		return nil, e
	}

	da.permMux.RLock()
	perms := da.perms
	da.permMux.RUnlock()

	var drive types.IDrive = NewListenerWrapper(
		NewPermissionWrapperDrive(da.rootDrive.Get(), perms.Filter(session)),
		types.DriveListenerContext{
			Session: &session,
			Drive:   da.rootDrive.Get(),
		},
		da.bus,
	)
	if chroot != nil {
		drive = NewChrootWrapper(drive, chroot)
	}

	return drive, nil
}

func (da *Access) GetRootDrive() types.IDrive {
	return NewListenerWrapper(da.rootDrive.Get(), types.DriveListenerContext{
		Drive: da.rootDrive.Get(),
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
