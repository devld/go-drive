package drive

import (
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

	ch           *registry.ComponentsHolder
	listeners    []types.IDriveListener
	listenersMux *sync.Mutex
}

func NewAccess(ch *registry.ComponentsHolder,
	rootDrive *RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	signer *utils.Signer) (*Access, error) {

	da := &Access{
		rootDrive:     rootDrive,
		permMux:       &sync.Mutex{},
		permissionDAO: permissionDAO,
		signer:        signer,
		ch:            ch,
		listenersMux:  &sync.Mutex{},
	}
	if e := da.ReloadPerm(); e != nil {
		return nil, e
	}

	ch.Add("driveAccess", da)
	return da, nil
}

func (da *Access) GetDrive(req *http.Request, session types.Session) types.IDrive {
	if da.listeners == nil {
		func() {
			da.listenersMux.Lock()
			defer da.listenersMux.Unlock()
			if da.listeners == nil {
				listeners := make([]types.IDriveListener, 0)
				listenerObjects := da.ch.Gets(func(c interface{}) bool {
					_, ok := c.(types.IDriveListener)
					return ok
				})
				for _, listener := range listenerObjects {
					listeners = append(listeners, listener.(types.IDriveListener))
				}
				da.listeners = listeners
			}
		}()
	}

	return NewListenerWrapper(
		NewPermissionWrapperDrive(
			req,
			da.rootDrive.Get(),
			da.perms.Filter(session),
			da.signer,
		),
		types.DriveListenerContext{
			Request: req,
			Session: session,
		},
		da.listeners,
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
