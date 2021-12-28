package server

import (
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"net/http"
	"sync"
)

type driveAccess struct {
	rootDrive *drive.RootDrive

	perms         utils.PermMap
	permMux       *sync.Mutex
	permissionDAO *storage.PathPermissionDAO

	signer *utils.Signer

	ch           *registry.ComponentsHolder
	listeners    []types.IDriveListener
	listenersMux *sync.Mutex
}

func newDriveAccess(ch *registry.ComponentsHolder,
	rootDrive *drive.RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	signer *utils.Signer) (*driveAccess, error) {

	da := &driveAccess{
		rootDrive:     rootDrive,
		permMux:       &sync.Mutex{},
		permissionDAO: permissionDAO,
		signer:        signer,
		ch:            ch,
		listenersMux:  &sync.Mutex{},
	}
	if e := da.reloadPerm(); e != nil {
		return nil, e
	}

	ch.Add("driveAccess", da)
	return da, nil
}

func (da *driveAccess) getDrive(req *http.Request, session types.Session) types.IDrive {
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

	return NewDriveListenerWrapper(
		NewPermissionWrapperDrive(
			req, session,
			da.rootDrive.Get(),
			da.perms,
			da.signer,
		),
		types.DriveListenerContext{
			Request: req,
			Session: session,
		},
		da.listeners,
	)
}

func (da *driveAccess) reloadPerm() error {
	da.permMux.Lock()
	defer da.permMux.Unlock()
	all, e := da.permissionDAO.GetAll()
	if e != nil {
		return e
	}
	da.perms = utils.NewPermMap(all)
	return nil
}
