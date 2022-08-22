package drive

import (
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"sync"
	"time"
)

const (
	anonymousRootPathKey = "anonymous.rootPath"
)

type Access struct {
	rootDrive *RootDrive

	perms         utils.PermMap
	permMux       *sync.Mutex
	permissionDAO *storage.PathPermissionDAO
	options       *storage.OptionsDAO

	ch  *registry.ComponentsHolder
	bus event.Bus
}

func NewAccess(ch *registry.ComponentsHolder,
	rootDrive *RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	options *storage.OptionsDAO, bus event.Bus) (*Access, error) {

	da := &Access{
		rootDrive:     rootDrive,
		permMux:       &sync.Mutex{},
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

func (da *Access) GetDrive(session types.Session, signer EntrySigner) (types.IDrive, error) {
	chroot, e := da.GetChroot(session)
	if e != nil {
		return nil, e
	}

	if signer != nil && chroot != nil {
		signer = &chrootEntrySigner{signer, chroot}
	}

	var drive types.IDrive = NewPermissionWrapperDrive(da.rootDrive.Get(), da.perms.Filter(session), signer)
	if chroot != nil {
		drive = NewChrootWrapper(drive, chroot)
	}

	return NewListenerWrapper(
		drive,
		types.DriveListenerContext{
			Session: session,
		},
		da.bus,
	), nil
}

func (da *Access) GetRootDrive() types.IDrive {
	return da.rootDrive.Get()
}

func (da *Access) GetPerms() utils.PermMap {
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

type chrootEntrySigner struct {
	signer EntrySigner
	chroot *Chroot
}

func (es *chrootEntrySigner) GetSignature(path string, notAfter time.Time) string {
	return es.signer.GetSignature(es.chroot.UnwrapPath(path), notAfter)
}

func (es *chrootEntrySigner) CheckSignature(path string) bool {
	return es.signer.CheckSignature(es.chroot.UnwrapPath(path))
}
