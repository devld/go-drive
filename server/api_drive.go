package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func InitDriveRoutes(router gin.IRouter,
	config common.Config,
	rootDrive *drive.RootDrive,
	permissionDAO *storage.PathPermissionDAO,
	thumbnail *Thumbnail,
	signer *utils.Signer,
	chunkUploader *ChunkUploader,
	runner task.Runner,
	tokenStore types.TokenStore) {

	dr := driveRoute{
		config:        config,
		rootDrive:     rootDrive,
		permissionDAO: permissionDAO,
		chunkUploader: chunkUploader,
		thumbnail:     thumbnail,
		runner:        runner,
		signer:        signer,
	}

	// get file content
	router.HEAD("/content/*path", dr.getContent)
	router.GET("/content/*path", dr.getContent)
	router.GET("/thumbnail/*path", dr.getThumbnail)

	r := router.Group("/", Auth(tokenStore))

	// list entries/drives
	r.GET("/entries/*path", dr.list)
	// get entry info
	r.GET("/entry/*path", dr.get)
	// mkdir
	r.POST("/mkdir/*path", dr.makeDir)
	// copy file
	r.POST("/copy", dr.copyEntry)
	// move file
	r.POST("/move", dr.move)
	// deleteEntry entry
	r.DELETE("/entry/*path", dr.deleteEntry)
	// get upload config
	r.POST("/upload/*path", dr.upload)
	// write file
	r.PUT("/content/*path", dr.writeContent)
	// chunk upload request
	r.POST("/chunk", dr.chunkUploadRequest)
	// chunk upload
	r.PUT("/chunk/:id/:seq", dr.chunkUpload)
	// chunk upload complete
	r.POST("/chunk-content/*path", dr.chunkUploadComplete)
	// delete chunk upload
	r.DELETE("/chunk/:id", dr.deleteChunkUpload)
	// get task
	r.GET("/task/:id", func(c *gin.Context) {
		t, e := dr.runner.GetTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = err.NewNotFoundMessageError(e.Error())
		}
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, t)
	})

	// cancel and delete task
	r.DELETE("/task/:id", func(c *gin.Context) {
		_, e := dr.runner.StopTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = err.NewNotFoundMessageError(e.Error())
		}
		if e != nil {
			_ = c.Error(e)
		}
	})
}

type driveRoute struct {
	config        common.Config
	rootDrive     *drive.RootDrive
	permissionDAO *storage.PathPermissionDAO
	chunkUploader *ChunkUploader
	thumbnail     *Thumbnail
	runner        task.Runner
	signer        *utils.Signer
}

func (dr *driveRoute) getDrive(c *gin.Context) types.IDrive {
	session := GetSession(c)
	return NewPermissionWrapperDrive(
		c.Request, session,
		dr.rootDrive.Get(),
		dr.permissionDAO,
		dr.signer,
	)
}

func (dr *driveRoute) list(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	entries, e := dr.getDrive(c).List(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	res := make([]entryJson, 0, len(entries))
	for _, v := range entries {
		res = append(res, *newEntryJson(v))
	}
	SetResult(c, res)
}

func (dr *driveRoute) get(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	entry, e := dr.getDrive(c).Get(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, newEntryJson(entry))
}

func (dr *driveRoute) makeDir(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	entry, e := dr.getDrive(c).MakeDir(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, newEntryJson(entry))
}

func (dr *driveRoute) copyEntry(c *gin.Context) {
	drive_ := dr.getDrive(c)
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	override := c.Query("override")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Copy(fromEntry, to, override != "", ctx)
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second)

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) move(c *gin.Context) {
	drive_ := dr.getDrive(c)
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	override := c.Query("override")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Move(fromEntry, to, override != "", ctx)
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second)

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func checkCopyOrMove(from, to string) error {
	if from == to {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_same_path_not_allowed"))
	}
	if strings.HasPrefix(to, from) && utils.PathDepth(from) != utils.PathDepth(to) {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	}
	return nil
}

func (dr *driveRoute) deleteEntry(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		return nil, dr.getDrive(c).Delete(path, ctx)
	}, 2*time.Second)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) upload(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := utils.ToInt64(c.Query("size"), -1)
	request := make(types.SM, 0)
	if e := c.Bind(&request); e != nil {
		_ = c.Error(e)
		return
	}
	config, e := dr.getDrive(c).Upload(path, size, override != "", request)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if config != nil {
		SetResult(c, uploadConfig{config.Provider, config.Config})
	}
}

func (dr *driveRoute) getContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	file, e := dr.getDrive(c).Get(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if content, ok := file.(types.IContent); ok {
		useProxy := c.Query("proxy")
		if dr.config.ProxyMaxSize > 0 && file.Size() > dr.config.ProxyMaxSize {
			useProxy = ""
		}
		if e := drive_util.DownloadIContent(content, c.Writer, c.Request, useProxy != ""); e != nil {
			_ = c.Error(e)
			return
		}
		return
	}
	_ = c.Error(err.NewNotAllowedError())
}

func (dr *driveRoute) getThumbnail(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	if !checkSignature(dr.signer, c.Request, path) {
		_ = c.Error(err.NewNotFoundError())
		return
	}
	entry, e := dr.getDrive(c).Get(path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if entry.Meta().Props != nil && entry.Meta().Thumbnail != "" {
		c.Redirect(http.StatusFound, entry.Meta().Thumbnail)
		return
	}
	file, e := dr.thumbnail.Create(entry)
	if e != nil {
		_ = c.Error(e)
		return
	}
	defer func() { _ = file.Close() }()
	stat, e := file.Stat()
	if e != nil {
		_ = c.Error(e)
		return
	}
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dr.config.ThumbnailCacheTTl.Seconds())))
	http.ServeContent(c.Writer, c.Request, "thumbnail.jpg", stat.ModTime(), file)
}

func (dr *driveRoute) writeContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := utils.ToInt64(c.GetHeader("Content-Length"), -1)
	defer func() { _ = c.Request.Body.Close() }()
	file, e := drive_util.CopyReaderToTempFile(c.Request.Body, task.DummyContext(), dr.config.TempDir)
	if e != nil {
		_ = c.Error(e)
		return
	}
	stat, e := file.Stat()
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(e)
		return
	}
	if size != stat.Size() {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_file_size")))
		return
	}
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		defer func() {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}()
		return dr.getDrive(c).Save(path, size, override != "", file, ctx)
	}, 2*time.Second)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) chunkUploadRequest(c *gin.Context) {
	size := utils.ToInt64(c.Query("size"), -1)
	chunkSize := utils.ToInt64(c.Query("chunk_size"), -1)
	if size <= 0 || chunkSize <= 0 {
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_size_or_chunk_size")))
		return
	}
	upload, e := dr.chunkUploader.CreateUpload(size, chunkSize)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, upload)
}

func (dr *driveRoute) chunkUpload(c *gin.Context) {
	id := c.Param("id")
	seq, e := strconv.Atoi(c.Param("seq"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	if e := dr.chunkUploader.ChunkUpload(id, seq, c.Request.Body); e != nil {
		_ = c.Error(e)
	}
}

func (dr *driveRoute) chunkUploadComplete(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	id := c.Query("id")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		file, e := dr.chunkUploader.CompleteUpload(id, ctx)
		if e != nil {
			return nil, e
		}
		stat, e := file.Stat()
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		ctx.Progress(0, true)
		entry, e := dr.getDrive(c).Save(path, stat.Size(), true, file, ctx)
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		_ = file.Close()
		e = dr.chunkUploader.DeleteUpload(id)
		return newEntryJson(entry), nil
	}, 2*time.Second)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) deleteChunkUpload(c *gin.Context) {
	id := c.Param("id")
	if e := dr.chunkUploader.DeleteUpload(id); e != nil {
		_ = c.Error(e)
	}
}

type entryJson struct {
	Path    string          `json:"path"`
	Name    string          `json:"name"`
	Type    types.EntryType `json:"type"`
	Size    int64           `json:"size"`
	Meta    types.M         `json:"meta"`
	ModTime int64           `json:"mod_time"`
}

func newEntryJson(e types.IEntry) *entryJson {
	entryMeta := e.Meta()
	meta := utils.CopyMap(entryMeta.Props)
	meta["can_write"] = entryMeta.CanWrite
	if entryMeta.Thumbnail != "" {
		meta["thumbnail"] = entryMeta.Thumbnail
	}
	return &entryJson{
		Path:    e.Path(),
		Name:    utils.PathBase(e.Path()),
		Type:    e.Type(),
		Size:    e.Size(),
		Meta:    meta,
		ModTime: e.ModTime(),
	}
}

type uploadConfig struct {
	Provider string      `json:"provider"`
	Config   interface{} `json:"config"`
}
