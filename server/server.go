package server

import (
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

func InitServer(config common.Config,
	ch *registry.ComponentsHolder,
	bus event.Bus,
	rootDrive *drive.RootDrive,
	driveAccess *drive.Access,
	searcher *search.Service,
	tokenStore types.TokenStore,
	thumbnail *thumbnail.Maker,
	signer *utils.Signer,
	chunkUploader *ChunkUploader,
	runner task.Runner,
	optionsDAO *storage.OptionsDAO,
	userDAO *storage.UserDAO,
	groupDAO *storage.GroupDAO,
	driveDAO *storage.DriveDAO,
	driveCacheDAO *storage.DriveCacheDAO,
	driveDataDAO *storage.DriveDataDAO,
	permissionDAO *storage.PathPermissionDAO,
	pathMountDAO *storage.PathMountDAO,
	messageSource i18n.MessageSource) (*gin.Engine, error) {

	if utils.IsDebugOn() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(Logger())
	engine.Use(apiResultHandler(messageSource))

	userAuth := NewUserAuth(userDAO)

	router := engine.Group(config.APIPath)

	if e := InitCommonRoutes(ch, router, tokenStore, runner); e != nil {
		return nil, e
	}
	if e := InitAuthRoutes(router, userAuth, tokenStore); e != nil {
		return nil, e
	}
	if e := InitAdminRoutes(router, ch, bus, driveAccess, rootDrive, searcher, tokenStore, optionsDAO,
		userDAO, groupDAO, driveDAO, driveCacheDAO, driveDataDAO, permissionDAO, pathMountDAO); e != nil {
		return nil, e
	}

	if e := InitDriveRoutes(router, driveAccess, searcher, config, thumbnail,
		signer, chunkUploader, runner, tokenStore, optionsDAO); e != nil {
		return nil, e
	}

	if config.WebDav.Enabled {
		if e := InitWebdavAccess(engine, config, driveAccess, userAuth); e != nil {
			return nil, e
		}
	}

	if config.WebDir != "" {
		webFiles := newWebFiles(config.WebDir, config, optionsDAO)
		s := http.StripPrefix(config.WebPath, webFiles)
		engine.NoRoute(func(c *gin.Context) { s.ServeHTTP(c.Writer, c.Request) })
	}

	ch.Add("runtimeStat", runtimeStat{})
	return engine, nil
}

func apiResultHandler(ms i18n.MessageSource) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			result, exists := GetResult(c)
			if exists {
				writeJSON(c, ms, 200, result)
			}
			return
		}
		e := c.Errors[0]
		code := 500
		result := types.M{
			"message": e.Err.Error(),
		}
		if re, ok := e.Err.(err.RequestError); ok {
			code = re.Code()
		}
		if red, ok := e.Err.(err.RequestErrorWithData); ok {
			result["data"] = red.Data()
		}
		writeJSON(c, ms, code, result)
	}
}

func writeJSON(c *gin.Context, ms i18n.MessageSource, code int, v interface{}) {
	if c.Writer.Written() {
		return
	}
	result := TranslateV(c, ms, v)
	rv := reflect.ValueOf(v)
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
		c.Status(code)
	} else {
		c.JSON(code, result)
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
		"Sys":          utils.FormatBytes(m.Sys, 2),
		"HeapSys":      utils.FormatBytes(m.HeapSys, 2),
		"HeapInUse":    utils.FormatBytes(m.HeapInuse, 2),
		"LastGC":       time.Unix(0, int64(m.LastGC)).Format(time.RubyDate),
		"StopTheWorld": fmt.Sprintf("%d ms", m.PauseTotalNs/uint64(time.Millisecond)),
		"NumGC":        fmt.Sprintf("%d", m.NumGC),
	}, nil
}
