package server

import (
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	apierr "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
	"net/http"
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
	args, err := artifactArgs(c)
	if err != nil {
		return artifact.Request{}, err
	}
	return artifact.Request{
		Source:  entry,
		Handler: c.Param("handler"),
		Args:    args,
	}, nil
}

const maxArtifactArgsBytes = 1 << 20

// artifactArgs reads a POST body as the args document. Handlers parse that
// string, including an empty body. LimitReader enforces the size. GET uses
// the args query parameter.
func artifactArgs(c *gin.Context) (string, error) {
	if c.Request.Method != http.MethodPost {
		return c.Query("args"), nil
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxArtifactArgsBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxArtifactArgsBytes {
		return "", apierr.NewBadRequestError("")
	}
	return string(body), nil
}

// getArtifact streams artifact bytes. ref only opens a cached artifact.
// args may generate one and the caller waits up to SyncWait. A timeout keeps
// generation running and returns 404 so <img> and download navigations never
// receive a task JSON body.
func (dr *driveRoute) getArtifact(c *gin.Context) {
	request, err := dr.artifactRequest(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if ref := c.Query("ref"); ref != "" {
		file, openErr := dr.artifacts.OpenCached(request.Handler, request.Source, ref)
		if openErr != nil {
			if apierr.IsNotFoundError(openErr) {
				c.Header("Cache-Control", "no-store")
				c.Status(http.StatusNotFound)
				return
			}
			_ = c.Error(openErr)
			return
		}
		defer func() { _ = file.Body.Close() }()
		streamArtifact(c, file)
		return
	}
	result, err := dr.artifacts.Fetch(c.Request.Context(), request, 30*time.Second)
	if err != nil {
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
	if file.Meta.MimeType != "" {
		c.Header("Content-Type", file.Meta.MimeType)
	}
	driveutil.SetContentDisposition(c.Writer.Header(), file.Meta.Name)
	http.ServeContent(c.Writer, c.Request, file.Meta.Name, file.Meta.ModTime, file.Body)
}
