package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/storage"
	"regexp"
	"sort"
)

func InitAdminRoutes(r gin.IRouter,
	ch *common.ComponentsHolder,
	rootDrive *drive.RootDrive,
	tokenStore types.TokenStore,
	userDAO *storage.UserDAO,
	groupDAO *storage.GroupDAO,
	driveDAO *storage.DriveDAO,
	driveCacheDAO *storage.DriveCacheDAO,
	driveDataDAO *storage.DriveDataDAO,
	permissionDAO *storage.PathPermissionDAO,
	pathMountDAO *storage.PathMountDAO) {

	r = r.Group("/admin", Auth(tokenStore), UserGroupRequired("admin"))

	// region user

	// list users
	r.GET("/users", func(c *gin.Context) {
		users, e := userDAO.ListUser()
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
		user, e := userDAO.GetUser(username)
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
		addUser, e := userDAO.AddUser(user)
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
		e := userDAO.UpdateUser(username, user)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete user
	r.DELETE("/user/:username", func(c *gin.Context) {
		username := c.Param("username")
		e := userDAO.DeleteUser(username)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// endregion

	// region group

	// list groups
	r.GET("/groups", func(c *gin.Context) {
		groups, e := groupDAO.ListGroup()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, groups)
	})

	// get group and it's users
	r.GET("/group/:name", func(c *gin.Context) {
		name := c.Param("name")
		group, e := groupDAO.GetGroup(name)
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
		addGroup, e := groupDAO.AddGroup(group)
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
		if e := groupDAO.UpdateGroup(name, gus); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete group
	r.DELETE("/group/:name", func(c *gin.Context) {
		name := c.Param("name")
		e := groupDAO.DeleteGroup(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})
	// endregion

	// region drive

	// get drives
	r.GET("/drives", func(c *gin.Context) {
		drives, e := driveDAO.GetDrives()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, drives)
	})

	// add drive
	r.POST("/drive", func(c *gin.Context) {
		d := types.Drive{}
		if e := c.Bind(&d); e != nil {
			_ = c.Error(e)
			return
		}
		if e := checkDriveName(d.Name); e != nil {
			_ = c.Error(e)
			return
		}
		d, e := driveDAO.AddDrive(d)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, d)
	})

	// update drive
	r.PUT("/drive/:name", func(c *gin.Context) {
		name := c.Param("name")
		if e := checkDriveName(name); e != nil {
			_ = c.Error(e)
			return
		}
		d := types.Drive{}
		if e := c.Bind(&d); e != nil {
			_ = c.Error(e)
			return
		}
		e := driveDAO.UpdateDrive(name, d)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete drive
	r.DELETE("/drive/:name", func(c *gin.Context) {
		name := c.Param("name")
		e := driveDAO.DeleteDrive(name)
		_ = driveCacheDAO.Remove(name)
		_ = driveDataDAO.Remove(name)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// get drive initialization information
	r.GET("/drive/:name/init", func(c *gin.Context) {
		name := c.Param("name")
		data, e := rootDrive.DriveInitConfig(name)
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
		if e := rootDrive.DriveInit(name, data); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// reload drives
	r.POST("/drives/reload", func(c *gin.Context) {
		if e := rootDrive.ReloadDrive(false); e != nil {
			_ = c.Error(e)
		}
	})

	// endregion

	// region permissions

	// get by path
	r.GET("/path-permissions/*path", func(c *gin.Context) {
		path := common.CleanPath(c.Param("path"))
		permissions, e := permissionDAO.GetByPath(path)
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
		if e := permissionDAO.SavePathPermissions(path, permissions); e != nil {
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
		if e := pathMountDAO.SaveMounts(mounts, true); e != nil {
			_ = c.Error(e)
			return
		}
		_ = rootDrive.ReloadMounts()
	})

	// endregion

	// region misc

	// clean all PathPermission and PathMount that is point to invalid path
	r.POST("/clean-permissions-mounts", func(c *gin.Context) {
		root := rootDrive.Get()
		pps, e := permissionDAO.GetAll()
		if e != nil {
			_ = c.Error(e)
			return
		}
		ms, e := pathMountDAO.GetMounts()
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
			if e := permissionDAO.DeleteByPath(p); e != nil {
				_ = c.Error(e)
				return
			}
			if e := pathMountDAO.DeleteByMountAt(p); e != nil {
				_ = c.Error(e)
				return
			}
			n++
		}
		SetResult(c, n)
	})

	// get service stats
	r.GET("/stats", func(c *gin.Context) {
		stats := ch.Gets(func(c interface{}) bool {
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
	})

	// endregion

}

type mountSource struct {
	Path string `json:"path" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type statItem struct {
	Name string   `json:"name"`
	Data types.SM `json:"data"`
}

var driveNamePattern = regexp.MustCompile("^[^/\\\\0:*\"<>|]+$")

func checkDriveName(name string) error {
	if name == "" || name == "." || name == ".." || !driveNamePattern.MatchString(name) {
		return common.NewBadRequestError("invalid drive name '" + name + "'")
	}
	return nil
}
