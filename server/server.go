package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"net/http"
)

func InitServer(components *ComponentsHolder, resDir string) (*gin.Engine, error) {

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

	if resDir != "" {
		engine.NoRoute(Static("/", resDir))
	}

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
	result := types.M{
		"message": e.Err.Error(),
	}
	if re, ok := e.Err.(common.RequestError); ok {
		code = re.Code()
	}
	if red, ok := e.Err.(common.RequestErrorWithData); ok {
		result["data"] = red.Data()
	}
	c.JSON(code, result)
}

func Static(prefix, root string) gin.HandlerFunc {
	s := http.StripPrefix(prefix, http.FileServer(http.Dir(root)))
	return func(c *gin.Context) {
		s.ServeHTTP(c.Writer, c.Request)
	}
}
