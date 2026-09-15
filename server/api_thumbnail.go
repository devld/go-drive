package server

import (
	"context"
	"fmt"
	"go-drive/common"
	"go-drive/common/types"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (dr *driveRoute) getThumbnail(c *gin.Context) {
	path, e := getQueryPath(c, "path")
	if e != nil {
		_ = c.Error(e)
		return
	}
	d := c.MustGet("drive").(types.IDrive)

	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if entry.Meta().ThumbnailURL != "" {
		c.Redirect(http.StatusFound, entry.Meta().ThumbnailURL)
		return
	}
	makeCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	file, e := dr.thumbnail.Make(
		makeCtx, dr.wrapEntryWithAccessKey(entry, c.Query(common.SignatureQueryKey)),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}
	defer func() { _ = file.Close() }()
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dr.config.Thumbnail.TTL.Seconds())))
	c.Header("Content-Type", file.MimeType())
	http.ServeContent(c.Writer, c.Request, "", file.ModTime(), file)
}
