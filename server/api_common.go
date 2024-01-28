package server

import (
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/storage"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	allowedOptionPrefix = []string{"web."}
)

func InitCommonRoutes(
	ch *registry.ComponentsHolder,
	r gin.IRouter,
	options *storage.OptionsDAO,
	tokenStore types.TokenStore,
	runner task.Runner) error {

	cr := &commonRoute{ch, options, tokenStore, runner}

	// get configuration
	r.GET("/config", cr.getConfig)

	authR := r.Group("/", TokenAuth(tokenStore))
	// get task
	authR.GET("/task/:id", cr.getTask)
	// cancel and delete task
	authR.DELETE("/task/:id", cr.cancelAndDeleteTask)

	authAdmin := authR.Group("/", AdminGroupRequired())
	// get tasks
	authAdmin.GET("/tasks", cr.getTasks)

	return nil
}

type commonRoute struct {
	ch         *registry.ComponentsHolder
	options    *storage.OptionsDAO
	tokenStore types.TokenStore
	runner     task.Runner
}

func (cr *commonRoute) getConfig(c *gin.Context) {
	cs := cr.ch.Gets(func(c interface{}) bool {
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

	optionsMap := make(map[string]string)
	for _, key := range strings.Split(c.Query("opts"), ",") {
		for _, prefix := range allowedOptionPrefix {
			if strings.HasPrefix(key, prefix) {
				value, e := cr.options.Get(key)
				if e != nil {
					_ = c.Error(e)
					return
				}
				optionsMap[key] = value
			}
		}
	}
	configMap["options"] = optionsMap

	SetResult(c, configMap)
}

func (cr *commonRoute) getTask(c *gin.Context) {
	t, e := cr.runner.GetTask(c.Param("id"))
	if e != nil && e == task.ErrorNotFound {
		e = err.NewNotFoundMessageError(e.Error())
	}
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (cr *commonRoute) cancelAndDeleteTask(c *gin.Context) {
	_, e := cr.runner.StopTask(c.Param("id"))
	if e != nil && e == task.ErrorNotFound {
		e = err.NewNotFoundMessageError(e.Error())
	}
	if e != nil {
		_ = c.Error(e)
	}
}

func (cr *commonRoute) getTasks(c *gin.Context) {
	group := c.Query("group")
	tasks, e := cr.runner.GetTasks(group)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, tasks)
}
