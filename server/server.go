package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
)

func InitServer(components *ComponentsHolder) (*gin.Engine, error) {

	if common.IsDebugOn() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(func(c *gin.Context) {
		c.Set(keyComponentsHolder, components)
	})

	engine.Use(gin.Recovery())
	engine.Use(apiResultHandler)
	engine.Use(gin.Logger())

	InitAuthRoutes(engine)
	InitAdminRoutes(engine)
	InitDriveRoutes(engine)

	return engine, nil
}

func apiResultHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) == 0 {
		result, exists := GetResult(c)
		if exists {
			c.JSON(200, result)
		}
		return
	}
	e := c.Errors[0]
	code := 500
	result := map[string]interface{}{
		"msg": e.Err.Error(),
	}
	if re, ok := e.Err.(common.RequestError); ok {
		code = re.Code()
	}
	if red, ok := e.Err.(common.RequestErrorWithData); ok {
		result["data"] = red.Data()
	}
	c.JSON(code, result)
}
