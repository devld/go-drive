package server

import (
	"archive/zip"
	"fmt"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func (dr *driveRoute) zipDownload(c *gin.Context) {
	files := utils.SplitLines(c.PostForm("files"))
	if len(files) == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	prefix := c.PostForm("prefix")
	drive := c.MustGet("drive").(types.IDrive)

	entries := make([]types.IEntry, 0, len(files))
	for _, f := range files {
		if f == "" {
			continue
		}
		file := utils.CleanPath(f)
		entry, e := drive.Get(c.Request.Context(), file)
		if e != nil {
			_ = c.Error(e)
			return
		}
		entries = append(entries, entry)
	}

	ctx := task.NewTaskContext(c.Request.Context())
	entriesTrees := make([]driveutil.EntryTreeNode, 0, len(entries))
	for _, entry := range entries {
		rootNode, e := driveutil.BuildEntriesTree(ctx, entry, true)
		if e != nil {
			return
		}
		entriesTrees = append(entriesTrees, rootNode)
	}

	totalSize := ctx.GetTotal()
	maxAllowedSizeOpt := dr.options.GetValue(maxZipSizeKey)
	maxAllowSize := maxAllowedSizeOpt.DataSize(-1)
	if maxAllowSize > 0 && totalSize > maxAllowSize {
		_ = c.Error(err.NewNotAllowedMessageError(i18n.T("api.zip.size_exceed", string(maxAllowedSizeOpt))))
		return
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=\""+
		url.QueryEscape(fmt.Sprintf("packaged_%d.zip", len(files)))+"\"")

	zipFile := zip.NewWriter(c.Writer)
	defer func() { _ = zipFile.Close() }()
	for _, node := range entriesTrees {
		if e := driveutil.VisitEntriesTree(node, func(entry types.IEntry) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			name := entry.Path()
			if entry.Type().IsDir() {
				name += "/"
			}
			if prefix != "" && strings.HasPrefix(name, prefix+"/") {
				name = strings.TrimPrefix(name, prefix+"/")
			}
			file, e := zipFile.Create(name)
			if e != nil {
				return e
			}
			if entry.Type().IsFile() {
				if e := driveutil.CopyIContent(task.NewContextWrapper(c.Request.Context()), entry, file); e != nil {
					return e
				}
			}
			return nil
		}); e != nil {
			return
		}
	}
}
