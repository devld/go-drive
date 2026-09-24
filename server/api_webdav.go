package server

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/logging"
	"go-drive/drive"
	"go-drive/server/auth"
	"go-drive/server/webdav"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

var webdavHTTPMethods = []string{
	"OPTIONS", "GET", "HEAD", "POST", "DELETE", "PUT",
	"MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", "PROPFIND", "PROPPATCH",
}

func InitWebdavAccess(router gin.IRouter, config common.Config,
	access *drive.Access, userAuth *auth.UserAuth) error {

	tempDir := filepath.Join(config.TempDir, "webdav")
	if e := os.MkdirAll(tempDir, 0700); e != nil {
		return e
	}
	cfp, e := driveutil.NewCacheFilePool(driveutil.CacheFilePoolOptions{
		MaxEntries: config.WebDav.MaxCacheItems,
		Dir:        tempDir,
	})
	if e != nil {
		return e
	}

	wa := &webdavAccess{
		access:  access,
		cfp:     cfp,
		config:  config,
		lockSys: webdav.NewMemLS(),
		tempDir: tempDir,
	}

	withAuth := router.Group(config.WebDav.Prefix, BasicAuth(userAuth, "webdav", config.WebDav.AllowAnonymous))
	withoutAuth := router.Group(config.WebDav.Prefix)

	for _, method := range webdavHTTPMethods {
		r := withAuth
		if method == "OPTIONS" {
			r = withoutAuth
		}
		r.Handle(method, "/*path", wa.ServeHTTP)
	}
	return nil
}

type webdavAccess struct {
	access  *drive.Access
	cfp     *driveutil.CacheFilePool
	lockSys webdav.LockSystem
	config  common.Config
	tempDir string
}

func (w *webdavAccess) ServeHTTP(c *gin.Context) {
	principal := GetPrincipal(c)

	drive, e := w.access.GetDrive(principal)
	if e != nil {
		logging.For("webdav").Errorf("GetDrive error: %v", e)
		c.AbortWithError(http.StatusInternalServerError, e)
		return
	}

	driveFs, e := driveutil.NewDriveFS(c.Request.Context(), drive, w.tempDir, w.cfp)
	if e != nil {
		logging.For("webdav").Errorf("DriveFS creation failed method=%s path=%s: %v",
			c.Request.Method, logging.Sanitize(c.Request.URL.Path), e)
		c.AbortWithError(http.StatusInternalServerError, e)
		return
	}

	started := time.Now()
	handler := webdav.Handler{
		Prefix:     w.config.WebDav.Prefix,
		FileSystem: webDavFS{driveFs},
		LockSystem: w.lockSys,
		Logger: func(r *http.Request, handlerErr error) {
			path := logging.Sanitize(r.URL.Path)
			if handlerErr != nil {
				logging.For("webdav").Warnf("request failed method=%s path=%s duration=%s: %v",
					r.Method, path, time.Since(started), handlerErr)
				return
			}
			logging.For("webdav").Debugf("request completed method=%s path=%s duration=%s",
				r.Method, path, time.Since(started))
		},
	}
	handler.ServeHTTP(c.Writer, c.Request)
}

type webDavFS struct {
	*driveutil.DriveFS
}

func (wfs webDavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return wfs.DriveFS.OpenFile(ctx, name, flag, perm)
}
