package server

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (dr *driveRoute) artifactRequest(c *gin.Context) (artifact.Request, error) {
	path, err := getQueryPath(c, "path")
	if err != nil {
		return artifact.Request{}, err
	}
	drive := c.MustGet("drive").(types.IDrive)
	entry, err := drive.Get(c.Request.Context(), path)
	if err != nil {
		return artifact.Request{}, err
	}
	entry = dr.wrapEntryWithAccessKey(entry, c.Query(common.SignatureQueryKey))
	return artifact.Request{
		Source:  entry,
		Handler: c.Param("handler"),
		Args:    c.Query("args"),
	}, nil
}

// getArtifact streams artifact bytes. Missing outputs are generated and the
// caller waits up to SyncWait. A timeout keeps generation running and returns
// 404 so <img> and download navigations never receive a task JSON body.
func (dr *driveRoute) getArtifact(c *gin.Context) {
	request, err := dr.artifactRequest(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	result, err := dr.artifacts.Fetch(c.Request.Context(), request, 30*time.Second)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		_ = c.Error(err)
		return
	}
	if result.Artifact != nil {
		defer func() { _ = result.Artifact.Body.Close() }()
		streamArtifact(c, result.Artifact)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNotFound)
}

// postArtifact starts or reuses generation. A ready artifact returns Info;
// otherwise the task is returned after at most AsyncWait.
func (dr *driveRoute) postArtifact(c *gin.Context) {
	request, err := dr.artifactRequest(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	result, err := dr.artifacts.Prepare(c.Request.Context(), request, 2*time.Second)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		_ = c.Error(err)
		return
	}
	if result.Task == nil {
		c.JSON(http.StatusOK, result.Info)
		return
	}
	c.Header("Location", fmt.Sprintf("%s/tasks/%s", dr.config.APIPath, result.Task.ID))
	c.JSON(http.StatusAccepted, result.Task)
}

func streamArtifact(c *gin.Context, file *artifact.Artifact) {
	contentType := file.Meta.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if file.Size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	}
	driveutil.SetContentDisposition(c.Writer.Header(), file.Meta.Name)
	_, _ = io.Copy(c.Writer, file.Body)
}
