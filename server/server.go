package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"net/http"
	"runtime"
	"time"
)

func init() {
	common.R().Register("httpServer", func(c *common.ComponentRegistry) interface{} {
		resDir := c.Get("config").(common.Config).GetResDir()
		engine, e := InitServer(resDir)
		common.PanicIfError(e)
		return engine
	}, 4096)

	common.R().Register("runtimeStat", func(c *common.ComponentRegistry) interface{} {
		return runtimeStat{}
	}, 0)
}

func InitServer(resDir string) (*gin.Engine, error) {
	if common.IsDebugOn() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(Logger())
	engine.Use(apiResultHandler)

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

func Logger() gin.HandlerFunc {
	logger := gin.Logger()
	return func(c *gin.Context) {
		if c.FullPath() == "" {
			// NoRoute static files
			c.Next()
			return
		}
		logger(c)
	}
}

type runtimeStat struct {
}

func (r runtimeStat) Status() (string, types.SM, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return "Runtime", types.SM{
		"GoRoutines":   fmt.Sprintf("%d", runtime.NumGoroutine()),
		"TotalAlloc":   fmt.Sprintf("%d", m.TotalAlloc),
		"Alloc":        fmt.Sprintf("%d", m.Alloc),
		"HeapObjects":  fmt.Sprintf("%d", m.HeapObjects),
		"Sys":          common.FormatBytes(m.Sys, 2),
		"HeapSys":      common.FormatBytes(m.HeapSys, 2),
		"HeapInUse":    common.FormatBytes(m.HeapInuse, 2),
		"LastGC":       time.Unix(0, int64(m.LastGC)).Format(time.RubyDate),
		"StopTheWorld": fmt.Sprintf("%d ms", m.PauseTotalNs/uint64(time.Millisecond)),
		"NumGC":        fmt.Sprintf("%d", m.NumGC),
	}, nil
}
