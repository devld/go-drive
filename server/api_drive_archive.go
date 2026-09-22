package server

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/artifact/archive"
	"time"

	"github.com/gin-gonic/gin"
)

type archiveExtractRequest struct {
	Source   string   `json:"source"`
	Dest     string   `json:"dest"`
	Members  []string `json:"members"`
	Override bool     `json:"override"`
}

func (dr *driveRoute) archiveExtract(c *gin.Context) {
	var request archiveExtractRequest
	if e := c.ShouldBindJSON(&request); e != nil {
		_ = c.Error(err.NewBadRequestError(e.Error()))
		return
	}
	if request.Source == "" || len(request.Members) == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}

	drive_ := c.MustGet("drive").(types.IDrive)
	sourcePath := utils.CleanPath(request.Source)
	destPath := utils.CleanPath(request.Dest)

	source, e := drive_.Get(c.Request.Context(), sourcePath)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if !source.Type().IsFile() {
		_ = c.Error(err.NewBadRequestError(i18n.T("api.archive.invalid")))
		return
	}

	dest, e := drive_.Get(c.Request.Context(), destPath)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if !dest.Type().IsDir() {
		_ = c.Error(err.NewBadRequestError(i18n.T("api.archive.dest_not_dir")))
		return
	}
	if !dest.Meta().Writable {
		_ = c.Error(err.NewPermissionDeniedNotFoundError(i18n.T("error.permission_denied")))
		return
	}

	handler, e := dr.artifacts.Handler("archive")
	if e != nil {
		_ = c.Error(e)
		return
	}
	extractor, ok := handler.(archive.MemberExtractor)
	if !ok {
		_ = c.Error(err.NewNotFoundMessageError("unknown artifact handler"))
		return
	}

	t, e := dr.runner.ExecuteAndWait(c.Request.Context(), func(ctx types.TaskCtx) (any, error) {
		return nil, extractor.ExtractMembers(ctx, drive_, source, destPath, request.Members, request.Override)
	}, 2*time.Second, task.WithNameGroup(sourcePath+" -> "+destPath, "drive/archive-extract"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}
