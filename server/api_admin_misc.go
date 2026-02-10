package server

import (
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/search"
	"go-drive/storage"
	"sort"

	"github.com/gin-gonic/gin"
)

type miscRoute struct {
	access        *drive.Access
	permissionDAO *storage.PathPermissionDAO
	pathMountDAO  *storage.PathMountDAO
	rootDrive     *drive.RootDrive
	search        *search.Service
	ch            *registry.ComponentsHolder
}

func (mr *miscRoute) updateSearcherIndexes(c *gin.Context) {
	root := utils.CleanPath(c.Param("path"))
	t, e := mr.search.TriggerIndexAll(root, true)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (mr *miscRoute) cleanupInvalidPathPermissionsAndMounts(c *gin.Context) {
	root := mr.rootDrive.Get()
	pps, e := mr.permissionDAO.GetAll()
	if e != nil {
		_ = c.Error(e)
		return
	}
	ms, e := mr.pathMountDAO.GetMounts()
	if e != nil {
		_ = c.Error(e)
		return
	}
	paths := make(map[string]bool)
	var reloadPermission, reloadMount bool
	for _, p := range pps {
		paths[*p.Path] = true
		reloadPermission = true
	}
	for _, m := range ms {
		paths[m.MountAt] = true
		reloadMount = true
	}
	for p := range paths {
		_, e := root.Get(c.Request.Context(), p)
		if e != nil {
			if err.IsNotFoundError(e) {
				paths[p] = false
				continue
			}
			_ = c.Error(e)
			return
		}
	}
	n := 0
	for p, ok := range paths {
		if ok {
			continue
		}
		if e := mr.permissionDAO.DeleteByPath(p); e != nil {
			_ = c.Error(e)
			return
		}
		if e := mr.pathMountDAO.DeleteByMountAt(p); e != nil {
			_ = c.Error(e)
			return
		}
		n++
	}
	if reloadMount {
		_ = mr.rootDrive.ReloadMounts()
	}
	if reloadPermission {
		_ = mr.access.ReloadPerm()
	}
	SetResult(c, n)
}

func (mr *miscRoute) getSystemStats(c *gin.Context) {
	stats := mr.ch.Gets(func(c any) bool {
		_, ok := c.(types.IStatistics)
		return ok
	})
	res := make([]statItem, len(stats))
	for i, s := range stats {
		name, data, e := s.(types.IStatistics).Status()
		if e != nil {
			_ = c.Error(e)
			return
		}
		res[i] = statItem{Name: name, Data: data}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	SetResult(c, res)
}

func (mr *miscRoute) clearDriveCache(c *gin.Context) {
	name := c.Param("name")
	e := mr.rootDrive.ClearDriveCache(name)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

type statItem struct {
	Name string   `json:"name"`
	Data types.SM `json:"data"`
}
