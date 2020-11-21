package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/storage"
	"regexp"
)

func InitAdminRoutes(r gin.IRouter) {
	r = r.Group("/admin", Auth(), UserGroupRequired("admin"))

	// region user

	// list users
	r.GET("/users", func(c *gin.Context) {
		users, e := GetUserStorage(c).ListUser()
		if e != nil {
			_ = c.Error(e)
			return
		}
		for _, u := range users {
			u.Password = ""
		}
		SetResult(c, users)
	})

	// get user by username
	r.GET("/user/:username", func(c *gin.Context) {
		username := c.Param("username")
		user, e := GetUserStorage(c).GetUser(username)
		if e != nil {
			_ = c.Error(e)
			return
		}
		user.Password = ""
		SetResult(c, user)
	})

	// create user
	r.POST("/user", func(c *gin.Context) {
		user := types.User{}
		if e := c.Bind(&user); e != nil {
			_ = c.Error(e)
			return
		}
		addUser, e := GetUserStorage(c).AddUser(user)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, addUser)
	})

	// update user
	r.PUT("/user/:username", func(c *gin.Context) {
		user := types.User{}
		if e := c.Bind(&user); e != nil {
			_ = c.Error(e)
			return
		}
		username := c.Param("username")
		e := GetUserStorage(c).UpdateUser(username, user)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete user
	r.DELETE("/user/:username", func(c *gin.Context) {
		username := c.Param("username")
		e := GetUserStorage(c).DeleteUser(username)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// endregion

	// region group

	// list groups
	r.GET("/groups", func(c *gin.Context) {
		groups, e := GetGroupStorage(c).ListGroup()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, groups)
	})

	// get group and it's users
	r.GET("/group/:name", func(c *gin.Context) {
		name := c.Param("name")
		group, e := GetGroupStorage(c).GetGroup(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, group)
	})

	// create group
	r.POST("/group", func(c *gin.Context) {
		group := storage.GroupWithUsers{}
		if e := c.Bind(&group); e != nil {
			_ = c.Error(e)
			return
		}
		addGroup, e := GetGroupStorage(c).AddGroup(group)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, addGroup)
	})

	r.PUT("/group/:name", func(c *gin.Context) {
		name := c.Param("name")
		gus := storage.GroupWithUsers{}
		if e := c.Bind(&gus); e != nil {
			_ = c.Error(e)
			return
		}
		if e := GetGroupStorage(c).UpdateGroup(name, gus); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete group
	r.DELETE("/group/:name", func(c *gin.Context) {
		name := c.Param("name")
		e := GetGroupStorage(c).DeleteGroup(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})
	// endregion

	// region drive

	// get drives
	r.GET("/drives", func(c *gin.Context) {
		drives, e := GetDriveStorage(c).GetDrives()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, drives)
	})

	// add drive
	r.POST("/drive", func(c *gin.Context) {
		drive := types.Drive{}
		if e := c.Bind(&drive); e != nil {
			_ = c.Error(e)
			return
		}
		if e := checkDriveName(drive.Name); e != nil {
			_ = c.Error(e)
			return
		}
		drive, e := GetDriveStorage(c).AddDrive(drive)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, drive)
	})

	// update drive
	r.PUT("/drive/:name", func(c *gin.Context) {
		name := c.Param("name")
		if e := checkDriveName(name); e != nil {
			_ = c.Error(e)
			return
		}
		drive := types.Drive{}
		if e := c.Bind(&drive); e != nil {
			_ = c.Error(e)
			return
		}
		e := GetDriveStorage(c).UpdateDrive(name, drive)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete drive
	r.DELETE("/drive/:name", func(c *gin.Context) {
		name := c.Param("name")
		e := GetDriveStorage(c).DeleteDrive(name)
		_ = GetDriveCacheStorage(c).Remove(name)
		_ = GetDriveDataStorage(c).Remove(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// get drive initialization information
	r.GET("/drive/:name/init", func(c *gin.Context) {
		name := c.Param("name")
		data, e := GetRootDrive(c).DriveInitConfig(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, data)
	})

	// init drive
	r.POST("/drive/:name/init", func(c *gin.Context) {
		name := c.Param("name")
		data := make(types.SM, 0)
		if e := c.Bind(&data); e != nil {
			_ = c.Error(e)
			return
		}
		if e := GetRootDrive(c).DriveInit(name, data); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// reload drives
	r.POST("/drives/reload", func(c *gin.Context) {
		if e := GetRootDrive(c).ReloadDrive(false); e != nil {
			_ = c.Error(e)
		}
	})

	// endregion

	// region permissions

	// get by path
	r.GET("/path-permissions/*path", func(c *gin.Context) {
		path := common.CleanPath(c.Param("path"))
		permissions, e := GetPermissionStorage(c).GetByPath(path)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, permissions)
	})

	// save path permissions
	r.PUT("/path-permissions/*path", func(c *gin.Context) {
		path := common.CleanPath(c.Param("path"))
		permissions := make([]types.PathPermission, 0)
		if e := c.Bind(&permissions); e != nil {
			_ = c.Error(e)
			return
		}
		if e := GetPermissionStorage(c).SavePathPermissions(path, permissions); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// endregion

	// region mount

	// save mounts
	r.POST("/mount/*to", func(c *gin.Context) {
		to := common.CleanPath(c.Param("to"))
		src := make([]mountSource, 0)
		if e := c.Bind(&src); e != nil {
			_ = c.Error(e)
			return
		}
		if len(src) == 0 {
			return
		}
		mounts := make([]types.PathMount, len(src))
		for i, p := range src {
			mounts[i] = types.PathMount{Path: &to, Name: p.Name, MountAt: p.Path}
		}
		if e := GetPathMountStorage(c).SaveMounts(mounts, true); e != nil {
			_ = c.Error(e)
			return
		}
		_ = GetRootDrive(c).ReloadMounts()
	})

	// endregion

	// region misc

	// clean all PathPermission and PathMount that is point to invalid path
	r.POST("/clean-permissions-mounts", func(c *gin.Context) {
		root := GetRootDrive(c).Get()
		pps, e := GetPermissionStorage(c).GetAll()
		if e != nil {
			_ = c.Error(e)
			return
		}
		ms, e := GetPathMountStorage(c).GetMounts()
		if e != nil {
			_ = c.Error(e)
			return
		}
		paths := make(map[string]bool)
		for _, p := range pps {
			paths[*p.Path] = true
		}
		for _, m := range ms {
			paths[m.MountAt] = true
		}
		for p := range paths {
			_, e := root.Get(p)
			if e != nil {
				if common.IsNotFoundError(e) {
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
			if e := GetPermissionStorage(c).DeleteByPath(p); e != nil {
				_ = c.Error(e)
				return
			}
			if e := GetPathMountStorage(c).DeleteByMountAt(p); e != nil {
				_ = c.Error(e)
				return
			}
			n++
		}
		SetResult(c, n)
	})

	// endregion

}

type mountSource struct {
	Path string `json:"path" binding:"required"`
	Name string `json:"name" binding:"required"`
}

var driveNamePattern = regexp.MustCompile("^[^/\\\\0:*\"<>|]+$")

func checkDriveName(name string) error {
	if name == "" || name == "." || name == ".." || !driveNamePattern.MatchString(name) {
		return common.NewBadRequestError("invalid drive name '" + name + "'")
	}
	return nil
}
