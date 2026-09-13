package server

import (
	"archive/zip"
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	archivepreview "go-drive/server/archive"
	"io"
	"mime"
	"net/url"
	"strconv"
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
	c.Writer.Header().Set("Content-Disposition",
		"attachment; filename=\""+
			url.QueryEscape(fmt.Sprintf("packaged_%d.zip", len(files)))+"\"")

	zipFile := zip.NewWriter(c.Writer)
	defer func() {
		_ = zipFile.Close()
	}()

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

func (dr *driveRoute) listArchive(c *gin.Context) {
	path, e := getQueryPath(c, "path")
	if e != nil {
		_ = c.Error(e)
		return
	}
	if dr.archive == nil {
		_ = c.Error(mapArchiveError(archivepreview.ErrUnsupportedFormat))
		return
	}
	d := c.MustGet("drive").(types.IDrive)
	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	result, e := dr.archive.List(c.Request.Context(), entry, c.Query("dir"))
	if e != nil {
		_ = c.Error(mapArchiveError(e))
		return
	}
	SetResult(c, result)
}

func (dr *driveRoute) getArchiveContent(c *gin.Context) {
	path, e := getQueryPath(c, "path")
	if e != nil {
		_ = c.Error(e)
		return
	}
	member := c.Query("entry")
	if member == "" {
		_ = c.Error(err.NewBadRequestError("archive member is required"))
		return
	}
	if dr.archive == nil {
		_ = c.Error(mapArchiveError(archivepreview.ErrUnsupportedFormat))
		return
	}
	d := c.MustGet("drive").(types.IDrive)
	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	opened, e := dr.archive.Open(c.Request.Context(), entry, member)
	if e != nil {
		_ = c.Error(mapArchiveError(e))
		return
	}
	defer func() { _ = opened.Reader.Close() }()

	contentType := opened.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if opened.Size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(opened.Size, 10))
	}
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": opened.Name,
	}))
	// Archive members are untrusted files. A sandboxed iframe can preview HTML
	// without giving it the application's origin or script access.
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("X-Content-Type-Options", "nosniff")
	if _, e := io.Copy(c.Writer, opened.Reader); e != nil {
		_ = c.Error(e)
	}
}

func mapArchiveError(e error) error {
	switch {
	case errors.Is(e, archivepreview.ErrArchiveTooLarge):
		return err.NewNotAllowedMessageError("archive exceeds the configured size limit")
	case errors.Is(e, archivepreview.ErrMemberTooLarge):
		return err.NewNotAllowedMessageError("archive member exceeds the configured size limit")
	case errors.Is(e, archivepreview.ErrTooManyEntries):
		return err.NewNotAllowedMessageError("archive contains too many entries")
	case errors.Is(e, archivepreview.ErrUnsupportedFormat):
		return err.NewBadRequestError("unsupported archive format")
	case errors.Is(e, archivepreview.ErrInvalidArchive):
		return err.NewBadRequestError("invalid archive")
	default:
		return e
	}
}
