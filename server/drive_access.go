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
	}
	if e := da.reloadPerm(); e != nil {
		return nil, e
	}

	ch.Add("driveAccess", da)
	return da, nil
}

func (da *driveAccess) getDrive(req *http.Request, session types.Session) types.IDrive {
	return NewPermissionWrapperDrive(
		req, session,
		da.rootDrive.Get(),
		da.perms,
		da.signer,
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
