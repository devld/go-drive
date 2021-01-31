package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common/registry"
	"go-drive/common/types"
)

func InitCommonRoutes(r gin.IRouter, ch *registry.ComponentsHolder) {

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

}
