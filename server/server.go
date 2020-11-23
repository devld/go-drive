package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/storage"
	"net/http"
	"runtime"
	"time"
)

func InitServer(config common.Config,
	ch *common.ComponentsHolder,
	rootDrive *drive.RootDrive,
	tokenStore types.TokenStore,
	thumbnail *Thumbnail,
	signer *common.Signer,
	chunkUploader *ChunkUploader,
	runner task.Runner,
	userDAO *storage.UserDAO,
	groupDAO *storage.GroupDAO,
	driveDAO *storage.DriveDAO,
	driveCacheDAO *storage.DriveCacheDAO,
	driveDataDAO *storage.DriveDataDAO,
	permissionDAO *storage.PathPermissionDAO,
	pathMountDAO *storage.PathMountDAO) *gin.Engine {

	if common.IsDebugOn() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(Logger())
	engine.Use(apiResultHandler)

	InitAuthRoutes(engine, tokenStore, userDAO)

	InitAdminRoutes(engine, ch, rootDrive, tokenStore, userDAO, groupDAO,
		driveDAO, driveCacheDAO, driveDataDAO, permissionDAO, pathMountDAO)

	InitDriveRoutes(engine, config, rootDrive, permissionDAO, thumbnail,
		signer, chunkUploader, runner, tokenStore)

	if config.GetResDir() != "" {
		engine.NoRoute(Static("/", config.GetResDir()))
	}

	ch.Add("runtimeStat", runtimeStat{})
	return engine
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
