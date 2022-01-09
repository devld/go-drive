package drive

import (
	"go-drive/common/event"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"net/http"
	"sync"
)

type Access struct {
	rootDrive *RootDrive

	perms         utils.PermMap
	permMux       *sync.Mutex
	permissionDAO *storage.PathPermissionDAO

	signer *utils.Signer

	ch  *registry.ComponentsHolder
	bus event.Bus
}

func NewAccess(ch *registry.ComponentsHolder,
	rootDrive *RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	signer *utils.Signer, bus event.Bus) (*Access, error) {

	da := &Access{
		rootDrive:     rootDrive,
		permMux:       &sync.Mutex{},
		permissionDAO: permissionDAO,
		signer:        signer,
		ch:            ch,
		bus:           bus,
	}
	if e := da.ReloadPerm(); e != nil {
		return nil, e
	}

	ch.Add("driveAccess", da)
	return da, nil
}

func (da *Access) GetDrive(req *http.Request, session types.Session) types.IDrive {
	return NewListenerWrapper(
		NewPermissionWrapperDrive(
			req,
			session,
			da.rootDrive.Get(),
			da.perms,
			da.signer,
		),
		types.DriveListenerContext{
			Request: req,
			Session: session,
		},
		da.bus,
	)
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
