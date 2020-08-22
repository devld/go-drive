package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common/types"
)

func InitAdminRoutes(r gin.IRouter) {
	r = r.Group("/admin", Auth(), UserGroupRequired("admin"))

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

}
