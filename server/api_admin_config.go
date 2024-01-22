package server

import (
	"go-drive/common/event"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	path2 "path"
	"strings"

	"github.com/gin-gonic/gin"
)

type configRoute struct {
	access        *drive.Access
	permissionDAO *storage.PathPermissionDAO
	pathMetaDAO   *storage.PathMetaDAO
	pathMountDAO  *storage.PathMountDAO
	optionsDAO    *storage.OptionsDAO
	rootDrive     *drive.RootDrive
	bus           event.Bus
}

func (cr *configRoute) getPathPermissions(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	permissions, e := cr.permissionDAO.GetByPath(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, permissions)
}

func (cr *configRoute) savePathPermissions(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	permissions := make([]types.PathPermission, 0)
	if e := c.Bind(&permissions); e != nil {
		_ = c.Error(e)
		return
	}
	if e := cr.permissionDAO.SavePathPermissions(path, permissions); e != nil {
		_ = c.Error(e)
		return
	}
	// permissions updated
	if e := cr.access.ReloadPerm(); e != nil {
		_ = c.Error(e)
		return
	}
}

func (cr *configRoute) getAllPathMeta(c *gin.Context) {
	res, e := cr.pathMetaDAO.GetAll()
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, res)
}

func (cr *configRoute) savePathMeta(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	data := types.PathMeta{}
	if e := c.Bind(&data); e != nil {
		_ = c.Error(e)
		return
	}
	data.Path = &path
	if e := cr.pathMetaDAO.Set(data); e != nil {
		_ = c.Error(e)
		return
	}
}

func (cr *configRoute) deletePathMeta(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	if e := cr.pathMetaDAO.Delete(path); e != nil {
		_ = c.Error(e)
		return
	}
}

func (cr *configRoute) savePathMounts(c *gin.Context) {
	s := GetSession(c)
	to := utils.CleanPath(c.Param("to"))
	src := make([]mountSource, 0)
	if e := c.Bind(&src); e != nil {
		_ = c.Error(e)
		return
	}
	if len(src) == 0 {
		return
	}

	dd := cr.rootDrive.Get()

	var e error
	mounts := make([]types.PathMount, len(src))
	for i, p := range src {
		mountPath := utils.CleanPath(path2.Join(to, p.Name))
		mountPath, e = dd.FindNonExistsEntryName(c.Request.Context(), dd, mountPath)
		if e != nil {
			_ = c.Error(e)
			return
		}
		mounts[i] = types.PathMount{Path: &to, Name: utils.PathBase(mountPath), MountAt: p.Path}
	}
	if e := cr.pathMountDAO.SaveMounts(mounts, true); e != nil {
		_ = c.Error(e)
		return
	}
	_ = cr.rootDrive.ReloadMounts()
	for _, m := range mounts {
		cr.bus.Publish(event.EntryUpdated, types.DriveListenerContext{
			Session: &s,
			Drive:   cr.rootDrive.Get(),
		}, path2.Join(*m.Path, m.Name), true)
	}
}

func (cr *configRoute) saveOptions(c *gin.Context) {
	options := make(map[string]string)
	if e := c.Bind(&options); e != nil {
		_ = c.Error(e)
		return
	}
	e := cr.optionsDAO.Sets(options)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (cr *configRoute) getOptions(c *gin.Context) {
	keys := strings.Split(c.Param("keys"), ",")
	value, e := cr.optionsDAO.Gets(keys...)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, value)
}

type fileBucketConfigRoute struct {
	fileBucketDAO *storage.FileBucketDAO
}

func (fbr *fileBucketConfigRoute) getAllBuckets(c *gin.Context) {
	buckets, e := fbr.fileBucketDAO.GetBuckets()
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, buckets)
}

func (fbr *fileBucketConfigRoute) createBucket(c *gin.Context) {
	bucket := types.FileBucket{}
	var e error
	if e = c.Bind(&bucket); e != nil {
		_ = c.Error(e)
		return
	}
	if e := CheckPathSegment(bucket.Name, "api.admin.invalid_file_bucket_name"); e != nil {
		_ = c.Error(e)
		return
	}
	if bucket, e = fbr.fileBucketDAO.AddBucket(bucket); e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, bucket)
}

func (fbr *fileBucketConfigRoute) updateBucket(c *gin.Context) {
	name := c.Param("name")
	bucket := types.FileBucket{}
	if e := c.Bind(&bucket); e != nil {
		_ = c.Error(e)
		return
	}
	if e := fbr.fileBucketDAO.UpdateBucket(name, bucket); e != nil {
		_ = c.Error(e)
		return
	}
}

func (fbr *fileBucketConfigRoute) deleteBucket(c *gin.Context) {
	name := c.Param("name")
	if e := fbr.fileBucketDAO.DeleteBucket(name); e != nil {
		_ = c.Error(e)
		return
	}
}

type mountSource struct {
	Path string `json:"path" binding:"required"`
	Name string `json:"name" binding:"required"`
}
