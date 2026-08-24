package server

import (
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	httpreq "go-drive/common/req"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/auth"
	"go-drive/server/job"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"
	"io/fs"
	"net/http"
	"runtime"
	"runtime/debug"
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
	driveDataDAO *storage.DriveDataDAO,
	permissionDAO *storage.PathPermissionDAO,
	pathMountDAO *storage.PathMountDAO,
	pathMetaDAO *storage.PathMetaDAO,
	jobDAO *storage.JobDAO,
	fileBucketDAO *storage.FileBucketDAO,
	jobExecutor *job.JobExecutor,
	messageSource i18n.MessageSource,
	webFS fs.FS) (*gin.Engine, error) {

	if logging.Enabled(logging.DebugLevel) {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	configureGinLogging()

	engine := gin.New()

	if len(config.TrustedProxies) > 0 {
		if e := engine.SetTrustedProxies(config.TrustedProxies); e != nil {
			return nil, e
		}
	} else {
		engine.SetTrustedProxies(nil)
	}

	engine.Use(gin.CustomRecoveryWithWriter(nil, handlePanic))

	engine.Use(Logger())

	engine.Use(apiResultHandler(messageSource))

	userAuth, e := auth.NewUserAuth(config.Auth.Providers, ch)
	if e != nil {
		return nil, e
	}
	ch.Add(registry.KeyUserAuth, userAuth)

	router := engine.Group(config.APIPath)

	failBanGroup := NewFailBanGroup(10 * time.Minute)
	ch.Add(registry.KeyFailBanGroup, failBanGroup)

	if e := InitCommonRoutes(ch, router, optionsDAO, tokenStore, runner); e != nil {
		return nil, e
	}
	if e := InitAuthRoutes(router, userAuth, tokenStore, failBanGroup); e != nil {
		return nil, e
	}
	if e := InitAdminRoutes(router, ch, config, bus, runner, jobExecutor, driveAccess, rootDrive, searcher, tokenStore, optionsDAO,
		userDAO, groupDAO, driveDAO, driveDataDAO, permissionDAO, pathMountDAO, pathMetaDAO, jobDAO, fileBucketDAO); e != nil {
		return nil, e
	}

	if e := InitDriveRoutes(router, driveAccess, searcher, config, thumbnail,
		signer, chunkUploader, runner, tokenStore, userDAO, optionsDAO, pathMetaDAO); e != nil {
		return nil, e
	}

	if e := InitFileBucketRoutes(router, config, driveAccess, fileBucketDAO, messageSource); e != nil {
		return nil, e
	}

	if config.WebDav.Enabled {
		if e := InitWebdavAccess(engine, config, driveAccess, userAuth); e != nil {
			return nil, e
		}
	}

	if webFS != nil {
		webFiles := newWebFiles(http.FS(webFS), config, optionsDAO)
		s := http.StripPrefix(config.WebPath, webFiles)
		engine.NoRoute(func(c *gin.Context) { s.ServeHTTP(c.Writer, c.Request) })
	}

	ch.Add(registry.KeyRuntimeStat, runtimeStat{})
	return engine, nil
}

func apiResultHandler(ms i18n.MessageSource) func(*gin.Context) {
	return func(c *gin.Context) {
		SetMessageSource(c, ms)
		c.Next()
		if len(c.Errors) == 0 {
			result, exists := GetResult(c)
			if exists {
				writeJSON(c, ms, http.StatusOK, result)
			}
			return
		}
		e := c.Errors[0]
		code := 500
		result := types.M{
			"message": e.Err.Error(),
		}
		if re, ok := e.Err.(err.Error); ok {
			code = re.Code()
		}
		if red, ok := e.Err.(err.ErrorWithData); ok {
			result["data"] = red.Data()
		}
		writeJSON(c, ms, code, result)
	}
}

func handlePanic(c *gin.Context, err any) {
	var msg string
	if ee, ok := err.(error); ok {
		msg = ee.Error()
	} else {
		msg = fmt.Sprintf("%v", err)
	}
	logging.For("http").Errorf("panic recovered: %s\n%s", msg, debug.Stack())
	c.JSON(http.StatusInternalServerError,
		types.M{
			"message": msg,
		},
	)
}

func configureGinLogging() {
	ginLog := logging.For("gin")
	gin.DefaultWriter = ginLog.Writer()
	gin.DefaultErrorWriter = ginLog.ErrorWriter()
	gin.DebugPrintFunc = func(format string, values ...any) {
		ginLog.Debugf(format, values...)
	}
	gin.DebugPrintRouteFunc = func(method, path, handler string, handlers int) {
		ginLog.Debugf("%-6s %-25s --> %s (%d handlers)", method, path, handler, handlers)
	}
}

func writeJSON(c *gin.Context, ms i18n.MessageSource, code int, v any) {
	if c.Writer.Written() {
		return
	}
	result := TranslateV(c, ms, v)
	c.JSON(code, result)
}

func Logger() gin.HandlerFunc {
	httpLog := logging.For("http")
	return func(c *gin.Context) {
		if c.FullPath() == "" {
			// NoRoute static files
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		httpLog.Infof("%s %s %d %s %s %d", c.Request.Method, httpreq.SanitizeURL(c.Request.URL.RequestURI()),
			status, time.Since(start), c.ClientIP(), c.Writer.Size())
		if errors := c.Errors.ByType(gin.ErrorTypePrivate).String(); errors != "" {
			httpLog.Errorf("request errors: %s", errors)
		}
	}
}

type runtimeStat struct {
}

func (r runtimeStat) Status() (string, types.SM, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return "Runtime", types.SM{
		"Alloc":        utils.FormatBytes(m.Alloc, 2),
		"Sys":          utils.FormatBytes(m.Sys, 2),
		"TotalAlloc":   utils.FormatBytes(m.TotalAlloc, 2),
		"HeapObjects":  fmt.Sprintf("%d", m.HeapObjects),
		"LastGC":       time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
		"StopTheWorld": fmt.Sprintf("%d ms", m.PauseTotalNs/uint64(time.Millisecond)),
		"NumGC":        fmt.Sprintf("%d", m.NumGC),
		"GoRoutines":   fmt.Sprintf("%d", runtime.NumGoroutine()),
	}, nil
}
