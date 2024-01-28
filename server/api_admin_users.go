package server

import (
	"go-drive/common/types"
	"go-drive/storage"

	"github.com/gin-gonic/gin"
)

type usersRoute struct {
	userDAO *storage.UserDAO
}

func (ar *usersRoute) listUsers(c *gin.Context) {
	users, e := ar.userDAO.ListUser()
	if e != nil {
		_ = c.Error(e)
		return
	}
	for i := range users {
		users[i].Password = ""
	}
	SetResult(c, users)
}

func (ar *usersRoute) getUser(c *gin.Context) {
	username := c.Param("username")
	user, e := ar.userDAO.GetUser(username)
	if e != nil {
		_ = c.Error(e)
		return
	}
	user.Password = ""
	SetResult(c, user)
}

func (ar *usersRoute) createUser(c *gin.Context) {
	user := types.User{}
	if e := c.Bind(&user); e != nil {
		_ = c.Error(e)
		return
	}
	addUser, e := ar.userDAO.AddUser(user)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, addUser)
}

func (ar *usersRoute) updateUser(c *gin.Context) {
	user := types.User{}
	if e := c.Bind(&user); e != nil {
		_ = c.Error(e)
		return
	}
	username := c.Param("username")
	e := ar.userDAO.UpdateUser(username, user)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (ar *usersRoute) deleteUser(c *gin.Context) {
	username := c.Param("username")
	e := ar.userDAO.DeleteUser(username)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

type groupsRoute struct {
	userDAO  *storage.UserDAO
	groupDAO *storage.GroupDAO
}

func (gr *groupsRoute) listGroups(c *gin.Context) {
	groups, e := gr.groupDAO.ListGroup()
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, groups)
}

func (gr *groupsRoute) getGroup(c *gin.Context) {
	name := c.Param("name")
	group, e := gr.groupDAO.GetGroup(name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, group)
}

func (gr *groupsRoute) createGroup(c *gin.Context) {
	group := storage.GroupWithUsers{}
	if e := c.Bind(&group); e != nil {
		_ = c.Error(e)
		return
	}
	addGroup, e := gr.groupDAO.AddGroup(group)
	if e != nil {
		_ = c.Error(e)
		return
	}
	gr.userDAO.EvictCache("")
	SetResult(c, addGroup)
}

func (gr *groupsRoute) updateGroup(c *gin.Context) {
	name := c.Param("name")
	gus := storage.GroupWithUsers{}
	if e := c.Bind(&gus); e != nil {
		_ = c.Error(e)
		return
	}
	if e := gr.groupDAO.UpdateGroup(name, gus); e != nil {
		_ = c.Error(e)
		return
	}
	gr.userDAO.EvictCache("")
}

func (gr *groupsRoute) deleteGroup(c *gin.Context) {
	name := c.Param("name")
	e := gr.groupDAO.DeleteGroup(name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	gr.userDAO.EvictCache("")
}
