package server

import (
	"github.com/gin-gonic/gin"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
)

func InitCommonRoutes(
	ch *registry.ComponentsHolder,
	r gin.IRouter,
	tokenStore types.TokenStore,
	runner task.Runner) error {

	// get configuration
	r.GET("/config", func(c *gin.Context) {
		cs := ch.Gets(func(c interface{}) bool {
			_, ok := c.(types.ISysConfig)
			return ok
		})

		configMap := make(types.M)

		for _, sc := range cs {
			name, m, e := sc.(types.ISysConfig).SysConfig()
			if e != nil {
				_ = c.Error(e)
				return
			}
			configMap[name] = m
		}

		SetResult(c, configMap)
	})

	authR := r.Group("/", TokenAuth(tokenStore))
	authAdmin := authR.Group("/", UserGroupRequired("admin"))

	// get task
	authR.GET("/task/:id", func(c *gin.Context) {
		t, e := runner.GetTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = err.NewNotFoundMessageError(e.Error())
		}
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, t)
	})

	// get tasks
	authAdmin.GET("/tasks", func(c *gin.Context) {
		group := c.Query("group")
		tasks, e := runner.GetTasks(group)
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, tasks)
	})

	// cancel and delete task
	authAdmin.DELETE("/task/:id", func(c *gin.Context) {
		_, e := runner.StopTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = err.NewNotFoundMessageError(e.Error())
		}
		if e != nil {
			_ = c.Error(e)
		}
	})

	return nil
}
