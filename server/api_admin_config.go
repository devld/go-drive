package server

import (
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	path2 "path"
	"strings"
	"sync"

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
	mountMux      sync.Mutex
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
	principal := GetPrincipal(c)
	to := utils.CleanPath(c.Param("to"))
	src := make([]mountSource, 0)
	if e := c.Bind(&src); e != nil {
		_ = c.Error(e)
		return
	}
	if len(src) == 0 {
		_ = c.Error(err.NewBadRequestError(i18n.T("drive.invalid_path")))
		return
	}
	cr.mountMux.Lock()
	defer cr.mountMux.Unlock()

	dd := cr.rootDrive.Get()

	existingMounts, e := cr.pathMountDAO.GetMounts()
	if e != nil {
		_ = c.Error(e)
		return
	}
	mounts := make([]types.PathMount, len(src))
	reserved := make(map[string]bool, len(existingMounts)+len(src))
	for _, mount := range existingMounts {
		reserved[path2.Join(*mount.Path, mount.Name)] = true
	}
	for i, p := range src {
		mountAt := utils.CleanPath(p.Path)
		name := utils.CleanPath(p.Name)
		if mountAt == "" || name == "" || name != p.Name || utils.PathBase(name) != name {
			_ = c.Error(err.NewBadRequestError(i18n.T("drive.invalid_path")))
			return
		}
		mountPath := path2.Join(to, name)
		mountPath, e = driveutil.FindNonExistsEntryNameWithReserved(c.Request.Context(), dd, mountPath, reserved)
		if e != nil {
			_ = c.Error(e)
			return
		}
		reserved[mountPath] = true
		mounts[i] = types.PathMount{Path: &to, Name: utils.PathBase(mountPath), MountAt: mountAt}
	}
	if e := cr.pathMountDAO.SaveMounts(mounts, false); e != nil {
		_ = c.Error(e)
		return
	}
	if e := cr.rootDrive.ReloadMounts(); e != nil {
		_ = c.Error(e)
		return
	}
	for _, m := range mounts {
		cr.bus.PublishEntryUpdated(types.DriveListenerContext{
			Principal: &principal,
			Drive:     cr.rootDrive.Get(),
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
