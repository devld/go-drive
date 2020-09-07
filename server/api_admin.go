package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common/types"
	"go-drive/storage"
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

	// update drives
	r.PUT("/drives", func(c *gin.Context) {
		drives := make([]types.Drive, 0)
		if e := c.Bind(&drives); e != nil {
			_ = c.Error(e)
			return
		}
		if e := GetDriveStorage(c).SaveDrives(drives); e != nil {
			_ = c.Error(e)
			return
		}
	})

	// get drives
	r.GET("/drives", func(c *gin.Context) {
		drives, e := GetDriveStorage(c).GetDrives()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, drives)
	})

	// reload drives
	r.POST("/drives/reload", func(c *gin.Context) {
		if e := GetRootDrive(c).ReloadDrive(); e != nil {
			_ = c.Error(e)
		}
	})

	// endregion

}
