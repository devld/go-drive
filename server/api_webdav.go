package server

import (
	"context"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/drive"
	"go-drive/server/webdav"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var webdavHTTPMethods = []string{
	"OPTIONS", "GET", "HEAD", "POST", "DELETE", "PUT",
	"MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", "PROPFIND", "PROPPATCH",
}

func InitWebdavAccess(router gin.IRouter, config common.Config,
	access *drive.Access, userAuth *UserAuth) error {

	cfp, e := drive_util.NewCacheFillPool(config.WebDav.MaxCacheItems, config.TempDir)
	if e != nil {
		return e
	}

	wa := &webdavAccess{
		access:  access,
		cfp:     cfp,
		config:  config,
		lockSys: webdav.NewMemLS(),
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
	cfp     *drive_util.CacheFilePool
	lockSys webdav.LockSystem
	config  common.Config
}

func (w *webdavAccess) ServeHTTP(c *gin.Context) {
	session := GetSession(c)

	drive, e := w.access.GetDrive(session)
	if e != nil {
		log.Printf("GetDrive error: %v", e)
		c.AbortWithError(http.StatusInternalServerError, e)
		return
	}

	driveFs, e := drive_util.NewDriveFS(drive, w.config.TempDir, w.cfp)
	if e != nil {
		c.AbortWithError(http.StatusInternalServerError, e)
		return
	}

	handler := webdav.Handler{
		Prefix:     w.config.WebDav.Prefix,
		FileSystem: webDavFS{driveFs},
		LockSystem: w.lockSys,
	}
	handler.ServeHTTP(c.Writer, c.Request)
}

type webDavFS struct {
	*drive_util.DriveFS
}

func (wfs webDavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return wfs.DriveFS.OpenFile(ctx, name, flag, perm)
}
